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
	Deployments     []pmxadapter.ServiceDeployment
	Spec            api.ReplicationController
	CreationError   error
	GetServiceError error
}

func (e *TestExecutor) GetReplicationController(id string) (string, string, error) {
	if e.GetServiceError != nil {
		return "", "", e.GetServiceError
	}

	for _, d := range e.Deployments {
		if d.ID == id {
			return d.ID, d.ActualState, nil
		}
	}
	return "", "", nil
}

func (e *TestExecutor) CreateReplicationController(spec api.ReplicationController) (string, string, error) {
	e.Spec = spec

	if e.CreationError != nil {
		return "", "", e.CreationError
	}
	return "TestID", "TestStatus", nil
}

var (
	adapter  KubernetesAdapter
	te       TestExecutor
	services []*pmxadapter.Service
)

func setup() {
	adapter = KubernetesAdapter{}
	te = TestExecutor{}
	DefaultExecutor = &te
}

func servicesSetup() {
	setup()
	services = []*pmxadapter.Service{
		{Name: "Test Service", Source: "redis", Deployment: pmxadapter.Deployment{Count: 1}},
	}
}

func TestSatisfiesAdapterInterface(t *testing.T) {
	assert.Implements(t, (*pmxadapter.PanamaxAdapter)(nil), adapter)
}

func TestSuccessfulGetService(t *testing.T) {
	setup()
	expected := pmxadapter.ServiceDeployment{ID: "TestID", ActualState: "Testing"}
	te.Deployments = []pmxadapter.ServiceDeployment{expected}
	sd, pmxErr := adapter.GetService("TestID")

	assert.Nil(t, pmxErr)
	assert.Equal(t, expected, sd)
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

func TestSuccessfulCreateServices(t *testing.T) {
	servicesSetup()
	sd, pmxErr := adapter.CreateServices(services)

	assert.Nil(t, pmxErr)
	assert.Equal(t, "test-service", te.Spec.ObjectMeta.Name)
	assert.Equal(t, 1, te.Spec.Spec.Replicas)
	cs := te.Spec.Spec.Template.Spec.Containers
	if assert.Len(t, cs, 1) {
		assert.Equal(t, "test-service", cs[0].Name)
		assert.Equal(t, "redis", cs[0].Image)
	}
	if assert.Len(t, sd, 1) {
		assert.Equal(t, "TestID", sd[0].ID)
		assert.Equal(t, "TestStatus", sd[0].ActualState)
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

func TestSanitizeServiceName(t *testing.T) {
	assert.Equal(t, "test", sanitizeServiceName("test"))
	assert.Equal(t, "test", sanitizeServiceName("Test"))
	assert.Equal(t, "test-service", sanitizeServiceName("Test Service"))
	assert.Equal(t, "test-service", sanitizeServiceName("Test_Service"))
	assert.Equal(t, "test-service", sanitizeServiceName("Test _ \n  Service"))
}
