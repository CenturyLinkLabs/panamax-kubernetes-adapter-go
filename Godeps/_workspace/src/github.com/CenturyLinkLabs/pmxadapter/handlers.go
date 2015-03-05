package pmxadapter

import (
	"encoding/json"
	"net/http"

	"github.com/codegangsta/martini"
)

// The sanitizeErrorCode function is used to guarantee
// return codes conform to standard HTTP codes.
func sanitizeErrorCode(code int) int {
	if http.StatusText(code) == "" {
		return http.StatusInternalServerError
	}
	return code
}

// The handler to get a list of services.
//
// Will return a status of 200 if successful or
// an internal error if there is an error.
// Refer to https://github.com/CenturyLinkLabs/panamax-ui/wiki/Adapter-Developer's-Guide
func getServices(e encoder, adapter PanamaxAdapter) (int, string) {
	data, err := adapter.GetServices()
	if err != nil {
		return sanitizeErrorCode(err.Code), err.Message
	}

	return http.StatusOK, e.Encode(data)
}

// The handler to get a service by its unique id.
//
// It will return a status of 200 and the service if successful.
// If the service cannot be found the return code will be 404
// otherwide the return code will be some internal error.
//
// Refer to https://github.com/CenturyLinkLabs/panamax-ui/wiki/Adapter-Developer's-Guide
func getService(e encoder, adapter PanamaxAdapter, params martini.Params) (int, string) {
	id := params["id"]

	data, err := adapter.GetService(id)
	if err != nil {
		return sanitizeErrorCode(err.Code), err.Message
	}

	return http.StatusOK, e.Encode(data)
}

// The handler to create a list of services.
//
// Services posted to this handler will be created in order
// and if successful the response will contain a list of id and actualState
// for each provided service.
//
// Refer to https://github.com/CenturyLinkLabs/panamax-ui/wiki/Adapter-Developer's-Guide
func createServices(e encoder, adapter PanamaxAdapter, r *http.Request) (int, string) {
	var services []*Service
	err := json.NewDecoder(r.Body).Decode(&services)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}

	res, pmxErr := adapter.CreateServices(services)
	if pmxErr != nil {
		return sanitizeErrorCode(pmxErr.Code), pmxErr.Message
	}

	return http.StatusCreated, e.Encode(res)
}

// The handler to update a service.
// Currently this action will only return a not implemented status.
func updateService(adapter PanamaxAdapter, params martini.Params, r *http.Request) (int, string) {
	return http.StatusNotImplemented, ""
}

// The handler to remove a service.
//
// If successful the return code will be no content but
// any application error code will be returned.
//
// Refer to https://github.com/CenturyLinkLabs/panamax-ui/wiki/Adapter-Developer's-Guide
func deleteService(adapter PanamaxAdapter, params martini.Params) (int, string) {
	id := params["id"]

	err := adapter.DestroyService(id)
	if err != nil {
		return sanitizeErrorCode(err.Code), err.Message
	}

	return http.StatusNoContent, ""
}

// The getMetadata function is a utility method to report the
// version and type of adapter.
func getMetadata(e encoder, adapter PanamaxAdapter) (int, string) {

	data := adapter.GetMetadata()

	return http.StatusOK, e.Encode(&data)
}
