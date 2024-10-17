package main

import (
	"context"

	"github.com/go-obvious/server"

	"github.com/cloudzero/cirrus-remote-write/app/remotewrite/internal/build"
	metrics "github.com/cloudzero/cirrus-remote-write/app/remotewrite/internal/service/metrics/http"
)

func main() {
	server.New(
		&server.ServerVersion{
			Revision: build.Rev,
			Tag:      build.Tag,
			Time:     build.Time,
		},
		metrics.NewService("/metrics"),
	).Run(context.Background())
}
