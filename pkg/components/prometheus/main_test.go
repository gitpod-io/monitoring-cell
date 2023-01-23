package prometheus_test

import (
	"testing"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
	prometheus "github.com/gitpod-io/monitoring-cell/pkg/components/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var minimalCell = &monitoringv1alpha1.Cell{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "monitoring.gitpod.io/v1alpha1",
		Kind:       "Cell",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "Cell",
		Namespace: "CellNamespace",
	},
	Spec: monitoringv1alpha1.CellSpec{
		ClusterName: "ClusterName",
	},
}

func TestExternalLabel(t *testing.T) {
	p := prometheus.Prometheus(minimalCell)
	if p.Spec.CommonPrometheusFields.ExternalLabels["cluster"] != minimalCell.Spec.ClusterName {
		t.Errorf("External label cluster has wrong value: %s", p.Spec.ExternalLabels["cluster"])
	}
}