// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package smoke

import "testing"

func TestSmoke_CollectorShipper_Runs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
}
