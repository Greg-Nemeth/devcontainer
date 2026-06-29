# Go Devcontainers CLI

An independent, native Go translation of the official [devcontainers/cli](https://github.com/devcontainers/cli) project.

## Project Goals

The main objectives of this project are:
1. **Zero Dependencies**: Provide a single, self-contained executable for running devcontainers without requiring Node.js, `npm`, or any external package managers to be installed on the host system.
2. **First-class Performance**: Leverage Go's performance, concurrency model, and fast startup times.
3. **Compatibility**: Maintain complete compatibility with the [Devcontainer Specification](https://github.com/devcontainers/spec).

---

## Features Matrix (Parity Status)

The Go CLI (`dc`) supports the following capabilities to reach near-complete parity with the TypeScript CLI:
* **OCI Features Downloader**: Pulls Features dynamically from public registries (such as `ghcr.io`), resolves Bearer tokens automatically, and extracts gzipped tarballs.
* **Dev Container Templates**: Scaffolds full developer project environments dynamically from OCI template repositories, resolving template placeholder options (`${templateOption:value}`) inside config files.
* **Interactive Terminal & Signal Forwarding**: Runs fully interactive shells inside containers (supporting arrow keys, tab completions, and text editors like Vim) by placing the terminal in raw mode and forwarding window resize events (`SIGWINCH`).
* **Headless IDE Server Injection**: Installs and starts headless servers (like OpenVSCode Server or JetBrains Projector) inside running containers.
* **SSH & GPG Agent Forwarding**: Dynamically maps host agent sockets inside containers using bind mounts and env injection to relay commit signatures and Git authentication safely.
* **WSL Path Translation**: Detects WSL and automatically translates Linux paths under `/mnt/` (e.g. `/mnt/c/project`) to Windows host formats (`C:\project`) for Docker/Podman Desktop compatibility.
* **Docker Compose Overlays**: Orchestrates compose services and dynamically compiles compose override overlay files, with built-in array-to-map conversions for Podman build arguments.

---

## Installation

### From GitHub Release/CI Artifacts
1. Go to the Releases page or Actions run artifacts page on GitHub.
2. Download the pre-built `devcontainer-linux-amd64` binary.
3. Extract and make the binary executable:
   ```bash
   chmod +x devcontainer-linux-amd64
   ```
4. Move the binary into your system `PATH`:
   ```bash
   sudo mv devcontainer-linux-amd64 /usr/local/bin/devcontainer
   ```

### Building From Source
Ensure you have Go installed (version 1.25 or higher), then run:
```bash
git clone https://github.com/Greg-Nemeth/devcontainer.git
cd devcontainer
go build -o devcontainer ./cmd/devcontainer/main.go
```
You can now run `./devcontainer` or copy it to your system paths.

---

## Configuration

The CLI supports parsing standard `devcontainer.json` files including comments (JSONC syntax). By default, it will look up configuration files in the following locations within your workspace folder:
1. `.devcontainer/devcontainer.json`
2. `.devcontainer.json`

### Flag Configuration
You can supply global and command-specific configuration parameters via flags:
- `--workspace-folder`: The path to the folder containing your project and devcontainer files.
- `--config`: Custom path to a `devcontainer.json` file.
- `--override-config`: Path to a config file containing overrides for the base configurations.
- `--log-level`: Set severity of logs (e.g. `info`, `debug`, `trace`).
- `--log-format`: Output logs in either `text` (default) or `json` formats.
- `--docker-path`: Custom path to the Docker or Podman CLI executable.

---

## Usage & Examples

Run `devcontainer --help` or `devcontainer [command] --help` to see complete instructions and available flags.

### 1. Scaffold a Project from a Template
Scaffolds a new project from a public dev container template repository, substituting template options:
```bash
mkdir -p /tmp/my-go-project
devcontainer templates apply \
  --template ghcr.io/devcontainers/templates/go:1 \
  --workspace-folder /tmp/my-go-project \
  --option goVersion=1.21
```

### 2. Create and Run a Dev Container (`up`)
Provision and start a dev container for your workspace. This automatically detects and mounts your host GPG/SSH agent sockets, and translates paths if executing inside WSL:
```bash
devcontainer up --workspace-folder /home/user/myproject --id-label app=development
```

### 3. Run Interactive Commands Inside the Container (`exec`)
Execute an interactive shell session with full arrow keys, terminal resize event relay, and tab completions:
```bash
devcontainer exec --workspace-folder /home/user/myproject -- /bin/bash
```

### 4. Inject a Headless IDE Server
Download, extract, and start a headless IDE server inside a running container:
```bash
devcontainer inject-server --workspace-folder /home/user/myproject --type openvscode
```

### 5. Inspect Dev Container Configurations
Read and inspect resolved configurations with comments stripped for validation:
```bash
devcontainer read-configuration --workspace-folder /home/user/myproject
```

---

## Development Methodology (TDD)

This project strictly adheres to **Test-Driven Development (TDD)** using the **Red-Green-Refactor** cycle:
1. **Red**: Write a failing unit or integration test defining the desired behavior before writing any production code.
2. **Green**: Write the minimal amount of production code required to make the test pass.
3. **Refactor**: Clean up the code while keeping tests green.

Run the test suite using standard Go commands:
```bash
go test -v ./...
```

---

## Workspace Structure

- `cli/` (git-ignored): Cloned copy of the upstream [devcontainers/cli](https://github.com/devcontainers/cli) repository.
- `spec/` (git-ignored): Cloned copy of the upstream [devcontainers/spec](https://github.com/devcontainers/spec) repository.
- `scripts/`: Utility scripts for syncing and comparing with upstream releases.
