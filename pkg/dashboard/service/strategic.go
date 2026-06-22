package service

import (
	"context"
	"strings"
)

// Strategic returns fleet overview KPIs and widgets from all latest runs in SQLite.
func (s *Service) Strategic(ctx context.Context, rangeKey string) (*StrategicResponse, error) {
	if rangeKey == "" {
		rangeKey = "30d"
	}
	label := "Last 30 days"
	switch rangeKey {
	case "7d":
		label = "Last 7 days"
	case "24h":
		label = "Last 24 hours"
	}

	since := rangeCutoff(rangeKey)
	runs, err := s.latestRunsByTargetSince(ctx, since)
	if err != nil {
		return nil, err
	}

	r := emptyStrategicRange(label)
	if len(runs) == 0 {
		return &StrategicResponse{Ranges: map[string]StrategicRange{rangeKey: r}}, nil
	}

	r.Servers = len(runs)

	var scoreSum float64
	var scoreN int
	var totalPass, totalFail, criticalTotal int
	passwordLeakHosts := map[string]bool{}

	for _, run := range runs {
		host := hostLabel(run)
		cis := decodeCISResults(run.Report)
		passN, failN, hostScore := runCISSummary(run)
		if len(cis) > 0 && passN+failN == 0 {
			passN, failN, hostScore = summarizeCISResults(cis)
		}
		checkFails := countCriticalCheckFailures(s.criticalChecksForRun(ctx, run))

		scoreSum += hostScore
		scoreN++
		totalPass += passN
		totalFail += failN
		criticalTotal += checkFails

		if passwordManagerText(run.Report) != "" || hasLogPasswordLeak(run.Report) {
			passwordLeakHosts[host] = true
		}
	}

	if scoreN > 0 {
		r.Health = int(scoreSum / float64(scoreN))
	}
	if totalPass+totalFail > 0 {
		r.CIS = int(float64(totalPass) / float64(totalPass+totalFail) * 100)
	} else if scoreN > 0 {
		r.CIS = r.Health
	}
	r.Critical = criticalTotal
	r.Grade, r.GradeColor = gradeFromHealth(r.Health)

	r.Privs, r.Hygiene, r.Cred = buildIdentityAccess(runs, passwordLeakHosts)

	net := buildNetworkConnectivity(runs)
	r.HBA = net.HBA
	r.HBAScanned = net.HBAScanned
	r.SSLEnforced = net.SSLEnforced
	r.SSLScanned = net.SSLScanned

	comp := buildComplianceSnapshot(ctx, s, runs, since, rangeKey)
	r.Drift = comp.Drift
	r.DriftLabels = comp.DriftLabels
	r.Audit = comp.Audit
	r.Heatmap = comp.Heatmap
	r.HeatmapColumns = comp.HeatmapColumns
	r.PiiScanned = comp.PiiScanned

	return &StrategicResponse{Ranges: map[string]StrategicRange{rangeKey: r}}, nil
}

func emptyStrategicRange(label string) StrategicRange {
	return StrategicRange{
		Label:          label,
		Privs:          []StrategicBar{},
		Hygiene:        StrategicHygiene{},
		HBA:            []StrategicHBA{},
		Drift:          []StrategicDrift{},
		DriftLabels:    []string{},
		Audit:          [][]string{},
		Heatmap:        [][]int{},
		HeatmapColumns: []string{},
	}
}

func hasLogPasswordLeak(report map[string]interface{}) bool {
	for _, e := range decodeLogParserEntries(report) {
		title, _ := e["title"].(string)
		if strings.Contains(strings.ToLower(title), "password") || strings.Contains(strings.ToLower(title), "leak") {
			return true
		}
	}
	return false
}
