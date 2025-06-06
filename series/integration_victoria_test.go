package series

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/tychoish/fun/ft"
	"github.com/tychoish/fun/testt"
)

// Shared HTTP client for all VictoriaMetrics interactions.
var httpClient = &http.Client{Timeout: 5 * time.Second}

type victoriaInstance struct {
	containerID string
	external    bool // true if we connected to an already-running instance
}

func (v *victoriaInstance) stop(t *testing.T) {
	t.Helper()

	if v == nil {
		return
	}

	if v.containerID != "" && !v.external {
		if err := exec.Command("docker", "kill", v.containerID).Run(); err != nil {
			t.Errorf("failed to kill victoria-metrics container %s: %v", v.containerID, err)
		}
		v.containerID = ""
	}
}

func startVictoriaMetrics(t *testing.T) *victoriaInstance {
	t.Helper()

	// If VictoriaMetrics is already running locally, just use it.
	if conn, err := net.DialTimeout("tcp", "127.0.0.1:8428", time.Second); err == nil {
		_ = conn.Close()
		t.Log("using pre-existing victoria-metrics on 127.0.0.1")
		inst := &victoriaInstance{external: true}
		t.Cleanup(func() { inst.stop(t) })
		return inst
	}

	if os.Getenv("GITHUB_ACTIONS") != "" {
		// TODO: figure out how to get the image to start correctly.
		t.Skip("victoria metrics fixture inoperable in CI")
	}

	// Docker must be available for a local container.
	if _, err := exec.LookPath("docker"); err != nil {
		t.Fatalf("victoria-metrics not running locally and docker unavailable: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	containerName := fmt.Sprintf("victoria-metrics-test-%d", time.Now().UnixNano())
	runArgs := []string{
		"run", "-d", "--name", containerName,
		"-p", "8428:8428", // HTTP API
		"-p", "2003:2003", // Graphite TCP
		"-p", "8125:8125/udp", // statsd UDP
		"victoriametrics/victoria-metrics:v1.118.0",
		"-opentsbListenAddr=:8125",
		"-graphiteListenAddr=:2003",
	}

	out, err := exec.CommandContext(ctx, "docker", runArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("failed launching victoria-metrics docker container: %v – output: %s", err, string(out))
	}

	inst := &victoriaInstance{containerID: strings.TrimSpace(string(out))}
	t.Cleanup(func() { inst.stop(t) })

	readyCtx, readyCancel := context.WithTimeout(t.Context(), time.Minute)
	defer readyCancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		req, _ := http.NewRequestWithContext(readyCtx, http.MethodGet, "http://127.0.0.1:8428/health", nil)
		resp, err := httpClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}

		select {
		case <-readyCtx.Done():
			t.Fatalf("victoria-metrics health endpoint not ready within timeout: %v", readyCtx.Err())
		case <-ticker.C:
		}
	}

	return inst
}

func victoriaHasMetric(ctx context.Context, t *testing.T, metric string) (_ bool, err error) {
	url := fmt.Sprintf("http://127.0.0.1:8428/api/v1/query?query=%s", metric)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { ft.IgnoreError(resp.Body.Close()) }()

	var payload struct {
		Status string `json:"status"`
		Data   struct {
			Type   string `json:"resultType"`
			Result []struct {
				Metadata map[string]string `json:"metric"`
				Value    []any             `json:"value"`
			} `json:"result"`
		} `json:"data"`
		Stats struct {
			SeriesFetched string `json:"seriesFetched"`
		} `json:"stats"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, err
	}
	testt.Logf(t, "%+v", payload)

	if payload.Status != "success" {
		return false, fmt.Errorf("unexpected status %q", payload.Status)
	}

	return len(payload.Data.Result) > 0, nil
}
