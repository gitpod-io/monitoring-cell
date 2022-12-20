package prometheus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/Jeffail/gabs"
	"k8s.io/client-go/rest"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// Response hold API response in a form similar to apiResponse struct from prometheus/client_golang
// https://github.com/prometheus/client_golang/blob/master/api/prometheus/v1/api.go
type Response struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

// apiRequest makes a request against specified Prometheus API endpoint
func apiRequest(endpoint string, selector string, query string, cell *monitoringv1alpha1.Cell, restClient rest.Interface) (Response, error) {
	req := restClient.Get().
		Namespace(cell.Namespace).
		Resource("pods").
		SubResource("proxy").
		Name(fmt.Sprintf("prometheus-%s-%s-0:9090", Name, cell.Name)).
		Suffix(endpoint).Param(selector, query)

	var data Response
	b, err := req.DoRaw(context.Background())
	if err != nil {
		return data, err
	}

	r := bytes.NewReader(b)
	decoder := json.NewDecoder(r)
	err = decoder.Decode(&data)
	if err != nil {
		return data, err
	}

	if data.Status != "success" {
		return data, fmt.Errorf("status of returned response was not successful; status: %s", data.Status)
	}

	return data, err
}

// Query makes a request against the Prometheus /api/v1/query endpoint.
func Query(query string, cell *monitoringv1alpha1.Cell, restClient rest.Interface) (int, error) {
	req := restClient.Get().
		Namespace(cell.Namespace).
		Resource("pods").
		SubResource("proxy").
		Name(fmt.Sprintf("prometheus-%s-%s-0:9090", Name, cell.Name)).
		Suffix("/api/v1/query").Param("query", query)

	b, err := req.DoRaw(context.Background())
	if err != nil {
		return 0, err
	}

	res, err := gabs.ParseJSON(b)
	if err != nil {
		return 0, err
	}

	n, err := res.ArrayCountP("data.result")
	return n, err
}

// metadata makes a request against the Prometheus /api/v1/targets/metadata endpoint.
// It returns all the metrics and its metadata.
func Metadata(query string, cell *monitoringv1alpha1.Cell, restClient rest.Interface) ([]promv1.MetricMetadata, error) {
	var metadata []promv1.MetricMetadata
	rsp, err := apiRequest("/api/v1/targets/metadata", "match_target", query, cell, restClient)
	if err != nil {
		return metadata, err
	}

	r := bytes.NewReader(rsp.Data)
	decoder := json.NewDecoder(r)
	err = decoder.Decode(&metadata)
	if err != nil {
		return metadata, err
	}
	return metadata, err
}

// targets makes a request against the Prometheus /api/v1/targets endpoint.
// It returns all targets registered in prometheus.
func Targets(cell *monitoringv1alpha1.Cell, restClient rest.Interface) (promv1.TargetsResult, error) {
	var targets promv1.TargetsResult
	rsp, err := apiRequest("/api/v1/targets", "state", "any", cell, restClient)
	if err != nil {
		return targets, err
	}

	r := bytes.NewReader(rsp.Data)
	decoder := json.NewDecoder(r)
	err = decoder.Decode(&targets)
	if err != nil {
		return targets, err
	}

	return targets, err
}
