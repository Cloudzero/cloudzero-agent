package config

type Database struct {
	Enabled     bool   `yaml:"enabled" default:"false" env:"DATABASE_ENABLED" env-description:"when enabled will write to persistent storage, otherwise only in memory sqlite"`
	StoragePath string `yaml:"storagePath" default:"/opt/insights" env:"DATABASE_STORAGE_PATH" env-description:"location where to write database"`
}
