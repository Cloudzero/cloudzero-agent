package storage

import (
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DatabaseWriter interface {
	WriteData(data ResourceTags) error
}

func NewWriter(db *gorm.DB) *Writer {
	return &Writer{db: db}
}

type Writer struct {
	db *gorm.DB
}

func (w *Writer) WriteData(data ResourceTags) error {
	result := w.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "type"}, {Name: "name"}, {Name: "namespace"}},
	}).Create(&data)
	if result.Error != nil {
		log.Printf("failed to upsert tag data: %v", result.Error)
		return result.Error
	}
	return nil
}
