package cert

import "path/filepath"

const (
	DefaultCertDir = "/etc/klouddbshield/certs"
	CAFile         = "ca.crt"
	CAKeyFile      = "ca.key"
	ServerCertFile = "server.crt"
	ServerKeyFile  = "server.key"
)

func CAPath(dir string) string         { return filepath.Join(dir, CAFile) }
func CAKeyPath(dir string) string      { return filepath.Join(dir, CAKeyFile) }
func ServerCertPath(dir string) string { return filepath.Join(dir, ServerCertFile) }
func ServerKeyPath(dir string) string  { return filepath.Join(dir, ServerKeyFile) }
