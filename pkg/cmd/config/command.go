package config

import (
	"context"
	_ "embed"
	"html/template"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/build"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/k8s"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/util/gh"
)

//go:embed internal/template.yml
var templateString string

var (
	configAlias = []string{"f"}
)

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
					&cli.StringFlag{Name: config.FlagConfigFile, Aliases: configAlias, Usage: "output configuration file. if omitted output will print to standard out", Required: false},
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

					configMap, err := k8s.GetConfigMap(ctx, clientset, namespace, configMapName)
					if err != nil {
						return err
					}

					kubeStateMetricsURL, nodeExporterURL, err := k8s.GetServiceURLs(ctx, clientset)
					if err != nil {
						return err
					}

					// Update the ConfigMap data
					configMap.Data["prometheus.kube_state_metrics_service_endpoint"] = kubeStateMetricsURL
					configMap.Data["prometheus.prometheus_node_exporter_service_endpoint"] = nodeExporterURL

					err = k8s.UpdateConfigMap(ctx, clientset, namespace, configMap)
					if err != nil {
						return err
					}

					return Generate(map[string]interface{}{ //nolint: gofmt
						"ChartVerson":         getCurrentChartVersion(),
						"AgentVersion":        getCurrentAgentVersion(),
						"AccountID":           c.String(config.FlagAccountID),
						"ClusterName":         c.String(config.FlagClusterName),
						"Region":              c.String(config.FlagRegion),
						"CloudzeroHost":       build.PlatformEndpoint,
						"KubeStateMetricsURL": kubeStateMetricsURL,
						"PromNodeExporterURL": nodeExporterURL,
					}, c.String(config.FlagConfigFile))
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

func Generate(values map[string]interface{}, outputFile string) error { //nolint: gofmt
	t := template.New("template")
	t, err := t.Parse(templateString)
	if err != nil {
		return errors.Wrap(err, "template parser")
	}
	var cleanup = func() {}
	out := os.Stdout
	if outputFile != "" {
		output, err := os.Create(outputFile)
		if err != nil {
			return errors.Wrap(err, "creating output file")
		}
		cleanup = func() {
			output.Close()
		}
		out = output
	}
	defer cleanup()
	return t.Execute(out, values)
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
