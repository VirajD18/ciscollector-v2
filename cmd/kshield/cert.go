package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/VirajD18/ciscollector-v2/pkg/cert"
)

func run(args []string) error {
	if len(args) == 0 || args[0] != "cert" {
		printUsage()
		return fmt.Errorf("unknown command")
	}
	if len(args) < 2 {
		return fmt.Errorf("cert subcommand required (init-ca, issue-server, trust-ca, verify, instructions)")
	}
	sub := args[1]
	rest := args[2:]

	fs := flag.NewFlagSet("kshield cert "+sub, flag.ExitOnError)
	certDir := fs.String("cert-dir", cert.DefaultCertDir, "certificate directory")
	force := fs.Bool("force", false, "overwrite existing files")
	san := fs.String("san", "", "comma-separated SANs (DNS names and IPs)")
	days := fs.Int("days", 0, "validity in days (CA default 3650, server default 825)")
	_ = fs.Parse(rest)

	switch sub {
	case "init-ca":
		if err := cert.InitCA(cert.InitCAOptions{CertDir: *certDir, Force: *force, ValidDays: *days}); err != nil {
			return err
		}
		fmt.Printf("CA created:\n  %s\n  %s\n", cert.CAPath(*certDir), cert.CAKeyPath(*certDir))
		return nil

	case "issue-server":
		sans := cert.SplitSANArg(*san)
		if err := cert.IssueServer(cert.IssueServerOptions{
			CertDir:   *certDir,
			SANs:      sans,
			ValidDays: *days,
			Force:     *force,
		}); err != nil {
			return err
		}
		if len(sans) == 0 {
			sans = cert.DefaultSANs()
		}
		fmt.Printf("Server certificate issued:\n  %s\n  %s\n  SANs: %s\n",
			cert.ServerCertPath(*certDir), cert.ServerKeyPath(*certDir), strings.Join(sans, ", "))
		return nil

	case "trust-ca":
		if err := cert.TrustCA(*certDir); err != nil {
			return err
		}
		fmt.Println("CA trusted successfully. Restart browsers, then open https://<your-host>:8081")
		return nil

	case "verify":
		if err := cert.VerifyServer(*certDir); err != nil {
			return err
		}
		fmt.Println("OK: server.crt is signed by ca.crt")
		return nil

	case "instructions":
		fmt.Print(cert.TrustInstructions(*certDir))
		return nil

	default:
		return fmt.Errorf("unknown cert subcommand %q", sub)
	}
}
