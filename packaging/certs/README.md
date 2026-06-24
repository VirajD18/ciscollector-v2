# KloudDB Shield — TLS certificates

## Phase 2 — KloudDB Shield internal CA (recommended on-prem)

```bash
# 1. Create internal CA (once per deployment)
sudo kshield cert init-ca --cert-dir /etc/klouddbshield/certs

# 2. Issue server certificate (localhost, LAN IP, hostname)
sudo kshield cert issue-server \
  --cert-dir /etc/klouddbshield/certs \
  --san localhost,127.0.0.1,192.168.1.50,your-hostname

# 3. Verify chain
kshield cert verify --cert-dir /etc/klouddbshield/certs

# 4. Restart main-server (HTTPS auto-enables when server.crt / server.key exist)
sudo systemctl restart klouddbshield-server
```

| File | Purpose |
|------|---------|
| `ca.crt` | KloudDB Shield CA — distribute to collectors and workstations |
| `ca.key` | CA private key — **keep secret** on main-server only (`chmod 600`) |
| `server.crt` | TLS certificate for main-server |
| `server.key` | TLS private key (`chmod 600`) |

## Phase 3 — Trust CA once (green lock without public domain)

**Linux (root):**

```bash
sudo kshield cert trust-ca --cert-dir /etc/klouddbshield/certs
# or manually:
sudo cp /etc/klouddbshield/certs/ca.crt /usr/local/share/ca-certificates/kshield-ca.crt
sudo update-ca-certificates
```

**Windows (Administrator PowerShell):**

```powershell
kshield cert trust-ca --cert-dir C:\etc\klouddbshield\certs
# or manually:
certutil -addstore -f ROOT C:\etc\klouddbshield\certs\ca.crt
```

After trusting `ca.crt`, `https://192.168.1.50:8081` shows as secure in the browser.

**Collector** (`kshieldconfig.toml`):

```toml
[mainserver]
url = "https://192.168.1.50:8081"
tls_ca_file = "/etc/klouddbshield/certs/ca.crt"
```

If the CA is in the OS trust store, `tls_ca_file` can be omitted on that host.

## Phase 1 — External CA (Let's Encrypt / corporate PKI)

Place admin-provided files:

| File | Purpose |
|------|---------|
| `server.crt` | TLS certificate (PEM). Use full chain for Let's Encrypt. |
| `server.key` | TLS private key (PEM). `chmod 600`. |

Let's Encrypt example:

```bash
sudo cp /etc/letsencrypt/live/your.domain/fullchain.pem /etc/klouddbshield/certs/server.crt
sudo cp /etc/letsencrypt/live/your.domain/privkey.pem   /etc/klouddbshield/certs/server.key
sudo chmod 600 /etc/klouddbshield/certs/server.key
sudo systemctl restart klouddbshield-server
```

For corporate PKI, distribute your CA bundle as `ca.crt` and set `mainserver.tls_ca_file` on collectors.
