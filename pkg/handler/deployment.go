package handler

import (
	"encoding/json"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	v1 "k8s.io/api/apps/v1"
)

type DeploymentHandler struct {
	hook.Handler
} // &v1.Deployment{}

// NewValidationHook creates a new instance of deployment validation hook
func NewDeploymentHandler(settings *config.Settings) hook.Handler {
	// TODO: Need to accept a Data object
	// for saving records to the database

	// Need little trick to protect internal data
	d := &DeploymentHandler{}
	d.Handler.Create = d.Create()
	d.Handler.Delete = d.Delete()
	return d.Handler
}

func (d *DeploymentHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		dp, err := d.parseV1(r.Object.Raw)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}

		if dp.Namespace == "special" {
			// log? You cannot create a deployment in `special` namespace
			return &hook.Result{Allowed: true}, nil
		}

		return &hook.Result{Allowed: true}, nil
	}
}

func (d *DeploymentHandler) Delete() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		dp, err := d.parseV1(r.OldObject.Raw)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}

		if dp.Namespace == "special-system" && dp.Annotations["skip"] == "false" {
			// log? You cannot remove a deployment from `special-system` namespace.
			return &hook.Result{Allowed: true}, nil
		}

		return &hook.Result{Allowed: true}, nil
	}
}

func (d *DeploymentHandler) parseV1(object []byte) (*v1.Deployment, error) {
	var dp v1.Deployment
	if err := json.Unmarshal(object, &dp); err != nil {
		return nil, err
	}
	return &dp, nil
}
