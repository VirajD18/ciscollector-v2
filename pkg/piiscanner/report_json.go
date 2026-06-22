package piiscanner

import (
	"fmt"
	"strings"

	"github.com/klouddb/klouddbshield/pkg/utils"
)

// ToReportJSON builds the payload stored in runs.pii_report_json for the dashboard API.
func ToReportJSON(out *DatabasePIIScanOutput, cnf Config) map[string]interface{} {
	if out == nil {
		return map[string]interface{}{}
	}
	dataCols := []string{"table", "column", "label", "confidence", "detector", "matched"}
	metaCols := []string{"table", "column", "label", "confidence"}
	var dataRows, metaRows [][]interface{}
	tablesWithPII := utils.NewSet[string]()
	tablesInTop := utils.NewSet[string]()

	for tablename, columns := range out.Data {
		for columnName, piidatas := range columns {
			for _, piidata := range piidatas {
				tablesWithPII.Add(tablename)
				if !cnf.printAllResults && piidata.Confidence != "High" {
					continue
				}
				conf := strings.TrimSpace(piidata.Confidence + " " + piidata.ConfidenceIcon)
				if piidata.DetectorType == DetectorType_ValueDetector {
					dataRows = append(dataRows, []interface{}{
						tablename, columnName, string(piidata.Label), conf,
						piidata.DetectorName,
						fmt.Sprintf("%d/%d", piidata.MatchedCount, piidata.ScanedValueCount),
					})
					tablesInTop.Add(tablename)
				} else {
					metaRows = append(metaRows, []interface{}{
						tablename, columnName, string(piidata.Label), conf,
					})
					tablesInTop.Add(tablename)
				}
			}
		}
	}

	var lowConf []string
	for _, t := range tablesWithPII.Slice() {
		if !tablesInTop.Contains(t) {
			lowConf = append(lowConf, t)
		}
	}

	runOpt := cnf.runOption.String()
	if cnf.useSpacy {
		runOpt = RunOption_SpacyScan_String
	}

	return map[string]interface{}{
		"high_confidence": map[string]interface{}{
			"columns": dataCols,
			"rows":    dataRows,
		},
		"meta": map[string]interface{}{
			"columns": metaCols,
			"rows":    metaRows,
		},
		"low_confidence_tables": lowConf,
		"scan_type":             out.ScanType,
		"run_option":            runOpt,
		"schema":                cnf.Schema,
	}
}
