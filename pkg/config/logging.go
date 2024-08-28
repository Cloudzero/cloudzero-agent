package config

type Logging struct {
	Level string `yaml:"level" default:"info" env:"LOG_LEVEL" env-description:"logging level such as debug, info, error"`
}
