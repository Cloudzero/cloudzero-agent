package config

import (
	"context"
	_ "embed"
	"os"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/build"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/k8s"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/util/gh"
)

//go:embed internal/scrape_config.tmpl
var scrapeConfigTemplate string

var (
	configAlias = []string{"f"}
)

type ScrapeConfigData struct {
	Targets        []string
	ClusterName    string
	CloudAccountID string
	Region         string
}

func NewCommand(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Name:  "config",
		Usage: "configuration utility commands",
		Subcommands: []*cli.Command{
			{
				Name:  "generate",
				Usage: "generates a generic config file",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: config.FlagAccountID, Aliases: []string{"a"}, Usage: config.FlagDescAccountID, Required: true},
					&cli.StringFlag{Name: config.FlagClusterName, Aliases: []string{"c"}, Usage: config.FlagDescClusterName, Required: true},
					&cli.StringFlag{Name: config.FlagRegion, Aliases: []string{"r"}, Usage: config.FlagDescRegion, Required: true},
					&cli.StringFlag{Name: "kubeconfig", Usage: "absolute path to the kubeconfig file", Required: false},
					&cli.StringFlag{Name: "namespace", Usage: "namespace of the cloudzero-agent pod", Required: true},
					&cli.StringFlag{Name: "configmap", Usage: "name of the ConfigMap", Required: true},
					&cli.StringFlag{Name: "pod", Usage: "name of the cloudzero-agent pod", Required: true},
				},
				Action: func(c *cli.Context) error {
					kubeconfigPath := c.String("kubeconfig")
					namespace := c.String("namespace")
					configMapName := c.String("configmap")

					clientset, err := k8s.BuildKubeClient(kubeconfigPath)
					if err != nil {
						return err
					}

					kubeStateMetricsURL, err := k8s.GetKubeStateMetricsURL(ctx, clientset, namespace)
					if err != nil {
						return err
					}

					targets := []string{kubeStateMetricsURL}
					scrapeConfigData := ScrapeConfigData{
						Targets:        targets,
						ClusterName:    c.String(config.FlagClusterName),
						CloudAccountID: c.String(config.FlagAccountID),
						Region:         c.String(config.FlagRegion),
					}

					configContent, err := Generate(scrapeConfigData)
					if err != nil {
						return err
					}

					configMapData := map[string]string{
						"prometheus.yml": configContent,
					}

					err = k8s.UpdateConfigMap(ctx, clientset, namespace, configMapName, configMapData)
					if err != nil {
						return err
					}

					return nil
				},
			},
			{
				Name:  "validate",
				Usage: "validates the config file",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name: config.FlagConfigFile, Aliases: configAlias,
						Usage: "input " + config.FlagDescConfFile, Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					configs := c.StringSlice(config.FlagConfigFile)
					if len(configs) == 0 {
						return errors.New("no configuration files specified")
					}

					cfg, err := config.NewSettings(configs...)
					if err != nil {
						return errors.Wrap(err, "config read")
					}
					err = cfg.Validate()
					if err != nil {
						return errors.Wrap(err, "config validation")
					}
					return nil
				},
			},
		},
	}
	return cmd
}

func Generate(data ScrapeConfigData) (string, error) {
	t, err := template.New("scrape_config").Parse(scrapeConfigTemplate)
	if err != nil {
		return "", errors.Wrap(err, "template parser")
	}

	var result strings.Builder
	err = t.Execute(&result, data)
	if err != nil {
		return "", errors.Wrap(err, "executing template")
	}

	return result.String(), nil
}

func getCurrentChartVersion() string {
	if v, err := gh.GetLatestRelease("", build.AuthorName, build.ChartsRepo); err == nil {
		return v
	}
	return "¯\\_(ツ)_/¯"
}

// homeDir returns the home directory for the current user
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func getCurrentAgentVersion() string {
	return "latest"
}
