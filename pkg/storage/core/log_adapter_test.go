// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package core_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage/core"
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
		run   func() error
		sql   string
		errOk bool
	}{
		{
			run: func() error { return db.Create(&Post{Title: "awesome"}).Error },
			sql: fmt.Sprintf(
				"INSERT INTO `posts` (`title`,`body`,`created_at`) VALUES (%q,%q,%q)",
				"awesome", "", strings.TrimSuffix(now.Format("2006-01-02 15:04:05.000"), "0"),
			),
			errOk: false,
		},
		{
			run:   func() error { return db.Model(&Post{}).Find(&[]*Post{}).Error },
			sql:   "SELECT * FROM `posts`",
			errOk: false,
		},
		{
			run: func() error {
				return db.Where(&Post{Title: "awesome", Body: "This is awesome post !"}).First(&Post{}).Error
			},
			sql: fmt.Sprintf(
				"SELECT * FROM `posts` WHERE `posts`.`title` = %q AND `posts`.`body` = %q ORDER BY `posts`.`title` LIMIT 1",
				"awesome", "This is awesome post !",
			),
			errOk: true,
		},
		{
			run:   func() error { return db.Raw("THIS is,not REAL sql").Scan(&Post{}).Error },
			sql:   "THIS is,not REAL sql",
			errOk: true,
		},
	}

	for _, c := range cases {
		mogger.Reset()

		err := c.run()

		if err != nil && !c.errOk {
			t.Fatalf("Unexpected error: %s (%T)", err, err)
		}

		// TODO: Must get from log entries
		entries := mogger.Entries

		if got, want := len(entries), 1; got != want {
			t.Errorf("Logger logged %d items, want %d items", got, want)
		} else {
			fieldByName := entries[0]

			if got, want := fieldByName["sql"].(string), c.sql; got != want {
				t.Errorf("Logged sql was %q, want %q", got, want)
			}
		}
	}
}
