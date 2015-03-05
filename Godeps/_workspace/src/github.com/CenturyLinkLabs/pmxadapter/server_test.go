package pmxadapter

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testServer *httptest.Server

func init() {
	martini := NewServer(new(NoOPAdapter)).(*martiniServer)
	testServer = httptest.NewServer(martini.svr)
}

type NoOPAdapter struct {
}

func (NoOPAdapter) GetServices() ([]ServiceDeployment, *Error) {
	return []ServiceDeployment{}, nil
}
func (NoOPAdapter) GetService(string) (ServiceDeployment, *Error) {
	return ServiceDeployment{}, nil
}
func (NoOPAdapter) CreateServices([]*Service) ([]ServiceDeployment, *Error) {
	return make([]ServiceDeployment, 0), nil
}
func (NoOPAdapter) DestroyService(string) *Error {
	return nil
}
func (NoOPAdapter) GetMetadata() Metadata {
	return Metadata{Type: "NOOP", Version: "0.1"}
}

func TestGetServicesRoute(t *testing.T) {
	res, _ := http.Get(fmt.Sprintf("%s/v1/services", testServer.URL))

	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestGetServiceRoute(t *testing.T) {
	res, _ := http.Get(fmt.Sprintf("%s/v1/services/1", testServer.URL))

	assert.Equal(t, http.StatusOK, res.StatusCode)
}

//func TestPostServiceRoute(t *testing.T) {
//  var body io.Reader
//  res, _ := http.Post(fmt.Sprintf("%s/v1/services", testServer.URL), "", body)

//  assert.Equal(t, http.StatusCreated, res.StatusCode)
//}

func TestPutServiceRoute(t *testing.T) {
	var body io.Reader
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/v1/services/1", testServer.URL), body)
	res, _ := http.DefaultClient.Do(req)

	assert.Equal(t, http.StatusNotImplemented, res.StatusCode)
}

func TestDeleteServiceRoute(t *testing.T) {
	var body io.Reader
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/services/1", testServer.URL), body)
	res, _ := http.DefaultClient.Do(req)

	assert.Equal(t, http.StatusNoContent, res.StatusCode)
}

func TestGetMetadataRoute(t *testing.T) {
	res, _ := http.Get(fmt.Sprintf("%s/v1/metadata", testServer.URL))
	m := Metadata{}
	j, _ := ioutil.ReadAll(res.Body)
	err := json.Unmarshal(j, &m)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "NOOP", m.Type)
	assert.Equal(t, "0.1", m.Version)
}

func TestNoRoute(t *testing.T) {
	res, _ := http.Get(fmt.Sprintf("%s/v1/nothere", testServer.URL))

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}
