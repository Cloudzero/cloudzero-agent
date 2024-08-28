package config

type Server struct {
	Port         string `yaml:"port" default:"8080" env:"PORT" env-description:"port to listen on"`
	ReadTimeout  int    `yaml:"readTimeout" default:"15" env:"READ_TIMEOUT" env-description:"server read timeout in seconds"`
	WriteTimeout int    `yaml:"writeTimeout" default:"15" env:"WRITE_TIMEOUT" env-description:"server write timeout in seconds"`
	IdleTimeout  int    `yaml:"idleTimeout" default:"60" env:"IDLE_TIMEOUT" env-description:"server idle timeout in seconds"`
}
