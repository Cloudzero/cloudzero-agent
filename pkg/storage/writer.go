package storage

import (
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DatabaseWriter interface {
	WriteData(data ResourceTags) error
	UpdateSentAtForRecords(data []ResourceTags, ct time.Time) error
}

func NewWriter(db *gorm.DB) *Writer {
	return &Writer{db: db, mu: sync.Mutex{}}
}

type Writer struct {
	db *gorm.DB
	mu sync.Mutex
}

func (w *Writer) WriteData(data ResourceTags) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := w.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "type"}, {Name: "name"}, {Name: "namespace"}},
	}).Create(&data)
	if result.Error != nil {
		log.Printf("failed to upsert tag data: %v", result.Error)
		return result.Error
	}
	return nil
}

func (w *Writer) UpdateSentAtForRecords(records []ResourceTags, ct time.Time) error {
	log.Debug().Msgf("Updating the sent_at column for %d records", len(records))
	cts := ct.Format(time.RFC3339)
	w.mu.Lock()
	defer w.mu.Unlock()

	var compositeKeys = make([][]interface{}, 0, len(records)) //nolint:gofmt
	for _, record := range records {
		item := []interface{}{ //nolint:gofmt
			strconv.Itoa(int(record.Type)),
			record.Name,
			*record.Namespace,
		}
		compositeKeys = append(compositeKeys, item)
	}
	result := w.db.Model(&ResourceTags{}).
		Where("updated_at < ? AND created_at < ?", cts, cts).
		Where("(type, name, namespace) IN ?", compositeKeys).
		Update("sent_at", ct).
		Update("updated_at", ct)
	if result.Error != nil {
		log.Error().Msgf("failed to update sent_at for records: %v", result.Error)
		return result.Error
	}
	log.Debug().Msgf("Updated sent_at for %d records", result.RowsAffected)
	return nil
}
