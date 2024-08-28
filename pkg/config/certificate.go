package config

type Certificate struct {
	Key  string `yaml:"key" env:"TLS_KEY" env-description:"path to the TLS key"`
	Cert string `yaml:"cert" env:"TLS_CERT" env-description:"path to the TLS certificate"`
}
