//go:build unit
// +build unit

package build_test

import (
	"testing"

	"github.com/cloudzero/cirrus-remote-write/app/internal/build"
	"github.com/stretchr/testify/assert"
)

func TestBuild(t *testing.T) {
	assert.NotEmpty(t, build.Version())
}
