package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
	cfg "github.com/go-obvious/server/config"
	"github.com/go-obvious/server/request"
	"github.com/go-obvious/timestamp"

	"github.com/cloudzero/cirrus-remote-write/app/remotewrite/internal/service/metrics/config"
	"github.com/cloudzero/cirrus-remote-write/app/remotewrite/internal/storage"
	"github.com/cloudzero/cirrus-remote-write/app/remotewrite/internal/validation"
)

const MaxPayloadSize = 256 * 1024 * 1024

type API struct {
	api.Service
	client     *s3.S3
	s3Cfg      config.S3Config
	metricsCfg config.MetricService
}

func NewService(base string) *API {
	a := &API{
		Service: api.Service{
			APIName: "metrics",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	a.Service.Mounts[base] = a.routes()
	cfg.Register(&a.s3Cfg, &a.metricsCfg)
	return a
}

func (a *API) Register(app server.Server) error {
	logrus.WithField("s3Cfg", a.s3Cfg.String()).WithField("metricsCfg", a.metricsCfg.String()).Debug("configurations")

	a.createS3Client()
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (a *API) createS3Client() {
	if a.s3Cfg.Endpoint == "" {
		logrus.Debug("DEFAULT s3 client")
		a.client = s3.New(session.Must(session.NewSession(&aws.Config{
			Region: aws.String(a.s3Cfg.Region),
		})))
		return
	}
	logrus.Debug("CUSTOMER s3 client")
	a.client = s3.New(session.Must(session.NewSession(&aws.Config{
		Region:           aws.String(a.s3Cfg.Region),
		Endpoint:         aws.String(a.s3Cfg.Endpoint),
		Credentials:      credentials.NewStaticCredentials(a.s3Cfg.AccessKey, a.s3Cfg.SecretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(true),
	})))
}

func (a *API) routes() *chi.Mux {
	r := chi.NewRouter()
	r.Post("/", a.PostMetrics)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		request.Reply(r, w, "hello", http.StatusOK)
	})
	return r
}

func (a *API) PostMetrics(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	contentLen := r.ContentLength

	if contentLen <= 0 {
		request.Reply(r, w, "empty body", http.StatusOK)
		return
	}

	if contentLen > MaxPayloadSize {
		request.Reply(r, w, "too big", http.StatusOK)
		return
	}

	organizationID := r.Header.Get("organization_id")
	if organizationID == "" {
		request.Reply(r, w, "organization_id is required", http.StatusBadRequest)
		return
	}

	clusterName := request.QS(r, "cluster_name")
	if err := validation.ValidateClusterName(clusterName); err != nil {
		request.Reply(r, w, "cluster_name is required", http.StatusBadRequest)
		return
	}

	cleanAccountID, err := validation.ValidateCloudAccountID(request.QS(r, "cloud_account_id"))
	if err != nil || cleanAccountID == "" {
		request.Reply(r, w, "cloud_account_id is required", http.StatusBadRequest)
		return
	}

	key := storage.BuildPath(
		storage.CompressedFilesPrefix(organizationID),
		timestamp.Now(),
		cleanAccountID,
		clusterName,
		storage.CompressedFileExt(),
	)

	logrus.WithFields(logrus.Fields{
		"organizationID": organizationID,
		"clusterName":    clusterName,
		"cloudAccountID": cleanAccountID,
		"key":            key,
	}).Trace("saved to s3")

	if err := a.upload(r.Context(), storage.BucketName(organizationID), key, contentLen, r.Body); err != nil {
		logrus.WithError(err).Error("failed to write to s3")
		request.Reply(r, w, "failed to write to s3", http.StatusInternalServerError)
		return
	}

	request.Reply(r, w, nil, http.StatusOK)
}

func (a *API) upload(ctx context.Context, bucket, key string, len int64, body io.ReadCloser) error {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, body); err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}

	if _, err := a.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(buf.Bytes()),
		ContentLength: aws.Int64(len),
		ContentType:   aws.String("application/octet-stream"),
	}); err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}
