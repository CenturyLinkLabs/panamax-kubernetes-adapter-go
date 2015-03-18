package adapter

import (
	"errors"
	"net/http"
	"testing"

	"github.com/CenturyLinkLabs/pmxadapter"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kerrors "github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/stretchr/testify/assert"
)

type TestExecutor struct {
	RCs                  []api.ReplicationController
	KServices            []api.Service
	CreatedSpec          api.ReplicationController
	CreateRCError        error
	CreateKServicesError error
	GetServicesError     error
	GetServiceError      error
	DeletionError        error
	DestroyedServiceID   string
	HealthCheckResult    bool
}

func (e *TestExecutor) GetReplicationControllers() ([]api.ReplicationController, error) {
	if e.GetServicesError != nil {
		return []api.ReplicationController{}, e.GetServicesError
	}

	return e.RCs, nil
}

func (e *TestExecutor) GetReplicationController(id string) (api.ReplicationController, error) {
	if e.GetServiceError != nil {
		return api.ReplicationController{}, e.GetServiceError
	}

	for _, rc := range e.RCs {
		if rc.ObjectMeta.Name == id {
			return rc, nil
		}
	}

	return api.ReplicationController{}, errors.New("Should never get here")
}

func (e *TestExecutor) CreateReplicationController(spec api.ReplicationController) (api.ReplicationController, error) {
	e.CreatedSpec = spec

	if e.CreateRCError != nil {
		return api.ReplicationController{}, e.CreateRCError
	}

	spec.Status.Replicas = 0
	return spec, nil
}

func (e *TestExecutor) DeleteReplicationController(id string) error {
	e.DestroyedServiceID = id
	if e.DeletionError != nil {
		return e.DeletionError
	}
	return nil
}

func (e *TestExecutor) CreateKServices(ks []api.Service) error {
	e.KServices = ks
	return e.CreateKServicesError
}

func (e *TestExecutor) IsHealthy() bool {
	return e.HealthCheckResult
}

var (
	adapter  KubernetesAdapter
	te       TestExecutor
	services []*pmxadapter.Service
	rcs      []api.ReplicationController
)

func setup() {
	adapter = KubernetesAdapter{}
	te = TestExecutor{}
	DefaultExecutor = &te
}

func TestSatisfiesAdapterInterface(t *testing.T) {
	assert.Implements(t, (*pmxadapter.PanamaxAdapter)(nil), adapter)
}

func setupRCs() {
	setup()
	rc := api.ReplicationController{
		ObjectMeta: api.ObjectMeta{Name: "test-service"},
		Spec:       api.ReplicationControllerSpec{Replicas: 1},
		Status:     api.ReplicationControllerStatus{Replicas: 0},
	}
	te.RCs = []api.ReplicationController{rc}
}

func TestSuccessfulGetServices(t *testing.T) {
	setupRCs()
	sds, err := adapter.GetServices()
	assert.NoError(t, err)
	if assert.Len(t, sds, 1) {
		assert.Equal(t, "test-service", sds[0].ID)
		assert.Equal(t, "pending", sds[0].ActualState)
	}
}

func TestErroredGetServices(t *testing.T) {
	setup()
	te.GetServicesError = errors.New("test error")
	sds, err := adapter.GetServices()
	assert.Empty(t, sds)
	assert.EqualError(t, err, "test error")
}

func TestSuccessfulGetService(t *testing.T) {
	setupRCs()
	sd, err := adapter.GetService("test-service")

	assert.NoError(t, err)
	assert.Equal(t, pmxadapter.ServiceDeployment{ID: "test-service", ActualState: "pending"}, sd)
}

func TestErroredNotFoundGetService(t *testing.T) {
	setup()
	te.GetServiceError = kerrors.NewNotFound("thing", "name")
	sd, err := adapter.GetService("UnknownID")

	assert.Equal(t, pmxadapter.ServiceDeployment{}, sd)
	pmxErr, ok := err.(*pmxadapter.Error)
	if assert.Error(t, pmxErr) && assert.True(t, ok) {
		assert.Equal(t, te.GetServiceError.Error(), pmxErr.Message)
		assert.Equal(t, http.StatusNotFound, pmxErr.Code)
	}
}

