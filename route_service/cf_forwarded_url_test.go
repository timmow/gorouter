package route_service_test

import (
	"net/http"

	"github.com/cloudfoundry/gorouter/route_service"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Route Service X-CF-Forwarded-Url", func() {
	Describe("ForwardedUrlFor", func() {
		Context("when the X-Forwarded-Proto header is not set", func() {
			It("uses http for the scheme", func() {
				request, err := http.NewRequest("GET", "http://test.com/", nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(route_service.CfForwardedUrlFor(request)).To(Equal("http://test.com/"))
			})
		})

		Context("when the X-Forwarded-Proto header is empty", func() {
			It("uses http for the scheme", func() {
				request, err := http.NewRequest("GET", "http://test.com/", nil)
				Expect(err).ToNot(HaveOccurred())
				request.Header.Set("X-Forwarded-Proto", "")
				Expect(route_service.CfForwardedUrlFor(request)).To(Equal("http://test.com/"))
			})
		})

		Context("when the X-Forwarded-Proto header is set to a single value", func() {
			It("uses the scheme in X-Forwarded-Proto", func() {
				request, err := http.NewRequest("GET", "http://test.com/", nil)
				Expect(err).ToNot(HaveOccurred())
				request.Header.Set("X-Forwarded-Proto", "https")
				Expect(route_service.CfForwardedUrlFor(request)).To(Equal("https://test.com/"))
			})
		})

		Context("when the X-Forwarded-Proto header is set to multiple values", func() {
			It("uses the first scheme in X-Forwarded-Proto", func() {
				request, err := http.NewRequest("GET", "http://test.com/", nil)
				Expect(err).ToNot(HaveOccurred())
				request.Header.Set("X-Forwarded-Proto", "https, http, http")
				Expect(route_service.CfForwardedUrlFor(request)).To(Equal("https://test.com/"))
			})
		})

		Context("when the URL has a path", func() {
			It("includes the path in the forwarded url", func() {
				request, err := http.NewRequest("GET", "http://test.com/path", nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(route_service.CfForwardedUrlFor(request)).To(Equal("http://test.com/path"))
			})
		})

		Context("when the URL has a query string", func() {
			It("includes the query string in the forwarded url", func() {
				request, err := http.NewRequest("GET", "http://test.com/path?a=b", nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(route_service.CfForwardedUrlFor(request)).To(Equal("http://test.com/path?a=b"))
			})
		})
	})
})
