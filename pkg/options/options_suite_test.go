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

var _ = Describe("Test parsing Hosts", func() {
	It("Parse hosts as FQDNs with various symbols", func() {
		Expect(utils.GetNodeFQDNs(
			[]string{"simpleHost", "host-with-dashes", "abc.ydb.nebius.dev"},
		)).To(Equal(
			[]string{"simpleHost", "host-with-dashes", "abc.ydb.nebius.dev"},
		))
	})

	DescribeTable("Parse host as ids",
		func(input []string, expected []uint32) {
			Expect(utils.GetNodeIds(input)).To(Equal(expected))
		},
		Entry("simplest case, three hosts each by themselves",
			[]string{"1", "2", "3"},
			[]uint32{1, 2, 3},
		),
		Entry("simplest range test",
			[]string{"1-5"},
			[]uint32{1, 2, 3, 4, 5},
		),
		Entry("real world example",
			[]string{"1", "2", "4-8", "3"},
			[]uint32{1, 2, 4, 5, 6, 7, 8, 3},
		),
	)
})
