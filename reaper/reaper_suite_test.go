package reaper_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestReaper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reaper Suite")
}
