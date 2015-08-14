package route_service

import (
	"net/http"
	"strings"
)

func CfForwardedUrlFor(request *http.Request) string {
	schemes := request.Header.Get("X-Forwarded-Proto")
	firstScheme := strings.Split(schemes, ", ")[0]
	if firstScheme == "" {
		firstScheme = "http"
	}

	return firstScheme + "://" + request.Host + request.URL.RequestURI()
}
