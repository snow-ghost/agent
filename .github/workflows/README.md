# GitHub Actions Workflows

This directory contains the CI/CD pipelines for the project:

## Pipeline Files

### `ci.yml`
- **Purpose**: Main CI pipeline that runs on every push and pull request
- **Triggers**: Push to `main` or `develop` branches, and pull requests to `main`
- **Actions**:
  - Runs linter (golangci-lint)
  - Runs go vet
  - Runs format check
  - Executes unit tests
  - Generates and uploads code coverage

### `test.yml`
- **Purpose**: Dedicated unit tests pipeline
- **Triggers**: Push to `main` or `develop` branches, and pull requests to `main`
- **Actions**:
  - Runs all unit tests
  - Generates code coverage report
  - Uploads coverage to Codecov

### `lint.yml`
- **Purpose**: Code quality checks
- **Triggers**: Push to `main` or `develop` branches, and pull requests to `main`
- **Actions**:
  - Runs golangci-lint
  - Runs go vet
  - Checks code formatting

### `build.yml`
- **Purpose**: Build verification pipeline
- **Triggers**: Push to `main` or `develop` branches, and pull requests to `main`
- **Actions**:
  - Builds all Go packages
  - Builds all binaries (worker, router, kb-indexer, llmrouter)
  - Verifies the created binaries

### `docker.yml`
- **Purpose**: Docker image build pipeline
- **Triggers**: Push to `main` or `develop` branches, and pull requests to `main`
- **Actions**:
  - Builds Docker images for worker, router and main application
  - Verifies the created images

### `release.yml` (Pre-existing)
- **Purpose**: Automated release pipeline
- **Triggers**: Git tags matching `v*` pattern or manual dispatch
- **Actions**:
  - Creates GitHub release with changelog
  - Builds and pushes Docker images to registry
  - Builds binaries for multiple platforms
  - Uploads release assets
  - Performs security scanning

### `deploy.yml` (Pre-existing)
- **Purpose**: Deployment pipeline
- **Triggers**: Push to `main` branch or manual dispatch
- **Actions**:
  - Deploys to staging or production environments
  - Performs health checks
  - Sends notifications

## Go Version

All pipelines use Go version 1.24.x to maintain consistency across the development and CI/CD environments.

## Configuration

- Linting is configured using `.golangci.yml` in the project root
- Dockerfiles: `Dockerfile`, `Dockerfile.worker`, `Dockerfile.router`
- Environment-specific secrets are managed through GitHub repository settings