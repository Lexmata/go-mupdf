#!/bin/bash

# Automated Release Script for Go MuPDF
# This script helps create and publish releases with proper validation

set -e

echo "🚀 Go MuPDF Release Automation Script"
echo "====================================="

# Configuration
CURRENT_VERSION=$(cat VERSION)
REPO_URL="bitbucket.org/lexmata/go-mupdf"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to validate semantic version
validate_semver() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9\.-]+)?(\+[a-zA-Z0-9\.-]+)?$ ]]; then
        return 1
    fi
    return 0
}

# Function to compare versions (strictly greater; equal versions are rejected)
#
# sort -V orders "1.4.7" before "1.4.7-rc.1", which is the opposite of
# semver precedence (a pre-release precedes its own release), so base
# versions and pre-release suffixes are compared separately.
version_greater() {
    [ "$1" = "$2" ] && return 1

    local a_base="${1%%-*}" b_base="${2%%-*}"

    if [ "$a_base" != "$b_base" ]; then
        printf '%s\n%s\n' "$b_base" "$a_base" | sort -V -C
        return
    fi

    # Same base version: a pre-release ranks below the plain release.
    case "$1:$2" in
        *-*:*-*) printf '%s\n%s\n' "$2" "$1" | sort -V -C ;;  # both pre-release
        *-*:*)   return 1 ;;                                  # $1 pre-release, $2 release
        *:*-*)   return 0 ;;                                  # $1 release, $2 pre-release
    esac
}

# Function to check if we're on main branch
check_main_branch() {
    local current_branch=$(git branch --show-current)
    if [ "$current_branch" != "main" ]; then
        log_error "Must be on main branch to create release. Current branch: $current_branch"
        exit 1
    fi
}

# Function to check that the release tag does not already exist
check_tag_absent() {
    local version=$1
    if git rev-parse -q --verify "refs/tags/v$version" >/dev/null; then
        log_error "Tag v$version already exists"
        exit 1
    fi
}

# Function to check if working directory is clean
check_clean_working_dir() {
    if ! git diff-index --quiet HEAD --; then
        log_error "Working directory is not clean. Please commit or stash changes."
        git status --porcelain
        exit 1
    fi
}

# Function to run tests
run_tests() {
    log_info "Running comprehensive test suite..."
    
    # Build MuPDF if needed
    if [ ! -f "third_party/mupdf/build/release/libmupdf.a" ]; then
        log_info "Building MuPDF library..."
        cd third_party/mupdf
        make -j$(nproc) libs USE_SYSTEM_LIBS=no HAVE_X11=no HAVE_GLUT=no build=release
        cd ../..
    fi
    
    # Run tests
    go test -race -coverprofile=coverage.out ./... || {
        log_error "Tests failed"
        exit 1
    }
    
    # Check coverage (RELEASE_COVERAGE is reused in the tag message)
    local coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    RELEASE_COVERAGE="$coverage"
    log_success "Test coverage: ${coverage}%"
    
    if [ $(echo "$coverage < 70" | bc) -eq 1 ]; then
        log_warning "Test coverage is below 70%"
    fi
    
    # Run quality checks
    go vet ./... || {
        log_error "go vet failed"
        exit 1
    }
    
    # Check formatting
    if [ -n "$(gofmt -l .)" ]; then
        log_error "Code is not properly formatted"
        gofmt -l .
        exit 1
    fi
    
    log_success "All tests and quality checks passed"
}

# Function to update version
update_version() {
    local new_version=$1
    
    log_info "Updating version from $CURRENT_VERSION to $new_version"
    
    # Update VERSION file
    echo "$new_version" > VERSION
    
    # Update go.mod if needed (though Go modules use git tags)
    log_success "Version updated to $new_version"
}

