package config

type Annotations struct {
	Enabled bool     `yaml:"enabled" default:"false" env:"ANNOTATIONS_ENABLED" env-description:"enable annotations"`
	Filters []string `yaml:"filters" env:"ANNOTATIONS_FILTERS" env-description:"list of annotations to filter"`
}
