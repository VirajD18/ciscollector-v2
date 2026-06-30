package main

import (
	"context"

	"github.com/VirajD18/ciscollector-v2/htmlreport"
	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
	"github.com/VirajD18/ciscollector-v2/postgres"
	"github.com/VirajD18/ciscollector-v2/postgres/configaudit"
)

type configAuditor struct {
	postgresConfig   *postgresdb.Postgres
	htmlReportHelper *htmlreport.HtmlReportHelper
}

func newConfigAuditor(postgresConfig *postgresdb.Postgres, htmlReportHelper *htmlreport.HtmlReportHelper) *configAuditor {
	return &configAuditor{
		postgresConfig:   postgresConfig,
		htmlReportHelper: htmlReportHelper,
	}
}

func (h *configAuditor) cronProcess(ctx context.Context) error {
	return h.run(ctx)
}

func (h *configAuditor) run(ctx context.Context) error {
	postgresStore, _, err := postgresdb.Open(*h.postgresConfig)
	if err != nil {
		return err
	}
	defer postgresStore.Close()

	result, err := configaudit.AuditConfig(ctx, postgresStore)
	if err != nil {
		return err
	}

	h.htmlReportHelper.RegisterConfigAudit(result)

	postgres.PrintConfigAuditSummary(result)

	return nil
}
