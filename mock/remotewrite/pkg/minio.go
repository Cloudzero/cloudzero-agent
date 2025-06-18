// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package remotewrite

import (
	"encoding/json"
	"net/http"

	"github.com/minio/minio-go/v7"
	"golang.org/x/exp/slog"
)

type QueryMinioResponse struct {
	Objects []*minio.ObjectInfo `json:"objects"`
	Length  int                 `json:"length"`
}

// QueryMinio scans the entire minio instance and returns all of the objects
// this can return a lot of information if there are a lot of objects stored
func (rw *RemoteWrite) QueryMinio(w http.ResponseWriter, r *http.Request) {
	objects := make([]*minio.ObjectInfo, 0)
	for obj := range rw.minioClient.ListObjects(r.Context(), bucketName, minio.ListObjectsOptions{}) {
		objects = append(objects, &obj)
	}

	response := QueryMinioResponse{
		Objects: objects,
		Length:  len(objects),
	}

	enc, err := json.Marshal(&response)
	if err != nil {
		slog.Default().Error("failed to encode json", "error", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	w.Write(enc)
}
