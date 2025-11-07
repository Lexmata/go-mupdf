# Local Pipeline Testing Scripts

## run-pipeline-local.sh

Run Bitbucket Pipelines steps locally using Docker to debug CI issues.

### Prerequisites

- Docker installed and running
- Access to pull Docker images

### Usage

```bash
# Run the test step (default)
./scripts/run-pipeline-local.sh

# Run the lint step
./scripts/run-pipeline-local.sh --step lint

# Use a different Go version
./scripts/run-pipeline-local.sh --step test --image golang:1.24

# Show help
./scripts/run-pipeline-local.sh --help
```

### What it does

1. **Lint Step**:
   - Installs all required system dependencies
   - Runs `gofmt` to check formatting
   - Runs `go vet`
   - Runs `golangci-lint`
   - Runs `staticcheck`

2. **Test Step**:
   - Installs all required system dependencies
   - Runs tests with race detection: `go test -v -race ./...`
   - Generates coverage reports
   - Creates HTML coverage report

### Benefits

- **Debug CI issues locally**: Reproduce CI failures on your machine
- **Faster iteration**: Test changes before pushing
- **Isolated environment**: Uses the same Docker image as CI
- **No CI credits used**: Test as many times as needed

### Tips

- The script mounts your current directory, so changes are reflected immediately
- Use `--image` to test with different Go versions
- Add `set -x` at the top of the Docker command for verbose output
- Check the exit code to see if the step passed or failed

### Example Output

```
=== Running Bitbucket Pipeline Locally ===
Image: golang:1.23
Step: test
Directory: /home/user/go-mupdf

Running test step...
Running tests with race detection...
Running tests with coverage...
Generating coverage report...

✅ Pipeline step 'test' completed successfully!
```

