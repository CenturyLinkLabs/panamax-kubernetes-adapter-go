package adapter

import (
	"errors"
	"net/http"
	"testing"

	"github.com/CenturyLinkLabs/pmxadapter"
	kerrors "github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/stretchr/testify/assert"
)

var services []*pmxadapter.Service

func servicesSetup() {
	adapterSetup()
	services = []*pmxadapter.Service{
		{
			Name:        "Test Service",
			Source:      "redis",
			Command:     "redis-server",
			Environment: []*pmxadapter.Environment{{Variable: "VAR_NAME", Value: "Var Value"}},
			Ports:       []*pmxadapter.Port{{HostPort: 31981, ContainerPort: 12345, Protocol: "TCP"}},
			Deployment:  pmxadapter.Deployment{Count: 1},
		},
	}
}

func TestSuccessfulCreateServices(t *testing.T) {
	servicesSetup()
	sd, err := adapter.CreateServices(services)

	assert.NoError(t, err)
	assert.Equal(t, "test-service", te.CreatedSpec.ObjectMeta.Name)
	assert.Equal(t, 1, te.CreatedSpec.Spec.Replicas)
	if assert.Len(t, sd, 1) {
		assert.Equal(t, "test-service", sd[0].ID)
		assert.Equal(t, "pending", sd[0].ActualState)
	}
	if assert.Len(t, te.KServices, 1) {
		assert.Equal(t, 31981, te.KServices[0].Spec.Port)
	}
}

func TestSuccessfulSetMissingReplicasCreateServices(t *testing.T) {
	servicesSetup()
	services[0].Deployment.Count = 0
	_, err := adapter.CreateServices(services)

	assert.NoError(t, err)
	assert.Equal(t, 1, te.CreatedSpec.Spec.Replicas)
}

func TestErroredKServiceCreationCreateServices(t *testing.T) {
	servicesSetup()
	te.CreateKServicesError = errors.New("test error")
	sd, err := adapter.CreateServices(services)

	assert.Len(t, sd, 0)
	assert.EqualError(t, err, "test error")
}

func TestErroredRCCreationCreateServices(t *testing.T) {
	servicesSetup()
	te.CreateRCError = errors.New("test error")
	sd, err := adapter.CreateServices(services)

	assert.Len(t, sd, 0)
	assert.EqualError(t, err, "test error")
}

func TestErroredConflictedCreateServices(t *testing.T) {
	servicesSetup()
	te.CreateRCError = kerrors.NewAlreadyExists("thing", "name")
	sd, err := adapter.CreateServices(services)

	assert.Len(t, sd, 0)
	pmxErr, ok := err.(*pmxadapter.Error)
	if assert.Error(t, pmxErr) && assert.True(t, ok) {
		assert.Equal(t, te.CreateRCError.Error(), pmxErr.Message)
		assert.Equal(t, http.StatusConflict, pmxErr.Code)
	}
}

func TestReplicationControllerFromService(t *testing.T) {
	servicesSetup()
	spec := replicationControllerSpecFromService(*services[0])

	assert.Equal(t, "test-service", spec.ObjectMeta.Name)
	assert.Equal(t, 1, spec.Spec.Replicas)

	podTemplate := spec.Spec.Template
	labels := podTemplate.ObjectMeta.Labels
	assert.Equal(t, "test-service", labels["service-name"])
	assert.Equal(t, "panamax", labels["panamax"])

	containers := podTemplate.Spec.Containers
	if assert.Len(t, containers, 1) {
		c := containers[0]
		assert.Equal(t, "test-service", c.Name)
		assert.Equal(t, "redis", c.Image)

		if assert.Len(t, c.Command, 1) {
			assert.Equal(t, "redis-server", c.Command[0])
		}

		if assert.Len(t, c.Env, 1) {
			assert.Equal(t, "VAR_NAME", c.Env[0].Name)
			assert.Equal(t, "Var Value", c.Env[0].Value)
		}

		if assert.Len(t, c.Ports, 1) {
			assert.Equal(t, 31981, c.Ports[0].HostPort)
			assert.Equal(t, 12345, c.Ports[0].ContainerPort)
			assert.Equal(t, "TCP", c.Ports[0].Protocol)
		}
	}
}

func TestNoCommandReplicationControllerFromService(t *testing.T) {
	servicesSetup()
	services[0].Command = ""
	spec := replicationControllerSpecFromService(*services[0])

	containers := spec.Spec.Template.Spec.Containers
	if assert.Len(t, containers, 1) {
		assert.Empty(t, containers[0].Command)
	}
}

