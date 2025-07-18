// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package telemetry contains code for posting telemetry data to the CloudZero API.
package telemetry

import (
	"bytes"
	"context"
	"fmt"
	net "net/http"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	http "github.com/cloudzero/cloudzero-agent/app/http/client"
	pb "github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	PostTimeout = 5 * time.Second
)

const (
	URLPath = "/v1/container-metrics/status"
)

const (
	// matches AWS API Gateway timeout
	Timeout = 15 * time.Second
)

func Post(ctx context.Context, client *net.Client, cfg *config.Settings, accessor pb.Accessor) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if accessor == nil {
		return errors.New("nil accessor")
	}

	if cfg.Cloudzero.Host == "" {
		return errors.New("missing cloudzero host")
	}

	if cfg.Cloudzero.Credential == "" {
		return errors.New("missing cloudzero api key")
	}

	// quietly exit
	if cfg.Cloudzero.DisableTelemetry {
		return nil
	}

	if client == nil {
		client = net.DefaultClient
	}

	var (
		err  error
		data []byte
	)
	accessor.ReadFromReport(func(cs *pb.ClusterStatus) {
		data, err = proto.Marshal(cs)
		logrus.Info("marshalled cluster status: " + strconv.Itoa(len(data)) + " bytes")
	})
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return errors.New("no data to post")
	}
	buf, err := compress(data)
	if err != nil {
		return err
	}

	logrus.Infof("compressed size is: %d bytes", buf.Len())

	endpoint := fmt.Sprintf("%s%s", cfg.Cloudzero.Host, URLPath)
	_, err = http.Do(ctx, client, net.MethodPost,
		map[string]string{
			http.HeaderAuthorization: "Bearer " + cfg.Cloudzero.Credential,
			http.HeaderContentType:   http.ContentTypeProtobuf,
		},
		map[string]string{
			http.QueryParamAccountID:   cfg.Deployment.AccountID,
			http.QueryParamRegion:      cfg.Deployment.Region,
			http.QueryParamClusterName: cfg.Deployment.ClusterName,
		},
		endpoint,
		&buf,
	)

	return err
}

func compress(data []byte) (bytes.Buffer, error) {
	var buf bytes.Buffer
	if _, err := buf.Write(data); err != nil {
		return bytes.Buffer{}, errors.Wrap(err, "write data to buffer")
	}

	// XXX: Compression is taking more space than the buffer alone! We gain nothing!
	// snappyWriter := snappy.NewBufferedWriter(&buf)
	// _, err := snappyWriter.Write(data)
	// if err != nil {
	// 	return bytes.Buffer{}, errors.Wrap(err, "data compress")
	// }

	// if err = snappyWriter.Close(); err != nil {
	// 	return bytes.Buffer{}, errors.Wrap(err, "close writer failed")
	// }

	return buf, nil
}
