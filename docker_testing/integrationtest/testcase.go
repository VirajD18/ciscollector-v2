package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strings"

	cons "github.com/klouddb/klouddbshield/pkg/const"
	"github.com/spf13/cobra"
)

// init for testing command
func init() {
	var filename string
	testCmd := cobra.Command{
		Use:   "test",
		Short: "this will test the postgres setup",
		Run: func(cmd *cobra.Command, args []string) {
			testInactiveUser(prefix, filename)
			// testMissingIPs(prefix, filename)
			testUniqueIPs(prefix, filename)
			testUnusedHbaLines(prefix, filename)
			testLeakedPasswordScanner(prefix, filename)
		},
	}

	testCmd.Flags().StringVarP(&filename, "file", "f", "", "pass file for testing")
	testCmd.PersistentFlags().StringVarP(&prefix, "prefix", "p", "", "prefix for setup")
	err := testCmd.MarkPersistentFlagRequired("prefix")
	if err != nil {
		fmt.Println("Got error while marking flag required in test command:", err)
		return
	}
	err = testCmd.MarkFlagRequired("file")
	if err != nil {
		fmt.Println("error while setting required flag:", err)
		os.Exit(1)
	}

	rootCmd.AddCommand(&testCmd)
}

func testUniqueIPs(prefix, file string) {
	cmd := exec.Command("ciscollector",
		"-logparser", cons.LogParserCMD_UniqueIPs,
		"-prefix", prefix,
		"-config", "./",
		"-file-path", file,
		"-output-type", "json",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("execution error from unique_ips:", err)
		os.Exit(1)
	}

	if !logParserOutputSucceeded(string(out)) {
		fmt.Println("Successful log is not available in output (unique_ips):", string(out))
		os.Exit(1)
	}

	payload, err := extractLogParserPayload(string(out))
	if err != nil {
		fmt.Println("failed to parse unique_ips JSON output:", err, string(out))
		os.Exit(1)
	}
	entry, err := summaryEntryForCommand(payload, cons.LogParserCMD_UniqueIPs)
	if err != nil {
		fmt.Println("unique_ips summary missing:", err, string(out))
		os.Exit(1)
	}
	ips, err := stringSliceFromValue(entry["Value"])
	if err != nil {
		fmt.Println("unique_ips value invalid:", err, string(out))
		os.Exit(1)
	}
	mustHave := []string{
		"192.168.0.25",
		"192.168.0.26",
		"192.168.0.27",
		"192.168.0.28",
		"192.168.0.29",
		"192.168.0.30",
	}
	for _, ip := range mustHave {
		if !slices.Contains(ips, ip) {
			fmt.Println("missing expected ip in unique_ips output:", ip, "got", ips, string(out))
			os.Exit(1)
		}
	}
	if len(ips) < len(mustHave) {
		fmt.Println("not enough unique ips in output:", ips, string(out))
		os.Exit(1)
	}

	fmt.Println("unique_ip test is working fine for prefix:", prefix)
}

func testInactiveUser(prefix, file string) {
	cmd := exec.Command("ciscollector",
		"-logparser", cons.LogParserCMD_InactiveUser,
		"-prefix", prefix,
		"-file-path", file,
		"-config", "./",
		"-output-type", "json",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("execution error from inactive_users:", err, string(out))
		os.Exit(1)
	}

	if !logParserOutputSucceeded(string(out)) {
		fmt.Println("Successful log is not available in output (inactive_users):", string(out))
		os.Exit(1)
	}

	payload, err := extractLogParserPayload(string(out))
	if err != nil {
		fmt.Println("failed to parse inactive_users JSON output:", err, string(out))
		os.Exit(1)
	}
	entry, err := summaryEntryForCommand(payload, cons.LogParserCMD_InactiveUser)
	if err != nil {
		fmt.Println("inactive_users summary missing:", err, string(out))
		os.Exit(1)
	}
	got, err := stringMatrixFromValue(entry["Value"])
	if err != nil {
		fmt.Println("inactive_users value invalid:", err, string(out))
		os.Exit(1)
	}
	want := [][]string{
		{"myuser", "user0", "user1", "user2", "user3", "user4", "user5"},
		{"myuser", "user0", "user1", "user2", "user3", "user4"},
		{"user5"},
	}
	if !reflect.DeepEqual(got, want) {
		fmt.Println("not getting valid users in output (inactive_users):", got, "want", want, string(out))
		os.Exit(1)
	}

	fmt.Println("Inactive user test is working fine for prefix:", prefix)
}

