package grpctransport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/lihongjie0209/application-service/internal/apperror"
	applicationdomain "github.com/lihongjie0209/application-service/internal/application"
	"github.com/lihongjie0209/application-service/internal/auth"
	"github.com/lihongjie0209/application-service/internal/buildinfo"
	"github.com/lihongjie0209/application-service/internal/config"
	"github.com/lihongjie0209/application-service/internal/environment"
	apphealth "github.com/lihongjie0209/application-service/internal/health"
	"github.com/lihongjie0209/application-service/internal/idempotency"
	"github.com/lihongjie0209/application-service/internal/observability"
	"github.com/lihongjie0209/application-service/internal/requestid"
	appPolicy "github.com/lihongjie0209/application-service/internal/routepolicy"
	platformauthz "github.com/lihongjie0209/microservice-platform-go/authz"
	platformidempotency "github.com/lihongjie0209/microservice-platform-go/idempotency"
	"github.com/lihongjie0209/microservice-platform-go/principal"
	platformpolicy "github.com/lihongjie0209/microservice-platform-go/routepolicy"

	applicationv1 "github.com/lihongjie0209/platform-protos/gen/go/platform/application/v1"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type Server struct {
	server  *grpc.Server
	address string
	logger  *slog.Logger
}

func NewServer(lc fx.Lifecycle, cfg config.Config, authService *auth.Service, authorizer platformauthz.Authorizer, policies *appPolicy.Manager, policyRepository *appPolicy.Repository, healthService *apphealth.Service, applicationService *applicationdomain.Service, idempotencyManager *idempotency.Manager, metrics *observability.Metrics, logger *slog.Logger) (*Server, error) {
	options := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(cfg.GRPC.MaxReceiveBytes),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(environmentInterceptor(cfg.Runtime.ActiveProfile), requestIDInterceptor, idempotencyInterceptor, recoveryInterceptor(logger), optionalAuthInterceptor(authService, cfg), databaseAuthorizationInterceptor(cfg.Authorization.Enabled, cfg.App.Name, policies, authorizer), platformidempotency.UnaryServerInterceptor(idempotencyManager, cfg.Idempotency.GRPCMethods, logger), errorMappingInterceptor, metricsInterceptor(metrics, logger)),
		grpc.ChainStreamInterceptor(environmentStreamInterceptor(cfg.Runtime.ActiveProfile), requestIDStreamInterceptor, idempotencyStreamInterceptor, recoveryStreamInterceptor(logger), optionalAuthStreamInterceptor(authService, cfg), databaseAuthorizationStreamInterceptor(cfg.Authorization.Enabled, cfg.App.Name, policies, authorizer), metricsStreamInterceptor(metrics, logger)),
	}
	if cfg.GRPC.TLS.Enabled {
		creds, err := serverCredentials(cfg.GRPC.TLS)
		if err != nil {
			return nil, err
		}
		options = append(options, grpc.Creds(creds))
	}
	grpcServer := grpc.NewServer(options...)
	applicationv1.RegisterApplicationServiceServer(grpcServer, &applicationServer{service: applicationService})
	grpc_health_v1.RegisterHealthServer(grpcServer, &healthServer{health: healthService})
	if cfg.GRPC.ReflectionEnabled {
		reflection.Register(grpcServer)
	}
	server := &Server{server: grpcServer, address: cfg.GRPC.Address, logger: logger}
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		if cfg.GRPC.Enabled && cfg.Authorization.Enabled {
			routes, err := discoveredGRPCRoutes(grpcServer, cfg.App.Name)
			if err != nil {
				return err
			}
			if err := policyRepository.SyncRoutes(ctx, routes, cfg.App.Name+":route-discovery"); err != nil {
				return fmt.Errorf("sync gRPC routes: %w", err)
			}
			if err := policies.RefreshSource(ctx, "startup-grpc"); err != nil {
				return fmt.Errorf("load gRPC route policies: %w", err)
			}
			if err := policies.ValidateRoutes(ctx, cfg.App.Name); err != nil {
				logger.WarnContext(ctx, "route authorization policy coverage is incomplete; uncovered routes fail closed", "error", err)
			}
		}
		return server.start(cfg.GRPC.Enabled)(ctx)
	}, OnStop: server.stop})
	return server, nil
}

func discoveredGRPCRoutes(server *grpc.Server, serviceName string) ([]platformpolicy.Route, error) {
	routes := []platformpolicy.Route{}
	for service, info := range server.GetServiceInfo() {
		if service == grpc_health_v1.Health_ServiceDesc.ServiceName {
			continue
		}
		for _, method := range info.Methods {
			path := "/" + service + "/" + method.Name
			route, err := platformpolicy.NewRoute("grpc", "call", path, serviceName, buildinfo.Version)
			if err != nil {
				return nil, err
			}
			route.Operation = path
			routes = append(routes, route)
		}
	}
	return routes, nil
}

type routePolicyEvaluator interface {
	EvaluateRoute(context.Context, string, string, string, string, platformauthz.Authorizer) error
}

