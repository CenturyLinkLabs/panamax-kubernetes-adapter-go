package adapter

import (
	"fmt"
	"strings"

	"github.com/CenturyLinkLabs/pmxadapter"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
)

func (a KubernetesAdapter) CreateServices(services []*pmxadapter.Service) ([]pmxadapter.ServiceDeployment, error) {
	deployments := make([]pmxadapter.ServiceDeployment, len(services))
	// TODO destroy all services (and RCs I guess!) if there is an error
	// anywhere. Otherwise they'll be orphaned and screw up subsequent deploys.
	kServices, err := kServicesFromServices(services)
	if err != nil {
		return nil, err
	}
	if err := DefaultExecutor.CreateKServices(kServices); err != nil {
		return nil, err
	}

	for i, s := range services {
		rcSpec := replicationControllerSpecFromService(*s)
		rc, err := DefaultExecutor.CreateReplicationController(rcSpec)
		if err != nil {
			if sErr, ok := err.(*errors.StatusError); ok && sErr.ErrStatus.Reason == api.StatusReasonAlreadyExists {
				return nil, pmxadapter.NewAlreadyExistsError(err.Error())
			}
			return nil, err
		}

		deployments[i].ID = rc.ObjectMeta.Name
		deployments[i].ActualState = statusFromReplicationController(rc)
	}

	return deployments, nil
}

func replicationControllerSpecFromService(s pmxadapter.Service) api.ReplicationController {
	ports := make([]api.Port, len(s.Ports))
	for i, p := range s.Ports {
		ports[i].HostPort = int(p.HostPort)
		ports[i].ContainerPort = int(p.ContainerPort)
		ports[i].Protocol = api.Protocol(p.Protocol)
	}

	env := make([]api.EnvVar, len(s.Environment))
	for i, e := range s.Environment {
		env[i].Name = e.Variable
		env[i].Value = e.Value
	}

	safeName := sanitizeServiceName(s.Name)
	commands := make([]string, 0)
	if s.Command != "" {
		commands = append(commands, s.Command)
	}

	replicas := s.Deployment.Count
	// The adapter seems to be in charge of adjusting missing replica count from
	// the JSON. The UI doesn't allow selection of 0 replicas, so this shouldn't
	// screw things up in the current state.
	if replicas == 0 {
		replicas = 1
	}

	return api.ReplicationController{
		ObjectMeta: api.ObjectMeta{
			Name: safeName,
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: replicas,
			Selector: map[string]string{"service-name": safeName},
			Template: &api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{
						"service-name": safeName,
						"panamax":      "panamax",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:    safeName,
							Image:   s.Source,
							Command: commands,
							Ports:   ports,
							Env:     env,
						},
					},
				},
			},
		},
	}
}

func kServicesFromServices(services []*pmxadapter.Service) ([]api.Service, error) {
	if err := validateServicesPorts(services); err != nil {
		return nil, err
	}

	if err := validateServicesAliases(services); err != nil {
		return nil, err
	}

	servicesByName := map[string]pmxadapter.Service{}
	for _, s := range services {
		servicesByName[s.Name] = *s
	}
	kServices := make([]api.Service, 0)

	// Create KServices by name for any configured ports.
	for _, s := range services {
		if len(s.Ports) == 0 {
			continue
		}

		ks := kServiceByNameAndPort(
			sanitizeServiceName(s.Name),
			*s.Ports[0],
		)
		kServices = append(kServices, ks)
	}

	// Create KServices by alias for any links with aliases.
	for _, s := range services {
		for _, l := range s.Links {
			if l.Alias == "" {
				continue
			}

			toService, exists := servicesByName[l.Name]
			if !exists {
				return nil, fmt.Errorf("linking to non-existant service '%v'", l.Name)
			}

			if len(toService.Ports) == 0 {
				return nil, fmt.Errorf("linked-to service '%v' exposes no ports", l.Name)
			}

			ks := kServiceByNameAndPort(
				sanitizeServiceName(l.Alias),
				*toService.Ports[0],
			)
			kServices = append(kServices, ks)
		}
	}

	return kServices, nil
}

// Once K8s allows multiple ports per service, we can lift the restriction on
// a single port. We can't do anything about it now because we need to mimic
// current Docker environment variables while satisfying K8s's requirement
// for unique service names.
func validateServicesPorts(services []*pmxadapter.Service) error {
	for _, s := range services {
		if len(s.Ports) > 1 {
			return pmxadapter.NewAlreadyExistsError(multiplePortsError)
		}
	}

	return nil
}

// The same alias name to different services can't be supported.
func validateServicesAliases(services []*pmxadapter.Service) error {
	aliases := map[string]string{}
	for _, s := range services {
		for _, l := range s.Links {
			if l.Alias == "" {
				continue
			}

			if name, exists := aliases[l.Alias]; exists && name != l.Name {
				return fmt.Errorf("multiple services with the same alias name '%v'", l.Alias)
			}

			aliases[l.Alias] = l.Name
		}
	}

	return nil
}

func kServiceByNameAndPort(name string, p pmxadapter.Port) api.Service {
	return api.Service{
		ObjectMeta: api.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"service-name": name},
		},
		Spec: api.ServiceSpec{
			// I'm unaware of any wildcard selector, we don't have a name for the
			// overarching application being started, and I can't specifically
			// target only certain RCs because we don't know if a Service exists
			// solely to allow external access. Shrug.
			Selector:      map[string]string{"panamax": "panamax"},
			Port:          int(p.HostPort),
			ContainerPort: util.NewIntOrStringFromInt(int(p.ContainerPort)),
			Protocol:      api.Protocol(p.Protocol),
			PublicIPs:     PublicIPs,
		},
	}
}

func sanitizeServiceName(n string) string {
	s := illegalNameCharacters.ReplaceAllString(n, "-")
	return strings.ToLower(s)
}
