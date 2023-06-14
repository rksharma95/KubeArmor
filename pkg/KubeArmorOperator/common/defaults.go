// SPDX-License-Identifier: Apache-2.0
// Copyright 2022 Authors of KubeArmor

package common

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"os"
	"strings"

	deployments "github.com/kubearmor/KubeArmor/deployments/get"
	opv1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorOperator/api/operator.kubearmor.com/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

const (
	// constants for CRD status
	CREATED  string = "Created"
	PENDING  string = "Pending"
	RUNNING  string = "Running"
	UPDATING string = "Updating"
	ERROR    string = "Error"

	// Status Messages
	CREATED_MSG  string = "Installaltion has been created"
	PENDING_MSG  string = "Kubearmor Installation is in-progess"
	RUNNING_MSG  string = "Kubearmor Application is Up and Running"
	UPDATING_MSG string = "Updating the Application Configuration"

	// Error Messages
	INSTALLATION_ERR_MSG    string = "Failed to install KubeArmor component(s)"
	MULTIPLE_CRD_ERR_MSG    string = "There's already a CRD exists to manage KubeArmor"
	UPDATION_FAILED_ERR_MSG string = "Failed to update KubeArmor configuration"
)

var OperatigConfigCrd *opv1.KubeArmorConfig

var (
	EnforcerLabel                   string = "kubearmor.io/enforcer"
	RuntimeLabel                    string = "kubearmor.io/runtime"
	RuntimeStorageLabel             string = "kubearmor.io/runtime-storage"
	SocketLabel                     string = "kubearmor.io/socket"
	RandLabel                       string = "kubearmor.io/rand"
	OsLabel                         string = "kubernetes.io/os"
	ArchLabel                       string = "kubernetes.io/arch"
	DeletAction                     string = "DELETE"
	AddAction                       string = "ADD"
	Namespace                       string = "kube-system"
	Privileged                      bool   = true
	OperatorImage                   string = "ttl.sh/kubearmor-operator:48h"
	KubeArmorServiceAccountName     string = "kubearmor"
	KubeArmorClusterRoleBindingName string = KubeArmorServiceAccountName
	KubeArmorSnitchRoleName         string = "kubearmor-snitch"

	// redhat ubi-based images
	KubeArmorRelayUbiImage      string = "kubearmor/kubearmor-relay-server:redhat-ubi"
	KubeArmorControllerUbiImage string = "kubearmor/kubearmor-controller:redhat-ubi"
	KubeArmorUbiImage           string = "kubearmor/kubearmor:redhat-ubi"
	KubeArmorInitUbiImage       string = "kubearmor/kubearmor-init:redhat-ubi"

	KubeArmorConfigMapName string = "kubearmor-config"

	// ConfigMap Data
	ConfigGRPC                       string = "gRPC"
	ConfigVisibility                 string = "visibility"
	ConfigCluster                    string = "cluster"
	ConfigDefaultFilePosture         string = "defaultFilePosture"
	ConfigDefaultCapabilitiesPosture string = "defaultCapabilitiesPosture"
	ConfigDefaultNetworkPosture      string = "defaultNetworkPosture"
)

var ConfigMapData = map[string]string{
	ConfigGRPC:                       "32767",
	ConfigCluster:                    "default",
	ConfigDefaultFilePosture:         "audit",
	ConfigDefaultCapabilitiesPosture: "audit",
	ConfigDefaultNetworkPosture:      "audit",
}

var ContainerRuntimeSocketMap = map[string][]string{
	"docker": {
		"/var/run/docker.sock",
		"/run/docker.sock",
	},
	"containerd": {
		"/var/snap/microk8s/common/run/containerd.sock",
		"/run/k3s/containerd/containerd.sock",
		"/run/containerd/containerd.sock",
		"/var/run/containerd/containerd.sock",
	},
	"crio": {
		"/var/run/crio/crio.sock",
		"/run/crio/crio.sock",
	},
}

var HostPathDirectory = corev1.HostPathDirectory
var HostPathSocket = corev1.HostPathSocket

var EnforcerVolumesMounts = map[string][]corev1.VolumeMount{
	"apparmor": {
		{
			Name:      "etc-apparmor-d-path",
			MountPath: "/etc/apparmor.d",
		},
	},
	"bpf": {
		{
			Name:      "sys-fs-bpf-path",
			MountPath: "/sys/fs/bpf",
		},
	},
}

var EnforcerVolumes = map[string][]corev1.Volume{
	"apparmor": {
		{
			Name: "etc-apparmor-d-path",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/apparmor.d",
					Type: &HostPathDirectory,
				},
			},
		},
	},
	"bpf": {

		{
			Name: "sys-fs-bpf-path",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/sys/fs/bpf",
					Type: &HostPathDirectory,
				},
			},
		},
	},
}

var RuntimeStorageVolumes = map[string][]string{
	"docker": {
		"/var/lib/docker",
	},
	"crio": {
		"/var/lib/containers/storage",
	},
	"containerd": {
		"/run/k3s/containerd",
		"/run/containerd",
	},
}

func ShortSHA(s string) string {
	sBytes := []byte(s)

	shaFunc := sha512.New()
	shaFunc.Write(sBytes)
	res := shaFunc.Sum(nil)
	return hex.EncodeToString(res)[:5]
}

var CommonVolumes = []corev1.Volume{
	{
		Name: "bpf",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	},
	{
		Name: "sys-kernel-security-path",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/sys/kernel/security",
				Type: &HostPathDirectory,
			},
		},
	},
	{
		Name: "sys-kernel-debug-path",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/sys/kernel/debug",
				Type: &HostPathDirectory,
			},
		},
	},
}

var CommonVolumesMount = []corev1.VolumeMount{
	{
		Name:      "bpf",
		MountPath: "/opt/kubearmor/BPF",
	},
	{
		Name:      "sys-kernel-security-path",
		MountPath: "/sys/kernel/security",
	},
	{
		Name:      "sys-kernel-debug-path",
		MountPath: "/sys/kernel/debug",
	},
}

func GetFreeRandSuffix(c *kubernetes.Clientset, namespace string) (suffix string, err error) {
	var found bool
	for {
		suffix = rand.String(5)
		found = false
		if _, err = c.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), deployments.AnnotationsControllerServiceName+"-"+suffix, metav1.GetOptions{}); err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return "", err
			}
		} else {
			found = true
		}

		if _, err = c.CoreV1().Services(namespace).Get(context.Background(), deployments.AnnotationsControllerServiceName+"-"+suffix, metav1.GetOptions{}); err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return "", err
			}
		} else {
			found = true
		}

		if _, err = c.AppsV1().Deployments(namespace).Get(context.Background(), deployments.AnnotationsControllerDeploymentName+"-"+suffix, metav1.GetOptions{}); err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return "", err
			}
		} else {
			found = true
		}

		if _, err = c.CoreV1().Secrets(namespace).Get(context.Background(), deployments.KubeArmorControllerSecretName+"-"+suffix, metav1.GetOptions{}); err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return "", err
			}
		} else {
			found = true
		}

		if !found {
			break
		}
	}
	return suffix, nil
}

func GetOperatorNamespace() string {
	ns := os.Getenv("KUBEARMOR_OPERATOR_NS")

	if ns == "" {
		return Namespace
	}

	return ns
}

func init() {
	Namespace = GetOperatorNamespace()
}
