package pmxadapter

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testEncoder = new(jsonEncoder)

type MockAdapter struct {
	returnError *Error
}

func (e MockAdapter) GetServices() ([]ServiceDeployment, *Error) {
	return nil, e.returnError
}
func (e MockAdapter) GetService(string) (ServiceDeployment, *Error) {
	return ServiceDeployment{}, e.returnError
}
func (e MockAdapter) CreateServices([]*Service) ([]ServiceDeployment, *Error) {
	return nil, e.returnError
}
func (e MockAdapter) DestroyService(string) *Error {
	return e.returnError
}
func (e MockAdapter) GetMetadata() Metadata {
	return Metadata{Type: "mock", Version: "0.1"}
}

func newMockAdapter(code int, message string) *MockAdapter {
	adapter := new(MockAdapter)
	adapter.returnError = NewError(code, message)

	return adapter
}

func TestCodeOutOfRange(t *testing.T) {
	code, _ := getServices(testEncoder, newMockAdapter(9090, ""))
	assert.Equal(t, http.StatusInternalServerError, code)
}

func TestSuccessfulGetServices(t *testing.T) {
	code, _ := getServices(testEncoder, newMockAdapter(200, ""))

	assert.Equal(t, http.StatusOK, code)
}

func TestSuccessfulGetService(t *testing.T) {
	params := map[string]string{
		"id": "test",
	}

	code, _ := getService(testEncoder, newMockAdapter(200, ""), params)

	assert.Equal(t, http.StatusOK, code)
}

//func TestSuccessfulCreateServices(t *testing.T) {
//  req, _ := http.NewRequest("POST", "http://localhost", strings.NewReader("{}"))
//  code, _ := createServices(testEncoder, newMockAdapter(201, ""), req)

//  assert.Equal(t, http.StatusCreated, code)
//}

func TestErroredCreateServices(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://localhost", strings.NewReader("BAD JSON"))
	code, message := createServices(testEncoder, newMockAdapter(201, ""), req)

	assert.Equal(t, http.StatusInternalServerError, code)
	assert.Contains(t, message, "invalid character")
}

func TestSuccessfulUpdateService(t *testing.T) {
	req, _ := http.NewRequest("PUT", "http://localhost", strings.NewReader("{}"))
	params := map[string]string{
		"id": "test",
	}
	code, _ := updateService(newMockAdapter(501, ""), params, req)

	assert.Equal(t, http.StatusNotImplemented, code)
}

func TestSuccessfulDeleteService(t *testing.T) {
	params := map[string]string{
		"id": "test",
	}

	code, _ := deleteService(newMockAdapter(204, ""), params)

	assert.Equal(t, http.StatusNoContent, code)
}

func TestSuccessfulGetMetadata(t *testing.T) {
	code, _ := getMetadata(testEncoder, newMockAdapter(200, ""))

	assert.Equal(t, http.StatusOK, code)
}

func TestGetServicesError(t *testing.T) {
	adapter := newMockAdapter(500, "internal error")
	code, _ := getServices(testEncoder, adapter)

	assert.Equal(t, http.StatusInternalServerError, code)
}

func TestGetServiceNotFound(t *testing.T) {
	adapter := newMockAdapter(404, "service not found")
	params := map[string]string{
		"id": "test",
	}

	code, body := getService(testEncoder, adapter, params)

	assert.Equal(t, http.StatusNotFound, code)
	assert.Equal(t, "service not found", body)
}

//func TestCreateServicesError(t *testing.T) {
//  req, _ := http.NewRequest("POST", "http://localhost", strings.NewReader("{}"))
//  code, body := createServices(testEncoder, newMockAdapter(500, "internal error"), req)

//  assert.Equal(t, http.StatusInternalServerError, code)
//  assert.Equal(t, "internal error", body)
//}

func TestDeleteServiceNotFound(t *testing.T) {
	adapter := newMockAdapter(404, "service not found")
	params := map[string]string{
		"id": "test",
	}

	code, body := deleteService(adapter, params)

	assert.Equal(t, http.StatusNotFound, code)
	assert.Equal(t, "service not found", body)
}
