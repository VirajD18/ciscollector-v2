package main

import (
	"context"

	"github.com/VirajD18/ciscollector-v2/htmlreport"
	"github.com/VirajD18/ciscollector-v2/mysql"
	"github.com/VirajD18/ciscollector-v2/pkg/config"
	"github.com/VirajD18/ciscollector-v2/pkg/mysqldb"
	"github.com/VirajD18/ciscollector-v2/simpletextreport"
)

type mysqlRunner struct {
	mysqlDatabase    *config.MySQL
	fileData         map[string]interface{}
	htmlReportHelper *htmlreport.HtmlReportHelper
	outputType       string
}

func newMySqlRunner(mysqlDatabase *config.MySQL, fileData map[string]interface{},
	htmlReportHelper *htmlreport.HtmlReportHelper, outputType string) *mysqlRunner {
	return &mysqlRunner{
		mysqlDatabase:    mysqlDatabase,
		fileData:         fileData,
		htmlReportHelper: htmlReportHelper,
		outputType:       outputType,
	}
}

func (m *mysqlRunner) cronProcess(ctx context.Context) error {
	return m.run(ctx)
}

func (m *mysqlRunner) run(ctx context.Context) error {
	mysqlStore, _, err := mysqldb.Open(*m.mysqlDatabase)
	if err != nil {
		return err
	}
	defer mysqlStore.Close()

	result, score := mysql.PerformAllChecks(mysqlStore, ctx)
	if m.outputType == "json" {
		m.fileData["MySQL Report"] = map[string]interface{}{
			"mysql": result, "score": score,
		}
	} else {
		m.fileData["MySQL Report"] = simpletextreport.PrintReportInFile(result, "")
	}

	m.htmlReportHelper.RegisterMysqlReportData(result, score)

	return nil
}
