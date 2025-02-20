// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"errors"
)

func (m *MetricShipper) HandleDisk() error {
	return errors.ErrUnsupported
}
