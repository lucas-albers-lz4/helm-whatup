# Implementation Plan: Go Version Upgrade & Unit Testing

## Phase 1: Platform Support & Release Automation ✅

### Cross-Platform Binary Support
- ✅ Added support for Apple Silicon Macs (darwin-arm64)
- ✅ Added support for ARM64 Linux (linux-arm64)
- ✅ Updated Makefile with cross-platform build targets
- ✅ Updated installation script to detect and support new architectures

### Automated Release Process
- ✅ Created GitHub Actions workflow for automated releases
- ✅ Configured build matrix for all supported platforms:
  - linux-amd64
  - linux-arm64
  - darwin-amd64
  - darwin-arm64
- ✅ Set up both manual and tag-based release triggers
- ✅ Implemented artifact collection and GitHub release creation

### GitHub Actions Setup
- ✅ Created workflow file in `.github/workflows/release.yml`
- ✅ Configured the workflow to:
  - Trigger manually and on tag creation
  - Build for all supported platforms (linux-amd64, linux-arm64, darwin-amd64, darwin-arm64)
  - Run tests before building
  - Create GitHub release with built artifacts
  - Add release notes automatically

### Release Process
1. Update version in plugin.yaml
2. Commit changes
3. Create and push a new tag
4. GitHub Actions will automatically build and publish the release
5. Or manually trigger the workflow using the "workflow_dispatch" event

## Phase 3: Testing & CI/CD

### Testing
- Add basic unit tests with proper mocking
- Add tests for fetchReleases and fetchIndices functions
- Add tests for other formatting outputs (plain, yaml, table)
- Implement integration tests if needed

### Build Infrastructure
- Enhance bootstrap process for dependency management
- Add lint targets to Makefile for quality checks
- Add test target for running unit tests
- Configure GitHub Actions for continuous integration

## Phase 4: Documentation & Release

### Documentation & Finalization
- Update version information in plugin.yaml
- Update README.md with updated platform support
- Create release notes for the upgrade
- Document development workflow

### Release
- Create a new release with updated binary artifacts
- Test the plugin with the latest Helm version
- Update the plugin in the Helm plugin repository (if applicable)

## Development Workflow

### Quality Checks
```bash
# Run after each code change to verify tests pass
make test

# Run linter to ensure code quality before committing
make lint

# Run a specific linter only
make lint-specific LINTER=wrapcheck
```

### Build Process
```bash
# Rebuild the binary after changes
make build

# Build for all supported platforms
make dist
```

Clone existing upstream issues so I can fix them:

```python
from github import Github
import os

# Authentication - use a Personal Access Token
g = Github(os.environ.get("GITHUB_TOKEN"))

# Source and target repositories
source_repo = g.get_repo("original-owner/original-repo")
target_repo = g.get_repo("your-username/your-fork")

# Get all issues from source repo
source_issues = source_repo.get_issues(state="all")

# Clone each issue to the target repo
for issue in source_issues:
    body = f"""
Original issue: {issue.html_url}

{issue.body}
"""
    new_issue = target_repo.create_issue(
        title=f"[UPSTREAM] {issue.title}",
        body=body,
        labels=[label.name for label in issue.labels]
    )
    print(f"Created issue #{new_issue.number} from upstream #{issue.number}")
```

NOTES:
Install dependencies: pip install PyGithub
Create a GitHub Personal Access Token with 'repo' scope
Set it as an environment variable: export GITHUB_TOKEN=your_token_here
Customize the repo paths and run the script
This will create copies of all 6 issues in your fork, clearly labeled as upstream issues with links back to the originals.