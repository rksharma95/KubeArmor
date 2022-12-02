// SPDX-License-Identifier: Apache-2.0
// Copyright 2022 Authors of KubeArmor

package kubearmor_config

import (
	"fmt"
	"time"

	. "github.com/kubearmor/KubeArmor/tests/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
	// install wordpress-mysql app
	err := K8sApply([]string{"manifests/ubuntu-deployment.yaml"})
	Expect(err).To(BeNil())

	// delete all KSPs
	KspDeleteAll()

	// enable kubearmor port forwarding
	err = KubearmorPortForward()
	Expect(err).To(BeNil())
})

var _ = AfterSuite(func() {
	KubearmorPortForwardStop()
})

func getUnannotatedPod(name string, ant string) string {
	pods, err := K8sGetPods(name, "unannotated", []string{ant}, 60)
	Expect(err).To(BeNil())
	Expect(len(pods)).To(Equal(1))
	return pods[0]
}

func getFullyAnnotatedPod(name string, ant string) string {
	pods, err := K8sGetPods(name, "fullyannotated", []string{ant}, 60)
	Expect(err).To(BeNil())
	Expect(len(pods)).To(Equal(1))
	return pods[0]
}

func getPartialyAnnotatedPod(name string, ant string) string {
	pods, err := K8sGetPods(name, "partialyannotated", []string{ant}, 60)
	Expect(err).To(BeNil())
	Expect(len(pods)).To(Equal(1))
	return pods[0]
}

