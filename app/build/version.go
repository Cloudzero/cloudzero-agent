// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"fmt"

	"github.com/go-obvious/server"
)

var (
	AuthorName       = "Cloudzero"
	ChartsRepo       = "cloudzero-charts"
	AuthorEmail      = "support@cloudzero.com"
	Copyright        = "© 2024-2025 Cloudzero, Inc."
	PlatformEndpoint = "https://api.cloudzero.com"
)

func Version() *server.ServerVersion {
	return &server.ServerVersion{Revision: Rev, Tag: Tag, Time: Time}
}

func GetVersion() string {
	return fmt.Sprintf("%s.%s-%s", Rev, Tag, Time)
}
