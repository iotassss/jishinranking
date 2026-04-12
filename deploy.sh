#!/bin/bash
set -e

BOLD="\033[1m"
CYAN="\033[36m"
GREEN="\033[32m"
YELLOW="\033[33m"
RESET="\033[0m"

# application build
echo -e "${BOLD}${CYAN}=============================${RESET}"
echo -e "${BOLD}${CYAN} Build${RESET}"
echo -e "${BOLD}${CYAN}=============================${RESET}"
make build

# deploy
echo -e "${BOLD}${GREEN}=============================${RESET}"
echo -e "${BOLD}${GREEN} Deploy infrastructure${RESET}"
echo -e "${BOLD}${GREEN}=============================${RESET}"
export AWS_PROFILE=worker
cd terraform
terraform apply
cd ../

# quick build in production environment
echo -e "${BOLD}${YELLOW}=============================${RESET}"
echo -e "${BOLD}${YELLOW} Invoke Lambda${RESET}"
echo -e "${BOLD}${YELLOW}=============================${RESET}"
aws lambda invoke \
  --function-name jishinranking-batch \
  --log-type Tail \
  --payload '{}' \
  /tmp/lambda_out.json \
  --query 'LogResult' --output text | base64 -d

cat /tmp/lambda_out.json
