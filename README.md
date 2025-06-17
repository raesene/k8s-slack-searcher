# Kubernetes Slack Searcher

A command-line tool to index and search through Slack workspace archives. This tool was built specifically for searching the Kubernetes Slack workspace archives but can be used with any Slack export data.

## Features

- **Fast Indexing**: Processes Slack export JSON files and creates SQLite databases with full-text search
- **Channel-based Databases**: Each channel gets its own searchable database
- **User Context**: Correlates messages with user information (real names, usernames)
- **Full-text Search**: SQLite FTS4-powered search with snippet highlighting
- **Progress Tracking**: Real-time progress during indexing operations
- **Human Messages Only**: Filters out bot messages and system notifications

## Installation

### Prerequisites

- SQLite3 (included with most systems)

### Download Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/raesene/k8s-slack-searcher/releases).

Available platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64) 
- Windows (amd64)

### Install via Go

```bash
go install github.com/raesene/k8s-slack-searcher@latest
```

### Build from Source

```bash
git clone https://github.com/raesene/k8s-slack-searcher.git
cd k8s-slack-searcher
go build -o k8s-slack-searcher
```

### Docker

```bash
docker pull ghcr.io/raesene/k8s-slack-searcher:latest
```

## Usage

### 1. Prepare Your Data

You'll need a Slack workspace export containing:
- `users.json` - User information
- `channels.json` - Channel metadata  
- Channel directories with daily JSON message files (e.g., `sig-auth/2019-01-15.json`)

Place these in a `source-data` directory:
```
source-data/
├── users.json
├── channels.json
├── sig-auth/
│   ├── 2019-01-15.json
│   ├── 2019-01-16.json
│   └── ...
└── other-channel/
    └── ...
```

### 2. Index a Channel

```bash
# Index the sig-auth channel
./k8s-slack-searcher ingest sig-auth

# Index with custom source directory
./k8s-slack-searcher ingest sig-auth --source /path/to/slack-export
```

This creates a database file at `databases/sig-auth.db`.

### 3. Search Messages

```bash
# Basic search
./k8s-slack-searcher search "authentication" --database sig-auth

# Search with more results
./k8s-slack-searcher search "RBAC OR authorization" --database sig-auth --limit 20

# Show database statistics
./k8s-slack-searcher search "certificates" --database sig-auth --stats
```

### 4. List Available Databases

```bash
./k8s-slack-searcher list
```

## Search Syntax

The search uses SQLite FTS4 syntax:

- **Simple terms**: `authentication`
- **Phrases**: `"pod security policy"`
- **Boolean operators**: `RBAC AND certificates`, `auth OR authentication`
- **Exclusion**: `security NOT policy`
- **Prefix matching**: `cert*` (matches certificate, certificates, etc.)

## Commands

### `ingest`

Index a Slack channel directory and create a searchable database.

```bash
k8s-slack-searcher ingest <channel-directory> [flags]

Flags:
  -s, --source string   Source data directory (default "source-data")
  -h, --help           Help for ingest
```

### `search`

Search messages in a channel database.

```bash
k8s-slack-searcher search <query> [flags]

Flags:
  -d, --database string   Database name (channel name) to search (required)
  -l, --limit int        Maximum number of results (default 10)
      --stats           Show database statistics
  -h, --help            Help for search
```

### `list`

List all available databases.

```bash
k8s-slack-searcher list
```

## Example Output

```bash
$ ./k8s-slack-searcher search "authentication" --database sig-auth --limit 2

Searching for: authentication
Database: sig-auth
Limit: 2

Found 2 result(s):

--- Result 1 ---
User: Eric Tune (erictune)
Date: 2016-02-23 17:44:22
File: 2016-02-23.json
Message: which allows <mark>authentication</mark> with open-id connect.

--- Result 2 ---
User: Rudi C (therc)
Date: 2016-03-31 16:31:50
File: 2016-03-31.json
Message: I'm interested in anything involving the 1.3 IAM work, per-pod cloud credentials through a 169.254.169.254 proxy or two-factor <mark>authentication</mark>
```

## Performance

- **Indexing**: ~2,400 files with 38K messages in under 30 seconds
- **Search**: Sub-second response times for typical queries
- **Storage**: ~50MB database for 38K messages with full-text index

## Data Privacy

- All data remains local - no external services are used
- Only human messages are indexed (bot messages are filtered out)
- Original Slack export files are not modified

## Troubleshooting

### "no such module: fts5" Error

The application uses SQLite FTS4 for compatibility. If you see FTS5 errors, the build should automatically fall back to FTS4.

### Database Not Found

Ensure you've run the `ingest` command for the channel before searching:
```bash
./k8s-slack-searcher list  # Check available databases
./k8s-slack-searcher ingest <channel-name>  # Create database if missing
```

### No Results Found

- Check that the channel contains human messages (not just bot messages)
- Try simpler search terms
- Verify the database was created successfully with `--stats`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable  
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built for searching the Kubernetes Slack workspace archives to help community members find historical discussions and decisions.