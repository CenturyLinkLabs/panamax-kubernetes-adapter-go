package adapter

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/CenturyLinkLabs/pmxadapter"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
)

const (
	metadataType = "Kubernetes"
)

var (
	metadataVersion       string
	DefaultExecutor       Executor
	illegalNameCharacters = regexp.MustCompile(`[\W_]+`)
	PublicIPs             []string
	DeletionWaitTime      = 1 * time.Second
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
		status, err := statusFromReplicationController(rc)
		if err != nil {
			return []pmxadapter.ServiceDeployment{}, err
		}

		sds[i].ID = rc.ObjectMeta.Name
		sds[i].ActualState = status
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

	status, err := statusFromReplicationController(rc)
	if err != nil {
		return pmxadapter.ServiceDeployment{}, err
	}
	sd := pmxadapter.ServiceDeployment{
		ID:          rc.ObjectMeta.Name,
		ActualState: status,
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

	// This is a timely (pun intended) hack to make sure that redeployment
	// through the agent doesn't fail. The call above must return before the pods
	// are actually deleted. Ideally I'd poll the list of resources until they're
	// all really gone, but this also gets the job done.
	time.Sleep(DeletionWaitTime)

	return nil
}

func (a KubernetesAdapter) GetMetadata() pmxadapter.Metadata {
	return pmxadapter.Metadata{
		Version:   metadataVersion,
		Type:      metadataType,
		IsHealthy: DefaultExecutor.IsHealthy(),
	}
}

func statusFromReplicationController(rc api.ReplicationController) (string, error) {
	desired := rc.Spec.Replicas
	actual := rc.Status.Replicas

	if actual < desired {
		return "pending", nil
	} else if desired == actual {
		selector := labels.OneTermEqualSelector("service-name", rc.ObjectMeta.Name)
		pods, err := DefaultExecutor.GetPods(selector)
		if err != nil {
			return "", err
		}
		runningCount := 0
		for _, p := range pods {
			if p.Status.Phase == api.PodRunning {
				runningCount++
			}
		}

		return fmt.Sprintf("running %v/%v", runningCount, desired), nil
	}

	return "unknown", nil
}
