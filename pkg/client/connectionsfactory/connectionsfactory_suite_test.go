package connectionsfactory

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConnectionsFactory(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Connections Factory Suite")
}
