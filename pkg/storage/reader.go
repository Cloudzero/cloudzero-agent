package storage

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"gorm.io/gorm"
)

type DatabaseReader interface {
	ReadData(time.Time) ([]ResourceTags, error)
}

func NewReader(db *gorm.DB, settings *config.Settings) *Reader {
	return &Reader{db: db, settings: settings}
}

type Reader struct {
	db       *gorm.DB
	settings *config.Settings
}

func (w *Reader) ReadData(ct time.Time) ([]ResourceTags, error) {
	records := []ResourceTags{}
	totalSize := 0
	offset := 0
	whereClause := fmt.Sprintf(`
		(updated_at < '%[1]s' AND created_at < '%[1]s' AND sent_at IS NULL)
		OR
		(sent_at IS NOT NULL AND updated_at > sent_at)
		`, ct.Format(time.RFC3339))
	for totalSize < w.settings.RemoteWrite.MaxBytesPerSend {
		var record ResourceTags
		result := w.db.Offset(offset).
			Where(whereClause).
			First(&record)
		if result.RowsAffected == 0 && result.Error.Error() == "record not found" {
			break
		}
		if result.Error != nil {
			log.Printf("failed to read tag data: %v", result.Error)
			return nil, result.Error
		}
		records = append(records, record)
		totalSize += record.Size
		offset++
	}
	return records, nil
}
