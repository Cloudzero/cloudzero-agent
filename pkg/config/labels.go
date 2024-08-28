package config

type Labels struct {
	Enabled bool     `yaml:"enabled" default:"false" env:"LABELS_ENABLED" env-description:"enable labels"`
	Filters []string `yaml:"filters" env:"LABELS_FILTERS" env-description:"list of labels to filter"`
}
