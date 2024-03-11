package tests

import (
  . "github.com/onsi/ginkgo/v2"
  . "github.com/onsi/gomega"
  "testing"
)

func TestRolling(t *testing.T) {
  RegisterFailHandler(Fail)
  RunSpecs(t, "Rolling Suite")
}
