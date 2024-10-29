package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func dbCleanup(db *gorm.DB) {
	instance, _ := db.DB()
	instance.Close()
}

func TestWriter_WriteData(t *testing.T) {
	db := SetupDatabase()

	writer := NewWriter(db)

	data := ResourceTags{
		Type:      1,
		Name:      "test-name",
		Namespace: func(s string) *string { return &s }("test-namespace"),
	}

	err := writer.WriteData(data)
	assert.NoError(t, err)

	var result ResourceTags
	err = db.Where("type = ? AND name = ? AND namespace = ?", data.Type, data.Name, data.Namespace).First(&result).Error
	assert.NoError(t, err)
	assert.Equal(t, data.Type, result.Type)
	assert.Equal(t, data.Name, result.Name)
	assert.Equal(t, data.Namespace, result.Namespace)
	t.Cleanup(func() {
		dbCleanup(db)
	})
}
func TestWriter_UpdateSentAtForRecords(t *testing.T) {
	db := SetupDatabase()
	writer := NewWriter(db)

	records := []ResourceTags{
		{
			Type:      1,
			Name:      "test-deployment-1",
			Namespace: func(s string) *string { return &s }("test-namespace-1"),
		},
		{
			Type:      2,
			Name:      "test-statefulset-2",
			Namespace: func(s string) *string { return &s }("test-namespace-2"),
		},
		{
			Type:      3,
			Name:      "test-pod-3",
			Namespace: func(s string) *string { return &s }("test-namespace-3"),
		},
		// {
		// 	Type: 4,
		// 	Name: "test-node-4",
		// },
		// {
		// 	Type: 5,
		// 	Name: "test-namespace-5",
		// },
	}

	for _, record := range records {
		err := writer.WriteData(record)
		assert.NoError(t, err)
	}

	ct := time.Now().UTC()
	_, err := writer.UpdateSentAtForRecords(records, ct)
	assert.NoError(t, err)

	for _, record := range records {
		var result ResourceTags
		err = db.Where("type = ? AND name = ?", record.Type, record.Name).First(&result).Error
		assert.NoError(t, err)
		// if result.SentAt == nil {
		// 	assert.Fail(t, "sent_at should not be nil")
		// }
		// assert that result.SentAt is not nil
		// assert.NotNil(t, result.SentAt)
		// assert.Equal(t, utils.FormatForStorage(ct), utils.FormatForStorage(*result.SentAt))
		// assert.Equal(t, utils.FormatForStorage(ct), utils.FormatForStorage(result.UpdatedAt))
	}
	t.Cleanup(func() {
		dbCleanup(db)
	})
}
func TestWriter_PurgeStaleData(t *testing.T) {
	db := SetupDatabase()
	writer := NewWriter(db)

	records := []ResourceTags{
		{
			Type:      1,
			Name:      "test-name-1",
			Namespace: func(s string) *string { return &s }("test-namespace-1"),
			CreatedAt: time.Now().Add(-48 * time.Hour),
			UpdatedAt: time.Now().Add(-48 * time.Hour),
			SentAt:    func(t time.Time) *time.Time { return &t }(time.Now().Add(-48 * time.Hour)),
		},
		{
			Type:      2,
			Name:      "test-name-2",
			Namespace: func(s string) *string { return &s }("test-namespace-2"),
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now().Add(-24 * time.Hour),
			SentAt:    func(t time.Time) *time.Time { return &t }(time.Now().Add(-24 * time.Hour)),
		},
		{
			Type:      3,
			Name:      "test-name-3",
			Namespace: func(s string) *string { return &s }("test-namespace-3"),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			SentAt:    func(t time.Time) *time.Time { return &t }(time.Now()),
		},
	}

	for _, record := range records {
		err := writer.WriteData(record)
		assert.NoError(t, err)
	}
	var resultsBefore []ResourceTags
	_ = db.Find(&resultsBefore).Error
	// purge data older than 36 hours
	err := writer.PurgeStaleData(36 * time.Hour)
	assert.NoError(t, err)

	var results []ResourceTags
	err = db.Find(&results).Error
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	for _, result := range results {
		assert.NotEqual(t, records[0].Name, result.Name)
	}
	t.Cleanup(func() {
		dbCleanup(db)
	})
}
