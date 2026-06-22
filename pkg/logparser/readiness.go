package logparser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	cons "github.com/klouddb/klouddbshield/pkg/const"
)

const LogReadinessReportKey = "Log Readiness Report"

var cisLogLinePrefixTokens = []string{"%m", "%p", "%l", "%d", "%u", "%a", "%h"}

var defaultLogParserCommands = []string{
	cons.LogParserCMD_InactiveUser,
	cons.LogParserCMD_UniqueIPs,
	cons.LogParserCMD_HBAUnusedLines,
	cons.LogParserCMD_PasswordLeakScanner,
}

// ReadinessInput is live GUC state plus configured parser commands.
type ReadinessInput struct {
	LogConnectionsOn bool
	LogLinePrefix    string
	Commands         []string
}

// BuildReadinessReport embeds CIS LMA status and per-command parser gates.
func BuildReadinessReport(in ReadinessInput) map[string]interface{} {
	commands := in.Commands
	if len(commands) == 0 {
		commands = append([]string(nil), defaultLogParserCommands...)
	}

	connDisplay := "off"
	if in.LogConnectionsOn {
		connDisplay = "on"
	}

	prefixCIS := evaluatePrefixCIS(in.LogLinePrefix)
	cisLMA := evaluateCISLMA(in.LogConnectionsOn, prefixCIS)

	gates := make([]map[string]interface{}, 0, len(commands))
	runnable := make([]string, 0, len(commands))
	menuNums := make([]string, 0, len(commands))

	for _, cmd := range commands {
		canRun, reason := parserGate(in.LogConnectionsOn, in.LogLinePrefix, cmd)
		menu := menuNumberForCommand(cmd)
		gates = append(gates, map[string]interface{}{
			"command": cmd,
			"can_run": canRun,
			"reason":  reason,
			"menu":    menu,
		})
		if canRun {
			runnable = append(runnable, cmd)
			if menu != "" {
				menuNums = append(menuNums, menu)
			}
		}
	}

	return map[string]interface{}{
		"log_connections":      connDisplay,
		"log_connections_on":   in.LogConnectionsOn,
		"log_line_prefix":      in.LogLinePrefix,
		"log_line_prefix_cis":  prefixCIS,
		"cis_lma":              cisLMA,
		"parser_gates":         gates,
		"can_run_menus":        formatCanRunMenus(menuNums, cisLMA),
		"runnable_commands":    runnable,
		"logparser_readiness":  evaluateLogParserReadiness(len(runnable)),
	}
}

func evaluateLogParserReadiness(runnableCount int) string {
	if runnableCount > 0 {
		return "PASS"
	}
	return "FAIL"
}

func evaluateCISLMA(logConnectionsOn bool, prefixCIS string) string {
	connOK := logConnectionsOn
	prefixOK := prefixCIS == "pass"
	if connOK && prefixOK {
		return "pass"
	}
	if !connOK && prefixCIS == "fail" {
		return "fail"
	}
	return "partial"
}

func evaluatePrefixCIS(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "fail"
	}
	missing := 0
	for _, tok := range cisLogLinePrefixTokens {
		if !strings.Contains(prefix, tok) {
			missing++
		}
	}
	if missing == 0 {
		return "pass"
	}
	if missing < len(cisLogLinePrefixTokens) {
		return "partial"
	}
	return "fail"
}

func parserGate(logConnectionsOn bool, logLinePrefix, command string) (bool, string) {
	prefix := logLinePrefix
	switch command {
	case cons.LogParserCMD_InactiveUser:
		if strings.Contains(prefix, "%u") || logConnectionsOn {
			return true, ""
		}
		return false, "set log_line_prefix to '%u' or enable log_connections"
	case cons.LogParserCMD_UniqueIPs:
		if strings.Contains(prefix, "%h") || strings.Contains(prefix, "%r") || logConnectionsOn {
			return true, ""
		}
		return false, "set log_line_prefix to '%h' or '%r' or enable log_connections"
	case cons.LogParserCMD_HBAUnusedLines:
		if logConnectionsOn {
			if (strings.Contains(prefix, "%h") || strings.Contains(prefix, "%r")) ||
				(strings.Contains(prefix, "%u") && strings.Contains(prefix, "%d")) {
				return true, ""
			}
			return false, "with log_connections enabled, set log_line_prefix to '%h' or '%r' or '%u' and '%d'"
		}
		if (strings.Contains(prefix, "%h") || strings.Contains(prefix, "%r")) &&
			strings.Contains(prefix, "%u") && strings.Contains(prefix, "%d") {
			return true, ""
		}
		return false, "set log_line_prefix to '%h' or '%r' or '%u' and '%d'"
	case cons.LogParserCMD_PasswordLeakScanner, cons.LogParserCMD_SqlInjectionScan:
		return true, ""
	default:
		return false, fmt.Sprintf("unknown log parser command %q", command)
	}
}

func menuNumberForCommand(command string) string {
	for i, item := range cons.CommandList {
		if item.CMD == command {
			return fmt.Sprintf("%d", i+1)
		}
	}
	return ""
}

func formatCanRunMenus(menuNums []string, cisLMA string) string {
	if len(menuNums) == 0 {
		if cisLMA == "fail" || cisLMA == "partial" {
			return "None until GUCs fixed"
		}
		return "None"
	}
	sort.Slice(menuNums, func(i, j int) bool {
		a, errA := strconv.Atoi(menuNums[i])
		b, errB := strconv.Atoi(menuNums[j])
		if errA != nil || errB != nil {
			return menuNums[i] < menuNums[j]
		}
		return a < b
	})
	return strings.Join(menuNums, " · ")
}
