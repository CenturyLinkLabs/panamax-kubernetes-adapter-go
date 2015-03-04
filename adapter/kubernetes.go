package adapter

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
)

type Executor interface {
	CreateReplicationController(api.ReplicationController) (string, string, error)
}

type KubernetesExecutor struct {
	APIEndpoint string
}

func NewKubernetesExecutor(url string) Executor {
	return KubernetesExecutor{APIEndpoint: url}
}

func (k KubernetesExecutor) CreateReplicationController(spec api.ReplicationController) (string, string, error) {
	client, err := client.New(&client.Config{Host: k.APIEndpoint})
	if err != nil {
		return "", "", err
	}
	_, err = client.ReplicationControllers("default").Create(&spec)
	if err != nil {
		return "", "", err
	}

	id := spec.ObjectMeta.Name
	return id, "pending", nil
}
