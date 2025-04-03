package shipper_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cloudzero/cloudzero-agent-validator/app/domain/shipper"
	"github.com/stretchr/testify/require"
)

func TestUnit_Shipper_ErrorsBasicWrapping(t *testing.T) {
	// test a basic case
	err := errors.Join(shipper.ErrCreateDirectory, fmt.Errorf("this is a wrapped error: %w", errors.New("base error")))
	statusCode := shipper.GetErrStatusCode(err)
	require.Equal(t, shipper.ErrCreateDirectory.Code(), statusCode)

	// test the reverse case
	err = errors.Join(errors.New("This is a basic error"), fmt.Errorf("this is a wrapped error: %w", shipper.ErrCreateDirectory))
	statusCode = shipper.GetErrStatusCode(err)
	require.Equal(t, shipper.ErrCreateDirectory.Code(), statusCode)
}

func TestUnit_Shipper_MultiWrapping(t *testing.T) {
	err := errors.Join(errors.New("this is a basic error"), fmt.Errorf("this is a wrapped error: %w", shipper.ErrCreateDirectory))
	err = errors.Join(shipper.ErrCreateLock, err)
	err = errors.Join(errors.New("this is a bogus error"), err)

	statusCode := shipper.GetErrStatusCode(err)
	require.Equal(t, statusCode, shipper.ErrCreateLock.Code())
}
