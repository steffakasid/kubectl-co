# Requirements — kubectl-co

## Overview

kubectl-co is a kubectl plugin that simplifies managing multiple kubeconfig files.
It allows users to add, delete, switch, and list kubeconfig files via symbolic links,
replacing manual shell functions with a single CLI tool.

---

## User Stories

### REQ-1: Add a config by copying an existing file

**As a** Kubernetes user,
**I want** to add an existing kubeconfig file to kubectl-co by providing its path and a name,
**so that** I can later switch to it by name.

**Acceptance criteria:**

- `kubectl co --add <name> <path>` copies the file at `<path>` to `~/.kube/co/<name>`.
- The copied file has owner-only permissions (0700).
- An informational message is printed on success.
- An error is returned if the source file does not exist.

### REQ-2: Add a config by creating an empty file

**As a** Kubernetes user,
**I want** to add a new empty kubeconfig file by providing only a name,
**so that** I can initialise it later.

**Acceptance criteria:**

- `kubectl co --add <name>` creates an empty file at `~/.kube/co/<name>`.
- The created file has owner-only permissions (0700).
- An informational message is printed indicating the file needs initialisation.

### REQ-3: Switch to a named config

**As a** Kubernetes user,
**I want** to switch to a named kubeconfig by providing its name as a positional argument,
**so that** `~/.kube/config` points to that config via symlink.

**Acceptance criteria:**

- `kubectl co <name>` creates a symlink `~/.kube/config -> ~/.kube/co/<name>`.
- The previous symlink target is recorded as the "previous" config.
- Any existing `~/.kube/config` symlink and `~/.kube/co/previous` symlink are cleaned up first.
- If the named config does not exist, nothing is changed (no error).
- The symlink permissions are set to 0700 to avoid kubectl warnings.

### REQ-4: Switch to the previous config

**As a** Kubernetes user,
**I want** to switch back to the previously active config with `--previous`,
**so that** I can quickly toggle between two configs.

**Acceptance criteria:**

- `kubectl co --previous` creates a symlink `~/.kube/config -> <previous config path>`.
- The flag accepts no additional arguments.
- An error is returned if no previous config path is recorded.

### REQ-5: Show the current config path

**As a** Kubernetes user,
**I want** to see the current config path with `--current`,
**so that** I know which config is active.

**Acceptance criteria:**

- `kubectl co --current` prints the resolved path of the current kubeconfig symlink.
- (Currently wired as a flag; the output path is the symlink target.)

### REQ-6: Delete a named config

**As a** Kubernetes user,
**I want** to delete a named config with `--delete <name>`,
**so that** unused configs are removed.

**Acceptance criteria:**

- `kubectl co --delete <name>` removes the file `~/.kube/co/<name>`.
- Before deletion, the kubeconfig symlink is re-linked to the previous config.
- An error is returned if the named config does not exist.
- An error is returned if re-linking fails (the file is not deleted in this case).

### REQ-7: List all available configs

**As a** Kubernetes user,
**I want** to list all stored configs by running `kubectl co` with no arguments,
**so that** I can see what is available.

**Acceptance criteria:**

- `kubectl co` with no flags or arguments lists all entries in `~/.kube/co/` except the `previous` symlink.
- The currently active config is highlighted in red.

### REQ-8: Shell completion

**As a** Kubernetes user,
**I want** bash and zsh shell completion,
**so that** I can tab-complete config names and flags.

**Acceptance criteria:**

- `kubectl-co completion bash` outputs a bash completion script.
- `kubectl-co completion zsh` outputs a zsh completion script (via bashcompinit).
- When `COMP_LINE` and `COMP_POINT` are set, the binary outputs matching config names or flags.

### REQ-9: Flag exclusivity

**As a** Kubernetes user,
**I want** the tool to reject conflicting flags,
**so that** I don't accidentally combine incompatible operations.

**Acceptance criteria:**

- `--add`, `--delete`, `--previous`, and `--current` are mutually exclusive.
- A clear error message is returned when more than one is provided.

### REQ-10: Debug mode

**As a** Kubernetes user,
**I want** a `--debug` flag to enable verbose output,
**so that** I can troubleshoot problems.

**Acceptance criteria:**

- `--debug` sets the log level to "debug" via eslog.
- Debug-level messages are visible when the flag is set.

### REQ-11: Version output

**As a** Kubernetes user,
**I want** a `--version` flag,
**so that** I can verify which version is installed.

**Acceptance criteria:**

- `--version` prints `kubectl-co version: <version>` using the build-time injected version string.

---

## Non-functional Requirements

### NFR-1: Security — file permissions

All config files and symlinks must use owner-only permissions (0700) to prevent
accidental exposure of cluster credentials.

### NFR-2: Compatibility — kubectl plugin convention

The binary must be named `kubectl-co` so that kubectl discovers it as `kubectl co`.

### NFR-3: Cross-platform builds

GoReleaser builds for `linux` and `darwin` (amd64 + arm64 implied by default).

### NFR-4: Homebrew distribution

A Homebrew formula is published to `steffakasid/homebrew-kubectl-co` on each release.

### NFR-5: CI/CD

- Tests run on every push/PR to `main` via GitHub Actions.
- Releases are triggered on a monthly schedule using go-semantic-release + GoReleaser.
- CodeQL analysis runs on push/PR and weekly.
