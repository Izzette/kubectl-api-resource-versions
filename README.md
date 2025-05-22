# kubectl-api-resource-versions

<img align="left" width="128" height="128" alt="Kubernetes Logo" src="https://raw.githubusercontent.com/cncf/artwork/main/projects/kubernetes/icon/color/kubernetes-icon-color.png">

A kubectl plugin that combines functionality from `kubectl api-resources` and `kubectl api-versions` to show API resources with their available group versions.

[![Go Version](https://img.shields.io/badge/go-1.24-blue)]() [![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE) <!--[![Go Report Card](https://goreportcard.com/badge/github.com/Izzette/kubectl-api-resource-versions)](https://goreportcard.com/report/github.com/Izzette/kubectl-api-resource-versions)-->

## Features

- List API resources with their available group versions in a single view
- Filter by API group, namespaced status, and preferred versions
- Multiple output formats: `wide` (default), `name` (kubectl-compatible)
- Sorting by resource name or kind
- Works with any Kubernetes cluster (v1.20+)
- Supports in-cluster and out-of-cluster configurations

## Installation

<!--
### Via Krew

```shell
kubectl krew install api-resource-versions
```
-->

### Install with `go install`

With this method, you can install the plugin directly from the source code using Go's package manager.

```shell
go install github.com/Izzette/kubectl-api-resource-versions/cmd/kubectl-api_resource_versions@latest

# Optionally install the completion script to your $GOPATH/bin
curl -fLo "$(go env GOPATH)/bin/kubectl_complete-api_resource_versions" \
  https://raw.githubusercontent.com/Izzette/kubectl-api-resource-versions/refs/heads/main/kubectl_complete-api_resource_versions
chmod +x "$(go env GOPATH)/bin/kubectl_complete-api_resource_versions"
```

### Manual Installation

1. Clone the repository:
   ```shell
   git clone https://github.com/Izzette/kubectl-api-resource-versions.git
   cd kubectl-api-resource-versions
   ```

2. Build the plugin:
   ```shell
   make build
   ```

3. Move the binary to your PATH (e.g., `/usr/local/bin`):
   ```shell
   sudo cp kubectl-api_resource_versions kubectl_complete-api_resource_versions /usr/local/bin/
   ```

### Standalone usage

You can also use the plugin without installing it globally. Just run the binary directly:

```shell
make build
./kubectl-api_resource_versions
```

## Usage

### Basic Usage

List all API resources with versions (including deprecated/unstable):

```shell
kubectl api-resource-versions
```

### More examples

Filter to non-preferred versions (these may be unstable APIs or deprecated):
```shell
kubectl api-resource-versions --preferred='false'
```

Filter to resources in specific API group:
```shell
kubectl api-resource-versions --api-group='apps'
```

List non-namespaced resources:
```shell
kubectl api-resource-versions --namespaced='false'
```

Show output in kubectl `name` format, and list those resources:
```shell
kubectl api-resource-versions --api-group='apps' --verbs='list,get' --namespaced='false' --output='name' |
  xargs -n1 kubectl get --show-kind
```

### Command Options

In additional to the normal `kubectl` options, the following options are available:

```text
Flags:
      --api-group string               Limit to resources in the specified API group.
      --cached                         Use the cached list of resources if available.
      --categories strings             Limit to resources that belong to the specified categories.
  -h, --help                           help for api-resource-versions
      --namespaced                     If false, non-namespaced resources will be returned, otherwise returning namespaced resources by default. (default true)
      --no-headers                     When using the default or custom-column output format, don't print headers (default print headers).
  -o, --output string                  Output format. One of: (wide, name).
      --preferred                      Filter resources by whether their group version is the preferred one.
      --sort-by string                 If non-empty, sort list of resources using specified field. One of (name, kind).
      --verbs strings                  Limit to resources that support the specified verbs.
```

## Documentation

Full command documentation:

```shell
kubectl api-resource-versions --help
```

Implementation details and API documentation available in the
[project repository](https://github.com/Izzette/kubectl-api-resource-versions)<!-- and
[GoDoc](https://pkg.go.dev/github.com/Izzette/kubectl-api-resource-versions)-->.

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Install pre-requisites:
   - Go 1.24.3 or later
   - Python 3.9 or later (for pre-commit)
   - pre-commit (https://pre-commit.com/)
   - Make (GNU Make recommended: https://www.gnu.org/software/make/)
   - Golangci-lint (https://golangci-lint.run/welcome/install/#local-installation):
     - ```shell
       go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
       ```

2. Set up development environment:
   ```shell
   # Install python virtual environment for pre-commit hooks
   pre-commit install
   ```

3. Update documentation accordingly.
   Use Godoc comments for public types and functions: https://go.dev/blog/godoc

4. Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commit titles.
   This is required for our automated release process, [Release Please](https://github.com/googleapis/release-please).

5. Open a pull request with a clear description of the changes and why they are needed.
   Include the CHANGELOG entry you would like to see in the release, it doesn't need to be perfect: we can refine it together.

### Development

```console
$ make help
all                            Run all the tests, linters and build the project
build                          Build the project (resulting binary is written to kubectl-api_resource_versions)
buildable                      Check if the project is buildable
clean                          Clean the working directory from binaries, coverage
lint                           Run the linters
```

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.

## Acknowledgements

⚠️ **Disclaimer**: This project is a derivative work adapted from the Kubernetes [`kubectl`](https://github.com/kubernetes/kubectl) implementation, but is **not** owned, maintained, or endorsed by The Kubernetes Authors. Kubernetes® is a registered trademark of the Linux Foundation, and the author(s) of kubectl-api-resource-versions have no affiliation with the Kubernetes Authors, the CNCF, or Linux Foundation.
