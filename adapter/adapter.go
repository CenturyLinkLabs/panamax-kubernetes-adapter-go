package adapter

import (
	"regexp"
	"strings"

	"github.com/CenturyLinkLabs/pmxadapter"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
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

func (a KubernetesAdapter) GetServices() ([]*pmxadapter.Service, *pmxadapter.Error) {
	return make([]*pmxadapter.Service, 0), nil
}

func (a KubernetesAdapter) GetService(id string) (*pmxadapter.Service, *pmxadapter.Error) {
	return &pmxadapter.Service{}, nil
}

func (a KubernetesAdapter) CreateServices(services []*pmxadapter.Service) ([]*pmxadapter.Service, *pmxadapter.Error) {
	for _, s := range services {
		safeName := sanitizeServiceName(s.Name)

		rcSpec := api.ReplicationController{
			ObjectMeta: api.ObjectMeta{
				Name: safeName,
			},
			Spec: api.ReplicationControllerSpec{
				Replicas: 1,
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

		// TODO You need to do something with the returned ID. These return objects
		// are wrong and just need to get fixed.
		_, err := DefaultExecutor.CreateReplicationController(rcSpec)
		if err != nil {
			services := make([]*pmxadapter.Service, 1)
			return services, pmxadapter.NewError(500, err.Error())
		}
	}

	return services, nil
}

func sanitizeServiceName(n string) string {
	s := illegalNameCharacters.ReplaceAllString(n, "-")
	return strings.ToLower(s)
}

func (a KubernetesAdapter) UpdateService(s *pmxadapter.Service) *pmxadapter.Error {
	return nil
}

func (a KubernetesAdapter) DestroyService(id string) *pmxadapter.Error {
	return nil
}

func (a KubernetesAdapter) GetMetadata() pmxadapter.Metadata {
	return pmxadapter.Metadata{Type: "Sample", Version: "0.1"}
}