func TestSuccessfulBasicKServicesFromServices(t *testing.T) {
	servicesSetup()
	kServices, err := kServicesFromServices(services)

	assert.NoError(t, err)
	if assert.Len(t, kServices, 1) {
		ks := kServices[0]
		assert.Equal(t, "test-service", ks.ObjectMeta.Name)
		assert.Equal(t, "test-service", ks.ObjectMeta.Labels["service-name"])
		assert.Equal(t, 12345, ks.Spec.ContainerPort.IntVal)
		assert.Equal(t, 31981, ks.Spec.Port)
		assert.Equal(t, "TCP", ks.Spec.Protocol)
		assert.Equal(t, map[string]string{"panamax": "panamax"}, ks.Spec.Selector)
		assert.Empty(t, ks.Spec.PublicIPs)
	}
}

func TestSuccessfulPublicIPsKServicesFromServices(t *testing.T) {
	servicesSetup()
	originalPublicIPs := PublicIPs
	PublicIPs = []string{"10.0.0.1"}
	kServices, _ := kServicesFromServices(services)

	if assert.Len(t, kServices, 1) {
		if assert.Len(t, kServices[0].Spec.PublicIPs, 1) {
			assert.Equal(t, "10.0.0.1", kServices[0].Spec.PublicIPs[0])
		}
	}

	PublicIPs = originalPublicIPs
}

func TestSuccessfulAliasesKServicesFromServices(t *testing.T) {
	servicesSetup()
	aliasing := pmxadapter.Service{
		Name:   "Other Service",
		Source: "example",
		Links:  []*pmxadapter.Link{{Name: "Test Service", Alias: "Alt Name"}},
	}
	services = append(services, &aliasing)
	kServices, err := kServicesFromServices(services)

	assert.NoError(t, err)
	if assert.Len(t, kServices, 2) {
		def := kServices[0]
		assert.Equal(t, "test-service", def.Name)
		alias := kServices[1]
		assert.Equal(t, "alt-name", alias.Name)

		assert.Equal(t, def.Spec.ContainerPort, alias.Spec.ContainerPort)
		assert.Equal(t, def.Spec.Port, alias.Spec.Port)
		assert.Equal(t, def.Spec.Protocol, alias.Spec.Protocol)
	}
}

func TestNoErrorPortlessServiceKServicesFromServices(t *testing.T) {
	servicesSetup()
	services[0].Ports = make([]*pmxadapter.Port, 0)
	kServices, err := kServicesFromServices(services)

	assert.NoError(t, err)
	assert.Empty(t, kServices)
}

func TestSuccessfulLinkedButUnaliasedKServiceFromServices(t *testing.T) {
	servicesSetup()
	aliasing := pmxadapter.Service{
		Name:   "Other Service",
		Source: "example",
		Links:  []*pmxadapter.Link{{Name: "Test Service"}},
	}
	services = append(services, &aliasing)
	kServices, err := kServicesFromServices(services)

	assert.NoError(t, err)
	assert.Len(t, kServices, 1)
}

func TestErroredLinkedButNonexistantKServiceFromServices(t *testing.T) {
	servicesSetup()
	services := []*pmxadapter.Service{{
		Name:   "Service",
		Source: "example",
		Links:  []*pmxadapter.Link{{Name: "Bad", Alias: "Foo"}},
	}}
	kServices, err := kServicesFromServices(services)

	assert.Empty(t, kServices)
	assert.EqualError(t, err, "linking to non-existant service 'Bad'")
}

func TestErroredNonExposedLinkKServicesFromServices(t *testing.T) {
	servicesSetup()
	aliasing := pmxadapter.Service{
		Name:   "Other Service",
		Source: "example",
		Links:  []*pmxadapter.Link{{Name: "Test Service", Alias: "Foo"}},
	}
	services = append(services, &aliasing)
	services[0].Ports = make([]*pmxadapter.Port, 0)
	kServices, err := kServicesFromServices(services)

	assert.Empty(t, kServices)
	assert.EqualError(t, err, "linked-to service 'Test Service' exposes no ports")
}

func TestErroredMismatchedAliasesKServicesFromServices(t *testing.T) {
	servicesSetup()
	foo := pmxadapter.Service{
		Name:   "A",
		Source: "example",
		Ports:  []*pmxadapter.Port{{HostPort: 1, ContainerPort: 1, Protocol: "TCP"}},
		Links:  []*pmxadapter.Link{{Name: "Test Service", Alias: "Alt"}},
	}
	bar := pmxadapter.Service{
		Name:   "B",
		Source: "example",
		Links:  []*pmxadapter.Link{{Name: "A", Alias: "Alt"}},
	}
	services = append(services, &foo)
	services = append(services, &bar)
	kServices, err := kServicesFromServices(services)

	assert.Empty(t, kServices)
	assert.EqualError(t, err, "multiple services with the same alias name 'Alt'")
}

func TestErroredMultiplePortsKServicesFromServices(t *testing.T) {
	servicesSetup()
	p := pmxadapter.Port{HostPort: 1, ContainerPort: 1, Protocol: "TCP"}
	services[0].Ports = append(services[0].Ports, &p)

	kServices, err := kServicesFromServices(services)

	assert.Empty(t, kServices)
	assert.Contains(t, err.Error(), "multiple ports")
}
