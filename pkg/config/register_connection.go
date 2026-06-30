package config

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
)

const postgresTestTimeout = 10 * time.Second

// TestPostgresConnection verifies credentials against Postgres with a bounded wait.
func TestPostgresConnection(ctx context.Context, host, port, user, password, dbname string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, postgresTestTimeout)
	defer cancel()

	if err := tcpReachable(ctx, host, port); err != nil {
		return err
	}

	pg := &postgresdb.Postgres{
		Host:      host,
		Port:      port,
		User:      user,
		Password:  password,
		DBName:    dbname,
		SSLmode:   "disable",
		PingCheck: false,
	}
	url := postgresdb.BuildConnectionString(*pg)
	if !strings.Contains(url, "connect_timeout=") {
		url += " connect_timeout=8"
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return context.DeadlineExceeded
		}
		return err
	}
	return nil
}

func tcpReachable(ctx context.Context, host, port string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "localhost"
	}
	port = strings.TrimSpace(port)
	if port == "" {
		port = "5432"
	}
	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

// PostgresTargetAddr formats host:port for error messages.
func PostgresTargetAddr(host string, port int) string {
	p := fmt.Sprintf("%d", port)
	if port <= 0 {
		p = "5432"
	}
	return net.JoinHostPort(strings.TrimSpace(host), p)
}
