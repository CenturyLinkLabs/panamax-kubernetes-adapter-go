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
	Spec          api.ReplicationController
	CreationError error
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
	services = []*pmxadapter.Service{
		{Name: "Test Service", Source: "redis", Deployment: pmxadapter.Deployment{Count: 1}},
	}
	DefaultExecutor = &te
}

func TestSatisfiesAdapterInterface(t *testing.T) {
	assert.Implements(t, (*pmxadapter.PanamaxAdapter)(nil), adapter)
}

func TestSuccessfulCreateServices(t *testing.T) {
	setup()
	ds, pmxErr := adapter.CreateServices(services)

	assert.Nil(t, pmxErr)
	assert.Equal(t, "test-service", te.Spec.ObjectMeta.Name)
	assert.Equal(t, 1, te.Spec.Spec.Replicas)
	cs := te.Spec.Spec.Template.Spec.Containers
	if assert.Len(t, cs, 1) {
		assert.Equal(t, "test-service", cs[0].Name)
		assert.Equal(t, "redis", cs[0].Image)
	}
	if assert.Len(t, ds, 1) {
		assert.Equal(t, "TestID", ds[0].ID)
		assert.Equal(t, "TestStatus", ds[0].ActualState)
	}
}

func TestErroredCreateServices(t *testing.T) {
	setup()
	te.CreationError = errors.New("test error")
	ds, pmxErr := adapter.CreateServices(services)

	assert.Len(t, ds, 0)
	if assert.NotNil(t, pmxErr) {
		assert.Equal(t, http.StatusInternalServerError, pmxErr.Code)
		assert.Equal(t, "test error", pmxErr.Message)
	}
}

func TestErroredConflictedCreateServices(t *testing.T) {
	setup()
	te.CreationError = kerrors.NewAlreadyExists("thing", "name")
	ds, pmxErr := adapter.CreateServices(services)

	assert.Len(t, ds, 0)
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
