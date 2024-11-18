package storage

import (
	"github.com/rs/zerolog/log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func newDriver() (*gorm.DB, error) {
	return gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func SetupDatabase() *gorm.DB {
	errHistory := []error{}
	db, err := newDriver()

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get access underlying sql object from gorm")
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	_, err = sqlDB.Exec("PRAGMA busy_timeout = 5000;") // 5000 milliseconds
	if err != nil {
		log.Error().Msgf("Failed to set busy timeout: %v", err)
	}
	if err := db.AutoMigrate(&RemoteWriteHistory{}); err != nil {
		errHistory = append(errHistory, err)
	}
	if err := db.AutoMigrate(&ResourceTags{}); err != nil {
		errHistory = append(errHistory, err)
	}

	if len(errHistory) > 0 {
		for _, err := range errHistory {
			log.Info().Err(err).Msgf("error creating table: %v", err)
		}
		log.Fatal().Msg("Unable to create db tables")
	}
	return db
}