# Function to update changelog from real git history since the last tag
update_changelog() {
    local version=$1
    local date=$(date +"%Y-%m-%d")

    log_info "Updating CHANGELOG.md for version $version"

    # Determine commit range: everything since the last tag (or full history)
    local last_tag range
    last_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    if [ -n "$last_tag" ]; then
        range="${last_tag}..HEAD"
    else
        range="HEAD"
    fi

    # Group commit subjects by conventional-commit prefix
    local added="" fixed="" changed="" subject text
    while IFS= read -r subject; do
        # Skip merge commits and version-bump/chore noise
        case "$subject" in
            "Merge "*) continue ;;
            "chore: bump version"*|"chore(release)"*) continue ;;
        esac

        # Strip the conventional-commit prefix for the changelog line
        text="$subject"
        if [[ "$subject" =~ ^[a-z]+(\([^\)]*\))?!?:\ (.*)$ ]]; then
            text="${BASH_REMATCH[2]}"
        fi

        # Classify on the parsed commit type so breaking-change forms
        # (feat!:, fix!:) are not misfiled under "Changed".
        local ctype="${subject%%:*}"
        ctype="${ctype%%(*}"
        ctype="${ctype%!}"

        case "$ctype" in
            feat) added+="- ${text}"$'\n' ;;
            fix)  fixed+="- ${text}"$'\n' ;;
            *)    changed+="- ${text}"$'\n' ;;
        esac
    done < <(git log --format='%s' "$range")

    if [ -z "$added" ] && [ -z "$fixed" ] && [ -z "$changed" ]; then
        log_error "No commits found since ${last_tag:-the beginning of history}; nothing to record in CHANGELOG.md"
        exit 1
    fi

    # Create a temporary file with the new entry
    local entry_file
    entry_file=$(mktemp)
    {
        echo "## [$version] - $date"
        echo ""
        if [ -n "$added" ]; then
            echo "### Added"
            printf '%s' "$added"
            echo ""
        fi
        if [ -n "$changed" ]; then
            echo "### Changed"
            printf '%s' "$changed"
            echo ""
        fi
        if [ -n "$fixed" ]; then
            echo "### Fixed"
            printf '%s' "$fixed"
            echo ""
        fi
    } > "$entry_file"

    # Insert the new entry after the [Unreleased] section
    sed -i "/## \[Unreleased\]/r $entry_file" CHANGELOG.md

    log_success "CHANGELOG.md updated"

    rm -f "$entry_file"
}

# Function to create git tag
create_git_tag() {
    local version=$1
    local tag_name="v$version"
    
    log_info "Creating git tag $tag_name"

    # Include the measured coverage line only when run_tests captured it
    local coverage_line=""
    if [ -n "${RELEASE_COVERAGE:-}" ]; then
        coverage_line="- ${RELEASE_COVERAGE}% test coverage
"
    fi

    # Create annotated tag with release notes
    git tag -a "$tag_name" -m "Release $version

This release includes:
- Complete Go MuPDF wrapper functionality
- Comprehensive documentation and examples
${coverage_line}- Production-ready CI/CD pipeline

See CHANGELOG.md for detailed changes."
    
    log_success "Git tag $tag_name created"
}

# Function to push release
push_release() {
    local version=$1
    local tag_name="v$version"
    
    log_info "Pushing release to remote repository"
    
    # Push changes and tag
    git push origin main
    git push origin "$tag_name"
    
    log_success "Release $tag_name pushed to repository"
    log_info "GitHub Actions will automatically build and publish release artifacts"
}

# Function to generate release summary
generate_release_summary() {
    local version=$1
    local tag_name="v$version"
    
    cat << EOF

🎉 Release $version Summary
========================

✅ Version: $version
✅ Git Tag: $tag_name
✅ Branch: main
✅ Tests: Passed with $(go tool cover -func=coverage.out | grep total | awk '{print $3}')
✅ Quality Checks: Passed
✅ Pipeline: Will trigger automatically

📦 What happens next:
1. GitHub Actions will detect the new tag
2. Comprehensive test suite will run
3. Binary artifacts will be built
4. Release notes will be generated
5. Artifacts will be published as GitHub Release assets

📋 Manual steps (if needed):
1. Review the pipeline execution at:
   https://bitbucket.org/lexmata/go-mupdf/addon/pipelines/home

2. Once pipeline completes, announce the release:
   - Update team wiki/documentation
   - Notify users of the new version
   - Consider creating a blog post or announcement

🔗 Installation for users:
   go get $REPO_URL@$tag_name

EOF
}

# Function to compute the next version from CURRENT_VERSION and a bump type
compute_bumped_version() {
    local bump=$1
    local base major minor patch
    base="${CURRENT_VERSION%%-*}"
    IFS='.' read -r major minor patch <<< "$base"

    case "$bump" in
        major)
            echo "$((major + 1)).0.0"
            ;;
        minor)
            echo "${major}.$((minor + 1)).0"
            ;;
        patch)
            echo "${major}.${minor}.$((patch + 1))"
            ;;
        pre)
            if [[ "$CURRENT_VERSION" =~ ^(.+)-rc\.([0-9]+)$ ]]; then
                echo "${BASH_REMATCH[1]}-rc.$((BASH_REMATCH[2] + 1))"
            else
                # CURRENT_VERSION is a plain release, so bump the patch
                # first: a pre-release of the current version would rank
                # BELOW the release that already shipped.
                echo "${major}.${minor}.$((patch + 1))-rc.1"
            fi
            ;;
    esac
}

# Main function
main() {
    echo ""

    # Parse command line arguments
    DRY_RUN=false
    local bump=""
    NEW_VERSION=""

    while [ $# -gt 0 ]; do
        case "$1" in
            --help|-h)
                cat << EOF
Usage: $0 [OPTIONS] [VERSION]

Create and publish a new release.

OPTIONS:
    --major     Increment major version (X.0.0)
    --minor     Increment minor version (x.Y.0)
    --patch     Increment patch version (x.y.Z)
    --pre       Create pre-release (add -rc.N suffix, or increment an existing one)
    --dry-run   Show what would be done without making changes
    --help      Show this help

EXAMPLES:
    $0 1.1.0               # Create specific version
    $0 --minor             # Auto-increment minor version
    $0 --patch             # Auto-increment patch version
    $0 --dry-run 1.1.0     # Test release process
    $0 --dry-run --patch   # Test auto-incremented patch release

EOF
                exit 0
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --major|--minor|--patch|--pre)
                if [ -n "$bump" ]; then
                    log_error "Only one of --major/--minor/--patch/--pre may be given"
                    exit 1
                fi
                bump="${1#--}"
                shift
                ;;
            -*)
                log_error "Unknown option: $1"
                echo "Run with --help for usage"
                exit 1
                ;;
            *)
                if [ -n "$NEW_VERSION" ]; then
                    log_error "Only one version argument may be given"
                    exit 1
                fi
                NEW_VERSION=$1
                shift
                ;;
        esac
    done

    # Determine new version
    if [ -n "$bump" ] && [ -n "$NEW_VERSION" ]; then
        log_error "Specify either an explicit version or a bump flag (--major/--minor/--patch/--pre), not both"
        exit 1
    fi

    if [ -n "$bump" ]; then
        if ! validate_semver "$CURRENT_VERSION"; then
            log_error "Current version in VERSION file is not valid semver: $CURRENT_VERSION"
            exit 1
        fi
        NEW_VERSION=$(compute_bumped_version "$bump")
        log_info "Auto-incrementing ($bump): $CURRENT_VERSION → $NEW_VERSION"
    fi

    if [ -z "$NEW_VERSION" ]; then
        echo "Usage: $0 [--dry-run] [--major|--minor|--patch|--pre] [version]"
        echo "Current version: $CURRENT_VERSION"
        echo "Run with --help for more options"
        exit 1
    fi

    # Validate version format
    if ! validate_semver "$NEW_VERSION"; then
        log_error "Invalid semantic version format: $NEW_VERSION"
        exit 1
    fi

    # Check if new version is greater than current
    if ! version_greater "$NEW_VERSION" "$CURRENT_VERSION"; then
        log_error "New version $NEW_VERSION must be greater than current version $CURRENT_VERSION"
        exit 1
    fi

    log_info "Preparing release $NEW_VERSION"

    # Pre-flight checks
    check_main_branch
    check_tag_absent "$NEW_VERSION"
    check_clean_working_dir
    
    # Run tests and quality checks
    run_tests
    
    if [ "${DRY_RUN:-false}" = "true" ]; then
        log_warning "DRY RUN MODE - No changes will be made"
        echo "Would perform the following actions:"
        echo "1. Update VERSION file: $CURRENT_VERSION → $NEW_VERSION"
        echo "2. Update CHANGELOG.md"
        echo "3. Commit changes"
        echo "4. Create git tag v$NEW_VERSION"
        echo "5. Push to remote repository"
        echo "6. Trigger automated release pipeline"
        exit 0
    fi
    
    # Confirm release
    echo ""
    read -p "Create release $NEW_VERSION? (y/N): " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        log_info "Release cancelled"
        exit 0
    fi
    
    # Create the release
    update_version "$NEW_VERSION"
    update_changelog "$NEW_VERSION"
    
    # Commit changes
    git add VERSION CHANGELOG.md
    git commit -m "Release $NEW_VERSION

- Update version to $NEW_VERSION
- Update changelog with release notes
- Ready for automated release publishing"
    
    create_git_tag "$NEW_VERSION"
    push_release "$NEW_VERSION"
    
    # Show summary
    generate_release_summary "$NEW_VERSION"
    
    log_success "Release $NEW_VERSION created successfully!"
}

# Check if bc is available for version comparison
if ! command -v bc >/dev/null 2>&1; then
    log_error "bc (basic calculator) is required but not installed"
    exit 1
fi

# Run main function
main "$@"