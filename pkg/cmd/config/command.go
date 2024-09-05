package config

import (
    "context"
    _ "embed"
    "fmt"
    "html/template"
    "os"
    "path/filepath"

    "github.com/pkg/errors"
    "github.com/urfave/cli/v2"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"

    "github.com/cloudzero/cloudzero-agent-validator/pkg/build"
    "github.com/cloudzero/cloudzero-agent-validator/pkg/config"
    "github.com/cloudzero/cloudzero-agent-validator/pkg/util/gh"
)

//go:embed internal/template.yml
var templateString string

var (
	configAlias = []string{"f"}
)

func NewCommand() *cli.Command {
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
				},
				Action: func(c *cli.Context) error {
					return Generate(map[string]interface{}{ //nolint: gofmt
						"ChartVerson":         getCurrentChartVersion(),
						"AgentVersion":        getCurrentAgentVersion(),
						"AccountID":           c.String(config.FlagAccountID),
						"ClusterName":         c.String(config.FlagClusterName),
						"Region":              c.String(config.FlagRegion),
						"CloudzeroHost":       build.PlatformEndpoint,
						"KubeStateMetricsURL": "http://kube-state-metrics.your-namespace.svc.cluster.local:8080",
						"PromNodeExporterURL": "http://node-exporter.your-namespace.svc.cluster.local:9100",
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
      {
          Name:  "list-services",
          Usage: "lists Kubernetes services in all namespaces",
          Flags: []cli.Flag{
              &cli.StringFlag{Name: "kubeconfig", Usage: "absolute path to the kubeconfig file", Required: false},
          },
          Action: func(c *cli.Context) error {
              // List all services in all namespaces
              return ListServices(c.String("kubeconfig"))
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

// ListServices lists all Kubernetes services in all namespaces
func ListServices(kubeconfigPath string) error {
    if kubeconfigPath == "" {
        kubeconfigPath = filepath.Join(homeDir(), ".kube", "config")
    }

    // Build the Kubernetes client configuration
    config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
    if err != nil {
        return errors.Wrap(err, "building kubeconfig")
    }

    // Create a new Kubernetes clientset
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return errors.Wrap(err, "creating clientset")
    }

    // List all services in all namespaces
    services, err := clientset.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        return errors.Wrap(err, "listing services")
    }

    // Print the names and namespaces of the services
    fmt.Println("Services in all namespaces:")
    for _, service := range services.Items {
        fmt.Printf(" - %s (Namespace: %s)\n", service.Name, service.Namespace)
    }

    return nil
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
