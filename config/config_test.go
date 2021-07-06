package config_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gomicro/avenues/config"

	"github.com/franela/goblin"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("Config File", func() {
		g.Describe("Parsing", func() {
			g.It("should parse a default config file", func() {
				c, err := config.ParseFromFile()
				Expect(err).To(BeNil())

				Expect(len(c.Routes)).To(Equal(4))
				Expect(c.Routes["/v1/foo"].Backend).To(Equal("http://foo:4567"))
				Expect(c.Routes["/v1/bar"].Backend).To(Equal("http://foo:4567"))
			})

			g.It("should parse a custom config file set in the environment", func() {
				os.Setenv("AVENUES_CONFIG_FILE", "./routes_other.yaml")

				c, err := config.ParseFromFile()
				Expect(err).To(BeNil())

				Expect(len(c.Routes)).To(Equal(1))
				Expect(c.Routes["/v1/baz"].Backend).To(Equal("http://baz:4567"))
			})

			g.It("should return an error when it can't read the file", func() {
				os.Setenv("AVENUES_CONFIG_FILE", "./routes_missing.yaml")

				c, err := config.ParseFromFile()
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("Failed to read config file"))
				Expect(c).To(BeNil())
			})

			g.It("should parse a config file with path settings", func() {
				os.Setenv("AVENUES_CONFIG_FILE", "./routes_other.yaml")

				c, err := config.ParseFromFile()
				Expect(err).To(BeNil())

				Expect(c.CertPath).To(Equal("dummy.cert"))
				Expect(c.Cert).To(ContainSubstring("-----BEGIN CERTIFICATE-----"))
			})
		})

		g.Describe("Serving", func() {
			g.It("should handle options", func() {
				c, _ := config.ParseFromFile()

				server := httptest.NewServer(c)
				defer server.Close()

				u := fmt.Sprintf("%v%v", server.URL, "/v1/foo")

				req, err := http.NewRequest("OPTIONS", u, nil)
				Expect(err).To(BeNil())

				client := http.Client{}
				res, err := client.Do(req)
				Expect(err).To(BeNil())
				defer res.Body.Close()

				Expect(res.StatusCode).To(Equal(http.StatusNoContent))

				Expect(res.Header.Get("Access-Control-Allow-Origin")).To(Equal("*"))
				Expect(res.Header.Get("Access-Control-Allow-Methods")).To(Equal("*"))
				Expect(res.Header.Get("Access-Control-Allow-Headers")).To(Equal("*, Authorization"))
				Expect(res.Header.Get("Access-Control-Max-Age")).To(Equal("60"))
				Expect(res.Header.Get("Cache-Control")).To(Equal("no-store, no-cache, must-revalidate, post-check=0, pre-check=0"))
				Expect(res.Header.Get("Vary")).To(Equal("Accept-Encoding"))
			})
		})
	})
}
