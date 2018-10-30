package config_test

import (
	"github.com/gomicro/avenues/config"
	"os"
	"testing"

	. "github.com/franela/goblin"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	g := Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("Config File", func() {
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
	})
}
