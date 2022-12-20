package kubestatemetrics

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
)

type replaceLabel struct {
	Source string
	Target string
}

func labelsReplaceAndDrop(replaceLabels []replaceLabel) []*monitoringv1.RelabelConfig {
	var configs []*monitoringv1.RelabelConfig
	for _, s := range replaceLabels {
		configs = append(configs, &monitoringv1.RelabelConfig{
			Action:       "replace",
			Regex:        "(.*)",
			Replacement:  "$1",
			SourceLabels: []monitoringv1.LabelName{monitoringv1.LabelName(s.Source)},
			TargetLabel:  s.Target,
		})
		configs = append(configs, &monitoringv1.RelabelConfig{
			Action: "labeldrop",
			Regex:  s.Source,
		})
	}
	return configs
}

func ServiceMonitor(cell *monitoringv1alpha1.Cell) *monitoringv1.ServiceMonitor {
	configs := labelsReplaceAndDrop([]replaceLabel{
		{
			Source: "label_cloud_google_com_gke_nodepool",
			Target: "nodepool",
		},
		{
			Source: "label_topology_kubernetes_io_region",
			Target: "region",
		},
		{
			Source: "label_component",
			Target: "component",
		},
		{
			Source: "label_workspace_type",
			Target: "workspace_type",
		},
		{
			Source: "label_owner",
			Target: "owner",
		},
		{
			Source: "label_meta_id",
			Target: "metaID",
		},
	})

	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
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
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{
					BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					Port:            "https-main",
					Interval:        "60s",
					ScrapeTimeout:   "30s",
					Scheme:          "https",
					HonorLabels:     true,
					TLSConfig: &monitoringv1.TLSConfig{
						SafeTLSConfig: monitoringv1.SafeTLSConfig{
							InsecureSkipVerify: true,
						},
					},
					// MetricRelabelConfigs: append(configs, common.DropMetricsRelabeling(ctx)...),
					MetricRelabelConfigs: configs,
					RelabelConfigs: []*monitoringv1.RelabelConfig{
						{
							Action: "labeldrop",
							Regex:  "(pod|service|endpoint|namespace)",
						},
					},
				},
				{
					BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					Port:            "https-self",
					Interval:        "60s",
					Scheme:          "https",
					TLSConfig: &monitoringv1.TLSConfig{
						SafeTLSConfig: monitoringv1.SafeTLSConfig{
							InsecureSkipVerify: true,
						},
					},
				},
			},
			JobLabel: "app.kubernetes.io/name",
			Selector: metav1.LabelSelector{
				MatchLabels: Labels(cell),
			},
		},
	}
}
