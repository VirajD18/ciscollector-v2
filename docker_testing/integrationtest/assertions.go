package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

func logParserOutputSucceeded(out string) bool {
	clean := stripANSI(out)
	return strings.Contains(clean, "Successfully parsed all files") ||
		strings.Contains(clean, "All lines parsed successfully")
}

func extractLogParserPayload(out string) (map[string]interface{}, error) {
	clean := stripANSI(out)
	idx := strings.Index(clean, "{")
	if idx < 0 {
		return nil, fmt.Errorf("no JSON object in log parser output")
	}
	dec := json.NewDecoder(strings.NewReader(clean[idx:]))
	var payload map[string]interface{}
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func summaryEntryForCommand(payload map[string]interface{}, command string) (map[string]interface{}, error) {
	raw, ok := payload["Log Parser Summary"]
	if !ok {
		return nil, fmt.Errorf("missing Log Parser Summary")
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid Log Parser Summary shape")
	}
	for _, item := range items {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if cmd, _ := entry["Command"].(string); cmd == command {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("command %q not found in summary", command)
}

func stringMatrixFromValue(v interface{}) ([][]string, error) {
	rows, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("value is not an array")
	}
	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		cells, ok := row.([]interface{})
		if !ok {
			return nil, fmt.Errorf("row is not an array")
		}
		line := make([]string, 0, len(cells))
		for _, c := range cells {
			s, ok := c.(string)
			if !ok {
				return nil, fmt.Errorf("cell is not a string")
			}
			line = append(line, s)
		}
		out = append(out, line)
	}
	return out, nil
}

func stringSliceFromValue(v interface{}) ([]string, error) {
	items, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("value is not an array")
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("item is not a string")
		}
		out = append(out, s)
	}
	return out, nil
}

func hbaLineNumbersFromValue(v interface{}) ([]int, error) {
	items, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("value is not an array")
	}
	out := make([]int, 0, len(items))
	for _, item := range items {
		switch row := item.(type) {
		case float64:
			out = append(out, int(row))
		case int:
			out = append(out, row)
		case map[string]interface{}:
			switch n := row["LineNo"].(type) {
			case float64:
				out = append(out, int(n))
			case int:
				out = append(out, n)
			default:
				return nil, fmt.Errorf("LineNo missing or invalid")
			}
		default:
			return nil, fmt.Errorf("item is not a line number or hba row")
		}
	}
	return out, nil
}

func sortedIntsEqual(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	g := append([]int(nil), got...)
	w := append([]int(nil), want...)
	sort.Ints(g)
	sort.Ints(w)
	return reflect.DeepEqual(g, w)
}
