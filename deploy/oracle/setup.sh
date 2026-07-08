#!/usr/bin/env bash
# One-time bootstrap for an Oracle Cloud Always Free ARM VM (Ubuntu 22.04).
# Run as the default 'ubuntu' user:  bash deploy/oracle/setup.sh
set -euo pipefail

echo "==> Installing Docker + compose plugin"
sudo apt-get update -y
sudo apt-get install -y ca-certificates curl git
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
| sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update -y
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker "$USER"

echo "==> Creating shared docker networks (idempotent)"
sudo docker network inspect web >/dev/null 2>&1 || sudo docker network create web
sudo docker network inspect db  >/dev/null 2>&1 || sudo docker network create db

echo "==> Opening ports 80 + 443 in the instance firewall (Oracle images ship with a strict iptables)"
sudo iptables -I INPUT 6 -m state --state NEW -p tcp --dport 80  -j ACCEPT
sudo iptables -I INPUT 6 -m state --state NEW -p tcp --dport 443 -j ACCEPT
sudo netfilter-persistent save || sudo apt-get install -y iptables-persistent

echo ""
echo "Done. Log out/in (so 'docker' works without sudo), then bring up the edge + apps:"
echo "  cd ~/Realtime_tracker"
echo "  cp deploy/oracle/edge/.env.example deploy/oracle/edge/.env && nano deploy/oracle/edge/.env"
echo "  docker compose --env-file deploy/oracle/edge/.env -f deploy/oracle/edge/docker-compose.yml up -d"
echo "  bash deploy/oracle/newdb.sh kinnect <KINNECT_DB_PASSWORD>   # provision this app's database"
echo "  cp deploy/oracle/.env.example deploy/oracle/.env && nano deploy/oracle/.env"
echo "  docker compose --env-file deploy/oracle/.env -f deploy/oracle/docker-compose.yml up -d --build"
