package kubearmor_config_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestKubearmorConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "KubearmorConfig Suite")
}
