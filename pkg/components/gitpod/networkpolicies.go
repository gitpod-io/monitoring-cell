package gitpod

import (
	"fmt"

	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
)

func NetworkPolicies(cell *monitoringv1alpha1.Cell) []*networkv1.NetworkPolicy {
	var networkPolicies []*networkv1.NetworkPolicy
	for _, target := range targets {
		networkPolicies = append(networkPolicies, &networkv1.NetworkPolicy{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "NetworkPolicy",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-allow-prometheus", target),
				Namespace: cell.Spec.GitpodNamespace,
				Labels:    labels(target),
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: cell.APIVersion,
						Kind:       cell.Kind,
						Name:       cell.Name,
						UID:        cell.UID,
					},
				},
			},
			Spec: networkv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"component": target,
					},
				},
				Ingress: []networkv1.NetworkPolicyIngressRule{
					{
						From: []networkv1.NetworkPolicyPeer{
							{
								NamespaceSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"kubernetes.io/metadata.name": cell.Namespace,
									},
								},
								PodSelector: &metav1.LabelSelector{
									MatchLabels: matchLabels,
								},
							},
						},
					},
				},
				PolicyTypes: []networkv1.PolicyType{
					networkv1.PolicyTypeIngress,
				},
			},
		})
	}

	networkPolicies = append(networkPolicies, messagebusNetworkPolicy(cell))
	networkPolicies = append(networkPolicies, proxyCaddyNetowrkPolicy(cell))
	return networkPolicies
}

func messagebusNetworkPolicy(cell *monitoringv1alpha1.Cell) *networkv1.NetworkPolicy {
	return &networkv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "NetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-allow-prometheus", "messagebus"),
			Namespace: cell.Spec.GitpodNamespace,
			Labels:    labels("messagebus"),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: cell.APIVersion,
					Kind:       cell.Kind,
					Name:       cell.Name,
					UID:        cell.UID,
				},
			},
		},
		Spec: networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"component": "messagebus",
				},
			},
			Ingress: []networkv1.NetworkPolicyIngressRule{
				{
					From: []networkv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": cell.Namespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: matchLabels,
							},
						},
					},
				},
			},
			PolicyTypes: []networkv1.PolicyType{
				networkv1.PolicyTypeIngress,
			},
		},
	}
}

func proxyCaddyNetowrkPolicy(cell *monitoringv1alpha1.Cell) *networkv1.NetworkPolicy {
	return &networkv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "NetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-allow-prometheus", "proxy-caddy"),
			Namespace: cell.Spec.GitpodNamespace,
			Labels:    labels("proxy-caddy"),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: cell.APIVersion,
					Kind:       cell.Kind,
					Name:       cell.Name,
					UID:        cell.UID,
				},
			},
		},
		Spec: networkv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"component": "proxy",
				},
			},
			Ingress: []networkv1.NetworkPolicyIngressRule{
				{
					From: []networkv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": cell.Namespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: matchLabels,
							},
						},
					},
				},
			},
			PolicyTypes: []networkv1.PolicyType{
				networkv1.PolicyTypeIngress,
			},
		},
	}
}
