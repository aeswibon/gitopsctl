# Installation

GitOpsCTL is a single binary written in Go. You can install it using several methods.

## Pre-built Binaries

The easiest way to install GitOpsCTL is to download the pre-built binary for your platform from the [Releases](https://github.com/aeswibon/gitopsctl/releases) page.

1. Download the archive for your OS and architecture (e.g., `gitopsctl_Darwin_arm64.tar.gz` for Apple Silicon).
2. Extract the archive.
3. Move the `gitopsctl` binary to a directory in your `PATH` (e.g., `/usr/local/bin`).

## Using Go

If you have Go 1.25 or later installed, you can install GitOpsCTL directly:

```bash
go install aeswibon.com/github/gitopsctl@latest
```

## Using Docker

You can run GitOpsCTL as a Docker container. This is useful for CI/CD environments or if you don't want to install Go locally.

```bash
docker pull ghcr.io/aeswibon/gitopsctl:latest
```

To run it:

```bash
docker run -it -v ~/.kube/config:/root/.kube/config -v $(pwd)/configs:/app/configs ghcr.io/aeswibon/gitopsctl:latest start
```

## Homebrew (macOS/Linux)

*Coming soon!*

## Building from Source

To build GitOpsCTL from source:

1. Clone the repository:
   ```bash
   git clone https://github.com/aeswibon/gitopsctl.git
   cd gitopsctl
   ```
2. Build the binary:
   ```bash
   go build -o gitopsctl main.go
   ```
3. (Optional) Run tests:
   ```bash
   go test ./...
   ```
