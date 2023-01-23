package gitpod_test

import (
	"testing"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
	gitpod "github.com/gitpod-io/monitoring-cell/pkg/components/gitpod"
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
		GitpodNamespace: "GitpodNamespace",
	},
}

func TestServicesNamespace(t *testing.T) {
	services := gitpod.Services(minimalCell)
	for _, svc := range services {
		if svc.Namespace != minimalCell.Spec.GitpodNamespace {
			t.Errorf("Service %s has wrong namespace: %s", svc.Name, svc.Namespace)
		}
	}
}

func TestNetworkPoliciesNamespace(t *testing.T) {
	networkPolicies := gitpod.NetworkPolicies(minimalCell)
	for _, np := range networkPolicies {
		if np.Namespace != minimalCell.Spec.GitpodNamespace {
			t.Errorf("NetworkPolicy %s has wrong namespace: %s", np.Name, np.Namespace)
		}

		if np.Spec.Ingress[0].From[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] != minimalCell.Namespace {
			t.Errorf("NetworkPolicy %s has wrong namespace selector: %s", np.Name, np.Spec.Ingress[0].From[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"])
		}
	}
}

func TestServiceMonitorsNamespace(t *testing.T) {
	serviceMonitors := gitpod.ServiceMonitors(minimalCell)
	for _, sm := range serviceMonitors {
		if sm.Namespace != minimalCell.Namespace {
			t.Errorf("ServiceMonitor %s has wrong namespace: %s", sm.Name, sm.Namespace)
		}

		if sm.Spec.NamespaceSelector.MatchNames[0] != minimalCell.Spec.GitpodNamespace {
			t.Errorf("ServiceMonitor %s has wrong namespace selector: %s", sm.Name, sm.Spec.NamespaceSelector.MatchNames[0])
		}
	}
}
