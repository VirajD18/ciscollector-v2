package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/klouddb/klouddbshield/model"
	"github.com/klouddb/klouddbshield/pkg/config"
	cons "github.com/klouddb/klouddbshield/pkg/const"
	"github.com/klouddb/klouddbshield/pkg/logparser"
	"github.com/klouddb/klouddbshield/pkg/postgresdb"
	"github.com/klouddb/klouddbshield/pkg/utils"
)

func refreshPgSettingsForLogParser(ctx context.Context, store *sql.DB, pgSettings *model.PgSettings) error {
	if store == nil || pgSettings == nil {
		return nil
	}
	ps, err := utils.GetPGSettings(ctx, store)
	if err != nil {
		return err
	}
	pgSettings.LogConnections = ps.LogConnections
	if prefix, err := utils.GetLoglinePrefix(ctx, store); err == nil && prefix != "" {
		pgSettings.LogLinePrefix = prefix
	}
	return nil
}

func embedLogReadinessInFileData(
	fileData map[string]interface{},
	logConnectionsOn bool,
	logLinePrefix string,
	commands []string,
	startedAt, finishedAt time.Time,
) {
	if fileData == nil {
		return
	}
	report := logparser.BuildReadinessReport(logparser.ReadinessInput{
		LogConnectionsOn: logConnectionsOn,
		LogLinePrefix:    logLinePrefix,
		Commands:         commands,
	})
	if !startedAt.IsZero() {
		report["started_at"] = startedAt.UTC()
	}
	if !finishedAt.IsZero() {
		report["finished_at"] = finishedAt.UTC()
	}
	fileData[logparser.LogReadinessReportKey] = report
}

func logParserCommandsFromConfig(cnf *config.Config) []string {
	if cnf == nil {
		return nil
	}
	var commands []string
	if cnf.LogParser != nil && len(cnf.LogParser.Commands) > 0 {
		commands = append(commands, cnf.LogParser.Commands...)
	} else {
		commands = parseCommaScanCommands(cnf.Collector.ScanCommands)
	}
	var out []string
	for _, cmd := range commands {
		if isLogParserScanCommand(cmd) {
			out = append(out, cmd)
		}
	}
	return out
}

func parseCommaScanCommands(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isLogParserScanCommand(cmd string) bool {
	switch strings.TrimSpace(cmd) {
	case cons.LogParserCMD_InactiveUser,
		cons.LogParserCMD_UniqueIPs,
		cons.LogParserCMD_HBAUnusedLines,
		cons.LogParserCMD_PasswordLeakScanner,
		cons.LogParserCMD_SqlInjectionScan,
		cons.LogParserCMD_All:
		return true
	default:
		return false
	}
}

func gucValuesFromFileData(fileData map[string]interface{}) (logConnections, logLinePrefix string, ok bool) {
	if fileData == nil {
		return "", "", false
	}
	raw, exists := fileData[gucSettingsReportKey]
	if !exists {
		return "", "", false
	}
	report, ok := raw.(map[string]interface{})
	if !ok {
		return "", "", false
	}
	settingsRaw, ok := report["settings"]
	if !ok {
		return "", "", false
	}
	b, err := json.Marshal(settingsRaw)
	if err != nil {
		return "", "", false
	}
	settings := map[string]string{}
	if err := json.Unmarshal(b, &settings); err != nil {
		return "", "", false
	}
	logConnections = strings.TrimSpace(settings["log_connections"])
	logLinePrefix = settings["log_line_prefix"]
	return logConnections, logLinePrefix, logConnections != "" || logLinePrefix != ""
}

func collectLogReadinessIntoScanPayload(
	cnf *config.Config,
	pg *postgresdb.Postgres,
	fileData map[string]interface{},
	startedAt, finishedAt time.Time,
) {
	if cnf == nil || fileData == nil {
		return
	}
	if _, exists := fileData[logparser.LogReadinessReportKey]; exists {
		return
	}

	commands := logParserCommandsFromConfig(cnf)
	logConnectionsOn := false
	logLinePrefix := ""

	if conn, prefix, ok := gucValuesFromFileData(fileData); ok {
		logConnectionsOn = strings.EqualFold(conn, "on") || strings.EqualFold(conn, "yes")
		logLinePrefix = prefix
	} else if cnf.LogParser != nil {
		logConnectionsOn = cnf.LogParser.PgSettings.LogConnections
		logLinePrefix = cnf.LogParser.PgSettings.LogLinePrefix
	}

	if pg != nil {
		if store, _, err := postgresdb.Open(*pg); err == nil && store != nil {
			defer store.Close()
			ctx := context.Background()
			if ps, err := utils.GetPGSettings(ctx, store); err == nil {
				logConnectionsOn = ps.LogConnections
			}
			if prefix, err := utils.GetLoglinePrefix(ctx, store); err == nil && prefix != "" {
				logLinePrefix = prefix
			}
		}
	}

	if logLinePrefix == "" && !logConnectionsOn && len(commands) == 0 {
		return
	}

	embedLogReadinessInFileData(fileData, logConnectionsOn, logLinePrefix, commands, startedAt, finishedAt)
}
