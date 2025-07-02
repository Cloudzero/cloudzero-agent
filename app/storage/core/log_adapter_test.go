// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package core_test

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cloudzero/cloudzero-agent/app/storage/core"
)

type MockWriter struct {
	Entries []map[string]interface{}
}

func NewMockWriter() *MockWriter {
	return &MockWriter{make([]map[string]interface{}, 0)}
}

func (m *MockWriter) Write(p []byte) (int, error) {
	entry := map[string]interface{}{}

	if err := json.Unmarshal(p, &entry); err != nil {
		panic(fmt.Sprintf("Failed to parse JSON %v: %s", p, err.Error()))
	}

	m.Entries = append(m.Entries, entry)

	return len(p), nil
}

func (m *MockWriter) Reset() {
	m.Entries = make([]map[string]interface{}, 0)
}

func Test_Logger_Sqlite(t *testing.T) {
	mogger := NewMockWriter()

	z := zerolog.New(mogger)

	now := time.Now()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{NowFunc: func() time.Time { return now }, Logger: core.ZeroLogAdapter{}})
	if err != nil {
		panic(err)
	}

	db = db.WithContext(z.WithContext(context.Background()))

	type Post struct {
		Title, Body string
		CreatedAt   time.Time
	}
	db.AutoMigrate(&Post{})

	cases := []struct {
		run        func() error
		sqlPattern string // Use regex pattern instead of exact string
		errOk      bool
	}{
		{
			run: func() error { return db.Create(&Post{Title: "awesome"}).Error },
			// Use regex pattern to match timestamp format flexibly
			sqlPattern: `INSERT INTO ` + "`posts`" + ` \(` + "`title`" + `,` + "`body`" + `,` + "`created_at`" + `\) VALUES \("awesome","","` + now.Format("2006-01-02 15:04:05") + `\.\d+"\)`,
			errOk:      false,
		},
		{
			run:        func() error { return db.Model(&Post{}).Find(&[]*Post{}).Error },
			sqlPattern: "SELECT \\* FROM `posts`",
			errOk:      false,
		},
		{
			run: func() error {
				return db.Where(&Post{Title: "awesome", Body: "This is awesome post !"}).First(&Post{}).Error
			},
			sqlPattern: fmt.Sprintf(
				"SELECT \\* FROM `posts` WHERE `posts`\\.`title` = %q AND `posts`\\.`body` = %q ORDER BY `posts`\\.`title` LIMIT 1",
				"awesome", "This is awesome post !",
			),
			errOk: true,
		},
		{
			run:        func() error { return db.Raw("THIS is,not REAL sql").Scan(&Post{}).Error },
			sqlPattern: "THIS is,not REAL sql",
			errOk:      true,
		},
	}

	for i, c := range cases {
		mogger.Reset()

		err := c.run()

		if err != nil && !c.errOk {
			t.Fatalf("Case %d: Unexpected error: %s (%T)", i, err, err)
		}

		// TODO: Must get from log entries
		entries := mogger.Entries

		if got, want := len(entries), 1; got != want {
			t.Errorf("Case %d: Logger logged %d items, want %d items", i, got, want)
		} else {
			fieldByName := entries[0]

			actualSQL := fieldByName["sql"].(string)

			// Use regex matching for flexible timestamp matching
			matched, err := regexp.MatchString(c.sqlPattern, actualSQL)
			if err != nil {
				t.Fatalf("Case %d: Invalid regex pattern %q: %v", i, c.sqlPattern, err)
			}

			if !matched {
				t.Errorf("Case %d: Logged sql %q does not match pattern %q", i, actualSQL, c.sqlPattern)
			}
		}
	}
}
