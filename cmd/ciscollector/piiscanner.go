package main

import (
	"context"
	"fmt"
	"time"

	"github.com/VirajD18/ciscollector-v2/htmlreport"
	"github.com/VirajD18/ciscollector-v2/pkg/config"
	"github.com/VirajD18/ciscollector-v2/pkg/mainserverclient"
	"github.com/VirajD18/ciscollector-v2/pkg/piiscanner"
	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
)

type piiDbScanner struct {
	postgresConfig   *postgresdb.Postgres
	cnf              *piiscanner.Config
	htmlReportHelper *htmlreport.HtmlReportHelper
	shieldConfig     *config.Config
}

func newPiiDbScanner(postgresConfig *postgresdb.Postgres, piiConfig *piiscanner.Config, htmlReportHelper *htmlreport.HtmlReportHelper, shieldConfig *config.Config) *piiDbScanner {
	return &piiDbScanner{
		postgresConfig:   postgresConfig,
		cnf:              piiConfig,
		htmlReportHelper: htmlReportHelper,
		shieldConfig:     shieldConfig,
	}
}

func (p *piiDbScanner) cronProcess(ctx context.Context) error {
	return p.run(ctx)
}

func (p *piiDbScanner) run(ctx context.Context) error {
	pgConfig := *p.postgresConfig
	if p.cnf.Database != "" {
		pgConfig.DBName = p.cnf.Database
	}

	store, _, err := postgresdb.Open(pgConfig)
	if err != nil {
		return fmt.Errorf("error opening postgres connection: %v", err)
	}

	dbHelper := piiscanner.NewPostgresDBHelper(p.cnf.Schema)
	piiScanner := piiscanner.NewDatabasePiiScanner(dbHelper, store, p.cnf)

	err = piiScanner.Scan(ctx)
	if err != nil {
		msg := "PII scan failed."
		piiJSON := piiscanner.ReportPayload(nil, *p.cnf, piiscanner.ReportStatusError, msg, err.Error())
		_ = pushPiiReportToMainServer(p.shieldConfig, &pgConfig, piiJSON)
		return fmt.Errorf("error scanning database for pii data: %v", err)
	}

	result, err := piiScanner.GetResults()
	if err != nil {
		msg := "PII scan failed while collecting results."
		piiJSON := piiscanner.ReportPayload(nil, *p.cnf, piiscanner.ReportStatusError, msg, err.Error())
		_ = pushPiiReportToMainServer(p.shieldConfig, &pgConfig, piiJSON)
		return fmt.Errorf("error getting pii scan results: %v", err)
	}

	if result == nil {
		msg := "No tables found in database matching the PII scan criteria."
		piiJSON := piiscanner.ReportPayload(nil, *p.cnf, piiscanner.ReportStatusNoTables, msg, "")
		_ = pushPiiReportToMainServer(p.shieldConfig, &pgConfig, piiJSON)
		piiscanner.PrintTerminalOutput(result, *p.cnf)
		return nil
	}

	piiscanner.PrintTerminalOutput(result, *p.cnf)

	p.htmlReportHelper.RegisterPIIReport(result)

	piiscanner.CreateTabularOutputfile(result, *p.cnf)

	piiJSON := piiscanner.ReportPayload(result, *p.cnf, piiscanner.ReportStatusSuccess, "", "")
	if err := pushPiiReportToMainServer(p.shieldConfig, &pgConfig, piiJSON); err != nil {
		fmt.Printf("> Warning: PII report push to main-server failed: %v\n", err)
	}

	return nil
}

func pushPiiReportToMainServer(cnf *config.Config, pg *postgresdb.Postgres, piiJSON map[string]interface{}) error {
	if cnf == nil || !cnf.MainServer.Enabled || pg == nil || len(piiJSON) == 0 {
		return nil
	}
	client, err := mainserverclient.New(cnf)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := mainserverclient.PushPiiReport(ctx, cnf, client, pg, piiJSON); err != nil {
		return err
	}
	_ = client.FlushRetries(ctx)
	return nil
}
