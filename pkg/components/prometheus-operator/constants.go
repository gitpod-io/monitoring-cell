package prometheusoperator

import monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"

const (
	Name     = "prometheus-operator"
	Version  = "0.58.0"
	ImageURL = "quay.io/prometheus-operator/prometheus-operator"
)

func Labels(cell *monitoringv1alpha1.Cell) map[string]string {
	c := cell.DeepCopy()
	labels := c.Labels
	labels["app.kubernetes.io/name"] = Name

	return labels
}
