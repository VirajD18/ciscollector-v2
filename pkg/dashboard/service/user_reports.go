package service

import (
	"context"
	"sort"
	"strings"
)

// InactiveUserReportRow is one inactive user across the fleet (grouped by instance).
type InactiveUserReportRow struct {
	User           string   `json:"user"`
	Host           string   `json:"host"`
	Instance       string   `json:"instance"`
	Databases      []string `json:"databases,omitempty"`
	DatabasesLabel string   `json:"databases_label"`
	Status         string   `json:"status"`
}

// InactiveUsersReportResponse is fleet-wide inactive users from log parser scans.
type InactiveUsersReportResponse struct {
	Rows      []InactiveUserReportRow `json:"rows"`
	HostCount int                     `json:"hostCount"`
	UserCount int                     `json:"userCount"`
	Message   string                  `json:"message,omitempty"`
}

// CommonUserReportRow is one login-capable DB role across the fleet (grouped by instance).
type CommonUserReportRow struct {
	User           string   `json:"user"`
	Host           string   `json:"host"`
	Instance       string   `json:"instance"`
	Databases      []string `json:"databases,omitempty"`
	DatabasesLabel string   `json:"databases_label"`
}

// CommonUsersReportResponse is fleet-wide login roles from Users Report scans.
type CommonUsersReportResponse struct {
	Rows      []CommonUserReportRow `json:"rows"`
	UserCount int                   `json:"userCount"`
	HostCount int                   `json:"hostCount"`
	Message   string                `json:"message,omitempty"`
}

type userInstanceAgg struct {
	user   string
	status string
	dbs    map[string]bool
}

func mergeUserInstanceRow(agg map[string]*userInstanceAgg, user, host, status string) {
	user = strings.TrimSpace(user)
	if user == "" || user == "-" {
		return
	}
	inst := fleetHostInstance(host)
	if inst == "" {
		inst = strings.TrimSpace(host)
	}
	db := fleetHostDatabase(host)
	key := user + "|" + inst
	row := agg[key]
	if row == nil {
		row = &userInstanceAgg{user: user, status: status, dbs: map[string]bool{}}
		agg[key] = row
	}
	if status != "" && row.status == "" {
		row.status = status
	}
	if db != "" {
		row.dbs[db] = true
	}
}

func userInstanceRowsFromAgg(agg map[string]*userInstanceAgg) []struct {
	user, inst, status, dbLabel string
	dbs                         []string
} {
	keys := make([]string, 0, len(agg))
	for k := range agg {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]struct {
		user, inst, status, dbLabel string
		dbs                         []string
	}, 0, len(keys))
	for _, k := range keys {
		row := agg[k]
		user, inst := row.user, ""
		if parts := strings.SplitN(k, "|", 2); len(parts) == 2 {
			if user == "" {
				user = parts[0]
			}
			inst = parts[1]
		}
		dbs := make([]string, 0, len(row.dbs))
		for db := range row.dbs {
			dbs = append(dbs, db)
		}
		sort.Strings(dbs)
		dbLabel := instanceDatabasesLabel(inst, map[string][]string{inst: dbs}, dbs)
		out = append(out, struct {
			user, inst, status, dbLabel string
			dbs                         []string
		}{user: user, inst: inst, status: row.status, dbLabel: dbLabel, dbs: dbs})
	}
	return out
}

// InactiveUsersReport aggregates inactive_users log-parser output from latest scans per host.
func (s *Service) InactiveUsersReport(ctx context.Context) (*InactiveUsersReportResponse, error) {
	runs, err := s.latestRunsByTargetWithInactiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	resp := &InactiveUsersReportResponse{Rows: []InactiveUserReportRow{}}
	agg := map[string]*userInstanceAgg{}

	for _, run := range runs {
		if run == nil || run.Report == nil {
			continue
		}
		host := hostLabel(run)
		for _, row := range inactiveUserRowsFromReport(host, run.Report) {
			if len(row) < 3 {
				continue
			}
			mergeUserInstanceRow(agg, row[0], row[1], row[2])
		}
	}

	instances := map[string]bool{}
	users := map[string]bool{}
	for _, item := range userInstanceRowsFromAgg(agg) {
		instances[item.inst] = true
		users[item.user] = true
		resp.Rows = append(resp.Rows, InactiveUserReportRow{
			User:           item.user,
			Host:           item.inst,
			Instance:       item.inst,
			Databases:      item.dbs,
			DatabasesLabel: item.dbLabel,
			Status:         item.status,
		})
	}

	resp.HostCount = len(instances)
	resp.UserCount = len(users)
	if len(resp.Rows) == 0 {
		resp.Message = "No inactive users found. Run collector with log parser inactive_users (menu 6) on each host."
	}
	return resp, nil
}

// CommonUsersReport aggregates login-capable roles from Users Report across latest scans.
func (s *Service) CommonUsersReport(ctx context.Context) (*CommonUsersReportResponse, error) {
	runs, err := s.latestRunsByTarget(ctx)
	if err != nil {
		return nil, err
	}
	resp := &CommonUsersReportResponse{Rows: []CommonUserReportRow{}}
	agg := map[string]*userInstanceAgg{}

	for _, run := range runs {
		if run == nil || run.Report == nil {
			continue
		}
		host := hostLabel(run)
		sections := decodeUsersReportSections(run.Report)
		for _, role := range loginRolesFromUsersList(sections) {
			mergeUserInstanceRow(agg, role, host, "")
		}
	}

	instances := map[string]bool{}
	users := map[string]bool{}
	for _, item := range userInstanceRowsFromAgg(agg) {
		instances[item.inst] = true
		users[item.user] = true
		resp.Rows = append(resp.Rows, CommonUserReportRow{
			User:           item.user,
			Host:           item.inst,
			Instance:       item.inst,
			Databases:      item.dbs,
			DatabasesLabel: item.dbLabel,
		})
	}

	resp.HostCount = len(instances)
	resp.UserCount = len(users)
	if len(resp.Rows) == 0 {
		resp.Message = "No login-capable DB users in latest scans. Run collector with Users Report (menu 9) on each host."
	}
	return resp, nil
}
