package service

import (
	"encoding/json"
	"strings"

	"github.com/VirajD18/ciscollector-v2/model"
	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
)

type networkSnapshot struct {
	HBA         []StrategicHBA
	HBAScanned  bool
	SSLEnforced int
	SSLScanned  bool
}

// buildNetworkConnectivity derives Network & connectivity widgets from SQLite scan reports.
func buildNetworkConnectivity(runs []*reportstore.RunRow) networkSnapshot {
	snap := networkSnapshot{}
	var sslPass, sslFail int

	for _, run := range runs {
		host := hostDisplayLabel(hostLabel(run))
		report := run.Report

		if hba := decodeHBAResults(report); len(hba) > 0 {
			snap.HBAScanned = true
			o, i, sec := hbaStackPercents(hba)
			snap.HBA = append(snap.HBA, StrategicHBA{L: host, O: o, I: i, S: sec})
		} else if o, i, sec, ok := connectionExposureStack(report); ok {
			snap.HBAScanned = true
			snap.HBA = append(snap.HBA, StrategicHBA{L: host, O: o, I: i, S: sec})
		}

		p, f := sslCheckCounts(report)
		sslPass += p
		sslFail += f
	}

	if sslPass+sslFail > 0 {
		snap.SSLScanned = true
		snap.SSLEnforced = sslPass * 100 / (sslPass + sslFail)
	}

	return snap
}

// hbaStackPercents maps HBA checks to Fail (open) vs Pass (secure) — same Pass/Fail model as HBA scanner page.
func hbaStackPercents(hba []model.HBAScannerResult) (open, inactive, secure int) {
	openN, secureN := 0, 0
	for _, h := range hba {
		if strings.EqualFold(strings.TrimSpace(h.Status), "Pass") {
			secureN++
		} else {
			openN++
		}
	}
	total := openN + secureN
	if total == 0 {
		return 0, 0, 0
	}
	return openN * 100 / total, 0, secureN * 100 / total
}

func sslCheckCounts(report map[string]interface{}) (pass, fail int) {
	p, f := countCISByFilter(report, isSSLRelatedCIS)
	pass, fail = p, f
	for _, h := range decodeHBAResults(report) {
		blob := strings.ToLower(h.Title + " " + h.Description)
		if !strings.Contains(blob, "ssl") && !strings.Contains(blob, "hostssl") {
			continue
		}
		if strings.EqualFold(h.Status, "Pass") {
			pass++
		} else if strings.EqualFold(h.Status, "Fail") {
			fail++
		}
	}
	for _, r := range decodeCloudResults(report, "SSL Report", "SSL Audit Report") {
		if strings.EqualFold(r.Status, "Pass") {
			pass++
		} else if strings.EqualFold(r.Status, "Fail") {
			fail++
		}
	}
	if pass+fail > 0 {
		return pass, fail
	}
	return countCISByFilter(report, isConnectionExposureCIS)
}

func countCISByFilter(report map[string]interface{}, pred func(model.Result) bool) (pass, fail int) {
	for _, r := range decodeCISResults(report) {
		if !pred(r) {
			continue
		}
		if strings.EqualFold(r.Status, "Pass") {
			pass++
		} else if strings.EqualFold(r.Status, "Fail") {
			fail++
		}
	}
	return pass, fail
}

// connectionExposureStack maps CIS connection/session logging checks to Open/Secure bar segments (postgres_cis).
func connectionExposureStack(report map[string]interface{}) (open, inactive, secure int, ok bool) {
	pass, fail := countCISByFilter(report, isConnectionExposureCIS)
	if pass+fail == 0 {
		return 0, 0, 0, false
	}
	total := pass + fail
	secure = pass * 100 / total
	open = fail * 100 / total
	return open, 0, secure, true
}

func isSSLRelatedCIS(r model.Result) bool {
	return cisMatchesAny(r, "ssl", "tls", "certificate", "hostssl", "scram-sha", "scram_sha")
}

func isConnectionExposureCIS(r model.Result) bool {
	if isSSLRelatedCIS(r) {
		return true
	}
	ctrl := strings.TrimSpace(r.Control)
	if strings.HasPrefix(ctrl, "6.") {
		return true
	}
	return cisMatchesAny(r,
		"log_connections", "log_disconnections", "log_line_prefix", "log_hostname",
		"connection", "pg_hba", "hostssl", "trust auth", "listen_addresses",
	)
}

func decodeCloudResults(report map[string]interface{}, keys ...string) []model.Result {
	var out []model.Result
	for _, key := range keys {
		raw, ok := report[key]
		if !ok {
			continue
		}
		out = append(out, decodeResultSlice(raw)...)
	}
	return out
}

func decodeResultSlice(raw interface{}) []model.Result {
	if raw == nil {
		return nil
	}
	if m, ok := asMap(raw); ok {
		if res, ok := m["result"]; ok {
			return decodeResultSlice(res)
		}
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var rows []model.Result
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil
	}
	return rows
}
