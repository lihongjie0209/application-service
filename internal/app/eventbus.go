package app

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lihongjie0209/application-service/internal/config"
	"github.com/lihongjie0209/microservice-platform-go/eventbus"
	"github.com/lihongjie0209/microservice-platform-go/operationlog"
	platformoutbox "github.com/lihongjie0209/microservice-platform-go/outbox"
	"github.com/lihongjie0209/microservice-platform-go/securitylog"
	commonv1 "github.com/lihongjie0209/platform-protos/gen/go/platform/common/v1"
	"go.uber.org/fx"
)

type applicationEventRuntime struct {
	config config.Config
	store  *platformoutbox.SQLStore
	logger *slog.Logger
	cancel context.CancelFunc
	wg     sync.WaitGroup
	bus    *eventbus.Bus
}

func newApplicationEventRuntime(lc fx.Lifecycle, cfg config.Config, store *platformoutbox.SQLStore, logger *slog.Logger) *applicationEventRuntime {
	r := &applicationEventRuntime{config: cfg, store: store, logger: logger}
	lc.Append(fx.Hook{OnStart: r.start, OnStop: r.stop})
	return r
}
func (r *applicationEventRuntime) start(ctx context.Context) error {
	if !r.config.EventBus.Enabled {
		return nil
	}
	if r.store == nil {
		return errors.New("enabled event bus requires database outbox")
	}
	bus, err := eventbus.New(ctx, eventbus.Config{URLs: r.config.EventBus.URLs, ClientName: r.config.App.Name, StreamName: r.config.EventBus.StreamName, Subjects: []string{"platform.>"}, Storage: r.config.EventBus.Storage, MaxAge: r.config.EventBus.MaxAge, DuplicateWindow: r.config.EventBus.DuplicateWindow, ConnectTimeout: r.config.EventBus.ConnectTimeout, PublishTimeout: r.config.EventBus.PublishTimeout})
	if err != nil {
		return err
	}
	dispatcher, err := platformoutbox.New(r.store, bus, platformoutbox.Config{BatchSize: r.config.EventBus.DispatchBatchSize, Lease: r.config.EventBus.DispatchLease, RetryDelay: r.config.EventBus.DispatchRetryDelay})
	if err != nil {
		_ = bus.Close()
		return err
	}
	r.bus = bus
	runCtx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	cleaner, err := platformoutbox.NewRetentionCleaner(r.store, platformoutbox.RetentionConfig{Retention: r.config.EventBus.PublishedRetention, BatchSize: r.config.EventBus.CleanupBatchSize})
	if err != nil {
		cancel()
		_ = bus.Close()
		return err
	}
	r.wg.Go(func() {
		ticker := time.NewTicker(r.config.EventBus.DispatchInterval)
		defer ticker.Stop()
		for {
			if _, err := dispatcher.RunOnce(runCtx); err != nil && !errors.Is(err, context.Canceled) {
				r.logger.ErrorContext(runCtx, "dispatch application outbox failed", "error", err)
			}
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
			}
		}
	})
	r.wg.Go(func() {
		ticker := time.NewTicker(r.config.EventBus.CleanupInterval)
		defer ticker.Stop()
		for {
			if deleted, runErr := cleaner.RunOnce(runCtx); runErr != nil && !errors.Is(runErr, context.Canceled) {
				r.logger.ErrorContext(runCtx, "clean published application outbox events", "error", runErr)
			} else if deleted > 0 {
				r.logger.InfoContext(runCtx, "published application outbox events cleaned", "deleted", deleted)
			}
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
			}
		}
	})
	return nil
}
func (r *applicationEventRuntime) stop(context.Context) error {
	if r.cancel != nil {
		r.cancel()
		r.wg.Wait()
	}
	if r.bus != nil {
		return r.bus.Close()
	}
	return nil
}
func (r *applicationEventRuntime) Publish(ctx context.Context, subject string, envelope *commonv1.EventEnvelope) error {
	if r == nil || r.bus == nil {
		return errors.New("application event bus is unavailable")
	}
	return r.bus.Publish(ctx, subject, envelope)
}
func newOperationLogRecorder(cfg config.Config, publisher *applicationEventRuntime) (operationlog.Recorder, error) {
	return operationlog.New(operationlog.Config{Enabled: cfg.OperationLog.Enabled, Subject: cfg.OperationLog.Subject, MaxPayloadBytes: cfg.OperationLog.MaxPayloadBytes}, publisher)
}
func newSecurityLogRecorder(cfg config.Config, publisher *applicationEventRuntime) (securitylog.Recorder, error) {
	return securitylog.New(securitylog.Config{Enabled: cfg.SecurityLog.Enabled, Subject: cfg.SecurityLog.Subject, MaxPayloadBytes: cfg.SecurityLog.MaxPayloadBytes, HashKey: cfg.SecurityLog.HashKey, FailClosed: cfg.SecurityLog.FailClosed}, publisher)
}
func newApplicationOutboxStore(db *sqlx.DB) (*platformoutbox.SQLStore, error) {
	if db == nil {
		return nil, nil
	}
	return platformoutbox.NewSQLStore(db, "application_outbox_events", platformoutbox.WithWorkerAuditActor("application-service:outbox"))
}

var EventBusModule = fx.Module("application-event-bus", fx.Provide(newApplicationOutboxStore, newApplicationEventRuntime, newOperationLogRecorder, newSecurityLogRecorder), fx.Invoke(func(*applicationEventRuntime) {}))
