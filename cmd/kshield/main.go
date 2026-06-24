package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`KloudDB Shield CLI

Usage:
  kshield cert init-ca       Create internal CA (ca.crt, ca.key)
  kshield cert issue-server  Issue server TLS cert signed by CA
  kshield cert trust-ca      Install ca.crt into OS trust store (Phase 3)
  kshield cert verify        Verify server.crt against ca.crt
  kshield cert instructions  Print manual trust steps for this OS

Global flags (after subcommand):
  --cert-dir <path>   Certificate directory (default: /etc/klouddbshield/certs)
  --force             Overwrite existing files
  --san <list>        Comma-separated SANs for issue-server
  --days <n>          Validity in days`)
}
