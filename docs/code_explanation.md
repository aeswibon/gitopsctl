# 🚀 Gitopsctl: Code Explanation

This document provides a comprehensive overview of the Gitopsctl Go codebase, explaining its structure, the purpose of each component, and the logic behind its operations.

## 💡 Overall Purpose

Gitopsctl is a lightweight, self-hosted GitOps controller designed to simplify Kubernetes application deployments. Its primary function is to continuously monitor specified Git repositories for changes in Kubernetes manifest files (YAMLs). When a new commit or modification is detected, it automatically applies those changes to the target Kubernetes cluster, ensuring that the cluster's state always reflects the desired state defined in Git.

This project is built in Go, leveraging its concurrency features and strong ecosystem for Kubernetes interaction.

## 📂 Project Structure

The project follows a standard Go project layout:

```txt
gitopsctl/
├── main.go               // Application entry point
├── go.mod                // Go module definition
├── go.sum                // Go module checksums
├── cmd/                  // Contains Cobra commands for CLI interface
│   ├── root.go           // Defines the base CLI command and global settings (e.g., logger)
│   ├── register.go       // Command to add a new GitOps application
│   ├── start.go          // Command to start the main controller loop
│   └── status.go         // Command to display the status of registered applications
├── internal/             // Private packages for core logic
│   ├── app/
│   │   ├── app.go        // Defines the `Application` struct and handles persistence of app definitions
│   ├── git/
│   │   ├── git.go        // Encapsulates Git operations (clone, pull, commit hash retrieval)
│   ├── k8s/
│   │   ├── k8s.go        // Handles Kubernetes API interactions (applying manifests)
│   └── controller/
│       ├── controller.go // The core orchestration logic for GitOps reconciliation
│       └── types.go      // (Currently empty, but reserved for shared types)
└── configs/              // Directory to store persistent application definitions (e.g., applications.json)
```

This structure promotes modularity and separation of concerns, making the codebase easier to understand, maintain, and extend. The `internal` directory ensures that its packages are not directly imported by external projects, keeping the core logic private to gitopsctl.

## 🔍 Codebase Breakdown

### 1. `main.go`

**Role**: The absolute entry point of the gitopsctl application.

**Logic**: Its sole responsibility is to call `cmd.Execute()`, which initializes and runs the Cobra CLI framework, delegating control to the defined subcommands (`register`, `start`, `status`).

---

### 2. `cmd/root.go`

**Role**: Sets up the root command for the gitopsctl CLI and defines global configurations that apply to all subcommands.

**Key Components**:

- `rootCmd`: A `*cobra.Command` instance representing the base gitopsctl command.
- `PersistentPreRunE`: A Cobra hook that runs before any subcommand. Here, it initializes the zap logger.

**Logic**:

- When gitopsctl is run, this file sets up the CLI parsing infrastructure and the global logging mechanism.

---

### 3. `cmd/register.go`

**Role**: Implements the `gitopsctl register` subcommand, allowing users to define a new application to be managed.

**Key Components**:

- `registerCmd`: A `*cobra.Command` defining the register subcommand.
- **Flags**: Uses Cobra's flag system (`--name`, `--repo`, `--path`, `--kubeconfig`, `--interval`) to capture application details from the command line.

**Logic**:

- Parses command-line arguments to extract application details.
- Loads existing application definitions from `configs/applications.json` using `app.LoadApplications()`.
- Creates a new `app.Application` struct with the provided details.
- Saves the updated collection back to `configs/applications.json` using `app.SaveApplications()`.

---

### 4. `cmd/start.go`

**Role**: Implements the `gitopsctl start` subcommand, which initiates the main GitOps reconciliation loop.

**Key Components**:

- `startCmd`: A `*cobra.Command` defining the start subcommand.

**Logic**:

- Loads all registered applications from `configs/applications.json`.
- Instantiates a `controller.Controller` with the loaded applications and the logger.
- Sets up signal handling for graceful shutdown.
- Calls `ctrl.Start()` to begin the reconciliation goroutines for each application.

---

### 5. `cmd/status.go`

**Role**: Implements the `gitopsctl status` subcommand, providing a tabular overview of all registered applications.

**Key Components**:

- `statusCmd`: A `*cobra.Command` defining the status subcommand.

**Logic**:

- Loads existing application definitions from `configs/applications.json`.
- Uses `text/tabwriter` to format and print a neat table showing each application's details.

---

### 6. `internal/app/app.go`

**Role**: Defines the data structure for a GitOps application and provides methods for persisting and loading these definitions to/from a JSON file.

**Key Components**:

- `Application` struct: Contains fields like `Name`, `RepoURL`, `Path`, `KubeconfigPath`, `Interval`, etc.
- `Applications` struct: A wrapper around `map[string]*Application` with a `sync.RWMutex` for thread-safe access.

**Important Functions**:

- `LoadApplications(filePath string)`: Reads the JSON file and populates the `Applications` map.
- `SaveApplications(apps *Applications, filePath string)`: Marshals the `Applications` map into JSON and writes it to the file.

---

### 7. `internal/git/git.go`

**Role**: Provides a clean interface for performing Git operations.

**Dependencies**: `github.com/go-git/go-git/v5`

**Key Functions**:

- `CloneOrPull(logger *zap.Logger, repoURL, branch, targetDir string)`: Clones or pulls the latest changes from a Git repository.
- `GetLatestCommitHash(logger *zap.Logger, repoPath string)`: Retrieves the current commit hash of a local Git repository.

---

### 8. `internal/k8s/k8s.go`

**Role**: Handles all interactions with the Kubernetes API.

**Dependencies**: `k8s.io/client-go`, `k8s.io/apimachinery`, `sigs.k8s.io/yaml`

**Key Components**:

- `ClientSet` struct: Holds the `dynamic.Interface` and `meta.RESTMapper`.

**Important Functions**:

- `NewClientSet(logger *zap.Logger, kubeconfigPath string)`: Initializes a Kubernetes client.
- `ApplyManifests(ctx context.Context, manifestsDir string)`: Applies Kubernetes manifests to the cluster.

---

### 9. `internal/controller/controller.go`

**Role**: The core orchestration logic for GitOps reconciliation.

**Key Components**:

- `Controller` struct: Holds the logger, a reference to the `app.Applications` collection, and a `context.Context` for graceful shutdown.

**Important Functions**:

- `Start(appConfigFile string)`: Starts the reconciliation loops for all registered applications.
- `Stop()`: Signals all running reconciliation loops to stop.
- `reconcileApp(application *app.Application, appConfigFile string)`: Runs the reconciliation loop for a single application.

---

## 🔁 Core Logic Flow

1. **Register an application**: `gitopsctl register ...`

   - Adds/updates the application definition in `applications.json`.

2. **Start the controller**: `gitopsctl start`

   - Launches a dedicated Go goroutine for each application's reconciliation loop.

3. **Reconciliation Loop**:

   - Polls Git for changes at the specified interval.
   - Detects changes and applies Kubernetes manifests if needed.
   - Updates the application's status and persists it to `applications.json`.

4. **Graceful Shutdown**:
   - Stops all reconciliation loops and cleans up resources.

---

## 🔒 Concurrency and Error Handling

- **Concurrency**: Each application's reconciliation loop runs in its own goroutine.
- **Mutex for App State**: A `sync.RWMutex` protects the shared in-memory collection of application definitions.
- **Graceful Shutdown**: Uses `context.Context` and signal handling for clean termination.
- **Basic Error Handling**: Logs errors and updates the application's status and message.

This structured approach forms a robust foundation for building out the full gitopsctl vision.
