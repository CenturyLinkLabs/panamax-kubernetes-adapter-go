package adapter

import (
	"errors"
	"net/http"
	"testing"

	"github.com/CenturyLinkLabs/pmxadapter"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kerrors "github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/stretchr/testify/assert"
)

func init() {
	DeletionWaitTime = 0
}

type TestExecutor struct {
	RCs                  []api.ReplicationController
	KServices            []api.Service
	Pods                 []api.Pod
	CreatedSpec          api.ReplicationController
	CreateRCError        error
	CreateKServicesError error
	GotPodsSelector      labels.Selector
	GetPodsError         error
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

func (e *TestExecutor) GetPods(s labels.Selector) ([]api.Pod, error) {
	e.GotPodsSelector = s
	return e.Pods, e.GetPodsError
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
	adapter KubernetesAdapter
	te      TestExecutor
	rcs     []api.ReplicationController
)

func adapterSetup() {
	adapter = KubernetesAdapter{}
	te = TestExecutor{}
	DefaultExecutor = &te
}

func TestSatisfiesAdapterInterface(t *testing.T) {
	assert.Implements(t, (*pmxadapter.PanamaxAdapter)(nil), adapter)
}

func setupRCs() {
	adapterSetup()
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
	adapterSetup()
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
	adapterSetup()
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
	adapterSetup()
	te.GetServiceError = errors.New("test error")
	sd, err := adapter.GetService("TestID")

	assert.Equal(t, pmxadapter.ServiceDeployment{}, sd)
	assert.EqualError(t, err, "test error")
}

func TestSuccessfulDestroyService(t *testing.T) {
	setupRCs()
	err := adapter.DestroyService("test-service")

	assert.NoError(t, err)
	assert.Equal(t, "test-service", te.DestroyedServiceID)
}

func TestErroredNotFoundDestroyService(t *testing.T) {
	adapterSetup()
	te.DeletionError = kerrors.NewNotFound("thing", "name")
	err := adapter.DestroyService("test-service")

	pmxErr, ok := err.(*pmxadapter.Error)
	if assert.Error(t, pmxErr) && assert.True(t, ok) {
		assert.Equal(t, te.DeletionError.Error(), pmxErr.Message)
		assert.Equal(t, http.StatusNotFound, pmxErr.Code)
	}
}

func TestErroredDestroyService(t *testing.T) {
	adapterSetup()
	te.DeletionError = errors.New("test error")
	err := adapter.DestroyService("test-service")

	assert.EqualError(t, err, "test error")
}

func TestSuccessfulGetMetadata(t *testing.T) {
	origVersion := metadataVersion
	metadataVersion = "3.9"
	adapterSetup()
	m := adapter.GetMetadata()
	metadataVersion = origVersion

	if assert.NotNil(t, m) {
		assert.Equal(t, metadataType, m.Type)
		assert.Equal(t, "3.9", m.Version)
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

func TestPendingStatusFromReplicationController(t *testing.T) {
	rc := api.ReplicationController{}
	rc.Spec.Replicas = 2
	rc.Status.Replicas = 1
	status, err := statusFromReplicationController(rc)

	assert.NoError(t, err)
	assert.Equal(t, "pending", status)
}

func TestUnkonwnStatusFromReplicationController(t *testing.T) {
	rc := api.ReplicationController{}
	rc.Spec.Replicas = 2
	rc.Status.Replicas = 3
	status, err := statusFromReplicationController(rc)

	assert.NoError(t, err)
	assert.Equal(t, "unknown", status)
}

func TestStartedStatusFromReplicationController(t *testing.T) {
	rc := api.ReplicationController{}
	rc.Spec.Replicas = 2
	rc.Status.Replicas = 2
	te.Pods = []api.Pod{
		{Status: api.PodStatus{Phase: api.PodPending}},
		{Status: api.PodStatus{Phase: api.PodPending}},
	}

	status, err := statusFromReplicationController(rc)
	assert.NoError(t, err)
	assert.Equal(t, "running 0/2", status)

	te.Pods[0].Status.Phase = api.PodRunning
	te.Pods[1].Status.Phase = api.PodRunning
	status, err = statusFromReplicationController(rc)
	assert.NoError(t, err)
	assert.Equal(t, "running 2/2", status)
}

func TestErroredStatusFromReplicationController(t *testing.T) {
	rc := api.ReplicationController{}
	rc.Spec.Replicas = 2
	rc.Status.Replicas = 2
	te.GetPodsError = errors.New("test error")
	status, err := statusFromReplicationController(rc)

	assert.Empty(t, status)
	assert.EqualError(t, err, "test error")
}
