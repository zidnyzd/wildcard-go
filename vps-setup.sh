#!/bin/bash
# ============================================================
# Wildcard Project — VPS Setup Script
# Cloudflare ACM SSL via SSL for SaaS (Custom Hostnames)
# ============================================================
# Support: Ubuntu 22.04+ / 24.04+ / 26.04+
#          Debian 11+ / 12+ / 13+
# ============================================================

set -e

# ─── Warna ───────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# ─── Deteksi OS ──────────────────────────────────────────
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        VERSION=$VERSION_ID
    elif command -v lsb_release &>/dev/null; then
        OS=$(lsb_release -si | tr '[:upper:]' '[:lower:]')
        VERSION=$(lsb_release -sr)
    else
        echo -e "${RED}❌ Tidak bisa mendeteksi OS.${NC}"
        exit 1
    fi

    case $OS in
        ubuntu|debian|linuxmint|pop|elementary|zorin)
            echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo -e "${GREEN}📋 OS:${NC} $OS $VERSION"
            echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            ;;
        *)
            echo -e "${RED}❌ OS tidak didukung: $OS${NC}"
            echo "   Support: Ubuntu 22.04+, Debian 11+"
            exit 1
            ;;
    esac
}

# ─── Root Check ──────────────────────────────────────────
check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        echo -e "${RED}❌ Jalankan dengan sudo: sudo ./vps-setup.sh${NC}"
        exit 1
    fi
}

# ─── Firewall ────────────────────────────────────────────
configure_firewall() {
    echo -e "\n${YELLOW}🔓 Membuka port 80/tcp dan 443/tcp...${NC}"

    if command -v ufw &>/dev/null; then
        ufw allow 80/tcp 2>/dev/null || true
        ufw allow 443/tcp 2>/dev/null || true
        ufw --force reload 2>/dev/null || true
        echo -e "   ${GREEN}✅ ufw: port 80 dan 443 dibuka${NC}"
    else
        echo -e "   ${YELLOW}⚠️  ufw tidak terinstall, lewati.${NC}"
        echo -e "   ${YELLOW}   Install: apt-get install -y ufw${NC}"
    fi
}

# ─── Install Nginx ───────────────────────────────────────
install_nginx() {
    echo -e "\n${YELLOW}📦 Memeriksa Nginx...${NC}"

    if command -v nginx &>/dev/null; then
        echo -e "   ${GREEN}✅ Nginx sudah terinstall: $(nginx -v 2>&1 | grep -oP '[\d.]+')${NC}"
        return 0
    fi

    echo -e "   ${YELLOW}⚡ Menginstall Nginx...${NC}"
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq nginx curl
    echo -e "   ${GREEN}✅ Nginx berhasil diinstall${NC}"
}

# ─── Setup ACME Challenge ────────────────────────────────
setup_acme() {
    echo -e "\n${YELLOW}📁 Membuat direktori ACME challenge...${NC}"

    local ACME_DIR="/var/www/acme/.well-known/acme-challenge"
    mkdir -p "$ACME_DIR"
    chmod -R 755 "/var/www/acme"

    # File test untuk debugging
    echo "ok" > "/var/www/acme/.well-known/acme-challenge/test.txt"

    echo -e "   ${GREEN}✅ Direktori: $ACME_DIR${NC}"
}

# ─── Nginx Config ────────────────────────────────────────
setup_nginx_config() {
    echo -e "\n${YELLOW}🔧 Konfigurasi Nginx...${NC}"

    local CONFIG_FILE="/etc/nginx/conf.d/acme-challenge.conf"

    # Hapus config lama dari sites-enabled (Debian/Ubuntu default)
    rm -f /etc/nginx/sites-enabled/acme-challenge 2>/dev/null || true

    cat > "$CONFIG_FILE" << 'EOF'
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;

    root /var/www/acme;

    location /.well-known/acme-challenge/ {
        root /var/www/acme;
        try_files $uri =404;
        access_log off;
        log_not_found off;
    }

    location / {
        access_log off;
        return 200 "OK\n";
        add_header Content-Type text/plain;
    }
}
EOF

    echo -e "   ${GREEN}✅ Config: $CONFIG_FILE${NC}"

    # Hapus default config biar gak konflik
    rm -f /etc/nginx/sites-enabled/default 2>/dev/null || true
}

# ─── Test Nginx ──────────────────────────────────────────
test_nginx() {
    echo -e "\n${YELLOW}🧪 Test konfigurasi Nginx...${NC}"

    nginx -t 2>&1 || {
        echo -e "${RED}❌ Test Nginx gagal. Cek error di atas.${NC}"
        exit 1
    }

    echo -e "   ${GREEN}✅ Konfigurasi Nginx OK${NC}"

    if systemctl is-active --quiet nginx 2>/dev/null; then
        systemctl reload nginx 2>/dev/null || systemctl restart nginx 2>/dev/null
        echo -e "   ${GREEN}✅ Nginx di-reload${NC}"
    else
        systemctl start nginx 2>/dev/null || true
        systemctl enable nginx 2>/dev/null || true
        echo -e "   ${GREEN}✅ Nginx di-start & enable on boot${NC}"
    fi
}

# ─── Test HTTP ─────────────────────────────────────────────
test_http() {
    echo -e "\n${YELLOW}🌐 Test HTTP response...${NC}"

    local IP
    IP=$(curl -s ifconfig.me 2>/dev/null || curl -s ip.sb 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}')

    if [ -z "$IP" ]; then
        IP="(gagal deteksi IP)"
    fi

    echo -e "   IP VPS: ${CYAN}$IP${NC}"

    # Test local
    if curl -s -o /dev/null -w "%{http_code}" http://localhost/ 2>/dev/null | grep -q "200"; then
        echo -e "   ${GREEN}✅ http://localhost → 200 OK${NC}"
    else
        echo -e "   ${RED}❌ http://localhost gagal${NC}"
    fi

    # Test ACME path
    if curl -s -o /dev/null -w "%{http_code}" http://localhost/.well-known/acme-challenge/test.txt 2>/dev/null | grep -q "200"; then
        echo -e "   ${GREEN}✅ ACME challenge path OK${NC}"
    else
        echo -e "   ${RED}❌ ACME challenge path gagal${NC}"
    fi

    echo -e ""
    echo -e "${CYAN}════════════════════════════════════════${NC}"
    echo -e "${GREEN}✅ VPS siap digunakan!${NC}"
    echo -e "${CYAN}════════════════════════════════════════${NC}"
    echo -e ""
    echo -e "   📝 Catat IP VPS Anda:"
    echo -e "   ${YELLOW}   $IP${NC}"
    echo -e ""
    echo -e "   🔗 Masukkan IP ini ke Web Panel → Credentials → IP VPS"
    echo -e ""
    echo -e "   🧹 Hapus file test setelah setup selesai:"
    echo -e "      rm /var/www/acme/.well-known/acme-challenge/test.txt"
    echo -e ""
}

# ─── Main ────────────────────────────────────────────────
main() {
    echo -e ""
    echo -e "${CYAN}╔═══════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║     Wildcard Project — VPS Setup Script      ║${NC}"
    echo -e "${CYAN}║           Ubuntu / Debian                    ║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════╝${NC}"
    echo -e ""

    check_root
    detect_os
    install_nginx
    configure_firewall
    setup_acme
    setup_nginx_config
    test_nginx
    test_http
}

main "$@"
