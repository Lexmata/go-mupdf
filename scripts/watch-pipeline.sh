#!/bin/bash
# Watch Bitbucket Pipeline Status
# Usage: ./scripts/watch-pipeline.sh [pipeline-uuid]

set -e

REPO_OWNER="lexmata"
REPO_SLUG="go-mupdf"
REFRESH_INTERVAL=10

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
GRAY='\033[0;90m'
NC='\033[0m'

# Function to get latest pipeline
get_latest_pipeline() {
    curl -s "https://api.bitbucket.org/2.0/repositories/${REPO_OWNER}/${REPO_SLUG}/pipelines/?sort=-created_on&pagelen=1" \
        | jq -r '.values[0] | {uuid: .uuid, build: .build_number, state: .state.name, target: .target.ref_name, created: .created_on}'
}

# Function to get pipeline details
get_pipeline_status() {
    local pipeline_uuid=$1
    curl -s "https://api.bitbucket.org/2.0/repositories/${REPO_OWNER}/${REPO_SLUG}/pipelines/${pipeline_uuid}" \
        | jq -r '. | {
            build: .build_number,
            state: .state.name,
            result: .state.result.name,
            target: .target.ref_name,
            trigger: .trigger.name,
            created: .created_on,
            completed: .completed_on,
            duration_seconds: .duration_in_seconds
        }'
}

# Function to get pipeline steps
get_pipeline_steps() {
    local pipeline_uuid=$1
    curl -s "https://api.bitbucket.org/2.0/repositories/${REPO_OWNER}/${REPO_SLUG}/pipelines/${pipeline_uuid}/steps/" \
        | jq -r '.values[] | [.uuid, .name, .state.name, .state.result.name // "PENDING"] | @tsv'
}

# Function to display status with color
status_color() {
    local state=$1
    case "$state" in
        SUCCESSFUL) echo -e "${GREEN}✅ $state${NC}" ;;
        FAILED) echo -e "${RED}❌ $state${NC}" ;;
        IN_PROGRESS) echo -e "${YELLOW}⏳ $state${NC}" ;;
        PENDING) echo -e "${GRAY}⏸  $state${NC}" ;;
        STOPPED) echo -e "${YELLOW}⏹  $state${NC}" ;;
        *) echo -e "${BLUE}$state${NC}" ;;
    esac
}

# Main monitoring function
watch_pipeline() {
    local pipeline_uuid=$1

    echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║         Bitbucket Pipeline Monitor - go-mupdf               ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    if [ -z "$pipeline_uuid" ]; then
        echo "Getting latest pipeline..."
        pipeline_info=$(get_latest_pipeline)
        pipeline_uuid=$(echo "$pipeline_info" | jq -r '.uuid' | tr -d '{}')
        echo "Monitoring pipeline: $pipeline_uuid"
        echo "$pipeline_info" | jq '.'
        echo ""
    fi

    local last_state=""
    local iteration=0

    while true; do
        clear
        echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${BLUE}║         Bitbucket Pipeline Monitor - go-mupdf               ║${NC}"
        echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
        echo ""

        # Get pipeline status
        status=$(get_pipeline_status "{$pipeline_uuid}")
        state=$(echo "$status" | jq -r '.state')

        echo -e "${BLUE}Pipeline Status:${NC}"
        echo "$status" | jq '.'
        echo ""

        echo -e "${BLUE}Pipeline Steps:${NC}"
        echo ""

        # Get and display steps
        get_pipeline_steps "{$pipeline_uuid}" | while IFS=$'\t' read -r step_uuid step_name step_state step_result; do
            printf "  %-40s " "$step_name"
            if [ "$step_state" = "COMPLETED" ]; then
                status_color "$step_result"
            else
                status_color "$step_state"
            fi
        done

        echo ""
        echo -e "${GRAY}───────────────────────────────────────────────────────────────${NC}"

        # Check if completed
        if [ "$state" = "COMPLETED" ]; then
            result=$(echo "$status" | jq -r '.result')
            echo ""
            echo -e "Pipeline completed with result: $(status_color "$result")"
            echo ""
            echo "View full details at:"
            echo "https://bitbucket.org/${REPO_OWNER}/${REPO_SLUG}/addon/pipelines/results/$pipeline_uuid"
            break
        fi

        echo ""
        echo -e "${GRAY}Refreshing in ${REFRESH_INTERVAL}s... (Ctrl+C to stop) [Iteration: $((++iteration))]${NC}"
        sleep $REFRESH_INTERVAL
    done
}

# Check dependencies
if ! command -v jq >/dev/null 2>&1; then
    echo "Error: jq is required but not installed."
    echo "Install with: sudo apt-get install jq"
    exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
    echo "Error: curl is required but not installed."
    exit 1
fi

# Show usage
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << EOF
Bitbucket Pipeline Monitor

Usage:
  $0                 Watch latest pipeline
  $0 [pipeline-uuid] Watch specific pipeline
  $0 --help          Show this help

Examples:
  $0
  $0 {12345678-1234-1234-1234-123456789012}

EOF
    exit 0
fi

# Run monitor
watch_pipeline "$1"

