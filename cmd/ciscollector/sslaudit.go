package main

import (
	"context"

	"github.com/klouddb/klouddbshield/htmlreport"
	"github.com/klouddb/klouddbshield/pkg/postgresdb"
	"github.com/klouddb/klouddbshield/postgres"
	"github.com/klouddb/klouddbshield/postgres/sslaudit"
)

type sslAuditor struct {
	postgresConfig   *postgresdb.Postgres
	fileData         map[string]interface{}
	htmlReportHelper *htmlreport.HtmlReportHelper
	outputType       string
}

func newSslAuditor(postgresConfig *postgresdb.Postgres, fileData map[string]interface{},
	htmlReportHelper *htmlreport.HtmlReportHelper, outputType string) *sslAuditor {
	return &sslAuditor{
		postgresConfig:   postgresConfig,
		fileData:         fileData,
		htmlReportHelper: htmlReportHelper,
		outputType:       outputType,
	}
}

func (h *sslAuditor) cronProcess(ctx context.Context) error {
	return h.run(ctx)
}

func (h *sslAuditor) run(ctx context.Context) error {
	postgresStore, _, err := postgresdb.Open(*h.postgresConfig)
	if err != nil {
		return err
	}
	defer postgresStore.Close()

	result, err := sslaudit.AuditSSL(ctx, postgresStore, h.postgresConfig.Host, h.postgresConfig.Port)
	if err != nil {
		return err
	}

	h.htmlReportHelper.RegisterSSLReport(result)

	if h.outputType == "json" && h.fileData != nil {
		h.fileData["SSL Report"] = result
	}

	postgres.PrintSSLAuditSummary(result)

	return nil
}
