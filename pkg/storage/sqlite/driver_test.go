package sqlite_test

import (
	"testing"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestNewSqlite3Driver(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{
			name:    "InMemoryDSN",
			dsn:     sqlite.InMemoryDSN,
			wantErr: false,
		},
		{
			name:    "MemorySharedCached",
			dsn:     sqlite.MemorySharedCached,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := sqlite.NewSQLiteDriver(tt.dsn)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
				assert.IsType(t, &gorm.DB{}, db)
			}
		})
	}
}
