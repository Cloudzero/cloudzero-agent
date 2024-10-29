package storage

import (
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/utils"
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
	ctf := utils.FormatForStorage(ct)
	whereClause := fmt.Sprintf(`
		(record_updated < '%[1]s' AND record_created < '%[1]s' AND sent_at IS NULL)
		OR
		(sent_at IS NOT NULL AND record_updated > sent_at)
		`, ctf)
	for totalSize < w.settings.RemoteWrite.MaxBytesPerSend {
		var record ResourceTags
		result := w.db.Offset(offset).
			Where(whereClause).
			First(&record)
		if result.RowsAffected == 0 && errors.Is(result.Error, gorm.ErrRecordNotFound) {
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
