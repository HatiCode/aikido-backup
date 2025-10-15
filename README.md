# aikido-backup

A Go application that monitors directories for changes and creates incremental backups in 5MB chunks, with full restore capability.

## Features

- Recursive directory monitoring with change detection
- Incremental backups in 5MB chunks
- SHA256-based change detection
- Full directory structure restoration
- Preserves file permissions and modification times
- Handles file deletions across backup runs

## Building

```bash
# Build the binary
make build

# Or manually
go build -o app
```

## Usage

### Watch Mode

Monitor a directory and automatically backup changes:

```bash
./app --watch <path> --backup <path> --refresh <seconds>
```

**Arguments:**
- `--watch`: Path to the directory to monitor
- `--backup`: Path where backup chunks will be stored
- `--refresh`: Scan interval in seconds (default: 60)

**Example:**
```bash
./app --watch /var/data --backup /var/backups --refresh 60
```

### Restore Mode

Restore files from backup chunks:

```bash
./app --restore <path> --backup <path>
```

**Arguments:**
- `--restore`: Path where files will be restored
- `--backup`: Path containing the backup chunks

**Example:**
```bash
./app --restore /var/restored --backup /var/backups
```

## How It Works

**Watch Mode:**
1. Recursively scans the watched directory every N seconds
2. Detects new, modified, and deleted files using SHA256 hashing
3. Collects changes and backs them up in 5MB chunks
4. Chunks are stored as `chunk_<timestamp>_<number>.dat` files

**Restore Mode:**
1. Reads all chunk files from the backup directory
2. Processes chunks in chronological order
3. Rebuilds the complete directory structure
4. Restores files with original permissions and timestamps
5. Handles deletions (files deleted in later backups won't be restored)

## Testing

```bash
# Run all tests
make test

# Or manually
go test -v ./...
```

## Development

```bash
# Format code
make fmt

# Run static analysis
make vet

# Clean build artifacts
make clean
```

## Project Structure

```
.
├── main.go       # CLI entry point
├── watch.go      # Directory monitoring and change detection
├── backup.go     # Chunking and backup logic
├── restore.go    # Restore functionality
└── Makefile      # Build automation
```