func TestErroredGetService(t *testing.T) {
	setup()
	te.GetServiceError = errors.New("test error")
	sd, err := adapter.GetService("TestID")

	assert.Equal(t, pmxadapter.ServiceDeployment{}, sd)
	assert.EqualError(t, err, "test error")
}

func servicesSetup() {
	setup()
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
	if assert.Len(t, sd, 1) {
		assert.Equal(t, "test-service", sd[0].ID)
		assert.Equal(t, "pending", sd[0].ActualState)
	}
	if assert.Len(t, te.KServices, 1) {
		assert.Equal(t, 31981, te.KServices[0].Spec.Port)
	}
}

func TestErroredKServiceCreationGetServices(t *testing.T) {
	servicesSetup()
	te.CreateKServicesError = errors.New("test error")
	sd, err := adapter.CreateServices(services)

	assert.Len(t, sd, 0)
	assert.EqualError(t, err, "test error")
}

func TestErroredRCCreationGetServices(t *testing.T) {
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
	}
}

//func TestSuccessfulAliasesKServicesFromServices(t *testing.T) {}
//func TestErroredMultiplePortsKServicesFromServices(t *testing.T) {}
//func TestErroredNonExposedLinkKServicesFromServices(t *testing.T) {}

func TestSuccessfulDestroyService(t *testing.T) {
	setupRCs()
	err := adapter.DestroyService("test-service")

	assert.NoError(t, err)
	assert.Equal(t, "test-service", te.DestroyedServiceID)
}

func TestErroredNotFoundDestroyService(t *testing.T) {
	setup()
	te.DeletionError = kerrors.NewNotFound("thing", "name")
	err := adapter.DestroyService("test-service")

	pmxErr, ok := err.(*pmxadapter.Error)
	if assert.Error(t, pmxErr) && assert.True(t, ok) {
		assert.Equal(t, te.DeletionError.Error(), pmxErr.Message)
		assert.Equal(t, http.StatusNotFound, pmxErr.Code)
	}
}

func TestErroredDestroyService(t *testing.T) {
	setup()
	te.DeletionError = errors.New("test error")
	err := adapter.DestroyService("test-service")

	assert.EqualError(t, err, "test error")
}

func TestSuccessfulGetMetadata(t *testing.T) {
	setup()
	m := adapter.GetMetadata()
	if assert.NotNil(t, m) {
		assert.Equal(t, metadataType, m.Type)
		assert.Equal(t, metadataVersion, m.Version)
		assert.False(t, m.IsHealthy)
	}

	te.HealthCheckResult = true
	m = adapter.GetMetadata()
	if assert.NotNil(t, m) {
		assert.True(t, m.IsHealthy)
	}
}

func TestSanitizeServiceName(t *testing.T) {
	assert.Equal(t, "test", sanitizeServiceName("test"))
	assert.Equal(t, "test", sanitizeServiceName("Test"))
	assert.Equal(t, "test-service", sanitizeServiceName("Test Service"))
	assert.Equal(t, "test-service", sanitizeServiceName("Test_Service"))
	assert.Equal(t, "test-service", sanitizeServiceName("Test _ \n  Service"))
}

func TestStatusFromReplicationController(t *testing.T) {
	rc := api.ReplicationController{}
	rc.Spec.Replicas = 2
	rc.Status.Replicas = 0
	assert.Equal(t, "pending", statusFromReplicationController(rc))
	rc.Status.Replicas = 1
	assert.Equal(t, "pending", statusFromReplicationController(rc))
	rc.Status.Replicas = 2
	assert.Equal(t, "running", statusFromReplicationController(rc))
	rc.Status.Replicas = 3
	assert.Equal(t, "unknown", statusFromReplicationController(rc))
}
