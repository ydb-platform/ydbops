package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRestart(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Restart e2e suite")
}