func testUnusedHbaLines(prefix, file string) {
	cmd := exec.Command("ciscollector",
		"-logparser", cons.LogParserCMD_HBAUnusedLines,
		"-prefix", prefix,
		"-config", "./",
		"-file-path", file,
		"-output-type", "json",
		"-hba-file", "./pg_hba.conf",
	)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		fmt.Println("Got error while parsing file:", err)
		os.Exit(1)
	}

	out := buf.String()
	if strings.Contains(out, "In logline prefix, please set '%u' and '%d'") || strings.Contains(out, "please set log_line_prefix") {
		fmt.Println("skipping test for unused files as required details are not available in prefix:", prefix)
		return
	}

	if !logParserOutputSucceeded(out) {
		fmt.Println("Got error while parsing file:", out)
		os.Exit(1)
	}

	payload, err := extractLogParserPayload(out)
	if err != nil {
		fmt.Println("failed to parse unused_lines JSON output:", err, out)
		os.Exit(1)
	}
	entry, err := summaryEntryForCommand(payload, cons.LogParserCMD_HBAUnusedLines)
	if err != nil {
		fmt.Println("unused_lines summary missing:", err, out)
		os.Exit(1)
	}
	lines, err := hbaLineNumbersFromValue(entry["Value"])
	if err != nil {
		fmt.Println("unused_lines value invalid:", err, out)
		os.Exit(1)
	}
	if sortedIntsEqual(lines, []int{11, 23, 28}) || sortedIntsEqual(lines, []int{11, 16, 17, 23, 28}) {
		fmt.Println("unused lines test is working fine for prefix:", prefix)
		return
	}

	fmt.Println("not getting valid unused lines:", lines, out)
	os.Exit(1)
}

func testLeakedPasswordScanner(prefix, file string) {
	cmd := exec.Command("ciscollector",
		"-logparser", cons.LogParserCMD_PasswordLeakScanner,
		"-prefix", prefix,
		"-config", "./",
		"-file-path", file,
		"-output-type", "json",
	)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		fmt.Println("Got error while parsing file:", err)
		os.Exit(1)
	}

	out := buf.String()
	if !logParserOutputSucceeded(out) {
		fmt.Println("Got error while parsing file:", out)
		os.Exit(1)
	}

	payload, err := extractLogParserPayload(out)
	if err != nil {
		fmt.Println("failed to parse password_leak_scanner JSON output:", err, out)
		os.Exit(1)
	}
	entry, err := summaryEntryForCommand(payload, cons.LogParserCMD_PasswordLeakScanner)
	if err != nil {
		fmt.Println("password_leak_scanner summary missing:", err, out)
		os.Exit(1)
	}
	leaks, ok := entry["Value"].([]interface{})
	if !ok || len(leaks) != 6 {
		fmt.Println("not getting valid password scanner output:", entry["Value"], out)
		os.Exit(1)
	}
	for i := 0; i < 6; i++ {
		row, ok := leaks[i].(map[string]interface{})
		if !ok {
			fmt.Println("password leak row invalid:", leaks[i], out)
			os.Exit(1)
		}
		query, _ := row["Query"].(string)
		password, _ := row["Password"].(string)
		wantUser := fmt.Sprintf("user%d", i)
		if !strings.Contains(query, wantUser) || password != "password" {
			fmt.Println("password leak row mismatch:", row, "want user", wantUser, out)
			os.Exit(1)
		}
	}

	fmt.Println("password leak scanner test is working fine for prefix:", prefix)
}
