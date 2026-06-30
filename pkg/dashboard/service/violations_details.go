package service

import (
	"regexp"
	"strings"

	"github.com/VirajD18/ciscollector-v2/model"
)

var reStripCmdNoise = regexp.MustCompile(`(?i)(cmd:|cmderr:|outerr:|execvpe|createprocess|wsl\s*\()`)

func violationDetailsFromCIS(r model.Result) string {
	fr := strings.TrimSpace(normalizeFailReason(r.FailReason))
	if fr != "" && !isNoisyFailReason(fr) {
		return trimForTable(fr, 120)
	}
	if guc := extractGucName(r); guc != "" {
		live := gucLiveValue(r)
		if live != "" && live != "-" && live != "ok" {
			return trimForTable(guc+" — "+live, 120)
		}
		if fr != "" {
			if msg := firstReadableFailLine(fr); msg != "" {
				return trimForTable(msg, 120)
			}
		}
		return trimForTable(guc+" (CIS fail)", 120)
	}
	if t := strings.TrimSpace(r.Title); t != "" {
		return trimForTable(t, 120)
	}
	if fr != "" {
		if msg := firstReadableFailLine(fr); msg != "" {
			return trimForTable(msg, 120)
		}
	}
	return strings.TrimSpace(r.Control)
}

func normalizeFailReason(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	for strings.Contains(s, "\n\t") {
		s = strings.ReplaceAll(s, "\n\t", "\n")
	}
	return strings.TrimSpace(s)
}

func isNoisyFailReason(s string) bool {
	if s == "" {
		return false
	}
	if strings.Contains(s, "\n") || strings.Contains(s, "\t") {
		return true
	}
	if len(s) > 160 {
		return true
	}
	return reStripCmdNoise.MatchString(s)
}

func firstReadableFailLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if isNoisyFailReason(line) {
			continue
		}
		if failSettingFromText(line) != "" || len(line) <= 120 {
			return line
		}
	}
	return ""
}
