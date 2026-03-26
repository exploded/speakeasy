#!/bin/bash
# Run as root on the Linode server:
#   curl -fsSL https://raw.githubusercontent.com/exploded/speakeasy/main/scripts/server-setup.sh | sudo bash
set -e

echo "=== SpeakEasy Server Setup ==="

# 1. Create deploy user if it doesn't exist
if ! id "deploy" &>/dev/null; then
    useradd -m -s /bin/bash deploy
    echo "Created deploy user"
else
    echo "deploy user already exists"
fi

# 2. Generate SSH key pair for GitHub Actions (reuse if exists)
KEY_FILE="/home/deploy/.ssh/github_actions"
mkdir -p /home/deploy/.ssh
chmod 700 /home/deploy/.ssh

if [ ! -f "$KEY_FILE" ]; then
    ssh-keygen -t ed25519 -C "speakeasy-github-actions" -f "$KEY_FILE" -N ""
    echo "Generated new SSH key pair"
else
    echo "SSH key already exists, reusing it"
fi

# Authorise the key for the deploy user
cat "$KEY_FILE.pub" >> /home/deploy/.ssh/authorized_keys
sort -u /home/deploy/.ssh/authorized_keys -o /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys
chown -R deploy:deploy /home/deploy/.ssh
echo "Public key added to authorized_keys"

# 3. Create application directory
mkdir -p /var/www/speakeasy
chown -R www-data:www-data /var/www/speakeasy
echo "Created /var/www/speakeasy"

# 4. Create .env template
ENV_FILE="/var/www/speakeasy/.env"
if [ ! -f "$ENV_FILE" ]; then
    cat > "$ENV_FILE" <<'ENVEOF'
PORT=8282
PROD=True
SPEAKEASY_DATA_DIR=/var/www/speakeasy
GOOGLE_TTS_API_KEY=your-google-tts-api-key-here
MONITOR_URL=
MONITOR_API_KEY=
ENVEOF
    chown www-data:www-data "$ENV_FILE"
    chmod 600 "$ENV_FILE"
    echo "Created .env template — edit with real values"
else
    echo ".env already exists, skipping"
fi

# 5. Install systemd service with security hardening
cat > /etc/systemd/system/speakeasy.service <<'SVCEOF'
[Unit]
Description=SpeakEasy Language Tutor
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/var/www/speakeasy
EnvironmentFile=/var/www/speakeasy/.env
ExecStart=/var/www/speakeasy/speakeasy
Restart=on-failure
RestartSec=5

# Security hardening (required)
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true

[Install]
WantedBy=multi-user.target
SVCEOF
systemctl daemon-reload
echo "Installed speakeasy.service with security hardening"

# 6. Install deploy script
DEPLOY_SCRIPT="/usr/local/bin/deploy-speakeasy"
cat > "$DEPLOY_SCRIPT" <<'DSEOF'
#!/bin/bash
set -e

DEPLOY_SRC="${1:-/tmp/speakeasy-deploy}"
DEPLOY_DIR=/var/www/speakeasy

# Self-update: if the bundle contains a newer version, install and re-exec
BUNDLE_SCRIPT="$DEPLOY_SRC/scripts/deploy-speakeasy"
if [ -f "$BUNDLE_SCRIPT" ] && ! diff -q /usr/local/bin/deploy-speakeasy "$BUNDLE_SCRIPT" > /dev/null 2>&1; then
    echo "[deploy] Updating deploy script from bundle..."
    install -m 755 "$BUNDLE_SCRIPT" /usr/local/bin/deploy-speakeasy
    exec /usr/local/bin/deploy-speakeasy "$@"
fi

SERVICE_USER=$(systemctl show speakeasy --property=User --value)
SERVICE_GROUP=$(systemctl show speakeasy --property=Group --value)

if [ -z "$SERVICE_USER" ]; then
    echo "[deploy] ERROR: Could not read User from speakeasy.service"
    exit 1
fi

echo "[deploy] Stopping service..."
systemctl stop speakeasy || true

echo "[deploy] Installing binary..."
rm -f "$DEPLOY_DIR/speakeasy"
cp "$DEPLOY_SRC/speakeasy" "$DEPLOY_DIR/speakeasy"
chmod +x "$DEPLOY_DIR/speakeasy"

echo "[deploy] Updating web assets..."
rm -rf "$DEPLOY_DIR/web"
cp -r "$DEPLOY_SRC/web" "$DEPLOY_DIR/web"
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$DEPLOY_DIR"

echo "[deploy] Starting service..."
systemctl start speakeasy

echo "[deploy] Verifying service is active..."
sleep 2
if ! systemctl is-active --quiet speakeasy; then
    echo "[deploy] ERROR: Service failed to start. Status:"
    systemctl status speakeasy --no-pager --lines=30
    exit 1
fi

echo "[deploy] Cleaning up..."
rm -rf "$DEPLOY_SRC"

echo "[deploy] Done — speakeasy is running."
DSEOF
chmod 755 "$DEPLOY_SCRIPT"
echo "Installed deploy script at $DEPLOY_SCRIPT"

# 7. Configure sudoers
SUDOERS_FILE="/etc/sudoers.d/speakeasy-deploy"
cat > "$SUDOERS_FILE" <<'EOF'
deploy ALL=(ALL) NOPASSWD: /usr/local/bin/deploy-speakeasy
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop speakeasy
EOF
chmod 440 "$SUDOERS_FILE"
visudo -cf "$SUDOERS_FILE" && echo "Sudoers configured OK" || (rm -f "$SUDOERS_FILE" && echo "ERROR: sudoers syntax check failed" && exit 1)

# 8. Print GitHub Secrets
echo ""
echo "============================================================"
echo "  PASTE THESE INTO GITHUB SECRETS"
echo "  https://github.com/exploded/speakeasy/settings/secrets/actions"
echo "============================================================"
echo ""
echo "--- Secret name: DEPLOY_SSH_KEY ---"
cat "$KEY_FILE"
echo ""
echo "--- Secret name: DEPLOY_USER ---"
echo "deploy"
echo ""
echo "--- Secret name: DEPLOY_HOST ---"
echo "$(hostname -f 2>/dev/null || hostname)"
echo ""
echo "--- Secret name: DEPLOY_PORT ---"
echo "22"
echo ""
echo "============================================================"
echo "Done! Copy each value above into the matching GitHub Secret."
echo "============================================================"
