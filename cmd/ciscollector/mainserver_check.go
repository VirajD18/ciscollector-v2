package main

import (
	"context"
	"fmt"

	"github.com/jedib0t/go-pretty/text"
	"github.com/klouddb/klouddbshield/pkg/config"
	"github.com/klouddb/klouddbshield/pkg/mainserverclient"
)

func runMainServerConnectionCheck(cnf *config.Config) int {
	fmt.Println("Main-server connection check")
	if !cnf.MainServer.Enabled {
		fmt.Println(text.FgHiRed.Sprint("✘"), "[mainserver] enabled = false in kshieldconfig.toml")
		return 1
	}

	client, err := mainserverclient.New(cnf)
	if err != nil {
		fmt.Println(text.FgHiRed.Sprint("✘"), "config:", err)
		return 1
	}

	fmt.Printf("  URL:      %s\n", client.BaseURL())
	fmt.Printf("  Node ID:  %s\n", client.NodeID())
	fmt.Printf("  Hostname: %s\n", client.Hostname())
	fmt.Println()

	report := client.Probe(context.Background())
	for _, step := range report.Steps {
		mark := text.FgGreen.Sprint("✔")
		if !step.OK {
			mark = text.FgHiRed.Sprint("✘")
		}
		fmt.Printf("  %s %-18s %s %s\n", mark, step.Name, step.Path, step.Detail)
		if !step.OK && step.Hint != "" {
			fmt.Printf("      hint: %s\n", step.Hint)
		}
	}
	fmt.Println()
	if report.OK() {
		fmt.Println(text.FgGreen.Sprint("Connection OK — collector can reach main-server"))
		return 0
	}
	fmt.Println(text.FgHiRed.Sprint("Connection FAILED — fix the steps above before running --setup-cron"))
	return 1
}
