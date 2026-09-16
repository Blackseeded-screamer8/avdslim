#!/usr/bin/env bash
# ==============================================================================
# monitor-pull.sh - Passive Pull & Adoption Monitor for avdslim
# Queries GitHub traffic, referrers, release downloads, and stargazers in 2s.
# ==============================================================================

set -eo pipefail

REPO="kdbhalala/avdslim"
BOLD="\033[1m"
GREEN="\033[0;32m"
BLUE="\033[0;34m"
YELLOW="\033[1;33m"
NC="\033[0m"

echo -e "${BOLD}${BLUE}=== AVD-SLIM Passive Adoption Pulse ===${NC}\n"

if command -v gh >/dev/null 2>&1; then
    # 1. GitHub 14-Day Traffic & Clones
    echo -e "${BOLD}[1/4] GitHub 14-Day Traffic & Clones:${NC}"
    CLONES_JSON=$(gh api "repos/$REPO/traffic/clones" 2>/dev/null || echo "{}")
    VIEWS_JSON=$(gh api "repos/$REPO/traffic/views" 2>/dev/null || echo "{}")

    TOTAL_CLONES=$(echo "$CLONES_JSON" | jq -r '.count // "N/A"')
    UNIQUE_CLONERS=$(echo "$CLONES_JSON" | jq -r '.uniques // "N/A"')
    TOTAL_VIEWS=$(echo "$VIEWS_JSON" | jq -r '.count // "N/A"')
    UNIQUE_VISITORS=$(echo "$VIEWS_JSON" | jq -r '.uniques // "N/A"')

    echo -e "  • ${GREEN}Clones (Git Pulls):${NC}    $TOTAL_CLONES total (${BOLD}$UNIQUE_CLONERS unique machines${NC})"
    echo -e "  • ${GREEN}Views (Repo Visits):${NC}   $TOTAL_VIEWS total (${BOLD}$UNIQUE_VISITORS unique visitors${NC})"

    # 2. Top Referrers
    echo -e "\n${BOLD}[2/4] Top Referral Sources (Last 14 Days):${NC}"
    REFERRERS=$(gh api "repos/$REPO/traffic/popular/referrers" 2>/dev/null || echo "[]")
    echo "$REFERRERS" | jq -r '.[] | "  - \(.referrer): \(.count) views (\(.uniques) unique)"' 2>/dev/null || echo "  (none recorded)"

    # 3. GitHub Engagement
    echo -e "\n${BOLD}[3/4] GitHub Community Engagement:${NC}"
    gh repo view "$REPO" --json stargazerCount,forkCount --jq '"  • Stars: \(.stargazerCount) | Forks: \(.forkCount)"' 2>/dev/null || echo "  (unavailable)"

    # 4. Release Binary Downloads (All versions / tarballs)
    echo -e "\n${BOLD}[4/4] Release Binary Downloads across Releases:${NC}"
    RELEASES_JSON=$(gh api "repos/$REPO/releases" 2>/dev/null || echo "[]")
    TOTAL_DOWNLOADS=$(echo "$RELEASES_JSON" | jq '[.[].assets[].download_count] | add // 0')
    echo -e "  • Total Release Binary Downloads: ${GREEN}${BOLD}$TOTAL_DOWNLOADS${NC}"
    echo "$RELEASES_JSON" | jq -r '.[] | "    - \(.tag_name): \([.assets[].download_count] | add // 0) downloads"' 2>/dev/null || echo "  (no releases found)"
else
    echo -e "${YELLOW}Notice: 'gh' CLI not found. Install gh to query GitHub traffic.${NC}"
fi

echo ""
