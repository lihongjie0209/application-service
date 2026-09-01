package bootstrapcli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/lihongjie0209/application-service/internal/bootstrap"
	"github.com/lihongjie0209/application-service/internal/buildinfo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type commandConfig struct {
	BaseURL       string        `mapstructure:"base_url"`
	Authorization string        `mapstructure:"authorization"`
	Manifest      string        `mapstructure:"manifest"`
	TenantIDs     []string      `mapstructure:"tenant_ids"`
	Timeout       time.Duration `mapstructure:"timeout"`
	Output        string        `mapstructure:"output"`
}

func NewRootCommand(stdout, stderr io.Writer) *cobra.Command {
	v := viper.New()
	var configFile string
	root := &cobra.Command{
		Use:           "platform-bootstrap",
		Short:         "Reconcile platform applications, menus, and tenant grants",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return loadConfig(v, configFile)
		},
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.PersistentFlags().StringVar(&configFile, "config", "", "optional YAML configuration file")
	root.PersistentFlags().String("base-url", "", "application-service base URL")
	root.PersistentFlags().String("authorization", "", "Authorization header, for example 'Bearer ...' or 'PSK ...'")
	root.PersistentFlags().String("manifest", "bootstrap/platform-applications.yaml", "platform application manifest")
	root.PersistentFlags().StringSlice("tenant-id", nil, "tenant ID to grant every manifest application; repeatable")
	root.PersistentFlags().Duration("timeout", 15*time.Second, "per-request timeout")
	root.PersistentFlags().String("output", "plain", "output format: plain or json")
	for key, name := range map[string]string{"base_url": "base-url", "authorization": "authorization", "manifest": "manifest", "tenant_ids": "tenant-id", "timeout": "timeout", "output": "output"} {
		if err := v.BindPFlag(key, root.PersistentFlags().Lookup(name)); err != nil {
			panic(err)
		}
	}
	root.AddCommand(newApplyCommand(v), newVersionCommand(), newCompletionCommand(root))
	return root
}

func newApplyCommand(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "apply",
		Short: "Idempotently reconcile the manifest",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var cfg commandConfig
			if err := v.Unmarshal(&cfg); err != nil {
				return fmt.Errorf("decode configuration: %w", err)
			}
			if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Authorization) == "" {
				return errors.New("base-url and authorization are required")
			}
			if cfg.Output != "plain" && cfg.Output != "json" {
				return errors.New("output must be plain or json")
			}
			manifest, err := bootstrap.LoadManifest(cfg.Manifest)
			if err != nil {
				return err
			}
			client, err := bootstrap.NewHTTPClient(cfg.BaseURL, cfg.Authorization, cfg.Timeout)
			if err != nil {
				return err
			}
			result, err := bootstrap.NewReconciler(client).Apply(cmd.Context(), manifest, cfg.TenantIDs)
			if err != nil {
				return err
			}
			if cfg.Output == "json" {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "applications: %d created, %d updated; menus: %d created, %d updated, %d published; grants: %d applied\n", result.ApplicationsCreated, result.ApplicationsUpdated, result.MenusCreated, result.MenusUpdated, result.MenusPublished, result.GrantsApplied)
			return err
		},
	}
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "version=%s commit=%s build_time=%s\n", buildinfo.Version, buildinfo.Commit, buildinfo.BuildTime)
			return err
		},
	}
}

func newCompletionCommand(root *cobra.Command) *cobra.Command {
	command := &cobra.Command{Use: "completion [bash|zsh|fish|powershell]", Short: "Generate shell completion", Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs)}
	command.ValidArgs = []string{"bash", "zsh", "fish", "powershell"}
	command.RunE = func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return root.GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return root.GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return root.GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return root.GenPowerShellCompletion(cmd.OutOrStdout())
		default:
			return errors.New("unsupported shell")
		}
	}
	return command
}

func loadConfig(v *viper.Viper, configFile string) error {
	v.SetEnvPrefix("PLATFORM_BOOTSTRAP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName(".platform-bootstrap")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
	}
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return fmt.Errorf("read configuration: %w", err)
		}
	}
	return nil
}

func Execute(ctx context.Context, stdout, stderr io.Writer, args []string) error {
	command := NewRootCommand(stdout, stderr)
	command.SetArgs(args)
	return command.ExecuteContext(ctx)
}
