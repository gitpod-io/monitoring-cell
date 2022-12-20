package gitpod

const (
	App = "gitpod"
)

var (
	matchLabels = map[string]string{
		"app.kubernetes.io/component": "prometheus",
		"app.kubernetes.io/name":      "prometheus",
		"app.kubernetes.io/part-of":   "monitoring-cell",
	}
	targets = []string{
		"agent-smith",
		"blobserve",
		"containerd-metrics",
		"content-service",
		"ide-metrics",
		"ide-service",
		"image-builder-mk3",
		"openvsx-proxy",
		"public-api-server",
		"registry-facade",
		"server",
		"slow-server",
		"usage",
		"ws-daemon",
		"ws-manager-bridge",
		"ws-manager",
		"ws-proxy",
		"ws-scheduler",
	}
)

func labels(target string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/component": target,
		"app.kubernetes.io/name":      App,
		"app.kubernetes.io/part-of":   "monitoring-cell",
	}
}
