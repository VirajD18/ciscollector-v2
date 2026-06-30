package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	dashboardsvc "github.com/VirajD18/ciscollector-v2/pkg/dashboard/service"
	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
	"github.com/gorilla/mux"
)

func (a *App) dashboardSvc() *dashboardsvc.Service {
	if a.DashboardSvc == nil {
		return nil
	}
	return a.DashboardSvc
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (a *App) hostsHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.HostsResponse{Rows: [][]string{}})
		return
	}
	resp, err := svc.Hosts(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) violationsHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.CriticalChecksResponse{Checks: []dashboardsvc.CriticalCheckDef{}})
		return
	}
	resp, err := svc.CriticalChecksFleet(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) criticalChecksHandler(w http.ResponseWriter, r *http.Request) {
	a.violationsHandler(w, r)
}

func (a *App) runsHandler(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.RunsResponse{Runs: []dashboardsvc.RunSummary{}})
		return
	}
	resp, err := svc.Runs(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) strategicHandler(w http.ResponseWriter, r *http.Request) {
	rangeKey := r.URL.Query().Get("range")
	if rangeKey == "" {
		rangeKey = "30d"
	}
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.StrategicResponse{
			Ranges: map[string]dashboardsvc.StrategicRange{
				"30d": {Label: "Last 30 days", Health: 0, Grade: "-", Servers: 0},
			},
		})
		return
	}
	resp, err := svc.Strategic(r.Context(), rangeKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) fleetCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.FleetCategoriesResponse{Categories: []dashboardsvc.FleetCategory{}})
		return
	}
	resp, err := svc.FleetCategories(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) gucDriftHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.GucDriftResponse{})
		return
	}
	resp, err := svc.GucDrift(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) gucBaselineGetHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.GucBaselineResponse{Settings: map[string]string{}})
		return
	}
	resp, err := svc.GucBaseline(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) gucBaselinePutHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Label    string            `json:"label"`
		Settings map[string]string `json:"settings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Settings == nil {
		http.Error(w, "settings is required", http.StatusBadRequest)
		return
	}
	if err := svc.PutGucBaseline(r.Context(), req.Label, req.Settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := svc.GucBaseline(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) gucSnapshotsHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.GucSnapshotsResponse{Snapshots: []dashboardsvc.GucSnapshotEntry{}})
		return
	}
	resp, err := svc.GucSnapshots(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) policiesHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.PoliciesResponse{})
		return
	}
	resp, err := svc.Policies(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) collectorConfigHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.CollectorConfigResponse{})
		return
	}
	resp, err := svc.CollectorConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) runHTMLHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["runId"]
	if a.Repo == nil {
		http.Error(w, "report database not configured", http.StatusServiceUnavailable)
		return
	}
	row, err := a.Repo.GetRunByID(r.Context(), runID)
	if err != nil || row == nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}
	if path := os.Getenv("KSHIELD_HTML_REPORT"); path != "" {
		if b, err := os.ReadFile(path); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if r.URL.Query().Get("download") == "1" {
				w.Header().Set("Content-Disposition", "attachment; filename=klouddbshield_report.html")
			}
			_, _ = w.Write(b)
			return
		}
	}
	title := hostLabelForExport(row)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-report.html", title))
	}
	_, _ = w.Write([]byte(dashboardsvc.RenderRunHTML(r.Context(), a.Repo, row)))
}

func hostLabelForExport(row *reportstore.RunRow) string {
	if row.TargetHost != "" {
		return row.TargetHost
	}
	return row.TargetID
}

func (a *App) hbaScannerHandler(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.HbaScannerResponse{Host: host})
		return
	}
	resp, err := svc.HbaScanner(r.Context(), host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) sslScannerHandler(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.SslScannerResponse{Host: host})
		return
	}
	resp, err := svc.SslScanner(r.Context(), host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) piiScannerHandler(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.PiiScannerResponse{Host: host})
		return
	}
	resp, err := svc.PiiScanner(r.Context(), host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) logParserScannerHandler(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.LogParserScannerResponse{Host: host})
		return
	}
	resp, err := svc.LogParserScanner(r.Context(), host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) logReadinessHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.LogReadinessFleetResponse{Rows: []dashboardsvc.LogReadinessHostRow{}})
		return
	}
	resp, err := svc.LogReadinessFleet(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) inactiveUsersReportHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.InactiveUsersReportResponse{Rows: []dashboardsvc.InactiveUserReportRow{}})
		return
	}
	resp, err := svc.InactiveUsersReport(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) commonUsersReportHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, dashboardsvc.CommonUsersReportResponse{Rows: []dashboardsvc.CommonUserReportRow{}})
		return
	}
	resp, err := svc.CommonUsersReport(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) overviewHandler(w http.ResponseWriter, r *http.Request) {
	svc := a.dashboardSvc()
	if svc == nil {
		writeJSON(w, http.StatusOK, OverviewResponse{
			Summary:   Summary{},
			UpdatedAt: time.Now(),
		})
		return
	}
	dash, err := svc.Overview(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := OverviewResponse{
		Summary: Summary{
			Servers:  dash.Summary.Servers,
			Healthy:  dash.Summary.Healthy,
			Warning:  dash.Summary.Warning,
			Critical: dash.Summary.Critical,
		},
		UpdatedAt: dash.UpdatedAt,
	}
	if len(dash.Servers) > 0 {
		resp.CentralServer = Server{
			ID:     dash.CentralServer.ID,
			Name:   dash.CentralServer.Name,
			IP:     dash.CentralServer.IP,
			Status: dash.CentralServer.Status,
			ServerSummary: ServerSummary{
				TotalCases:  dash.CentralServer.ServerSummary.TotalCases,
				PassedCases: dash.CentralServer.ServerSummary.PassedCases,
			},
		}
		for _, s := range dash.Servers {
			resp.Servers = append(resp.Servers, Server{
				ID:     s.ID,
				Name:   s.Name,
				IP:     s.IP,
				Status: s.Status,
				ServerSummary: ServerSummary{
					TotalCases:  s.ServerSummary.TotalCases,
					PassedCases: s.ServerSummary.PassedCases,
				},
			})
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *App) serverHandler(w http.ResponseWriter, r *http.Request) {
	hostQ := strings.TrimSpace(r.URL.Query().Get("host"))
	instanceQ := strings.TrimSpace(r.URL.Query().Get("instance"))
	serverID := hostQ
	if serverID == "" {
		serverID = instanceQ
	}
	if serverID == "" {
		serverID = strings.TrimSpace(mux.Vars(r)["serverId"])
	}
	svc := a.dashboardSvc()
	if svc == nil {
		http.Error(w, "report database not available", http.StatusServiceUnavailable)
		return
	}

	wantOverview := instanceQ != ""
	parsed := dashboardsvc.ParseHostKey(serverID)
	if !wantOverview && parsed.Database == "" && serverID != "" {
		wantOverview = true
	}

	if wantOverview {
		inst := instanceQ
		if inst == "" {
			inst = parsed.Instance
		}
		overview, err := svc.HostInstanceOverview(r.Context(), inst)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if overview == nil {
			http.Error(w, "server not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, overview)
		return
	}

	reportKey := parsed.HostKey
	if reportKey == "" {
		reportKey = serverID
	}
	detail, err := svc.HostReport(r.Context(), reportKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if detail == nil {
		http.Error(w, "server not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}
