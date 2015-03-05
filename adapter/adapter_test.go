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
	RCs                []api.ReplicationController
	CreatedSpec        api.ReplicationController
	CreationError      error
	GetServicesError   error
	GetServiceError    error
	DeletionError      error
	DestroyedServiceID string
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

	if e.CreationError != nil {
		return api.ReplicationController{}, e.CreationError
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
	sds, pmxErr := adapter.GetServices()
	assert.Nil(t, pmxErr)
	if assert.Len(t, sds, 1) {
		assert.Equal(t, "test-service", sds[0].ID)
		assert.Equal(t, "pending", sds[0].ActualState)
	}
}

func TestErroredGetServices(t *testing.T) {
	setup()
	te.GetServicesError = errors.New("test error")
	sds, pmxErr := adapter.GetServices()
	assert.Empty(t, sds)
	if assert.NotNil(t, pmxErr) {
		assert.Equal(t, http.StatusInternalServerError, pmxErr.Code)
		assert.Equal(t, "test error", pmxErr.Message)
	}
}

func TestSuccessfulGetService(t *testing.T) {
	setupRCs()
	sd, pmxErr := adapter.GetService("test-service")

	assert.Nil(t, pmxErr)
	assert.Equal(t, pmxadapter.ServiceDeployment{ID: "test-service", ActualState: "pending"}, sd)
}

func TestErroredNotFoundGetService(t *testing.T) {
	setup()
	te.GetServiceError = kerrors.NewNotFound("thing", "name")
	sd, pmxErr := adapter.GetService("UnknownID")

	assert.Equal(t, pmxadapter.ServiceDeployment{}, sd)
	if assert.NotNil(t, pmxErr) {
		assert.Equal(t, `thing "name" not found`, pmxErr.Message)
		assert.Equal(t, http.StatusNotFound, pmxErr.Code)
	}
}

func TestErroredGetService(t *testing.T) {
	setup()
	te.GetServiceError = errors.New("test error")
	sd, pmxErr := adapter.GetService("TestID")

	assert.Equal(t, pmxadapter.ServiceDeployment{}, sd)
	if assert.NotNil(t, pmxErr) {
		assert.Equal(t, "test error", pmxErr.Message)
		assert.Equal(t, http.StatusInternalServerError, pmxErr.Code)
	}
}

func servicesSetup() {
	setup()
	services = []*pmxadapter.Service{
		{Name: "Test Service", Source: "redis", Deployment: pmxadapter.Deployment{Count: 1}},
	}
}

func TestSuccessfulCreateServices(t *testing.T) {
	servicesSetup()
	sd, pmxErr := adapter.CreateServices(services)

	assert.Nil(t, pmxErr)
	assert.Equal(t, "test-service", te.CreatedSpec.ObjectMeta.Name)
	assert.Equal(t, 1, te.CreatedSpec.Spec.Replicas)
	cs := te.CreatedSpec.Spec.Template.Spec.Containers
	if assert.Len(t, cs, 1) {
		assert.Equal(t, "test-service", cs[0].Name)
		assert.Equal(t, "redis", cs[0].Image)
	}
	if assert.Len(t, sd, 1) {
		assert.Equal(t, "test-service", sd[0].ID)
		assert.Equal(t, "pending", sd[0].ActualState)
	}
}

func TestErroredCreateServices(t *testing.T) {
	servicesSetup()
	te.CreationError = errors.New("test error")
	sd, pmxErr := adapter.CreateServices(services)

	assert.Len(t, sd, 0)
	if assert.NotNil(t, pmxErr) {
		assert.Equal(t, http.StatusInternalServerError, pmxErr.Code)
		assert.Equal(t, "test error", pmxErr.Message)
	}
}

func TestErroredConflictedCreateServices(t *testing.T) {
	servicesSetup()
	te.CreationError = kerrors.NewAlreadyExists("thing", "name")
	sd, pmxErr := adapter.CreateServices(services)

	assert.Len(t, sd, 0)
	if assert.NotNil(t, pmxErr) {
		assert.Equal(t, http.StatusConflict, pmxErr.Code)
		assert.Equal(t, te.CreationError.Error(), pmxErr.Message)
	}
}

func TestSuccessfulDestroyService(t *testing.T) {
	setupRCs()
	pmxErr := adapter.DestroyService("test-service")

	assert.Nil(t, pmxErr)
	assert.Equal(t, "test-service", te.DestroyedServiceID)
}

func TestErroredNotFoundDestroyService(t *testing.T) {
	setup()
	te.DeletionError = kerrors.NewNotFound("thing", "name")
	pmxErr := adapter.DestroyService("test-service")

	if assert.NotNil(t, pmxErr) {
		assert.Equal(t, `thing "name" not found`, pmxErr.Message)
		assert.Equal(t, http.StatusNotFound, pmxErr.Code)
	}
}

func TestErroredDestroyService(t *testing.T) {
	setup()
	te.DeletionError = errors.New("test error")
	pmxErr := adapter.DestroyService("test-service")

	if assert.NotNil(t, pmxErr) {
		assert.Equal(t, http.StatusInternalServerError, pmxErr.Code)
		assert.Equal(t, "test error", pmxErr.Message)
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
