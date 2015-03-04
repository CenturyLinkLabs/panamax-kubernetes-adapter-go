package adapter

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
)

const namespace = "default"

type Executor interface {
	GetReplicationController(string) (string, string, error)
	CreateReplicationController(api.ReplicationController) (string, string, error)
}

type KubernetesExecutor struct {
	APIEndpoint string
}

func NewKubernetesExecutor(url string) Executor {
	return KubernetesExecutor{APIEndpoint: url}
}

func (k KubernetesExecutor) GetReplicationController(id string) (string, string, error) {
	// TODO hello duplication. Figure out client instantiation.
	client, err := client.New(&client.Config{Host: k.APIEndpoint})
	if err != nil {
		return "", "", err
	}

	rc, err := client.ReplicationControllers(namespace).Get(id)
	if err != nil {
		return "", "", err
	}

	status := "pending"
	if rc.Spec.Replicas == rc.Status.Replicas {
		status = "running"
	}

	return rc.ObjectMeta.Name, status, nil
}

func (k KubernetesExecutor) CreateReplicationController(spec api.ReplicationController) (string, string, error) {
	client, err := client.New(&client.Config{Host: k.APIEndpoint})
	if err != nil {
		return "", "", err
	}
	_, err = client.ReplicationControllers(namespace).Create(&spec)
	if err != nil {
		return "", "", err
	}

	id := spec.ObjectMeta.Name
	return id, "pending", nil
}
