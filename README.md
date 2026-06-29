# Go Devcontainers CLI

An independent, native Go translation of the official [devcontainers/cli](https://github.com/devcontainers/cli) project.

## Project Goals

The main objectives of this project are:
1. **Zero Dependencies**: Provide a single, self-contained executable for running devcontainers without requiring Node.js, `npm`, or any external package managers to be installed on the host system.
2. **First-class Performance**: Leverage Go's performance, concurrency model, and fast startup times.
3. **Compatibility**: Maintain complete compatibility with the [Devcontainer Specification](https://github.com/devcontainers/spec).

---

## Installation

### From GitHub Release/CI Artifacts
1. Go to the Actions run artifacts page on GitHub.
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

---

## Usage & Examples

Run `devcontainer --help` or `devcontainer [command] --help` to see complete instructions and available flags.

### 1. Build a Dev Container Image
Builds a dev container image as defined by the workspace configurations:
```bash
devcontainer build --workspace-folder /home/user/myproject
```

### 2. Create and Run a Dev Container (`up`)
Provision and start a dev container for your workspace:
```bash
devcontainer up --workspace-folder /home/user/myproject --id-label app=development
```

### 3. Run Commands Inside the Container (`exec`)
Execute a shell command inside your running dev container:
```bash
devcontainer exec --workspace-folder /home/user/myproject -- whoami
```

### 4. Inspect Dev Container Configurations
Read and inspect resolved configurations for verification:
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
