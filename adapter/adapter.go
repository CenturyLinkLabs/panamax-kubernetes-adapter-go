package adapter

import "github.com/CenturyLinkLabs/pmxadapter"

type KubernetesAdapter struct{}

func (a KubernetesAdapter) GetServices() ([]*pmxadapter.Service, *pmxadapter.Error) {
	var apiErr *pmxadapter.Error
	services := make([]*pmxadapter.Service, 1)
	service := new(pmxadapter.Service)
	services[0] = service

	return services, apiErr
}

func (a KubernetesAdapter) GetService(id string) (*pmxadapter.Service, *pmxadapter.Error) {
	var apiErr *pmxadapter.Error
	service := new(pmxadapter.Service)

	return service, apiErr
}

func (a KubernetesAdapter) CreateServices(services []*pmxadapter.Service) ([]*pmxadapter.Service, *pmxadapter.Error) {
	var apiErr *pmxadapter.Error
	list := make([]*pmxadapter.Service, 1)
	service := new(pmxadapter.Service)
	list[0] = service

	return list, apiErr
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
