package cert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCAAndIssueServer(t *testing.T) {
	tests := []struct {
		name      string
		sans      []string
		force     bool
		wantErr   bool
		errSubstr string
	}{
		{
			name: "default_sans",
			sans: nil,
		},
		{
			name: "custom_sans",
			sans: []string{"localhost", "127.0.0.1", "192.168.1.50", "shield-host"},
		},
		{
			name:      "missing_ca",
			sans:      []string{"localhost"},
			wantErr:   true,
			errSubstr: "CA not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.name != "missing_ca" {
				if err := InitCA(InitCAOptions{CertDir: dir}); err != nil {
					t.Fatalf("InitCA: %v", err)
				}
				for _, f := range []string{CAFile, CAKeyFile} {
					if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
						t.Fatalf("missing %s: %v", f, err)
					}
				}
			}

			err := IssueServer(IssueServerOptions{CertDir: dir, SANs: tc.sans, Force: tc.force})
			if (err != nil) != tc.wantErr {
				t.Fatalf("IssueServer err=%v wantErr=%v", err, tc.wantErr)
			}
			if tc.wantErr {
				if tc.errSubstr != "" && err != nil && !strings.Contains(err.Error(), tc.errSubstr) {
					t.Fatalf("err=%v want substring %q", err, tc.errSubstr)
				}
				return
			}

			if err := VerifyServer(dir); err != nil {
				t.Fatalf("VerifyServer: %v", err)
			}
			for _, f := range []string{ServerCertFile, ServerKeyFile} {
				if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
					t.Fatalf("missing %s: %v", f, err)
				}
			}
		})
	}
}

func TestInitCARefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := InitCA(InitCAOptions{CertDir: dir}); err != nil {
		t.Fatal(err)
	}
	err := InitCA(InitCAOptions{CertDir: dir})
	if err == nil {
		t.Fatal("expected error when CA exists")
	}
	if err := InitCA(InitCAOptions{CertDir: dir, Force: true}); err != nil {
		t.Fatalf("force overwrite: %v", err)
	}
}
