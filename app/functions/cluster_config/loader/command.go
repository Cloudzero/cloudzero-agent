package loader

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudzero/cloudzero-agent/app/domain/k8s"
	"github.com/cloudzero/cloudzero-agent/app/types/cluster_config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	cfg_gator "github.com/cloudzero/cloudzero-agent/app/config/gator"
	cfg_validator "github.com/cloudzero/cloudzero-agent/app/config/validator"
	cfg_webhook "github.com/cloudzero/cloudzero-agent/app/config/webhook"
)

const (
	FLAG_ACCOUNT           = "account"
	FLAG_REGION            = "region"
	FLAG_CLUSTER_NAME      = "cluster-name"
	FLAG_DEPLOY_NAME       = "deploy-name"
	FLAG_CHART_VERSION     = "chart-version"
	FLAG_AGENT_VERSION     = "agent-version"
	FLAG_VALUES_FILE       = "values-file"
	FLAG_CONFIG_VALIDATOR  = "config-validator"
	FLAG_CONFIG_WEBHOOK    = "config-webhook"
	FLAG_CONFIG_AGGREGATOR = "config-aggregator"
)

func NewCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "load",
		Usage:   "load the configuration",
		Aliases: []string{"l"},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: FLAG_ACCOUNT, Usage: "account name", Required: true},
			&cli.StringFlag{Name: FLAG_REGION, Usage: "region", Required: true},
			&cli.StringFlag{Name: FLAG_CLUSTER_NAME, Usage: "cluster name", Required: true},
			&cli.StringFlag{Name: FLAG_DEPLOY_NAME, Usage: "deploy name", Required: true},
			&cli.StringFlag{Name: FLAG_CHART_VERSION, Usage: "current chart version", Required: true},
			&cli.StringFlag{Name: FLAG_AGENT_VERSION, Usage: "current agent version", Required: true},
			&cli.StringFlag{Name: FLAG_VALUES_FILE, Usage: "rendered values file", Required: true},
			&cli.StringFlag{Name: FLAG_CONFIG_VALIDATOR, Usage: "list of validator config files", Required: true},
			&cli.StringFlag{Name: FLAG_CONFIG_WEBHOOK, Usage: "list of webhook config files", Required: true},
			&cli.StringFlag{Name: FLAG_CONFIG_AGGREGATOR, Usage: "list of aggregator config files", Required: true},
		},
		Action: func(c *cli.Context) error {
			return run(c)
		},
	}
	return cmd
}

func run(c *cli.Context) error {
	// create an errors array
	errs := make([]string, 0)

	// get the namespace
	ns, err := k8s.GetNamespace()
	if err != nil {
		errs = append(errs, err.Error())
	}

	// get the provider id
	providerId, err := k8s.GetProviderID(c.Context)
	if err != nil {
		errs = append(errs, err.Error())
	}

	// get the k8s version
	k8sVersion, err := k8s.GetVersion()
	if err != nil {
		errs = append(errs, err.Error())
	}

	// parse the validator config
	settingsValidatorB64 := ""
	settingsValidator, err := cfg_validator.NewSettings(c.StringSlice(FLAG_CONFIG_VALIDATOR)...)
	if err != nil {
		errs = append(errs, err.Error())
	} else {
		enc, err2 := settingsValidator.ToBytes()
		if err2 != nil {
			errs = append(errs, fmt.Errorf("failed to encode the settings: %w", err2).Error())
		} else {
			settingsValidatorB64 = base64.StdEncoding.EncodeToString(enc)
		}
	}

	// parse the validator config
	settingsWebhookB64 := ""
	settingsWebhook, err := cfg_webhook.NewSettings(c.StringSlice(FLAG_CONFIG_WEBHOOK)...)
	if err != nil {
		errs = append(errs, err.Error())
	} else {
		enc, err2 := settingsWebhook.ToBytes()
		if err2 != nil {
			errs = append(errs, fmt.Errorf("failed to encode the settings: %w", err2).Error())
		} else {
			settingsWebhookB64 = base64.StdEncoding.EncodeToString(enc)
		}
	}

	// parse the validator config
	settingsAggregatorB64 := ""
	settingsAggregator, err := cfg_gator.NewSettings(c.StringSlice(FLAG_CONFIG_AGGREGATOR)...)
	if err != nil {
		errs = append(errs, err.Error())
	} else {
		enc, err2 := settingsAggregator.ToBytes()
		if err2 != nil {
			errs = append(errs, fmt.Errorf("failed to encode the settings: %w", err2).Error())
		} else {
			settingsAggregatorB64 = base64.StdEncoding.EncodeToString(enc)
		}
	}

	// create a new cluster config object
	cfg := cluster_config.ClusterConfig{
		Account:                   c.String(FLAG_ACCOUNT),
		Region:                    c.String(FLAG_REGION),
		Namespace:                 ns,
		ProviderId:                providerId,
		ClusterName:               c.String(FLAG_CLUSTER_NAME),
		K8SVersion:                k8sVersion,
		DeploymentName:            c.String(FLAG_DEPLOY_NAME),
		ChartVersion:              c.String(FLAG_CHART_VERSION),
		AgentVersion:              c.String(FLAG_AGENT_VERSION),
		ConfigValuesBase64:        base64.StdEncoding.EncodeToString([]byte(c.String(FLAG_VALUES_FILE))),
		ConfigValidatorBase64:     settingsValidatorB64,
		ConfigWebhookServerBase64: settingsWebhookB64,
		ConfigAggregatorBase64:    settingsAggregatorB64,
		Errors:                    errs,
	}

	if log.Ctx(c.Context).GetLevel() <= zerolog.DebugLevel {
		enc, err := json.MarshalIndent(&cfg, "", "  ")
		if err == nil {
			log.Ctx(c.Context).Debug().Msg("Rendered ClusterConfig:")
			fmt.Println(string(enc))
		}
	}

	if len(errs) > 0 {
		for _, err := range errs {
			log.Ctx(c.Context).Error().Msg(err)
		}

		return errors.New(strings.Join(errs, ","))
	}

	return nil
}
