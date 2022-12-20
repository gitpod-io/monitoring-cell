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

	Metrics MetricsSpec `json:"metrics,omitempty"`
	Logs    LogsSpec    `json:"logs,omitempty"`
	Traces  TracesSpec  `json:"traces,omitempty"`
}

// MetricsSpec defines how metrics are handled within a monitoring cell
type MetricsSpec struct {
	// UpstreamRemoteWrites defines the remote-write configuration used by the Prometheus instance
	UpstreamRemoteWrites []pov1.RemoteWriteSpec `json:"upstream_remote_writes"`

	// Droplist defines metrics that will be dropped during scrape time. Metrics added to Droplist won't be available at any stage of our metrics pipeline
	Droplist []string `json:"drop_list,omitempty"`

	// UpstreamAllowList defines which metrics are allowed to be remote-written to upstream
	UpstreamAllowlist []string `json:"upstream_allow_list,omitempty"`
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

	// PrometheusReady reports whether Prometheus is in a ready or broken state
	PrometheusReady *bool `json:"prometheus_ready,omitempty"`
}

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
