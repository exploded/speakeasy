#!/bin/bash
echo "=== systemd service status ==="
systemctl status speakeasy --no-pager

echo ""
echo "=== last 30 lines of service journal ==="
journalctl -u speakeasy -n 30 --no-pager

echo ""
echo "=== service file ==="
cat /etc/systemd/system/speakeasy.service 2>/dev/null || echo "NOT FOUND at /etc/systemd/system/speakeasy.service"

echo ""
echo "=== binary ==="
ls -la /var/www/speakeasy/speakeasy-linux 2>/dev/null || echo "NOT FOUND"

echo ""
echo "=== /var/www/speakeasy contents ==="
ls -la /var/www/speakeasy/

echo ""
echo "=== nginx config ==="
cat /etc/nginx/sites-enabled/speakeasy* 2>/dev/null || echo "No speakeasy nginx config found in sites-enabled"

echo ""
echo "=== what is listening on port 8282 ==="
ss -tlnp | grep 8282 || echo "Nothing on 8282"
