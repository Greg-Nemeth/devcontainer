# Antigravity CLI (Go Devcontainers CLI)

An independent, native Go translation of the official [devcontainers/cli](https://github.com/devcontainers/cli) project.

## Project Goals

The main objectives of this project are:
1. **Zero Dependencies**: Provide a single, self-contained executable for running devcontainers without requiring Node.js, `npm`, or any external package managers to be installed on the host system.
2. **First-class Performance**: Leverage Go's performance, concurrency model, and fast startup times.
3. **Compatibility**: Maintain complete compatibility with the [Devcontainer Specification](https://github.com/devcontainers/spec).

## Development Methodology (TDD)

This project strictly adheres to **Test-Driven Development (TDD)** using the **Red-Green-Refactor** cycle:
1. **Red**: Write a failing unit or integration test defining the desired behavior before writing any production code.
2. **Green**: Write the minimal amount of production code required to make the test pass.
3. **Refactor**: Clean up the code while keeping tests green.


---

## Workspace Structure

The workspace is set up to track upstream changes without committing their large history to our repository:

- `cli/` (git-ignored): Cloned copy of the upstream [devcontainers/cli](https://github.com/devcontainers/cli) repository.
- `spec/` (git-ignored): Cloned copy of the upstream [devcontainers/spec](https://github.com/devcontainers/spec) repository.
- `scripts/`: Utility scripts for syncing and comparing with upstream releases.

---

## Syncing & Upstream Maintenance Workflow

To ensure we can easily catch up with new releases of the upstream Node.js CLI:

1. **Check for Upstream Releases**:
   Run the update script to fetch tags and show the latest release of the upstream repository:
   ```bash
   ./scripts/update-upstream.sh
   ```

2. **Analyzing Upstream Changes**:
   When a new version (e.g., `v0.88.0` to `v0.89.0`) is released upstream, you can compare changes directly in the local clones:
   ```bash
   cd cli
   git diff v0.88.0..v0.89.0
   ```

3. **Incremental Translation**:
   Identify the files modified upstream (primarily under the JavaScript/TypeScript source code) and apply corresponding translations to the Go codebase.

4. **Tagging and Releasing**:
   Tag the Go repository matching the upstream version we are compatible with (e.g., `v0.88.0-go.1` or matching `v0.88.0` directly once parity is achieved).
