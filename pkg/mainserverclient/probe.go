package mainserverclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ProbeStep is one connection check against the main-server.
type ProbeStep struct {
	Name   string
	Path   string
	OK     bool
	Detail string
	Hint   string
}

// ProbeReport summarizes collector → main-server connectivity.
type ProbeReport struct {
	BaseURL  string
	NodeID   string
	Hostname string
	Steps    []ProbeStep
}

// OK is true when every probe step succeeded.
func (r ProbeReport) OK() bool {
	for _, s := range r.Steps {
		if !s.OK {
			return false
		}
	}
	return len(r.Steps) > 0
}

// Probe checks reachability, token auth (heartbeat), and register API without queueing retries.
func (c *Client) Probe(ctx context.Context) ProbeReport {
	report := ProbeReport{
		BaseURL:  c.baseURL,
		NodeID:   c.nodeID,
		Hostname: c.hostname,
	}

	report.Steps = append(report.Steps, c.probeReachability(ctx))
	report.Steps = append(report.Steps, c.probeHeartbeat(ctx))
	report.Steps = append(report.Steps, c.probeRegister(ctx))
	return report
}

func (c *Client) probeReachability(ctx context.Context) ProbeStep {
	step := ProbeStep{Name: "reachability", Path: "/api/collector/config"}
	status, detail, err := c.getStatus(ctx, "/api/collector/config", false)
	if err != nil {
		step.Detail = err.Error()
		step.Hint = "start main-server first (e.g. main-server.exe -addr :8081 -dbdir C:\\etc\\klouddbshield\\db)"
		return step
	}
	step.OK = status >= 200 && status < 300
	step.Detail = fmt.Sprintf("HTTP %d", status)
	if !step.OK {
		step.Detail = formatHTTPDetail(status, detail)
		step.Hint = "confirm mainserver.url points at the KloudDB Shield main-server"
	}
	return step
}

func (c *Client) probeHeartbeat(ctx context.Context) ProbeStep {
	step := ProbeStep{Name: "auth (heartbeat)", Path: "/api/collector/heartbeat"}
	err := c.postDirect(ctx, "/api/collector/heartbeat", heartbeatRequest{
		SchemaVersion: "v1",
		NodeID:        c.nodeID,
		Hostname:      c.hostname,
		Timestamp:     time.Now().UTC(),
		Status: HeartbeatPayload{
			CronRunning:   false,
			ScheduledJobs: 0,
		},
	})
	if err == nil {
		step.OK = true
		step.Detail = "HTTP 200"
		return step
	}
	step.Detail = err.Error()
	if ae, ok := err.(*APIError); ok {
		switch ae.StatusCode {
		case 401:
			step.Hint = "match [mainserver] token in kshieldconfig.toml with server-node.yaml on the main-server host"
		case 404:
			step.Hint = "rebuild and restart main-server (heartbeat API missing)"
		case 503:
			step.Hint = "configure collector token on main-server (server-node.yaml)"
		}
	} else {
		step.Hint = "start main-server and verify mainserver.url is reachable"
	}
	return step
}

func (c *Client) probeRegister(ctx context.Context) ProbeStep {
	step := ProbeStep{Name: "register", Path: "/api/collector/register"}
	err := c.postDirect(ctx, "/api/collector/register", registerRequest{
		SchemaVersion: "v1",
		NodeID:        c.nodeID,
		Hostname:      c.hostname,
		Timestamp:     time.Now().UTC(),
		ScheduledJobs: 0,
	})
	if err == nil {
		step.OK = true
		step.Detail = "HTTP 200"
		return step
	}
	step.Detail = err.Error()
	if ae, ok := err.(*APIError); ok {
		switch ae.StatusCode {
		case 401:
			step.Hint = "match [mainserver] token in kshieldconfig.toml with server-node.yaml on the main-server host"
		case 404:
			step.Hint = "rebuild and restart main-server (register API missing on this binary)"
		case 503:
			step.Hint = "configure collector token on main-server (server-node.yaml)"
		}
	} else {
		step.Hint = "start main-server and verify mainserver.url is reachable"
	}
	return step
}

func (c *Client) getStatus(ctx context.Context, path string, withAuth bool) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return 0, "", err
	}
	if withAuth {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, strings.TrimSpace(string(body)), nil
}

func formatHTTPDetail(status int, body string) string {
	if body == "" {
		return fmt.Sprintf("HTTP %d", status)
	}
	if len(body) > 120 {
		body = body[:120] + "..."
	}
	return fmt.Sprintf("HTTP %d (%s)", status, body)
}
