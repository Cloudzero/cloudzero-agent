package config

type Versions struct {
	ChartVersion string `yaml:"chart_version" env:"CHART_VERSION" env-description:"Chart Version"`
	AgentVersion string `yaml:"agent_version" env:"AGENT_VERSION" env-description:"Agent Version"`
}

func (s *Versions) Validate() error {
	return nil
}
