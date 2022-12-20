package kubestatemetrics

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
)

func rbacProxyContainerSpec(portName string, portNumber, listenAddress int32) corev1.Container {
	return corev1.Container{
		Name:  fmt.Sprintf("kube-rbac-proxy-%s", portName),
		Image: fmt.Sprintf("%s:v%s", rbacURL, rbacVersion),
		Args: []string{
			"--logtostderr",
			fmt.Sprintf("--secure-listen-address=:%d", listenAddress),
			"--tls-cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
			fmt.Sprintf("--upstream=http://127.0.0.1:%d/", portNumber),
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("20m"),
				"memory": resource.MustParse("20Mi"),
			},
			Limits: corev1.ResourceList{
				"cpu":    resource.MustParse("40m"),
				"memory": resource.MustParse("40Mi"),
			},
		},
		Ports: []corev1.ContainerPort{{
			ContainerPort: listenAddress,
			Name:          fmt.Sprintf("https-%s", portName),
		}},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: pointer.Bool(false),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			ReadOnlyRootFilesystem:   pointer.Bool(true),
			RunAsUser:                pointer.Int64(65532),
			RunAsGroup:               pointer.Int64(65532),
			RunAsNonRoot:             pointer.Bool(true),
		},
	}
}

func Deployment(cell *monitoringv1alpha1.Cell) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", Name, cell.Name),
			Namespace: cell.Namespace,
			Labels:    Labels(cell),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: cell.APIVersion,
					Kind:       cell.Kind,
					Name:       cell.Name,
					UID:        cell.UID,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: Labels(cell)},
			Replicas: pointer.Int32(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: Labels(cell),
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": "kube-state-metrics",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:           fmt.Sprintf("%s-%s", Name, cell.Name),
					AutomountServiceAccountToken: pointer.Bool(true),
					// NodeSelector:                 ctx.Config.NodeSelector,
					Containers: []corev1.Container{
						{
							Name:  Name,
							Image: fmt.Sprintf("%s:v%s", ImageURL, Version),
							Args: []string{
								"--host=127.0.0.1",
								"--port=8081",
								"--telemetry-host=127.0.0.1",
								"--telemetry-port=8082",
								"--metric-labels-allowlist=nodes=[cloud.google.com/gke-nodepool,topology.kubernetes.io/region],pods=[component,workspaceType,owner,metaID]",
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("10m"),
									"memory": resource.MustParse("190Mi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: pointer.Bool(false),
								Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
								ReadOnlyRootFilesystem:   pointer.Bool(true),
								RunAsUser:                pointer.Int64(65534),
							},
						},
						rbacProxyContainerSpec("main", 8081, 8443),
						rbacProxyContainerSpec("self", 8082, 9443),
					},
				},
			},
		},
	}
}
