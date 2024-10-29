package storage

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/utils"
	"github.com/rs/zerolog/log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DatabaseWriter interface {
	WriteData(data ResourceTags) error
	UpdateSentAtForRecords(data []ResourceTags, ct time.Time) (int64, error)
	PurgeStaleData(rt time.Duration) error
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

func (w *Writer) UpdateSentAtForRecords(records []ResourceTags, ct time.Time) (int64, error) {
	log.Debug().Msgf("Updating the sent_at column for %d records", len(records))
	ctf := utils.FormatForStorage(ct)
	w.mu.Lock()
	defer w.mu.Unlock()

	var conditions []string
	var args []interface{} //nolint:gofmt
	for _, record := range records {
		if record.Namespace != nil {
			conditions = append(conditions, "(type = ? AND name = ? AND namespace = ?)")
			args = append(args, record.Type, record.Name, *record.Namespace)
		} else {
			conditions = append(conditions, "(type = ? AND name = ? AND namespace IS NULL)")
			args = append(args, record.Type, record.Name)
		}
	}

	whereClause := fmt.Sprintf("updated_at < ? AND created_at < ? AND (%s)", strings.Join(conditions, " OR "))
	args = append([]interface{}{ctf, ctf}, args...) //nolint:gofmt
	result := w.db.Model(&ResourceTags{}).
		Where(whereClause, args...).
		Update("sent_at", ct).
		Update("updated_at", ct)
	if result.Error != nil {
		log.Error().Msgf("failed to update sent_at for records: %v", result.Error)
		return 0, result.Error
	}
	updatedRows := result.RowsAffected
	log.Debug().Msgf("Updated sent_at for %d records", updatedRows)
	return updatedRows, nil
}

func (w *Writer) PurgeStaleData(rt time.Duration) error {
	retentionTime := utils.FormatForStorage(time.Now().UTC().Add(-1 * rt))
	log.Debug().Msgf("Starting data purge process for stale records older than %s", retentionTime)
	w.mu.Lock()
	defer w.mu.Unlock()
	whereClause := fmt.Sprintf("sent_at < '%[1]s' AND created_at < '%[1]s' AND updated_at < '%[1]s' AND sent_at IS NOT NULL", retentionTime)
	result := w.db.Where(whereClause).Delete(&ResourceTags{})

	if result.Error != nil {
		log.Printf("failed to delete old tag data: %v", result.Error)
		return result.Error
	}
	log.Debug().Msgf("Deleted %d records", result.RowsAffected)
	return nil
}
