package nodeexporter

import monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"

const (
	Name     = "node-exporter"
	Version  = "1.3.1"
	ImageURL = "quay.io/prometheus/node-exporter"
)

func Labels(cell *monitoringv1alpha1.Cell) map[string]string {
	c := cell.DeepCopy()
	labels := c.Labels
	labels["app.kubernetes.io/name"] = Name

	return labels
}
