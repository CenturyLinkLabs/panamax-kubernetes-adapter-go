package adapter

import (
	"log"
	"os"
	"regexp"

	"github.com/CenturyLinkLabs/pmxadapter"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
)

const (
	metadataType = "Kubernetes"
)

var (
	metadataVersion       string
	DefaultExecutor       Executor
	illegalNameCharacters = regexp.MustCompile(`[\W_]+`)
	PublicIPs             []string
)

func init() {
	metadataVersion = os.Getenv("ADAPTER_VERSION")

	if publicIP := os.Getenv("SERVICE_PUBLIC_IP"); publicIP != "" {
		PublicIPs = []string{publicIP}
	}

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

func statusFromReplicationController(rc api.ReplicationController) string {
	desired := rc.Spec.Replicas
	actual := rc.Status.Replicas

	if actual < desired {
		return "pending"
	} else if desired == actual {
		return "scheduled"
	}
	return "unknown"
}
