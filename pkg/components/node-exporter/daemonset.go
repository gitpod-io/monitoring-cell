package nodeexporter

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
)

func Daemonset(cell *monitoringv1alpha1.Cell) *appsv1.DaemonSet {
	hostToContainer := v1.MountPropagationHostToContainer
	maxUnavailable := intstr.FromString("10%")

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
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
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: Labels(cell),
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: Labels(cell),
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-container": "node-exporter",
					},
				},
				Spec: v1.PodSpec{
					AutomountServiceAccountToken: pointer.Bool(true),
					HostNetwork:                  true,
					HostPID:                      true,
					SecurityContext: &v1.PodSecurityContext{
						RunAsNonRoot: pointer.Bool(true),
						RunAsUser:    pointer.Int64(65534),
					},
					ServiceAccountName: fmt.Sprintf("%s-%s", Name, cell.Name),
					Tolerations: []v1.Toleration{
						{
							Operator: v1.TolerationOpExists,
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "sys",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/sys",
								},
							},
						},
						{
							Name: "root",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/",
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Args: []string{
								"--web.listen-address=127.0.0.1:9100",
								"--path.sysfs=/host/sys",
								"--path.rootfs=/host/root",
								"--no-collector.wifi",
								"--no-collector.hwmon",
								"--collector.filesystem.mount-points-exclude=^/(dev|proc|sys|run/k3s/containerd/.+|var/lib/docker/.+|var/lib/kubelet/pods/.+)($|/)",
								"--collector.netclass.ignored-devices=^(veth.*|[a-f0-9]{15})$",
								"--collector.netdev.device-exclude=^(veth.*|[a-f0-9]{15})$",
							},
							Image: fmt.Sprintf("%s:v%s", ImageURL, Version),
							Name:  Name,
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("100m"),
									v1.ResourceMemory: resource.MustParse("180Mi"),
								},
							},
							SecurityContext: &v1.SecurityContext{
								AllowPrivilegeEscalation: pointer.Bool(false),
								Capabilities: &v1.Capabilities{
									Add:  []v1.Capability{"SYS_TIME"},
									Drop: []v1.Capability{"ALL"},
								},
								ReadOnlyRootFilesystem: pointer.Bool(true),
							},
							VolumeMounts: []v1.VolumeMount{
								{
									MountPath:        "/host/sys",
									Name:             "sys",
									ReadOnly:         true,
									MountPropagation: &hostToContainer,
								},
								{
									MountPath:        "/host/root",
									Name:             "root",
									ReadOnly:         true,
									MountPropagation: &hostToContainer,
								},
							},
						},
						{
							Args: []string{
								"--logtostderr",
								"--secure-listen-address=[$(IP)]:9100",
								"--upstream=http://127.0.0.1:9100/",
							},
							Env: []v1.EnvVar{
								{
									Name: "IP",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
							},
							Image: "quay.io/brancz/kube-rbac-proxy:v0.13.0",
							Name:  "kube-rbac-proxy",
							Ports: []v1.ContainerPort{
								{
									Name:          "https",
									HostPort:      9100,
									ContainerPort: 9100,
								},
							},
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("60m"),
									v1.ResourceMemory: resource.MustParse("40Mi"),
								},
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse("10m"),
									v1.ResourceMemory: resource.MustParse("20Mi"),
								},
							},
							SecurityContext: &v1.SecurityContext{
								AllowPrivilegeEscalation: pointer.Bool(false),
								Capabilities: &v1.Capabilities{
									Drop: []v1.Capability{"ALL"},
								},
								ReadOnlyRootFilesystem: pointer.Bool(true),
								RunAsGroup:             pointer.Int64(65532),
								RunAsNonRoot:           pointer.Bool(true),
								RunAsUser:              pointer.Int64(65532),
							},
						},
					},
				},
			},
		},
	}
}
