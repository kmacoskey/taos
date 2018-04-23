package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTaos(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Taos Suite")
}
