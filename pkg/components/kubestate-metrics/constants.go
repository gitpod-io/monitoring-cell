package kubestatemetrics

import monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"

const (
	Name        = "kube-state-metrics"
	Version     = "2.5.0"
	ImageURL    = "k8s.gcr.io/kube-state-metrics/kube-state-metrics"
	rbacURL     = "quay.io/brancz/kube-rbac-proxy"
	rbacVersion = "0.13.0"
)

func Labels(cell *monitoringv1alpha1.Cell) map[string]string {
	c := cell.DeepCopy()
	labels := c.Labels
	labels["app.kubernetes.io/name"] = Name

	return labels
}
