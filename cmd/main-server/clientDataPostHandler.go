package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// collectorBodyTokenAllowed returns true when the body token is empty (Bearer auth already
// validated) or matches the configured server token.
func collectorBodyTokenAllowed(bodyToken, expected string) bool {
	bodyToken = strings.TrimSpace(bodyToken)
	expected = strings.TrimSpace(expected)
	if bodyToken == "" {
		return true
	}
	return bodyToken == expected
}

func sqliteDSN(path string) string {
	return "file:" + path + "?mode=rwc&_busy_timeout=5000&_journal_mode=WAL&_txlock=immediate&_foreign_keys=on"
}

func (a *App) clientDataPostHandler(w http.ResponseWriter, r *http.Request) {
	body, err := readRequestBody(r)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	var scanReq ScanDataRequest
	if err := json.Unmarshal(body, &scanReq); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if scanReq.SchemaVersion != "v1" {
		http.Error(w, "unsupported schema", http.StatusBadRequest)
		return
	}
	if !collectorBodyTokenAllowed(scanReq.Node.AgentConfig.Server.Token, a.ServerConfig.Token) {
		http.Error(w, "Token not matched", http.StatusBadRequest)
		return
	}
	if err := a.storeScanResult(r.Context(), &scanReq); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func readRequestBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(io.LimitReader(r.Body, 32<<20))
}
