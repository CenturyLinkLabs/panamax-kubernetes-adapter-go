package adapter

import (
	"errors"
	"testing"

	"github.com/CenturyLinkLabs/pmxadapter"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/stretchr/testify/assert"
)

type TestExecutor struct {
	Spec          api.ReplicationController
	CreationError error
}

func (e *TestExecutor) CreateReplicationController(spec api.ReplicationController) (string, error) {
	e.Spec = spec

	if e.CreationError != nil {
		return "", e.CreationError
	}
	return "Unused String", nil
}

var (
	adapter  KubernetesAdapter
	te       TestExecutor
	services []*pmxadapter.Service
)

func setup() {
	adapter = KubernetesAdapter{}
	te = TestExecutor{}
	services = []*pmxadapter.Service{{Name: "Test Service", Source: "redis"}}
	DefaultExecutor = &te
}

func TestSuccessfulCreateServices(t *testing.T) {
	setup()
	// TODO test return value once it's fixed
	_, pmxErr := adapter.CreateServices(services)

	assert.Nil(t, pmxErr)
	assert.Equal(t, "test-service", te.Spec.ObjectMeta.Name)
	cs := te.Spec.Spec.Template.Spec.Containers
	if assert.Len(t, cs, 1) {
		assert.Equal(t, "test-service", cs[0].Name)
		assert.Equal(t, "redis", cs[0].Image)
	}
}

func TestErroredCreateServices(t *testing.T) {
	setup()
	te.CreationError = errors.New("test error")
	// TODO test return value once it's fixed
	_, pmxErr := adapter.CreateServices(services)
	if assert.NotNil(t, pmxErr) {
		assert.Equal(t, 500, pmxErr.Code)
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
