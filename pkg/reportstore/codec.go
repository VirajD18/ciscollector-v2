package reportstore

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
)

func encodeReport(fileData map[string]interface{}) ([]byte, error) {
	raw, err := json.Marshal(fileData)
	if err != nil {
		return nil, fmt.Errorf("marshal report: %w", err)
	}
	return raw, nil
}

func decodeReport(blob []byte) (map[string]interface{}, error) {
	if len(blob) == 0 {
		return map[string]interface{}{}, nil
	}
	// Plain JSON (report_json column type JSON).
	if blob[0] == '{' || blob[0] == '[' {
		var out map[string]interface{}
		if err := json.Unmarshal(blob, &out); err != nil {
			return nil, fmt.Errorf("unmarshal report: %w", err)
		}
		return out, nil
	}
	// Legacy gzip-compressed BLOB rows.
	zr, err := gzip.NewReader(bytes.NewReader(blob))
	if err != nil {
		return nil, fmt.Errorf("gzip open: %w", err)
	}
	defer zr.Close()
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("gzip read: %w", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal report: %w", err)
	}
	return out, nil
}
