package proxy_test

import (
	"crypto/tls"
	"net/http/httptest"
	"time"

	fakelogger "github.com/cloudfoundry/gorouter/access_log/fakes"
	"github.com/cloudfoundry/gorouter/proxy"
	"github.com/cloudfoundry/gorouter/registry"
	fakeregistry "github.com/cloudfoundry/gorouter/registry/fakes"
	"github.com/cloudfoundry/gorouter/test_util"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Proxy Unit tests", func() {
	var (
		proxyObj         proxy.Proxy
		fakeAccessLogger *fakelogger.FakeAccessLogger
		fakeRegistry     *fakeregistry.FakeRegistryInterface

		r *registry.RouteRegistry
	)

	BeforeEach(func() {
		tlsConfig := &tls.Config{
			CipherSuites:       conf.CipherSuites,
			InsecureSkipVerify: conf.SSLSkipValidation,
		}

		fakeAccessLogger = &fakelogger.FakeAccessLogger{}
		fakeRegistry = &fakeregistry.FakeRegistryInterface{}

		mbus := fakeyagnats.Connect()
		r = registry.NewRouteRegistry(conf, mbus)

		proxyObj = proxy.NewProxy(proxy.ProxyArgs{
			EndpointTimeout:     conf.EndpointTimeout,
			Ip:                  conf.Ip,
			TraceKey:            conf.TraceKey,
			Registry:            r,
			Reporter:            nullVarz{},
			AccessLogger:        fakeAccessLogger,
			SecureCookies:       conf.SecureCookies,
			TLSConfig:           tlsConfig,
			RouteServiceEnabled: conf.RouteServiceEnabled,
			RouteServiceTimeout: conf.RouteServiceTimeout,
			Crypto:              crypto,
			CryptoPrev:          cryptoPrev,
		})

	})

	It("logs response time for TCP connections", func() {
		// Create TCP Upgrade HTTP Request
		// Create HTTP response writer
		// Fake out AccessLogger

		req := test_util.NewRequest("UPGRADE", "unit-test", "/", nil)
		req.Header.Set("Upgrade", "tcp")
		req.Header.Set("Connection", "upgrade")
		resp := httptest.NewRecorder()

		proxyObj.ServeHTTP(resp, req)
		Expect(fakeAccessLogger.LogCallCount()).To(Equal(1))
		Expect(fakeAccessLogger.LogArgsForCall(0).FinishedAt).NotTo(Equal(time.Time{}))
	})
})
