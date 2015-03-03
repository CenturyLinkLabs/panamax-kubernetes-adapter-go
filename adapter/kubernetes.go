package adapter

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
)

type Executor interface {
	CreateReplicationController(api.ReplicationController) (string, error)
}

type KubernetesExecutor struct {
	APIEndpoint string
}

func NewKubernetesExecutor(url string) Executor {
	return KubernetesExecutor{APIEndpoint: url}
}

func (k KubernetesExecutor) CreateReplicationController(spec api.ReplicationController) (string, error) {
	client, err := client.New(&client.Config{Host: k.APIEndpoint})
	if err != nil {
		return "", err
	}
	// TODO Capture this and figure out if there's a conflict, since that's an
	// allowable status according to our docs.
	_, err = client.ReplicationControllers("default").Create(&spec)
	if err != nil {
		return "", err
	}

	// TODO what about actualState? That's in the spec but Brian doesn't supply
	// it in response to Create. Meanwhile the pmxadapter seems to be combining
	// two JSON objects into one, and you're getting additional fields from the
	// service that aren't in the spec.
	return spec.ObjectMeta.Name + "-replication-controller", nil
}
