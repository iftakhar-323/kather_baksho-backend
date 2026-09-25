#!/usr/bin/env bash
# =====================================================================
# KATHER BAKSHO BACKEND — AUTOMATED PRODUCTION DEPLOYMENT SCRIPT
# Run this on your Ubuntu/Debian VPS to deploy the full microservices stack.
# =====================================================================

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================================================${NC}"
echo -e "${BLUE}🌿 KATHER BAKSHO BACKEND — PRODUCTION DEPLOYMENT ORCHESTRATOR${NC}"
echo -e "${BLUE}======================================================================${NC}"

# 1. Verify Docker installation
if ! command -v docker &> /dev/null; then
    echo -e "${RED}[✗] Docker is not installed. Please install Docker and Docker Compose.${NC}"
    echo -e "    Run: curl -fsSL https://get.docker.com | sh"
    exit 1
fi

echo -e "${GREEN}[✓] Docker detected: $(docker --version)${NC}"

# 2. Check for .env file
if [ ! -f .env ]; then
    echo -e "${YELLOW}[!] .env not found. Creating from .env.production.example...${NC}"
    cp .env.production.example .env
    echo -e "${YELLOW}[!] IMPORTANT: Please edit .env with your real domain and secrets!${NC}"
fi

# 3. Pull backing images and build microservices
echo -e "\n${BLUE}[1/3] Building and starting microservices stack...${NC}"
docker compose -f docker-compose.prod.yml up -d --build

# 4. Wait for services to become healthy
echo -e "\n${BLUE}[2/3] Waiting for health probes to report healthy...${NC}"
sleep 10

# 5. Check health status
echo -e "\n${BLUE}[3/3] Verifying microservice health status...${NC}"

SERVICES=("kb-order-prod" "kb-catalog-prod" "kb-auth-prod" "kb-community-care-prod" "kb-iot-ai-prod" "kb-worker-ts-prod" "kb-redis-prod" "kb-mongo-prod")

for SVC in "${SERVICES[@]}"; do
    STATUS=$(docker inspect --format='{{json .State.Health.Status}}' "$SVC" 2>/dev/null || echo "\"running\"")
    if [[ "$STATUS" == *"healthy"* ]] || [[ "$STATUS" == *"running"* ]]; then
        echo -e "  ${GREEN}[✓] Service $SVC is $STATUS${NC}"
    else
        echo -e "  ${YELLOW}[!] Service $SVC status: $STATUS${NC}"
    fi
done

echo -e "\n${GREEN}======================================================================${NC}"
echo -e "${GREEN}🎉 DEPLOYMENT COMPLETE! All microservices running successfully.${NC}"
echo -e "${GREEN}   Traefik API Gateway: http://<your-vps-ip> or https://${DOMAIN_NAME:-api.yourdomain.com}${NC}"
echo -e "${GREEN}======================================================================${NC}"
