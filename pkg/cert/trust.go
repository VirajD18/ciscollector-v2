package cert

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

const (
	linuxTrustPath = "/usr/local/share/ca-certificates/kshield-ca.crt"
)

// TrustCA installs ca.crt into the OS trust store (Phase 3).
func TrustCA(certDir string) error {
	dir := certDir
	if dir == "" {
		dir = DefaultCertDir
	}
	caPath := CAPath(dir)
	if !fileExists(caPath) {
		return fmt.Errorf("ca.crt not found at %s — run 'kshield cert init-ca' first", caPath)
	}

	switch runtime.GOOS {
	case "linux":
		return trustCALinux(caPath)
	case "windows":
		return trustCAWindows(caPath)
	case "darwin":
		return trustCADarwin(caPath)
	default:
		return fmt.Errorf("trust-ca is not supported on %s — see docs for manual steps", runtime.GOOS)
	}
}

func trustCALinux(caPath string) error {
	data, err := os.ReadFile(caPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(linuxTrustPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s (requires root): %w", linuxTrustPath, err)
	}
	cmd := exec.Command("update-ca-certificates")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("update-ca-certificates: %w (%s)", err, string(out))
	}
	return nil
}

func trustCAWindows(caPath string) error {
	cmd := exec.Command("certutil", "-addstore", "-f", "ROOT", caPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("certutil (run as Administrator): %w (%s)", err, string(out))
	}
	return nil
}

func trustCADarwin(caPath string) error {
	keychain := "/Library/Keychains/System.keychain"
	cmd := exec.Command("security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", keychain, caPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("security add-trusted-cert (requires root): %w (%s)", err, string(out))
	}
	return nil
}

// TrustInstructions returns manual trust steps for the current OS.
func TrustInstructions(certDir string) string {
	dir := certDir
	if dir == "" {
		dir = DefaultCertDir
	}
	caPath := CAPath(dir)
	switch runtime.GOOS {
	case "linux":
		return fmt.Sprintf(`Linux (run as root):
  sudo cp %s %s
  sudo update-ca-certificates
`, caPath, linuxTrustPath)
	case "windows":
		return fmt.Sprintf(`Windows (run PowerShell as Administrator):
  certutil -addstore -f ROOT %s
`, caPath)
	case "darwin":
		return fmt.Sprintf(`macOS (run as root):
  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s
`, caPath)
	default:
		return fmt.Sprintf("Import %s into your OS trusted root CA store.", caPath)
	}
}
