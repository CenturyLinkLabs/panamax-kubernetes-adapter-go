package adapter

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
)

const (
	namespace          = "default"
	multiplePortsError = "multiple ports from a single container is not currently supported"
)

type Executor interface {
	GetReplicationControllers() ([]api.ReplicationController, error)
	GetReplicationController(string) (api.ReplicationController, error)
	GetPods(labels.Selector) ([]api.Pod, error)
	CreateReplicationController(api.ReplicationController) (api.ReplicationController, error)
	DeleteReplicationController(string) error
	CreateKServices([]api.Service) error
	IsHealthy() bool
}

type KubernetesExecutor struct {
	client *client.Client
}

func NewKubernetesExecutor(url string, username string, password string) (Executor, error) {
	config := client.Config{Host: url, Username: username, Password: password}
	client, err := client.New(&config)
	if err != nil {
		return KubernetesExecutor{}, err
	}

	return KubernetesExecutor{client: client}, nil
}

func (k KubernetesExecutor) GetReplicationControllers() ([]api.ReplicationController, error) {
	rcList, err := k.client.ReplicationControllers(namespace).List(labels.Everything())
	if err != nil {
		return []api.ReplicationController{}, err
	}

	return rcList.Items, nil
}

func (k KubernetesExecutor) GetReplicationController(id string) (api.ReplicationController, error) {
	rc, err := k.client.ReplicationControllers(namespace).Get(id)
	if err != nil {
		return api.ReplicationController{}, err
	}

	return *rc, nil
}

func (k KubernetesExecutor) GetPods(s labels.Selector) ([]api.Pod, error) {
	ps, err := k.client.Pods(namespace).List(s)
	return ps.Items, err
}

func (k KubernetesExecutor) CreateReplicationController(spec api.ReplicationController) (api.ReplicationController, error) {
	rc, err := k.client.ReplicationControllers(namespace).Create(&spec)
	if err != nil {
		return api.ReplicationController{}, err
	}

	return *rc, nil
}

func (k KubernetesExecutor) DeleteReplicationController(id string) error {
	// Maybe find the desired ReplicationController
	rc, err := k.GetReplicationController(id)
	if err != nil {
		return err
	}

	// Maybe find Services labeled for that ReplicationController
	forService := labels.OneTermEqualSelector("service-name", rc.ObjectMeta.Name)
	sl, err := k.client.Services(namespace).List(forService)
	if err != nil {
		return err
	}

	// Delete all found Services
	for _, s := range sl.Items {
		if err := k.client.Services(namespace).Delete(s.ObjectMeta.Name); err != nil {
			return err
		}
	}

	// Scale down the ReplicationController, deleting Pods
	rc.Spec.Replicas = 0
	if _, err := k.client.ReplicationControllers(namespace).Update(&rc); err != nil {
		return err
	}

	// Delete the ReplicationController
	if err := k.client.ReplicationControllers(namespace).Delete(id); err != nil {
		return err
	}

	// Profit?
	return nil
}

func (k KubernetesExecutor) CreateKServices(ks []api.Service) error {
	for _, s := range ks {
		_, err := k.client.Services(namespace).Create(&s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (k KubernetesExecutor) IsHealthy() bool {
	if _, err := k.client.Nodes().List(); err != nil {
		return false
	}

	return true
}