var _ = Describe("KubeArmor-Config", func() {
	var unannotated string
	var partialyAnnotated string
	var fullyAnnotated string
	BeforeEach(func() {
		unannotated = getUnannotatedPod("unannotated-", "container.apparmor.security.beta.kubernetes.io/ubuntu-1: localhost/kubearmor-unannotated-unannotated-deployment-ubuntu-1")
		partialyAnnotated = getPartialyAnnotatedPod("partialyannotated-", "container.apparmor.security.beta.kubernetes.io/ubuntu-1: localhost/kubearmor-partialyannotated-partialyannotated-deployment-ubuntu-1")
		fullyAnnotated = getFullyAnnotatedPod("fullyannotated-", "container.apparmor.security.beta.kubernetes.io/ubuntu-1: localhost/kubearmor-fullyannotated-fullyannotated-deployment-ubuntu-1")
	})

	AfterEach(func() {
		KarmorLogStop()
		KspDeleteAll()
		CreateKAConfigMap("audit", "audit", "audit")
		time.Sleep(5 * time.Second)
	})

	Describe("Unannotated", func() {

		It("default posture will be set to global posture configs", Label("unannotated"), func() {
			Skip("Not getting alert, expecting alert for file access operation in audit mode")
			// apply a allow based policy
			err := K8sApply([]string{"manifests/ksp-unannotated-allow.yaml"})
			Expect(err).To(BeNil())

			// wait for policy
			time.Sleep(5 * time.Second)

			err = KarmorLogStart("policy", "unannotated", "File", unannotated)
			Expect(err).To(BeNil())

			// initialy global defaults posture is audit
			sout, _, err := K8sExecInPodWithContainer(unannotated, "unannotated", "ubuntu-1", []string{"bash", "-c", "cat /credentials/keys/priv.key"})
			Expect(err).To(BeNil())
			fmt.Printf("---START---\n%s---END---\n", sout)
			Expect(sout).NotTo(MatchRegexp(".*Permission denied"))

			// should get an alert with success
			// check policy violation alert
			_, alerts, err := KarmorGetLogs(5*time.Second, 1)
			Expect(err).To(BeNil())
			fmt.Println("No of Alerts Generated: ", len(alerts))
			Expect(len(alerts)).To(BeNumerically(">=", 1))
			Expect(alerts[0].PolicyName).To(Equal("DefaultPosture"))
			Expect(alerts[0].ContainerName).To(Equal("ubuntu-1"))

			// change global default posture to block using configmap
			err = CreateKAConfigMap("block", "block", "block") // will create a configMap with default posture as block
			Expect(err).To(BeNil())

			// wait for policy updation
			time.Sleep(5 * time.Second)

			// now defaults posture should be changed to block
			sout, _, err = K8sExecInPodWithContainer(unannotated, "unannotated", "ubuntu-1", []string{"bash", "-c", "cat /credentials/keys/priv.key"})
			Expect(err).To(BeNil())
			fmt.Printf("---START---\n%s---END---\n", sout)
			Expect(sout).To(MatchRegexp(".*Permission denied"))

		})
	})

	Describe("Partialy Annotated", Label("partial"), func() {

		It("default posture will be set to global config posture for unannotated posture only", func() {

			// apply a allow based policy
			err := K8sApply([]string{"manifests/ksp-partialyAnnotated-allow.yaml"})
			Expect(err).To(BeNil())

			// wait for policy
			time.Sleep(5 * time.Second)

			err = KarmorLogStart("policy", "partialyannotated", "File", partialyAnnotated)
			Expect(err).To(BeNil())

			// initialy namespace defaults posture (annotated) is audit for File
			sout, _, err := K8sExecInPodWithContainer(partialyAnnotated, "partialyannotated", "ubuntu-1", []string{"bash", "-c", "cat /credentials/keys/priv.key"})
			Expect(err).To(BeNil())
			fmt.Printf("---START---\n%s---END---\n", sout)
			Expect(sout).NotTo(MatchRegexp(".*Permission denied"))

			// should get an alert with success
			// check policy violation alert
			_, alerts, err := KarmorGetLogs(5*time.Second, 16)
			Expect(err).To(BeNil())
			fmt.Println("No of Alerts Generated: ", len(alerts))
			Expect(len(alerts)).To(BeNumerically(">=", 1))
			fmt.Printf("---Alert---\n%s", alerts[0].String())
			Expect(alerts[0].PolicyName).To(Equal("DefaultPosture"))
			Expect(alerts[0].ContainerName).To(Equal("ubuntu-1"))

			// change global default posture to block using configmap
			err = CreateKAConfigMap("block", "block", "block") // will create a configMap with default posture as block
			Expect(err).To(BeNil())

			// wait for policy updation
			time.Sleep(5 * time.Second)

			err = KarmorLogStart("policy", "partialyannotated", "Network", partialyAnnotated)
			Expect(err).To(BeNil())

			// defaults posture should be block for network
			sout, _, err = K8sExecInPodWithContainer(partialyAnnotated, "partialyannotated", "ubuntu-1", []string{"bash", "-c", "curl google.com"})
			Expect(err).To(BeNil())
			fmt.Printf("---START---\n%s---END---\n", sout)
			Expect(sout).To(MatchRegexp(".*not resolve"))

			// should get an alert with failure
			// check policy violation alert
			_, alerts, err = KarmorGetLogs(5*time.Second, 1)
			Expect(err).To(BeNil())
			fmt.Println("No of Alerts Generated: ", len(alerts))
			Expect(len(alerts)).To(BeNumerically(">=", 1))
			Expect(alerts[0].PolicyName).To(Equal("DefaultPosture"))
			Expect(alerts[0].ContainerName).To(Equal("ubuntu-1"))
			Expect(alerts[0].Action).To(Equal("Block"))

			err = KarmorLogStart("policy", "partialyannotated", "File", partialyAnnotated)
			Expect(err).To(BeNil())

			// defaults posture should still be audit for File
			sout, _, err = K8sExecInPodWithContainer(partialyAnnotated, "partialyannotated", "ubuntu-1", []string{"bash", "-c", "cat /credentials/keys/priv.key"})
			Expect(err).To(BeNil())
			fmt.Printf("---START---\n%s---END---\n", sout)
			Expect(sout).NotTo(MatchRegexp(".*Permission denied"))

		})
	})

	Describe("Fully Annotated", Label("full"), func() {

		It("default posture will be unchanged after global configs changed", func() {

			// apply a allow based policy
			err := K8sApply([]string{"manifests/ksp-fullyAnnotated-allow.yaml"})
			Expect(err).To(BeNil())

			// wait for policy
			time.Sleep(5 * time.Second)

			err = KarmorLogStart("policy", "fullyannotated", "Network", fullyAnnotated)
			Expect(err).To(BeNil())

			// initialy namespace defaults posture is block (annotated fully)
			sout, _, err := K8sExecInPodWithContainer(fullyAnnotated, "fullyannotated", "ubuntu-1", []string{"bash", "-c", "curl google.com"})
			Expect(err).To(BeNil())
			fmt.Printf("---START---\n%s---END---\n", sout)
			Expect(sout).To(MatchRegexp(".*has moved"))

			// should get an alert with success
			// check policy violation alert
			_, alerts, err := KarmorGetLogs(5*time.Second, 1)
			Expect(err).To(BeNil())
			Expect(len(alerts)).To(BeNumerically(">=", 1))
			Expect(alerts[0].PolicyName).To(Equal("DefaultPosture"))
			Expect(alerts[0].ContainerName).To(Equal("ubuntu-1"))

			// change global default posture to block using configmap
			err = CreateKAConfigMap("block", "block", "block") // will create a configMap with default posture as block
			Expect(err).To(BeNil())

			// wait for policy updation
			time.Sleep(5 * time.Second)

			// defaults posture should still be audit for network
			sout, _, err = K8sExecInPodWithContainer(fullyAnnotated, "fullyannotated", "ubuntu-1", []string{"bash", "-c", "curl google.com"})
			Expect(err).To(BeNil())
			fmt.Printf("---START---\n%s---END---\n", sout)
			Expect(sout).To(MatchRegexp(".*has moved"))

		})
	})

})
