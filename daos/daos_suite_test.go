package daos_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDaos(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Daos Suite")
}
