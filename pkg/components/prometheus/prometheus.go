package prometheus

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
)

func Prometheus(cell *monitoringv1alpha1.Cell) *monitoringv1.Prometheus {
	return &monitoringv1.Prometheus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "Prometheus",
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
		Spec: monitoringv1.PrometheusSpec{
			RuleSelector: &metav1.LabelSelector{},
			CommonPrometheusFields: monitoringv1.CommonPrometheusFields{
				Image: pointer.String(fmt.Sprintf("%s:v%s", ImageURL, Version)),
				PodMetadata: &monitoringv1.EmbeddedObjectMetadata{
					Labels: Labels(cell),
				},
				Replicas: pointer.Int32(1),
				SecurityContext: &corev1.PodSecurityContext{
					FSGroup:      pointer.Int64(2000),
					RunAsUser:    pointer.Int64(1000),
					RunAsNonRoot: pointer.Bool(true),
				},
				ServiceAccountName: fmt.Sprintf("%s-%s", Name, cell.Name),
				ExternalLabels: map[string]string{
					"cluster": cell.Spec.ClusterName,
				},
				// NodeSelector:           ctx.Config.NodeSelector,
				// RemoteWrite:            build from spec.Metrics.UpstreamRemoteWrites
				Version:                Version,
				ServiceMonitorSelector: &metav1.LabelSelector{},
				PodMonitorSelector:     &metav1.LabelSelector{},
			},
		},
	}
}
