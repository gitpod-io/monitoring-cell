package prometheus

import monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"

const (
	Name     = "prometheus"
	Version  = "2.37.0"
	ImageURL = "quay.io/prometheus/prometheus"
)

func Labels(cell *monitoringv1alpha1.Cell) map[string]string {
	c := cell.DeepCopy()
	if c.Labels == nil {
		c.Labels = map[string]string{}
	}
	labels := c.Labels
	labels["app.kubernetes.io/name"] = Name

	return labels
}
