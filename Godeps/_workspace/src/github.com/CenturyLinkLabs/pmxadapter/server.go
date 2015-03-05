package pmxadapter

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/codegangsta/martini"
)

// The Version of the API exposed by AdapterServer.
const APIVersion = "v1"

// The AdapterServer serves your PanamaxAdapter-implementing adapter via the
// standard API that Panamax speaks.
type AdapterServer interface {
	Start()
}

type martiniServer struct {
	svr *martini.Martini
}

// NewServer creates an instance of a martini server. The
// adapterInst parameter is the adapter type the server will
// use when dispatching requests.
func NewServer(adapterInst PanamaxAdapter) AdapterServer {
	s := martini.New()

	// Setup middleware
	s.Use(martini.Recovery())
	s.Use(martini.Logger())
	s.Use(mapEncoder)
	s.Use(func(c martini.Context, w http.ResponseWriter, r *http.Request) {
		c.Map(adapterInst)
	})
	// Setup routes
	router := martini.NewRouter()
	router.Group(fmt.Sprintf("/%s", APIVersion), func(r martini.Router) {
		r.Get(`/services`, getServices)
		r.Get(`/services/:id`, getService)
		r.Post(`/services`, createServices)
		r.Put(`/services/:id`, updateService)
		r.Delete(`/services/:id`, deleteService)
		r.Get(`/metadata`, getMetadata)
	})

	// Add the router action
	s.Action(router.Handle)
	server := martiniServer{svr: s}

	return &server
}

// Start the server.
func (m *martiniServer) Start() {
	err := http.ListenAndServe(":8001", m.svr)
	if err != nil {
		log.Fatal(err)
	}
}

// The regex to check for the requested format (allows an optional trailing
// slash)
var rxExt = regexp.MustCompile(`(\.(?:json))\/?$`)

// MapEncoder intercepts the request's URL, detects the requested format,
// and injects the correct encoder dependency for this request. It rewrites
// the URL to remove the format extension, so that routes can be defined
// without it.
func mapEncoder(c martini.Context, w http.ResponseWriter, r *http.Request) {
	// Get the format extension
	matches := rxExt.FindStringSubmatch(r.URL.Path)
	ft := ".json"
	if len(matches) > 1 {
		// Rewrite the URL without the format extension
		l := len(r.URL.Path) - len(matches[1])
		if strings.HasSuffix(r.URL.Path, "/") {
			l--
		}
		r.URL.Path = r.URL.Path[:l]
		ft = matches[1]
	}
	// Inject the requested encoder
	switch ft {
	// Add cases for other formats
	default:
		c.MapTo(jsonEncoder{}, (*encoder)(nil))
		w.Header().Set("Content-Type", "application/json")
	}
}
