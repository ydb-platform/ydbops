package options

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/ydb-platform/ydbops/pkg/utils"
)

func TestOptions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Options Suite")
}

var _ = Describe("Test parsing SSHArgs", func() {
	DescribeTable("SSH arguments parsing",
		func(input string, expected []string) {
			Expect(utils.ParseSSHArgs(input)).To(Equal(expected))
		},
		Entry("whitespace separated arguments",
			"arg1 arg2 arg3",
			[]string{"arg1", "arg2", "arg3"},
		),
		Entry("not split by whitespace inside quotes",
			"arg1 arg2 ProxyCommand=\\\"arg1 arg2\\\"",
			[]string{"arg1", "arg2", "ProxyCommand=\"arg1 arg2\""},
		),
		Entry("not split by comma",
			"arg1,arg2",
			[]string{"arg1,arg2"},
		),
		Entry("real world example",
			"ssh -i ~/yandex -o ProxyCommand=\\\"ssh -W %h:%p -i ~/yandex ubuntu@static-node-1.ydb-cluster.com\\\"",
			[]string{"ssh", "-i", "~/yandex", "-o", "ProxyCommand=\"ssh -W %h:%p -i ~/yandex ubuntu@static-node-1.ydb-cluster.com\""},
		),
	)
})
