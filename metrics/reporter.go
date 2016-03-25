package metrics
import (
"net/http"
"github.com/cloudfoundry/gorouter/route"
"time"
)

//go:generate counterfeiter -o fakes/fake_reporter.go . ProxyReporter
type ProxyReporter interface {
	CaptureBadRequest(req *http.Request)
	CaptureBadGateway(req *http.Request)
	CaptureRoutingRequest(b *route.Endpoint, req *http.Request)
	CaptureRoutingResponse(b *route.Endpoint, res *http.Response, t time.Time, d time.Duration)
}

//go:generate counterfeiter -o fakes/fake_registry_reporter.go . RouteRegistryReporter
type RouteRegistryReporter interface {
	CaptureRouteStats(totalRoutes int, msSinceLastUpdate uint64)
	CaptureRegistryMessage(msg ComponentTagged)
}
