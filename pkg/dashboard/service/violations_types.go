package service

import (
	"strings"

	"github.com/VirajD18/ciscollector-v2/model"
)

func violationTypeFromCIS(r model.Result) string {
	blob := strings.ToLower(r.Title + " " + r.Control + " " + r.FailReason + " " + r.Description)
	switch {
	case strings.Contains(blob, "ssl") || strings.Contains(blob, "tls") || strings.Contains(blob, "certificate"):
		return "SSL Violation"
	case strings.Contains(blob, "password") || strings.Contains(blob, "leak") || strings.Contains(blob, "credential"):
		return "Password Leak"
	case strings.Contains(blob, "superuser") || strings.Contains(blob, "privilege") ||
		strings.Contains(blob, "createrole") || strings.Contains(blob, "bypassrls"):
		return "Unauthorized Superuser"
	case strings.Contains(blob, "pii") || strings.Contains(blob, "personal data"):
		return "PII Exposure"
	default:
		return "Critical Config"
	}
}

func configSectionForType(vtype string) string {
	switch vtype {
	case "SSL Violation":
		return "sub-ssl-tls"
	case "Password Leak":
		return "section-pg-log"
	case "Unauthorized Superuser":
		return "sub-roles"
	case "PII Exposure":
		return "block-data"
	case "HBA Violation":
		return "sub-pg-hba"
	default:
		return "block-config"
	}
}
