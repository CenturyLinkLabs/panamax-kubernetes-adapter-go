package adapter

import (
	"github.com/CenturyLinkLabs/pmxadapter"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

const (
	namespace          = "default"
	multiplePortsError = "multiple ports from a single container is not currently supported"
)

type Executor interface {
	GetReplicationControllers() ([]api.ReplicationController, error)
	GetReplicationController(string) (api.ReplicationController, error)
	CreateReplicationController(api.ReplicationController) (api.ReplicationController, error)
	DeleteReplicationController(string) error
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

func (k KubernetesExecutor) CreateReplicationController(spec api.ReplicationController) (api.ReplicationController, error) {
	// Once K8s allows multiple ports per service, we can lift the restriction on
	// a single port and the rest of this code (creation and deletion) ought to
	// work fine because it's already looping through ports and using labels to
	// determine what to delete.
	ports := portsFromReplicationController(spec)
	if len(ports) > 1 {
		return api.ReplicationController{}, pmxadapter.NewAlreadyExistsError(multiplePortsError)
	}

	rc, err := k.client.ReplicationControllers(namespace).Create(&spec)
	if err != nil {
		return api.ReplicationController{}, err
	}

	rcName := spec.ObjectMeta.Name
	for _, p := range ports {
		serviceName := rcName
		serviceSpec := api.Service{
			ObjectMeta: api.ObjectMeta{
				Name:   serviceName,
				Labels: map[string]string{"service-name": rcName},
			},
			Spec: api.ServiceSpec{
				Port:          p.HostPort,
				Protocol:      p.Protocol,
				ContainerPort: util.NewIntOrStringFromInt(p.ContainerPort),
				Selector:      map[string]string{"service-name": rcName},
			},
		}
		_, err := k.client.Services(namespace).Create(&serviceSpec)
		if err != nil {
			return *rc, err
		}
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

func (k KubernetesExecutor) IsHealthy() bool {
	if _, err := k.client.Nodes().List(); err != nil {
		return false
	}

	return true
}

func portsFromReplicationController(rc api.ReplicationController) []api.Port {
	ports := make([]api.Port, 0)

	for _, c := range rc.Spec.Template.Spec.Containers {
		for _, p := range c.Ports {
			ports = append(ports, p)
		}
	}

	return ports
}
