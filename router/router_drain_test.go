package router_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"errors"

	"github.com/cloudfoundry/gorouter/access_log"
	vcap "github.com/cloudfoundry/gorouter/common"
	cfg "github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/metrics/fakes"
	"github.com/cloudfoundry/gorouter/proxy"
	rregistry "github.com/cloudfoundry/gorouter/registry"
	"github.com/cloudfoundry/gorouter/route"
	. "github.com/cloudfoundry/gorouter/router"
	"github.com/cloudfoundry/gorouter/test"
	"github.com/cloudfoundry/gorouter/test_util"
	vvarz "github.com/cloudfoundry/gorouter/varz"
	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/cloudfoundry/yagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Router", func() {
	var natsRunner *natsrunner.NATSRunner
	var config *cfg.Config

	var mbusClient yagnats.NATSConn
	var registry *rregistry.RouteRegistry
	var varz vvarz.Varz
	var router *Router
	var natsPort uint16
	var errChan chan error

	BeforeEach(func() {
		natsPort = test_util.NextAvailPort()
		natsRunner = natsrunner.NewNATSRunner(int(natsPort))
		natsRunner.Start()

		proxyPort := test_util.NextAvailPort()
		statusPort := test_util.NextAvailPort()

		config = test_util.SpecConfig(natsPort, statusPort, proxyPort)
		config.EndpointTimeout = 5 * time.Second

		mbusClient = natsRunner.MessageBus
		registry = rregistry.NewRouteRegistry(config, mbusClient, new(fakes.FakeRouteReporter))
		varz = vvarz.NewVarz(registry)
		logcounter := vcap.NewLogCounter()
		proxy := proxy.NewProxy(proxy.ProxyArgs{
			EndpointTimeout: config.EndpointTimeout,
			Ip:              config.Ip,
			TraceKey:        config.TraceKey,
			Registry:        registry,
			Reporter:        varz,
			AccessLogger:    &access_log.NullAccessLogger{},
		})

		errChan = make(chan error, 2)
		r, err := NewRouter(config, proxy, mbusClient, registry, varz, logcounter, errChan)
		Expect(err).ToNot(HaveOccurred())
		router = r

		signals := make(chan os.Signal)
		readyChan := make(chan struct{})
		go r.Run(signals, readyChan)
		select {
		case <-readyChan:
		}
	})

	AfterEach(func() {
		if natsRunner != nil {
			natsRunner.Stop()
		}
	})

	Context("Drain", func() {

		AfterEach(func() {
			if router != nil {
				router.Stop()
			}
		})

		It("waits until the last request completes", func() {
			app := test.NewTestApp([]route.Uri{"drain.vcap.me"}, config.Port, mbusClient, nil, "")
			blocker := make(chan bool)
			resultCh := make(chan bool, 2)
			app.AddHandler("/", func(w http.ResponseWriter, r *http.Request) {
				blocker <- true

				_, err := ioutil.ReadAll(r.Body)
				defer r.Body.Close()
				Expect(err).ToNot(HaveOccurred())

				<-blocker

				w.WriteHeader(http.StatusNoContent)
			})

			app.Listen()

			Eventually(func() bool {
				return appRegistered(registry, app)
			}).Should(BeTrue())

			drainTimeout := 1 * time.Second

			go func() {
				defer GinkgoRecover()
				req, err := http.NewRequest("GET", app.Endpoint(), nil)
				Expect(err).ToNot(HaveOccurred())

				client := http.Client{}
				resp, err := client.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				defer resp.Body.Close()
				_, err = ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())
				resultCh <- false
			}()

			<-blocker
			go func() {
				defer GinkgoRecover()
				err := router.Drain(drainTimeout)
				Expect(err).ToNot(HaveOccurred())
				resultCh <- true
			}()

			Consistently(resultCh, drainTimeout/10).ShouldNot(Receive())

			blocker <- false

			var result bool
			Eventually(resultCh).Should(Receive(&result))
			Expect(result).To(BeTrue())
		})

		It("times out if it takes too long", func() {
			app := test.NewTestApp([]route.Uri{"draintimeout.vcap.me"}, config.Port, mbusClient, nil, "")

			blocker := make(chan bool)
			resultCh := make(chan error, 2)
			app.AddHandler("/", func(w http.ResponseWriter, r *http.Request) {
				blocker <- true

				_, err := ioutil.ReadAll(r.Body)
				defer r.Body.Close()
				Expect(err).ToNot(HaveOccurred())

				time.Sleep(1 * time.Second)
			})
			app.Listen()

			Eventually(func() bool {
				return appRegistered(registry, app)
			}).Should(BeTrue())

			go func() {
				defer GinkgoRecover()
				req, err := http.NewRequest("GET", app.Endpoint(), nil)
				Expect(err).ToNot(HaveOccurred())

				client := http.Client{}
				resp, err := client.Do(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				defer resp.Body.Close()
			}()

			<-blocker

			go func() {
				defer GinkgoRecover()
				err := router.Drain(500 * time.Millisecond)
				resultCh <- err
			}()

			var result error
			Eventually(resultCh).Should(Receive(&result))
			Expect(result).To(Equal(DrainTimeout))
		})
	})

	Context("OnErrOrSignal", func() {
				/*
			- when an error is received in the error chan, it drains existing connections
			when a USR1 signal is sent, it drains and stops the router
			when a term signal is sent, it stops the router
			when a interruption signal is sent, it stops the router
			when USR1 is the first of multiple signals, it drains and stops the router
			when USR1 is not the first of multiple signals, it stops the router
			when a non handled signal is sent, it does nothing.
		*/

		BeforeEach(func() {
		})

		AfterEach(func() {

		})

		Context("when an error is received in the error chan", func() {
			It("it drains existing connections", func() {

				app := test.NewTestApp([]route.Uri{"drain.vcap.me"}, config.Port, mbusClient, nil, "")
				blocker := make(chan bool)
				resultCh := make(chan bool, 2)
				app.AddHandler("/", func(w http.ResponseWriter, r *http.Request) {
					blocker <- true

					_, err := ioutil.ReadAll(r.Body)
					defer r.Body.Close()
					Expect(err).ToNot(HaveOccurred())

					<-blocker

					w.WriteHeader(http.StatusNoContent)
				})

				app.Listen()

				Eventually(func() bool {
					return appRegistered(registry, app)
				}).Should(BeTrue())

				drainTimeout := 1 * time.Second

				go func() {
					defer GinkgoRecover()
					req, err := http.NewRequest("GET", app.Endpoint(), nil)
					Expect(err).ToNot(HaveOccurred())

					client := http.Client{}
					resp, err := client.Do(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(resp).ToNot(BeNil())
					defer resp.Body.Close()
					_, err = ioutil.ReadAll(resp.Body)
					Expect(err).ToNot(HaveOccurred())
					resultCh <- false
				}()
				<-blocker
				go func() {
					errChan <- errors.New("Fake error")
				}()

				Consistently(resultCh, drainTimeout/10).ShouldNot(Receive())

				blocker <- false

				var result bool
				Eventually(resultCh).Should(Receive(&result))
				Expect(result).To(BeFalse())
			})
		})
	})
})
