package adapter

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
)

const namespace = "default"

type Executor interface {
	GetReplicationControllers() ([]api.ReplicationController, error)
	GetReplicationController(string) (api.ReplicationController, error)
	CreateReplicationController(api.ReplicationController) (api.ReplicationController, error)
	DeleteReplicationController(string) error
}

type KubernetesExecutor struct {
	APIEndpoint string
}

func NewKubernetesExecutor(url string) Executor {
	return KubernetesExecutor{APIEndpoint: url}
}

func (k KubernetesExecutor) GetReplicationControllers() ([]api.ReplicationController, error) {
	// TODO hello duplication. Figure out client instantiation.
	client, err := client.New(&client.Config{Host: k.APIEndpoint})
	if err != nil {
		return []api.ReplicationController{}, err
	}

	rcList, err := client.ReplicationControllers(namespace).List(labels.Everything())
	if err != nil {
		return []api.ReplicationController{}, err
	}

	return rcList.Items, nil
}

func (k KubernetesExecutor) GetReplicationController(id string) (api.ReplicationController, error) {
	// TODO hello duplication. Figure out client instantiation.
	client, err := client.New(&client.Config{Host: k.APIEndpoint})
	if err != nil {
		return api.ReplicationController{}, err
	}

	rc, err := client.ReplicationControllers(namespace).Get(id)
	if err != nil {
		return api.ReplicationController{}, err
	}

	return *rc, nil
}

func (k KubernetesExecutor) CreateReplicationController(spec api.ReplicationController) (api.ReplicationController, error) {
	client, err := client.New(&client.Config{Host: k.APIEndpoint})
	if err != nil {
		return api.ReplicationController{}, err
	}
	rc, err := client.ReplicationControllers(namespace).Create(&spec)
	if err != nil {
		return api.ReplicationController{}, err
	}

	return *rc, nil
}

func (k KubernetesExecutor) DeleteReplicationController(id string) error {
	// TODO hello duplication. Figure out client instantiation.
	client, err := client.New(&client.Config{Host: k.APIEndpoint})
	if err != nil {
		return err
	}

	rc, err := k.GetReplicationController(id)
	if err != nil {
		return err
	}

	rc.Spec.Replicas = 0
	if _, err := client.ReplicationControllers(namespace).Update(&rc); err != nil {
		return err
	}

	if err := client.ReplicationControllers(namespace).Delete(id); err != nil {
		return err
	}

	return nil
}