func databaseAuthorizationInterceptor(enabled bool, serviceName string, policies routePolicyEvaluator, authorizer platformauthz.Authorizer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if enabled {
			if err := policies.EvaluateRoute(ctx, "grpc", "call", info.FullMethod, serviceName, authorizer); err != nil {
				if errors.Is(err, platformpolicy.ErrDenied) {
					return nil, status.Error(codes.PermissionDenied, "permission denied")
				}
				return nil, status.Error(codes.Unavailable, "authorization decision is unavailable")
			}
			if platformAdministrationMethod(info.FullMethod) {
				ctx = applicationdomain.WithPlatformAdministration(ctx)
			}
		}
		return handler(ctx, request)
	}
}

func databaseAuthorizationStreamInterceptor(enabled bool, serviceName string, policies routePolicyEvaluator, authorizer platformauthz.Authorizer) grpc.StreamServerInterceptor {
	return func(server any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		if enabled {
			if err := policies.EvaluateRoute(ctx, "grpc", "call", info.FullMethod, serviceName, authorizer); err != nil {
				if errors.Is(err, platformpolicy.ErrDenied) {
					return status.Error(codes.PermissionDenied, "permission denied")
				}
				return status.Error(codes.Unavailable, "authorization decision is unavailable")
			}
			if platformAdministrationMethod(info.FullMethod) {
				ctx = applicationdomain.WithPlatformAdministration(ctx)
			}
		}
		return handler(server, &contextServerStream{ServerStream: stream, ctx: ctx})
	}
}
func platformAdministrationMethod(method string) bool {
	return method != applicationv1.ApplicationService_GetPublishedNavigation_FullMethodName && method != applicationv1.ApplicationService_ListTenantApplications_FullMethodName && method != applicationv1.ApplicationService_BatchCheckTenantApplications_FullMethodName
}

func errorMappingInterceptor(ctx context.Context, request any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	response, err := handler(ctx, request)
	if err == nil || status.Code(err) != codes.Unknown {
		return response, err
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		return nil, status.Error(codes.Internal, "internal server error")
	}
	code := codes.Internal
	switch appErr.Code {
	case apperror.CodeInvalidArgument:
		code = codes.InvalidArgument
	case apperror.CodeUnauthorized:
		code = codes.Unauthenticated
	case apperror.CodeForbidden:
		code = codes.PermissionDenied
	case apperror.CodeNotFound:
		code = codes.NotFound
	case apperror.CodeConflict:
		code = codes.Aborted
	case apperror.CodeDependencyUnavailable:
		code = codes.Unavailable
	}
	return nil, status.Error(code, appErr.Message)
}

func (s *Server) start(enabled bool) func(context.Context) error {
	return func(context.Context) error {
		if !enabled {
			s.logger.Warn("grpc server is disabled")
			return nil
		}
		listener, err := net.Listen("tcp", s.address)
		if err != nil {
			return fmt.Errorf("listen grpc: %w", err)
		}
		go func() {
			if err := s.server.Serve(listener); err != nil {
				s.logger.Error("grpc server stopped unexpectedly", "error", err)
			}
		}()
		s.logger.Info("grpc server started", "address", s.address)
		return nil
	}
}
func (s *Server) stop(ctx context.Context) error {
	stopped := make(chan struct{})
	go func() { s.server.GracefulStop(); close(stopped) }()
	select {
	case <-stopped:
		return nil
	case <-ctx.Done():
		s.server.Stop()
		return ctx.Err()
	}
}

type healthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	health *apphealth.Service
}

func (s *healthServer) Check(ctx context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	_, ready := s.health.Ready(ctx)
	serving := grpc_health_v1.HealthCheckResponse_NOT_SERVING
	if ready {
		serving = grpc_health_v1.HealthCheckResponse_SERVING
	}
	return &grpc_health_v1.HealthCheckResponse{Status: serving}, nil
}
func (s *healthServer) List(context.Context, *grpc_health_v1.HealthListRequest) (*grpc_health_v1.HealthListResponse, error) {
	return &grpc_health_v1.HealthListResponse{Statuses: map[string]*grpc_health_v1.HealthCheckResponse{"": {Status: grpc_health_v1.HealthCheckResponse_SERVING}}}, nil
}

func requestIDInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	id := ""
	if values := metadata.ValueFromIncomingContext(ctx, "x-request-id"); len(values) > 0 && requestid.Valid(values[0]) {
		id = values[0]
	}
	if id == "" {
		id = requestid.Generate()
	}
	header := metadata.Pairs("x-request-id", id)
	_ = grpc.SetHeader(ctx, header)
	return handler(requestid.WithContext(ctx, id), req)
}
func idempotencyInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	values := metadata.ValueFromIncomingContext(ctx, "idempotency-key")
	if len(values) == 0 {
		return handler(ctx, req)
	}
	if !idempotency.Valid(values[0]) {
		return nil, status.Error(codes.InvalidArgument, "invalid idempotency-key")
	}
	return handler(idempotency.WithContext(ctx, values[0]), req)
}
func environmentInterceptor(profile string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(environment.WithContext(ctx, profile), req)
	}
}
func optionalAuthInterceptor(service *auth.Service, cfg config.Config) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		authCtx, err := authenticateGRPCOptional(ctx, service, cfg)
		if err != nil {
			return nil, err
		}
		return handler(authCtx, req)
	}
}

