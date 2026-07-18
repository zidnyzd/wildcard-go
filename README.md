# Wildcard-Go

Cloudflare Wildcard ACM SSL Certificate manager via SSL for SaaS (Custom Hostnames). Dapatkan SSL wildcard **gratis** untuk 100 hostname pertama tanpa biaya berlangganan.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-Proprietary-red)

---

## Fitur

- **Dashboard** — panduan setup step-by-step
- **Credentials** — simpan Cloudflare API Token, Zone ID, SSH credentials
- **Setup Fallback** — buat DNS records + set fallback origin (1 klik)
- **Create Hostname** — buat custom hostname wildcard + auto upload ACME challenge via SSH
- **Bulk Hostnames** — proses banyak domain sekaligus
- **List Hostnames** — lihat, cek status, hapus custom hostnames
- **DNS Manager** — CRUD DNS records langsung dari web panel
- **Auth** — login/register dengan bcrypt + session cookies
- **Dark Mode** — toggle dark/light theme

---

## Cara Kerja

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────┐
│  Browser UI  │ ──▶ │  Go Backend      │ ──▶ │  Cloudflare  │
│  (Bootstrap) │ ◀── │  (net/http)      │ ◀── │  API         │
└─────────────┘     └──────────────────┘     └─────────────┘
```

1. Cloudflare **SSL for SaaS** menerbitkan sertifikat SSL untuk custom hostname
2. **Fallback origin** adalah server asal yang diminta sertifikatnya saat validasi HTTP
3. DNS wildcard `*` mengarahkan semua subdomain ke VPS Anda
4. Go backend mengelola semua operasi Cloudflare API + SSH

---

## Instalasi

### 1. Setup VPS (Server)

Jalankan di VPS Ubuntu/Debian Anda:

```bash
curl -sS -o vps-setup https://raw.githubusercontent.com/zidnyzd/wildcard-go/main/vps-setup && chmod +x vps-setup && sudo ./vps-setup
```

Script akan:
- Install Nginx
- Buat folder ACME challenge (`/var/www/acme/.well-known/acme-challenge/`)
- Konfigurasi firewall (port 80, 443)
- Tampilkan IP VPS

### 2. Deploy Web Panel

```bash
# Download binary
curl -sL -o wildcard-go https://raw.githubusercontent.com/zidnyzd/wildcard-go/main/wildcard-go
chmod +x wildcard-go

# Download templates & static assets
git clone --depth 1 https://github.com/zidnyzd/wildcard-go.git /tmp/wildcard-assets
cp -r /tmp/wildcard-assets/templates /tmp/wildcard-assets/static .

# Jalankan
./wildcard-go
```

Server berjalan di `http://127.0.0.1:8083`

### 3. Systemd Service (opsional)

```bash
cat > /etc/systemd/system/wildcard-go.service << 'EOF'
[Unit]
Description=Wildcard-Go
After=network.target

[Service]
Type=simple
WorkingDirectory=/root/wildcard-go
ExecStart=/root/wildcard-go/wildcard-go
Restart=always
RestartSec=3
MemoryMax=64M

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now wildcard-go
```

### 4. Cloudflare Tunnel (opsional)

Tambahkan ke config cloudflared:

```yaml
ingress:
  - hostname: wildcard.yourdomain.com
    service: http://127.0.0.1:8083
  - service: http_status:404
```

```bash
cloudflared tunnel route dns <tunnel-id> wildcard.yourdomain.com
systemctl restart cloudflared
```

---

## Prasyarat

| Item | Keterangan |
|---|---|
| **Akun Cloudflare** | Free plan sudah cukup |
| **Domain sendiri** | Terdaftar dan di-manage di Cloudflare |
| **Payment method** | CC/PayPal terverifikasi di Cloudflare (~$1 auth, direfund) |
| **VPS** | Ubuntu 22.04+ / Debian 11+, port 80 terbuka |
| **OS Web Panel** | Linux x86_64 |

---

## API Token Permissions

Buat custom token di **Profile → API Tokens**:

- `Zone:DNS:Edit`
- `Zone:SSL:Edit`
- Zone Resources: **Include → Specific zone → pilih domain Anda**

---

## Stack

- **Backend:** Go (net/http + html/template)
- **Database:** SQLite (pure Go, no CGO)
- **Auth:** bcrypt + in-memory session
- **Frontend:** Sneat Bootstrap 5 Admin Template
- **API:** Cloudflare API v4
- **SSH:** golang.org/x/crypto/ssh

---

## Resource Usage

| Metric | Value |
|---|---|
| Memory | ~3 MB |
| Binary size | 13 MB |
| CPU | Near zero at idle |
| Dependencies | None (single binary) |

---

## License

Proprietary — free to use, **jangan diperjualbelikan**.

---

## Credits

- [Cloudflare SSL for SaaS](https://developers.cloudflare.com/ssl/ssl-for-saas/)
- [Sneat Bootstrap Admin Template](https://github.com/themeselection/sneat-html-admin-template-free)
- Original method by [HidePulsa](https://github.com/HidePulsa/Method-Wildcard-New)
