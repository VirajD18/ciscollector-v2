package piiscanner

import "testing"

func TestToReportJSONShape(t *testing.T) {
	out := &DatabasePIIScanOutput{
		ScanType: "Data Scan",
		Data: map[string]TableDetailOutput{
			"users": {
				"email": {
					{
						Label:            PIILabel_Email,
						Confidence:       "High",
						DetectorType:     DetectorType_ValueDetector,
						DetectorName:     "regex",
						MatchedCount:     4,
						ScanedValueCount: 4,
					},
				},
			},
		},
	}
	cnf := Config{runOption: RunOption_DataScan}
	payload := ToReportJSON(out, cnf)
	hc, ok := payload["high_confidence"].(map[string]interface{})
	if !ok {
		t.Fatal("expected high_confidence map")
	}
	rows, _ := hc["rows"].([][]interface{})
	if len(rows) != 1 {
		t.Fatalf("rows: %v", rows)
	}
}
