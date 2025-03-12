// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// GetShipperID will return a unique id for this shipper. This id is stored on the filesystem,
// and is meant to represent a relation between an uploaded file and which shipper this file came from.
// The id is not an id this instance of the shipper, but more an id of the filesystem in which the
// file came from
func (m *MetricShipper) GetShipperID() (string, error) {
	if m.shipperID == "" {
		// where the shipper id lives
		loc := filepath.Join(m.GetBaseDir(), ".shipperid")

		data, err := os.ReadFile(loc)
		if err == nil { //nolint:gocritic // impossible to do if/else statement here
			// file was read successfully
			m.shipperID = strings.TrimSpace(string(data))
		} else if os.IsNotExist(err) {
			// create the file
			file, err := os.Create(loc) //nolint:govet // err was in-fact read for what was needed
			if err != nil {
				return "", fmt.Errorf("failed to create the shipper id file: %w", err)
			}
			defer file.Close()

			// write an id to the file
			id := uuid.NewString()
			if _, err := file.WriteString(id); err != nil {
				return "", fmt.Errorf("failed to write an id to the id file: %w", err)
			}

			m.shipperID = id
		} else {
			return "", fmt.Errorf("unknown error getting the shipper id: %w", err)
		}
	}

	return m.shipperID, nil
}
