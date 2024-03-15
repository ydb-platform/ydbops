package tests

import (
  . "github.com/onsi/ginkgo/v2"
  . "github.com/onsi/gomega"
  "testing"
)

func TestRestart(t *testing.T) {
  RegisterFailHandler(Fail)
  RunSpecs(t, "Restart Suite")
}
