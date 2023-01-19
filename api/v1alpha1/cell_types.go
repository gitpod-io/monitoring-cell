/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	pov1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CellSpec defines the desired state of Cell
type CellSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ClusterName will be added as extra data to all metrics, logs and traces when being sent to a remote storage
	ClusterName string `json:"cluster_name,omitempty"`

	// GitpodNamespace identifies the namespace where Gitpod components were deployed to
	GitpodNamespace string      `json:"gitpodNamespace,omitempty"`
	Metrics         MetricsSpec `json:"metrics,omitempty"`
	Logs            LogsSpec    `json:"logs,omitempty"`
	Traces          TracesSpec  `json:"traces,omitempty"`
}

// MetricsSpec defines how metrics are handled within a monitoring cell
type MetricsSpec struct {
	// UpstreamRemoteWrites defines the remote-write configuration used by the Prometheus instance
	UpstreamRemoteWrites []pov1.RemoteWriteSpec `json:"upstreamRemoteWrite"`

	// Droplist defines metrics that will be dropped during scrape time. Metrics added to Droplist won't be available at any stage of our metrics pipeline
	Droplist []string `json:"dropList,omitempty"`

	// UpstreamAllowList defines which metrics are allowed to be remote-written to upstream
	UpstreamAllowlist []string `json:"upstreamAllowList,omitempty"`
}

// LogsSpec defines how logs are handled within a monitoring cell
type LogsSpec struct {
}

// TracesSpec defines how traces are handled within a monitoring cell
type TracesSpec struct {
}

// CellStatus defines the observed state of Cell
type CellStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PrometheusOperatorReady reports whether Prometheus-Operator is in a ready or broken state
	PrometheusOperatorReady OperatorReconciliationStatus `json:"prometheusOperatorReady,omitempty"`

	// PrometheusReady reports whether Prometheus is in a ready or broken state
	PrometheusReady OperatorReconciliationStatus `json:"prometheusReady,omitempty"`

	// NodeExporterReady reports whether Prometheus is able to scrape node-exporter metrics or not
	NodeExporterReady ExporterReconciliationStatus `json:"nodeExporterReady,omitempty"`

	// KubeStateMetricsReady reports whether Prometheus is able to scrape node-exporter metrics or not
	KubeStateMetricsReady ExporterReconciliationStatus `json:"kubeStateMetricsReady,omitempty"`

	// KubeletReady reports whether Prometheus is able to scrape node-exporter metrics or not
	KubeletReady ExporterReconciliationStatus `json:"kubeletReady,omitempty"`

	// APIServerReady reports whether Prometheus is able to scrape node-exporter metrics or not
	APIServerReady ExporterReconciliationStatus `json:"apiServerReady,omitempty"`
}

type OperatorReconciliationStatus struct {
	LastModified metav1.Time `json:"lastModified,omitempty"`

	Status OperatorStatusType `json:"status,omitempty"`

	StatusMessage string `json:"message,omitempty"`
}

type OperatorStatusType string

const (
	OperatorStatusUnkown      OperatorStatusType = "Unknown"
	OperatorStatusReconciling OperatorStatusType = "Reconciling"
	OperatorStatusReady       OperatorStatusType = "Ready"
)

type ExporterReconciliationStatus struct {
	LastModified metav1.Time `json:"lastModified,omitempty"`

	Status ExporterStatusType `json:"status,omitempty"`

	StatusMessage string `json:"message,omitempty"`
}

type ExporterStatusType string

const (
	ExporterReconciling    ExporterStatusType = "Reconciling"
	ExporterUnknown        ExporterStatusType = "Unknown"
	ExporterMetricNotFount ExporterStatusType = "MetricNotFound"
	ExporterReady          ExporterStatusType = "Ready"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Cell is the Schema for the cells API
type Cell struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CellSpec   `json:"spec,omitempty"`
	Status CellStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CellList contains a list of Cell
type CellList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cell `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cell{}, &CellList{})
}
