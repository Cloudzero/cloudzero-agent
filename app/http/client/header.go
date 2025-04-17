// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package http

import "github.com/cloudzero/cloudzero-agent/app/build"

const (
	HeaderAccept          = "Accept"
	HeaderUserAgent       = "User-Agent"
	HeaderAuthorization   = "Authorization"
	HeaderContentEncoding = "Content-Encoding"
	HeaderContentType     = "Content-Type"
	HeaderAcceptEncoding  = "Accept-Encoding"
)

const (
	ContentTypeGzip      = "gzip"
	ContentTypeProtobuf  = "application/x-protobuf"
	ContentTypeJSON      = "application/json"
	ContentTypeValueBin  = "application/octet-stream"
	ContentTypeValueTxt  = "text/plain"
	ContentTypeValueYAML = "text/yaml"
	ContentTypeValueCSV  = "text/csv"
)

func setUserAgent(headers map[string]string) {
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["User-Agent"] = "cloudzero/" + build.GetVersion()
}
