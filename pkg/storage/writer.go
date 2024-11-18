package storage

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/utils"
	"github.com/rs/zerolog/log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DatabaseWriter interface {
	WriteData(data ResourceTags, isCreate bool) error
	UpdateSentAtForRecords(data []ResourceTags, ct time.Time) (int64, error)
	PurgeStaleData(rt time.Duration) error
}

func NewWriter(db *gorm.DB, settings *config.Settings) *Writer {
	return &Writer{db: db, mu: sync.Mutex{}, clock: utils.Clock{}, settings: settings}
}

type Writer struct {
	db       *gorm.DB
	mu       sync.Mutex
	clock    utils.Clock
	settings *config.Settings
}

func (w *Writer) WriteData(data ResourceTags, isCreate bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	ct := w.clock.GetCurrentTime()
	if isCreate {
		data.RecordCreated = ct
		data.RecordUpdated = ct
	} else {
		data.RecordUpdated = ct
	}
	result := w.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "type"}, {Name: "name"}, {Name: "namespace"}},
	}).Create(&data)
	if result.Error != nil {
		log.Printf("failed to upsert tag data: %v", result.Error)
		return result.Error
	}
	return nil
}

func (w *Writer) UpdateSentAtForRecords(records []ResourceTags, ct time.Time) (int64, error) {
	log.Debug().Msgf("Updating the sent_at column for %d records", len(records))
	ctf := utils.FormatForStorage(ct)
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(records) == 0 {
		return 0, nil
	}

	var conditions []string
	for _, record := range records {
		if record.Namespace != nil {
			conditions = append(conditions, fmt.Sprintf("(type = '%d' AND name = '%s' AND namespace = '%s')", record.Type, record.Name, *record.Namespace))
		} else {
			conditions = append(conditions, fmt.Sprintf("(type = '%d' AND name = '%s' AND namespace IS NULL)", record.Type, record.Name))
		}
	}

	var batchSize = w.settings.Database.BatchUpdateSize
	var updatedRows int64

	for i := 0; i < len(conditions); i += batchSize {
		end := i + batchSize
		if end > len(conditions) {
			end = len(conditions)
		}
		batchConditions := conditions[i:end]
		whereClause := fmt.Sprintf("record_updated < '%s' AND record_created < '%s' AND (%s)", ctf, ctf, strings.Join(batchConditions, " OR "))
		result := w.db.Model(&ResourceTags{}).
			Where(whereClause).
			Update("sent_at", ct).
			Update("record_updated", ct)
		if result.Error != nil {
			log.Error().Msgf("failed to update sent_at for records: %v", result.Error)
			return 0, result.Error
		}
		updatedRows += result.RowsAffected
		log.Debug().Msgf("Updated sent_at for %d records in this batch", result.RowsAffected)
	}
	return updatedRows, nil
}

func (w *Writer) PurgeStaleData(rt time.Duration) error {
	retentionTime := utils.FormatForStorage(time.Now().UTC().Add(-1 * rt))
	log.Debug().Msgf("Starting data purge process for stale records older than %s", retentionTime)
	w.mu.Lock()
	defer w.mu.Unlock()
	whereClause := fmt.Sprintf("sent_at < '%[1]s' AND record_created < '%[1]s' AND record_updated < '%[1]s' AND sent_at IS NOT NULL", retentionTime)
	result := w.db.Where(whereClause).Delete(&ResourceTags{})

	if result.Error != nil {
		log.Printf("failed to delete old tag data: %v", result.Error)
		return result.Error
	}
	log.Debug().Msgf("Deleted %d records", result.RowsAffected)
	return nil
}
