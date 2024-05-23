package options

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOptions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Options Suite")
}

var _ = Describe("Test parsing SSHArgs", func() {
	DescribeTable("SSH arguments parsing",
		func(input string, expected []string) {
			Expect(parseSSHArgs(input)).To(Equal(expected))
		},
		Entry("whitespace separated arguments",
			"arg1 arg2 arg3",
			[]string{"arg1", "arg2", "arg3"},
		),
		Entry("not split by whitespace inside quotes",
			"arg1 arg2 ProxyCommand=\"cmd\"",
			[]string{"arg1", "arg2", "ProxyCommand=\"cmd\""},
		),
		Entry("not split by comma",
			"arg1,arg2 ProxyCommand=\"cmd\"",
			[]string{"arg1,arg2", "ProxyCommand=\"cmd\""},
		),
	)
})
