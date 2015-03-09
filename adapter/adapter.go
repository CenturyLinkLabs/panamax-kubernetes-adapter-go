package adapter

import (
	"log"
	"os"
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
	e, err := NewKubernetesExecutor(
		os.Getenv("KUBERNETES_MASTER"),
		os.Getenv("KUBERNETES_USERNAME"),
		os.Getenv("KUBERNETES_PASSWORD"),
	)
	if err != nil {
		log.Fatalf("There was a problem with your Kubernetes connection: %v", err)
	}

	DefaultExecutor = e
}

type KubernetesAdapter struct{}

func (a KubernetesAdapter) GetServices() ([]pmxadapter.ServiceDeployment, error) {
	rcs, err := DefaultExecutor.GetReplicationControllers()
	if err != nil {
		return []pmxadapter.ServiceDeployment{}, err
	}

	sds := make([]pmxadapter.ServiceDeployment, len(rcs))
	for i, rc := range rcs {
		sds[i].ID = rc.ObjectMeta.Name
		sds[i].ActualState = statusFromReplicationController(rc)
	}
	return sds, nil
}

func (a KubernetesAdapter) GetService(id string) (pmxadapter.ServiceDeployment, error) {
	rc, err := DefaultExecutor.GetReplicationController(id)
	if err != nil {
		if sErr, ok := err.(*errors.StatusError); ok && sErr.ErrStatus.Reason == api.StatusReasonNotFound {
			return pmxadapter.ServiceDeployment{}, pmxadapter.NewNotFoundError(err.Error())
		}

		return pmxadapter.ServiceDeployment{}, err
	}

	sd := pmxadapter.ServiceDeployment{
		ID:          rc.ObjectMeta.Name,
		ActualState: statusFromReplicationController(rc),
	}
	return sd, nil
}

func (a KubernetesAdapter) CreateServices(services []*pmxadapter.Service) ([]pmxadapter.ServiceDeployment, error) {
	deployments := make([]pmxadapter.ServiceDeployment, len(services))

	for i, s := range services {
		rcSpec := replicationControllerSpecFromService(*s)
		rc, err := DefaultExecutor.CreateReplicationController(rcSpec)
		if err != nil {
			if sErr, ok := err.(*errors.StatusError); ok && sErr.ErrStatus.Reason == api.StatusReasonAlreadyExists {
				return nil, pmxadapter.NewAlreadyExistsError(err.Error())
			}
			return nil, err
		}

		deployments[i].ID = rc.ObjectMeta.Name
		deployments[i].ActualState = statusFromReplicationController(rc)
	}

	return deployments, nil
}

func (a KubernetesAdapter) DestroyService(id string) error {
	err := DefaultExecutor.DeleteReplicationController(id)
	if err != nil {
		if sErr, ok := err.(*errors.StatusError); ok && sErr.ErrStatus.Reason == api.StatusReasonNotFound {
			return pmxadapter.NewNotFoundError(err.Error())
		}

		return err
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

func replicationControllerSpecFromService(s pmxadapter.Service) api.ReplicationController {
	ports := make([]api.Port, len(s.Ports))
	for i, p := range s.Ports {
		ports[i].HostPort = int(p.HostPort)
		ports[i].ContainerPort = int(p.ContainerPort)
		ports[i].Protocol = api.Protocol(p.Protocol)
	}

	env := make([]api.EnvVar, len(s.Environment))
	for i, e := range s.Environment {
		env[i].Name = e.Variable
		env[i].Value = e.Value
	}

	safeName := sanitizeServiceName(s.Name)

	return api.ReplicationController{
		ObjectMeta: api.ObjectMeta{
			Name: safeName,
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: s.Deployment.Count,
			Selector: map[string]string{"service-name": safeName},
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{"service-name": safeName},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:    safeName,
							Image:   s.Source,
							Command: []string{s.Command},
							Ports:   ports,
							Env:     env,
						},
					},
				},
			},
		},
	}
}