func authenticateGRPCOptional(ctx context.Context, service *auth.Service, cfg config.Config) (context.Context, error) {
	values := metadata.ValueFromIncomingContext(ctx, "authorization")
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return ctx, nil
	}
	header := strings.TrimSpace(values[0])
	scheme, raw, ok := strings.Cut(header, " ")
	if !ok || raw == "" {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization credential")
	}
	var identity principal.Principal
	switch {
	case strings.EqualFold(scheme, "Bearer"):
		verified, err := service.Verify(ctx, raw)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}
		identity = verified
	case strings.EqualFold(scheme, "PSK"):
		if !cfg.Auth.PSK.Enabled || !auth.VerifyPSK(header, cfg.Auth.PSK.Key) {
			return nil, status.Error(codes.Unauthenticated, "invalid PSK")
		}
		identity = principal.Principal{ID: cfg.App.Name + ":psk", Type: principal.TypeServiceAccount}
	default:
		return nil, status.Error(codes.Unauthenticated, "unsupported authorization scheme")
	}
	authenticated := principal.WithContext(ctx, identity)
	return platformauthz.WithCallerCredential(authenticated, header), nil
}

type contextServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *contextServerStream) Context() context.Context { return s.ctx }

func environmentStreamInterceptor(profile string) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &contextServerStream{ServerStream: stream, ctx: environment.WithContext(stream.Context(), profile)})
	}
}

func requestIDStreamInterceptor(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := stream.Context()
	id := ""
	if values := metadata.ValueFromIncomingContext(ctx, "x-request-id"); len(values) > 0 && requestid.Valid(values[0]) {
		id = values[0]
	}
	if id == "" {
		id = requestid.Generate()
	}
	if err := stream.SetHeader(metadata.Pairs("x-request-id", id)); err != nil {
		return status.Error(codes.Internal, "set request metadata")
	}
	return handler(srv, &contextServerStream{ServerStream: stream, ctx: requestid.WithContext(ctx, id)})
}

func idempotencyStreamInterceptor(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	values := metadata.ValueFromIncomingContext(stream.Context(), "idempotency-key")
	if len(values) == 0 {
		return handler(srv, stream)
	}
	if !idempotency.Valid(values[0]) {
		return status.Error(codes.InvalidArgument, "invalid idempotency-key")
	}
	return handler(srv, &contextServerStream{ServerStream: stream, ctx: idempotency.WithContext(stream.Context(), values[0])})
}

func optionalAuthStreamInterceptor(service *auth.Service, cfg config.Config) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := authenticateGRPCOptional(stream.Context(), service, cfg)
		if err != nil {
			return err
		}
		return handler(srv, &contextServerStream{ServerStream: stream, ctx: ctx})
	}
}

func recoveryStreamInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(stream.Context(), "grpc stream panic recovered", "method", info.FullMethod, "panic", recovered)
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(srv, stream)
	}
}

func metricsStreamInterceptor(metrics *observability.Metrics, logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		started := time.Now()
		err := handler(srv, stream)
		code := status.Code(err)
		if metrics.Enabled() {
			metrics.GRPCRequests.WithLabelValues(info.FullMethod, code.String()).Inc()
			metrics.GRPCDuration.WithLabelValues(info.FullMethod).Observe(time.Since(started).Seconds())
		}
		requestID, _ := requestid.FromContext(stream.Context())
		logger.InfoContext(stream.Context(), "grpc stream", "request_id", requestID, "method", info.FullMethod, "code", code.String(), "duration", time.Since(started))
		return err
	}
}

func recoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(ctx, "grpc panic recovered", "method", info.FullMethod, "panic", recovered)
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}
func metricsInterceptor(metrics *observability.Metrics, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		started := time.Now()
		response, err := handler(ctx, req)
		code := status.Code(err)
		if metrics.Enabled() {
			metrics.GRPCRequests.WithLabelValues(info.FullMethod, code.String()).Inc()
			metrics.GRPCDuration.WithLabelValues(info.FullMethod).Observe(time.Since(started).Seconds())
		}
		span := trace.SpanFromContext(ctx).SpanContext()
		requestID, _ := requestid.FromContext(ctx)
		logger.InfoContext(ctx, "grpc request", "request_id", requestID, "trace_id", span.TraceID().String(), "span_id", span.SpanID().String(), "method", info.FullMethod, "code", code.String(), "duration", time.Since(started))
		return response, err
	}
}

func serverCredentials(cfg config.GRPCTLS) (credentials.TransportCredentials, error) {
	certificate, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load grpc certificate: %w", err)
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate}}
	if cfg.ClientCAFile != "" {
		pem, err := os.ReadFile(cfg.ClientCAFile)
		if err != nil {
			return nil, fmt.Errorf("read grpc client CA: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("parse grpc client CA")
		}
		tlsConfig.ClientCAs = pool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return credentials.NewTLS(tlsConfig), nil
}

var Module = fx.Module("grpc", fx.Provide(NewServer), fx.Invoke(func(*Server) {}))
