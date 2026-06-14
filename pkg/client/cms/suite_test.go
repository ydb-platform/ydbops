package cms_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCMS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CMS Suite")
}
