#!/bin/bash

# Pipeline Validation Script
# This script validates the Bitbucket Pipeline configuration

set -e

echo "=== Bitbucket Pipeline Validation ==="

# Function to check if file exists
check_file() {
    if [ -f "$1" ]; then
        echo "✅ $1 exists"
        return 0
    else
        echo "❌ $1 missing"
        return 1
    fi
}

# Function to validate YAML syntax
validate_yaml() {
    echo "📋 Validating YAML syntax..."
    if command -v python3 >/dev/null 2>&1 && python3 -c 'import yaml' >/dev/null 2>&1; then
        if python3 -c 'import yaml,sys; yaml.safe_load(open("bitbucket-pipelines.yml"))'; then
            echo "✅ YAML parses successfully (python3/pyyaml)"
        else
            echo "❌ YAML syntax errors found"
            return 1
        fi
    elif command -v yamllint >/dev/null 2>&1; then
        # Optional linter fallback when pyyaml is unavailable
        if yamllint bitbucket-pipelines.yml; then
            echo "✅ YAML syntax is valid (yamllint)"
        else
            echo "❌ YAML syntax errors found"
            return 1
        fi
    else
        echo "⚠️  No YAML validator available (python3+pyyaml or yamllint), skipping syntax validation (optional check)"
    fi
}

# Function to check pipeline structure
check_pipeline_structure() {
    echo "🔍 Checking pipeline structure..."
    
    # Check for required sections
    if grep -q "pipelines:" bitbucket-pipelines.yml; then
        echo "✅ pipelines section found"
    else
        echo "❌ pipelines section missing"
        return 1
    fi
    
    if grep -q "default:" bitbucket-pipelines.yml; then
        echo "✅ default pipeline found"
    else
        echo "❌ default pipeline missing"
        return 1
    fi
    
    if grep -q "branches:" bitbucket-pipelines.yml; then
        echo "✅ branch-specific pipelines found"
    else
        echo "❌ branch-specific pipelines missing"
        return 1
    fi
    
    if grep -q "pull-requests:" bitbucket-pipelines.yml; then
        echo "✅ pull-request pipeline found"
    else
        echo "❌ pull-request pipeline missing"
        return 1
    fi
}

# Function to check cache configuration
check_cache_config() {
    echo "🗄️  Checking cache configuration..."
    
    if grep -q "caches:" bitbucket-pipelines.yml; then
        echo "✅ Cache configuration found"
        
        if grep -q "go" bitbucket-pipelines.yml; then
            echo "✅ Go cache configured"
        else
            echo "⚠️  Go cache not configured"
        fi
        
        if grep -q "mupdf" bitbucket-pipelines.yml; then
            echo "✅ MuPDF cache configured"
        else
            echo "⚠️  MuPDF cache not configured"
        fi
    else
        echo "❌ No cache configuration found"
        return 1
    fi
}

# Function to validate helper scripts referenced by the pipeline
validate_scripts() {
    echo "📜 Validating helper scripts referenced by bitbucket-pipelines.yml..."

    local scripts script failed=false
    scripts=$(grep -oE 'scripts/[a-z-]+\.sh' bitbucket-pipelines.yml | sort -u)

    if [ -z "$scripts" ]; then
        echo "⚠️  No helper scripts referenced in bitbucket-pipelines.yml"
        return 0
    fi

    for script in $scripts; do
        if [ ! -f "$script" ]; then
            echo "❌ $script is referenced by the pipeline but missing"
            failed=true
            continue
        fi
        echo "✅ $script exists"

        # Check if executable -- report only, do not modify during validation
        if [ -x "$script" ]; then
            echo "✅ $script is executable"
        else
            echo "❌ $script is not executable (fix with: chmod +x $script)"
            failed=true
        fi

        # Basic syntax check
        if bash -n "$script"; then
            echo "✅ $script syntax is valid"
        else
            echo "❌ $script has syntax errors"
            failed=true
        fi
    done

    [ "$failed" = false ]
}

# Function to check project structure
check_project_structure() {
    echo "🏗️  Checking project structure..."
    
    REQUIRED_DIRS=(
        "pkg/mupdf"
        "third_party/mupdf"
        "scripts"
        "docs"
    )
    
    for dir in "${REQUIRED_DIRS[@]}"; do
        if [ -d "$dir" ]; then
            echo "✅ $dir directory exists"
        else
            echo "❌ $dir directory missing"
            return 1
        fi
    done
    
    REQUIRED_FILES=(
        "go.mod"
        "bitbucket-pipelines.yml"
        "docs/CI_CD_PIPELINE.md"
    )
    
    for file in "${REQUIRED_FILES[@]}"; do
        check_file "$file" || return 1
    done
}

# Validation result tracking (populated by run_check, consumed by generate_report)
CHECK_RESULTS=()
CHECKS_PASSED=0
CHECKS_FAILED=0
VALIDATION_PASSED=true

# Function to run a named check and record its result
run_check() {
    local name=$1
    shift

    if "$@"; then
        CHECK_RESULTS+=("✅ PASS: $name")
        CHECKS_PASSED=$((CHECKS_PASSED + 1))
    else
        CHECK_RESULTS+=("❌ FAIL: $name")
        CHECKS_FAILED=$((CHECKS_FAILED + 1))
        VALIDATION_PASSED=false
    fi
}

# Function to generate validation report from the actual check results
generate_report() {
    echo "📊 Generating validation report..."

    REPORT_FILE="pipeline-validation-report.txt"

    {
        echo "# Bitbucket Pipeline Validation Report"
        echo "Generated: $(date)"
        echo ""
        echo "## Check Results"
        local result
        for result in "${CHECK_RESULTS[@]}"; do
            echo "- $result"
        done
        echo ""
        echo "## Summary"
        echo "- Checks passed: $CHECKS_PASSED"
        echo "- Checks failed: $CHECKS_FAILED"
        echo "- Overall: $([ "$VALIDATION_PASSED" = true ] && echo "✅ PASSED" || echo "❌ FAILED")"
        echo ""
        if [ "$VALIDATION_PASSED" != true ]; then
            echo "## Next Steps"
            echo "1. Fix the failed checks listed above"
            echo "2. Re-run scripts/validate-pipeline.sh until all checks pass"
            echo ""
        fi
    } > "$REPORT_FILE"

    echo "✅ Validation report generated: $REPORT_FILE"
}

# Main validation function
main() {
    echo "Starting pipeline validation..."

    # Run all validation checks
    run_check "bitbucket-pipelines.yml present" check_file "bitbucket-pipelines.yml"
    run_check "YAML syntax" validate_yaml
    run_check "Pipeline structure" check_pipeline_structure
    run_check "Cache configuration" check_cache_config
    run_check "Helper scripts (exist, executable, valid syntax)" validate_scripts
    run_check "Project structure" check_project_structure

    # Generate report
    generate_report

    if [ "$VALIDATION_PASSED" = true ]; then
        echo ""
        echo "🎉 All validations passed!"
        echo "✅ Pipeline is ready for deployment"
        echo ""
        echo "Next steps:"
        echo "1. git add bitbucket-pipelines.yml scripts/ docs/CI_CD_PIPELINE.md"
        echo "2. git commit -m 'Add Bitbucket CI/CD pipeline'"
        echo "3. git push origin <branch-name>"
        echo ""
        return 0
    else
        echo ""
        echo "❌ Some validations failed!"
        echo "🔧 Please fix the issues above before deploying"
        echo ""
        return 1
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi