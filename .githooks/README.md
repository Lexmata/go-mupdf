# Git Hooks

This directory contains Git hooks for the Go MuPDF Wrapper project.

## Installation

Run the install script from the repository root:

```bash
./install.sh
```

This will configure Git to use hooks from this directory.

## Available Hooks

### pre-commit

Runs before each commit to ensure code quality and enforce git-flow workflow:

**Git Flow Branch Protection:**
- **Main branch**: Only allows merges from `develop` branch (blocks direct commits)
- **Develop branch**: Allows all commits
- **Feature/Bugfix/Hotfix/Release branches**: Allows all commits
- **Other branches**: Warns but allows commits

**Code Quality Checks:**
- **Code Formatting**: Checks that all Go files are formatted with `gofmt`
- **Static Analysis**: Runs `go vet` to catch common errors
- **Linting**: Runs `golangci-lint` if installed (non-blocking)
- **Tests**: Runs quick tests on changed packages and the main package

The hook will:
- Automatically build the MuPDF library if needed
- Only test packages with staged changes (for speed)
- Block commits if formatting or tests fail
- Warn (but not block) on golangci-lint issues
- Enforce git-flow workflow rules

### commit-msg

Validates commit message format using [Conventional Commits](https://www.conventionalcommits.org/) standard.

**Required Format:**
```
<type>: <description>
```

**Valid Types:**
- `feat` - A new feature
- `fix` or `bugfix` - A bug fix
- `docs` - Documentation only changes
- `style` - Code style changes (formatting, etc.)
- `refactor` - Code refactoring
- `perf` - Performance improvements
- `test` - Adding or updating tests
- `build` - Build system changes
- `ci` - CI/CD configuration changes
- `chore` - Other maintenance tasks
- `revert` - Reverting a previous commit

**Examples:**
```bash
feat: add PDF page rotation support
fix: resolve memory leak in context cleanup
docs: update README with installation instructions
refactor: simplify document loading logic
test: add unit tests for page extraction
ci: add Bitbucket Pipelines configuration
```

**Rules:**
- Must start with a valid type followed by a colon
- Must have a description after the colon
- Subject line should be under 72 characters (warning if longer)
- Merge and revert commits are automatically allowed

### post-merge

Creates git tags when release branches are merged to main:

- **Detects Release Merges**: Identifies when a `release/X.Y.Z` branch was merged to main
- **Creates Git Tags**: Creates an annotated git tag `vX.Y.Z` for the release

**How it works:**
1. Runs after every merge on the main branch
2. Checks if the merge included a release branch (`release/X.Y.Z`)
3. Extracts the version number from the branch name
4. Creates git tag `vX.Y.Z` if it doesn't exist

**Example:**
```bash
# Merge release branch to main
git checkout main
git merge release/1.2.0

# prepare-commit-msg hook (runs during merge):
# ✅ Updates VERSION file to 1.2.0
# ✅ Stages VERSION file (included in merge commit)

# post-merge hook (runs after merge):
# ✅ Creates tag v1.2.0
```

**Note:** The VERSION file is updated by the `prepare-commit-msg` hook as part of the merge commit. This hook only creates the release tag.

### prepare-commit-msg

Automatically updates the VERSION file when merging release branches to main:

- **Runs During Merge**: Executes when creating a merge commit
- **Detects Release Branches**: Identifies `release/X.Y.Z` branches being merged
- **Updates VERSION File**: Updates the VERSION file with the new version
- **Stages Changes**: Stages the VERSION file so it's included in the merge commit

**How it works:**
1. Runs when creating a merge commit on the main branch
2. Checks if the merge includes a release branch (`release/X.Y.Z`)
3. Extracts the version number from the branch name
4. Updates `VERSION` file if needed
5. Stages the VERSION file so it becomes part of the merge commit

**Example:**
```bash
# Merge release branch to main
git checkout main
git merge release/1.2.0

# prepare-commit-msg hook automatically:
# ✅ Detects release/1.2.0 merge
# ✅ Updates VERSION file to 1.2.0
# ✅ Stages VERSION file
# ✅ VERSION update is included in the merge commit

# post-merge hook then:
# ✅ Creates tag v1.2.0
```

**Note:** This ensures the version update is part of the merge commit itself, not a separate commit.

## Git Flow Workflow

The pre-commit hook enforces the following git-flow workflow:

### Branch Rules

1. **Main Branch** (`main` or `master`)
   - ❌ **Blocks direct commits**
   - ✅ **Allows merges from `develop` only**
   - Used for production releases

2. **Develop Branch** (`develop`)
   - ✅ **Allows all commits**
   - Used for integration of features

3. **Feature Branches** (`feature/*`)
   - ✅ **Allows all commits**
   - Created from `develop`
   - Merged back to `develop` when complete

4. **Bugfix Branches** (`bugfix/*`)
   - ✅ **Allows all commits**
   - Created from `develop`
   - Merged back to `develop` when complete

5. **Hotfix Branches** (`hotfix/*`)
   - ✅ **Allows all commits**
   - Created from `main`
   - Merged to both `develop` and `main`

6. **Release Branches** (`release/*`)
   - ✅ **Allows all commits**
   - Created from `develop`
   - Merged to both `develop` and `main`

### Workflow Example

```bash
# 1. Start a new feature
git checkout develop
git checkout -b feature/add-pdf-rotation

# 2. Make commits (pre-commit hook will validate)
git commit -m "feat: add PDF rotation support"

# 3. Merge to develop
git checkout develop
git merge feature/add-pdf-rotation

# 4. Create release branch and merge to main
git checkout -b release/1.1.0 develop
# ... final testing and bug fixes ...
git checkout main
git merge release/1.1.0  # ✅ Allowed (merge from release branch)
# 🎉 Hooks automatically:
#    prepare-commit-msg: Updates VERSION to 1.1.0 (included in merge commit)
#    post-merge: Creates tag v1.1.0
```

## Uninstalling

To stop using these hooks:

```bash
git config --unset core.hooksPath
```

## Manual Hook Execution

You can manually run hooks:

```bash
# Test pre-commit hook
.githooks/pre-commit

# Test commit-msg hook (requires a commit message file)
echo "feat: test commit message" | .githooks/commit-msg /dev/stdin

# Post-merge hook runs automatically after merges
# To test manually, simulate a merge scenario
```

## Troubleshooting

### Hook not running

Make sure you've run `./install.sh` to configure Git to use these hooks.

### MuPDF library build fails

If the hook fails to build MuPDF automatically, build it manually:

```bash
cd third_party/mupdf
make libs
cd ../..
```

### Tests are slow

The hook uses `-short` flag to run quick tests. For full tests, run:

```bash
go test ./pkg/mupdf/
```

### Bypassing hooks (not recommended)

If you need to bypass hooks in an emergency:

```bash
git commit --no-verify
```

**Warning**: Only use this if absolutely necessary, as it bypasses all quality checks.

