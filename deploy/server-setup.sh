#!/bin/bash
# Run as root on the Linode server: sudo bash server-setup.sh
set -e

echo "=== SpeakEasy Server Setup ==="

# 1. Create deploy user if it doesn't exist
if ! id "deploy" &>/dev/null; then
    useradd -m -s /bin/bash deploy
    echo "Created deploy user"
else
    echo "deploy user already exists"
fi

# 2. Generate SSH key pair for GitHub Actions
KEY_FILE="/home/deploy/.ssh/github_actions"
mkdir -p /home/deploy/.ssh
chmod 700 /home/deploy/.ssh

if [ ! -f "$KEY_FILE" ]; then
    ssh-keygen -t ed25519 -C "speakeasy-github-actions" -f "$KEY_FILE" -N ""
    echo "Generated new SSH key pair"
else
    echo "SSH key already exists, reusing it"
fi

# 3. Authorise the key for the deploy user
cat "$KEY_FILE.pub" >> /home/deploy/.ssh/authorized_keys
sort -u /home/deploy/.ssh/authorized_keys -o /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys
chown -R deploy:deploy /home/deploy/.ssh
echo "Public key added to authorized_keys"

# 4. Create app directory
mkdir -p /var/www/speakeasy
chown -R www-data /var/www/speakeasy
echo "Created /var/www/speakeasy"

# 5. Sudoers — allow deploy user to manage the speakeasy service and files
# Note: avoid colons in sudoers rules (causes syntax errors); use chown without group
SUDOERS_FILE="/etc/sudoers.d/speakeasy-deploy"
cat > "$SUDOERS_FILE" <<'EOF'
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop speakeasy
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl start speakeasy
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl is-active speakeasy
deploy ALL=(ALL) NOPASSWD: /usr/bin/journalctl -u speakeasy *
deploy ALL=(ALL) NOPASSWD: /usr/bin/rm -f /var/www/speakeasy/speakeasy
deploy ALL=(ALL) NOPASSWD: /usr/bin/cp /home/deploy/speakeasy /var/www/speakeasy/speakeasy
deploy ALL=(ALL) NOPASSWD: /usr/bin/rm -rf /var/www/speakeasy/web
deploy ALL=(ALL) NOPASSWD: /usr/bin/cp -r /home/deploy/web /var/www/speakeasy/web
deploy ALL=(ALL) NOPASSWD: /usr/bin/chown -R www-data /var/www/speakeasy
deploy ALL=(ALL) NOPASSWD: /usr/bin/chmod 755 /var/www/speakeasy/speakeasy
EOF
chmod 440 "$SUDOERS_FILE"
# Validate the sudoers file
visudo -cf "$SUDOERS_FILE" && echo "Sudoers configured OK" || (rm -f "$SUDOERS_FILE" && echo "ERROR: sudoers syntax check failed" && exit 1)

# 6. Fix existing broken sudoers file if present
if [ -f /etc/sudoers.d/speakeasy-deploy ]; then
    visudo -cf /etc/sudoers.d/speakeasy-deploy 2>/dev/null || rm -f /etc/sudoers.d/speakeasy-deploy
fi

# 7. Print the GitHub Secrets
HOSTNAME=$(hostname -f 2>/dev/null || hostname)
echo ""
echo "============================================================"
echo "  PASTE THESE INTO GITHUB SECRETS"
echo "  https://github.com/exploded/speakeasy/settings/secrets/actions"
echo "============================================================"
echo ""
echo "--- Secret name: SSH_PRIVATE_KEY ---"
cat "$KEY_FILE"
echo ""
echo "--- Secret name: DEPLOY_USER ---"
echo "deploy"
echo ""
echo "--- Secret name: DEPLOY_HOST ---"
echo "speakeasy.mchugh.au"
echo ""
echo "============================================================"
echo "Done! Copy each value above into the matching GitHub Secret."
echo "============================================================"
