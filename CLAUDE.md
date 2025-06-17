# GoReleaser + GitHub Actions Best Practices

This document captures lessons learned from implementing automated releases with GoReleaser and GitHub Actions, particularly for CGO-enabled Go projects.

## Quick Setup for Standard Go Projects

### 1. GitHub Actions Workflow (`.github/workflows/release.yml`)

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          distribution: goreleaser
          version: '~> v1'  # Use v1 for compatibility
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### 2. Basic GoReleaser Config (`.goreleaser.yml`)

```yaml
project_name: your-project-name

before:
  hooks:
    - go mod tidy
    - go mod download

builds:
  - binary: your-binary-name
    env:
      - CGO_ENABLED=0  # Start with CGO disabled
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64
    flags:
      - -trimpath
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE

checksum:
  name_template: 'checksums.txt'

changelog:
  sort: asc
  use: github
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^ci:'
      - Merge pull request
      - Merge branch
```

### 3. Version Support in main.go

```go
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version information",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("your-app %s\n", version)
        fmt.Printf("Commit: %s\n", commit)
        fmt.Printf("Date: %s\n", date)
    },
}
```

## CGO Projects: Major Considerations

### The Reality Check
**CGO + Cross-compilation is challenging.** Consider these strategies in order of preference:

1. **Avoid CGO if possible** - Use pure Go alternatives
2. **Native builds only** - Build on each target platform
3. **Single platform** - Provide binaries for Linux only
4. **Go install fallback** - Let users build locally

### CGO-Specific Configuration

When CGO is required (e.g., SQLite, C libraries):

```yaml
# Option 1: Linux-only (Recommended for CGO)
builds:
  - binary: your-binary-name
    env:
      - CGO_ENABLED=1
    goos:
      - linux
    goarch:
      - amd64
    flags:
      - -trimpath
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

# Option 2: Multiple platforms (Higher complexity)
builds:
  - binary: your-binary-name
    env:
      - CGO_ENABLED=1
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      # Remove problematic combinations
      - goos: linux
        goarch: arm64
      - goos: windows
        goarch: arm64
      - goos: darwin
        goarch: amd64
```

### Common CGO Cross-Compilation Issues

1. **Linux ARM64**: Assembly code compatibility issues
   ```
   Error: no such instruction: `stp x29,x30,[sp,'
   ```

2. **Windows**: Missing cross-compilation toolchain
   ```
   gcc: error: unrecognized command-line option '-mthreads'
   ```

3. **macOS**: Cross-compilation from Linux
   ```
   clang: error: unsupported option '-arch' for target 'x86_64-pc-linux-gnu'
   ```

### CGO Solutions

#### Strategy 1: Linux-Only Releases (Recommended)
- Provide Linux AMD64 binaries via GoReleaser
- Document `go install` for other platforms
- Most reliable and fast CI

#### Strategy 2: Platform-Specific Runners
```yaml
strategy:
  matrix:
    include:
      - os: ubuntu-latest
        goos: linux
        goarch: amd64
      - os: macos-latest
        goos: darwin
        goarch: arm64
      - os: windows-latest
        goos: windows
        goarch: amd64
```

#### Strategy 3: Docker-Based Cross-Compilation
- Use containers with proper cross-compilation toolchains
- More complex but supports more platforms
- Requires custom Docker images

## Best Practices

### Repository Structure
```
.github/
  workflows/
    release.yml
.goreleaser.yml
.gitignore        # Include dist/
main.go          # With version variables
README.md        # Installation instructions
```

### Version Strategy
- Use semantic versioning: `v1.0.0`, `v1.1.0`, etc.
- GoReleaser triggers on `v*` tags
- Version injection via ldflags

### README Template for CGO Projects
```markdown
## Installation

### Download Pre-built Binaries
Download from [releases](https://github.com/user/repo/releases):
- Linux (amd64)

### Install via Go (All Platforms)
```bash
go install github.com/user/repo@latest
```

### Build from Source
```bash
git clone https://github.com/user/repo.git
cd repo
go build
```
```

### .gitignore Additions
```
# GoReleaser artifacts
dist/

# Binaries
your-binary-name
your-binary-name.exe
```

## Testing Locally

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Validate configuration
goreleaser check

# Test build without release
goreleaser build --snapshot --clean

# Test full release process (without publishing)
goreleaser release --snapshot --clean
```

## Troubleshooting

### GoReleaser Version Issues
- Stick with v1 configuration format for compatibility
- Use `version: '~> v1'` in GitHub Actions
- v2 format requires newer GoReleaser versions

### CGO Debugging
```bash
# Test local CGO build
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build

# Check for required libraries
ldd your-binary

# Test on target platform
file your-binary
```

### Release Process
1. Commit all changes
2. Tag version: `git tag v1.0.0`
3. Push tag: `git push origin v1.0.0`
4. Monitor GitHub Actions
5. Verify release artifacts

## Summary

- **Pure Go**: Use full cross-compilation matrix
- **CGO projects**: Start with Linux-only, expand carefully
- **Always provide**: `go install` as fallback option
- **Test locally**: Use `goreleaser --snapshot` before tagging
- **Document clearly**: Which platforms have pre-built binaries

The key is balancing automation complexity with reliability. It's better to have working Linux binaries than broken cross-compilation for all platforms.