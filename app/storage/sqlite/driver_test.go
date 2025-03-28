// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package sqlite_test

import (
	"sync"
	"testing"

	"github.com/cloudzero/cloudzero-insights-controller/app/storage/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSqlite3_NewDriver(t *testing.T) {
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

// Based on a test from: https://gist.github.com/jmunson/7b1974215d34439688b0
func TestSqlite3_ConcurrentAccess(t *testing.T) {
	gormdb, err := sqlite.NewSQLiteDriver(sqlite.MemorySharedCached)
	require.NoError(t, err, "failed to create sqlite driver")

	db, err := gormdb.DB()
	require.NoError(t, err, "failed to get underlying database")

	_, err = db.Exec("CREATE TABLE 'test' (key TEXT, value TEXT)")
	require.NoError(t, err, "failed to create test table")

	_, err = db.Exec("INSERT into 'test' VALUES ('a', 'aye'), ('b', 'bee'), ('c', 'cee')")
	require.NoError(t, err, "failed to isnert into test table")

	// values that are in our db
	kv := map[string]string{
		"a": "aye",
		"b": "bee",
		"c": "cee",
	}

	var wg sync.WaitGroup

	// We spawn 20 goroutines that each try to query every value in 'kv' and verify the results are correct
	// Each goroutine is given a waitgroup and a thread number just to track the output.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, thread int) {
			defer wg.Done()
			var errs int // number of failed results
			for id, knownvalue := range kv {
				var value string
				err := db.QueryRow("SELECT value FROM 'test' where key=?", id).Scan(&value)
				if err != nil || value != knownvalue {
					t.Errorf("[%d] [Error:%s]  looking up [%s] got [%s] should be [%s]", thread, err, id, value, knownvalue)
					errs++
				}
			}
		}(&wg, i)
	}

	wg.Wait()
}
