package adapter

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/CenturyLinkLabs/pmxadapter"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
)

const (
	metadataType    = "Kubernetes"
	metadataVersion = "0.1"
)

var (
	DefaultExecutor       Executor
	illegalNameCharacters = regexp.MustCompile(`[\W_]+`)
)

func init() {
	// TODO Need to instantiate with the correct connection info?
	DefaultExecutor = NewKubernetesExecutor("http://104.131.157.89:8080")
}

type KubernetesAdapter struct{}

func (a KubernetesAdapter) GetServices() ([]pmxadapter.ServiceDeployment, *pmxadapter.Error) {
	rcs, err := DefaultExecutor.GetReplicationControllers()
	if err != nil {
		pmxErr := pmxadapter.NewError(http.StatusInternalServerError, err.Error())
		return []pmxadapter.ServiceDeployment{}, pmxErr
	}

	sds := make([]pmxadapter.ServiceDeployment, len(rcs))
	for i, rc := range rcs {
		sds[i].ID = rc.ObjectMeta.Name
		sds[i].ActualState = statusFromReplicationController(rc)
	}
	return sds, nil
}

func (a KubernetesAdapter) GetService(id string) (pmxadapter.ServiceDeployment, *pmxadapter.Error) {
	rc, err := DefaultExecutor.GetReplicationController(id)
	if err != nil {
		if sErr, ok := err.(*errors.StatusError); ok && sErr.ErrStatus.Reason == api.StatusReasonNotFound {
			return pmxadapter.ServiceDeployment{}, pmxadapter.NewError(http.StatusNotFound, err.Error())
		}

		pmxErr := pmxadapter.NewError(http.StatusInternalServerError, err.Error())
		return pmxadapter.ServiceDeployment{}, pmxErr
	}

	sd := pmxadapter.ServiceDeployment{
		ID:          rc.ObjectMeta.Name,
		ActualState: statusFromReplicationController(rc),
	}
	return sd, nil
}

func (a KubernetesAdapter) CreateServices(services []*pmxadapter.Service) ([]pmxadapter.ServiceDeployment, *pmxadapter.Error) {
	deployments := make([]pmxadapter.ServiceDeployment, len(services))

	for i, s := range services {
		safeName := sanitizeServiceName(s.Name)

		rcSpec := api.ReplicationController{
			ObjectMeta: api.ObjectMeta{
				Name: safeName,
			},
			Spec: api.ReplicationControllerSpec{
				Replicas: s.Deployment.Count,
				Selector: map[string]string{"name": safeName},
				Template: &api.PodTemplateSpec{
					ObjectMeta: api.ObjectMeta{
						Labels: map[string]string{"name": safeName},
					},
					Spec: api.PodSpec{
						Containers: []api.Container{
							{
								// You're still missing a huge number of these things, from
								// Brian's 'manifest'.
								//container[:command] = command if command
								//container[:ports] = port_mapping if ports.any?
								//container[:env] = environment_mapping if environment.any?
								Name:    safeName,
								Image:   s.Source,
								Command: []string{s.Command},
							},
						},
					},
				},
			},
		}

		rc, err := DefaultExecutor.CreateReplicationController(rcSpec)
		if err != nil {
			if sErr, ok := err.(*errors.StatusError); ok && sErr.ErrStatus.Reason == api.StatusReasonAlreadyExists {
				return nil, pmxadapter.NewError(http.StatusConflict, err.Error())
			}
			return nil, pmxadapter.NewError(http.StatusInternalServerError, err.Error())
		}

		deployments[i].ID = rc.ObjectMeta.Name
		deployments[i].ActualState = statusFromReplicationController(rc)
	}

	return deployments, nil
}

func (a KubernetesAdapter) DestroyService(id string) *pmxadapter.Error {
	err := DefaultExecutor.DeleteReplicationController(id)
	if err != nil {
		if sErr, ok := err.(*errors.StatusError); ok && sErr.ErrStatus.Reason == api.StatusReasonNotFound {
			return pmxadapter.NewError(http.StatusNotFound, err.Error())
		}

		return pmxadapter.NewError(http.StatusInternalServerError, err.Error())
	}

	return nil
}

func (a KubernetesAdapter) GetMetadata() pmxadapter.Metadata {
	return pmxadapter.Metadata{
		Version:   metadataVersion,
		Type:      metadataType,
		IsHealthy: DefaultExecutor.IsHealthy(),
	}
}

func sanitizeServiceName(n string) string {
	s := illegalNameCharacters.ReplaceAllString(n, "-")
	return strings.ToLower(s)
}

func statusFromReplicationController(rc api.ReplicationController) string {
	desired := rc.Spec.Replicas
	actual := rc.Status.Replicas

	if actual < desired {
		return "pending"
	} else if desired == actual {
		return "running"
	}
	return "unknown"
}
