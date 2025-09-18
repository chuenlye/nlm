# nlm - NotebookLM CLI Tool 📚

`nlm` is a command-line interface for Google's NotebookLM, allowing you to manage notebooks, sources, and audio overviews from your terminal.

🔊 Listen to an Audio Overview of this tool here: [https://notebooklm.google.com/notebook/437c839c-5a24-455b-b8da-d35ba8931811/audio](https://notebooklm.google.com/notebook/437c839c-5a24-455b-b8da-d35ba8931811/audio).

## Installation 🚀

```bash
go install github.com/tmc/nlm/cmd/nlm@latest
```

### Usage 

```shell
Usage: nlm <command> [arguments]

Notebook Commands:
  list, ls          List all notebooks
  create <title>    Create a new notebook
  rm <id>           Delete a notebook
  analytics <id>    Show notebook analytics

Source Commands:
  sources <id>      List sources in notebook
  add <id> <input>  Add source to notebook
  rm-source <id> <source-id>  Remove source
  rename-source <source-id> <new-name>  Rename source
  check-source <notebook-id> <source-id>  Check source freshness
  refresh-source <notebook-id> <source-id>  Refresh source content
  batch-sync <notebook-id> [--google-docs-only] [--force]  Batch sync sources

Note Commands:
  notes <id>        List notes in notebook
  new-note <id> <title>  Create new note
  edit-note <id> <note-id> <content>  Edit note
  rm-note <note-id>  Remove note

Audio Commands:
  audio-create <id> <instructions>  Create audio overview
  audio-get <id>    Get audio overview
  audio-rm <id>     Delete audio overview
  audio-share <id>  Share audio overview

Generation Commands:
  generate-guide <id>  Generate notebook guide
  generate-outline <id>  Generate content outline
  generate-section <id>  Generate new section

Other Commands:
  auth              Setup authentication
```

<details>
<summary>📦 Installing Go (if needed)</summary>

### Option 1: Using Package Managers

**macOS (using Homebrew):**
```bash
brew install go
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt update
sudo apt install golang
```

**Linux (Fedora):**
```bash
sudo dnf install golang
```

### Option 2: Direct Download

1. Visit the [Go Downloads page](https://go.dev/dl/)
2. Download the appropriate version for your OS
3. Follow the installation instructions:

**macOS:**
- Download the .pkg file
- Double-click to install
- Follow the installer prompts

**Linux:**
```bash
# Example for Linux AMD64 (adjust version as needed)
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
```

### Post-Installation Setup

Add Go to your PATH by adding these lines to your `~/.bashrc`, `~/.zshrc`, or equivalent:
```bash
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$(go env GOPATH)/bin
```

Verify installation:
```bash
go version
```
</details>

## Authentication 🔑

First, authenticate with your Google account:

```bash
nlm auth
```

This will launch Chrome to authenticate with your Google account. The authentication tokens will be saved in `.env` file.

## Usage 💻

### Notebook Operations

```bash
# List all notebooks
nlm list

# Create a new notebook
nlm create "My Research Notes"

# Delete a notebook
nlm rm <notebook-id>

# Get notebook analytics
nlm analytics <notebook-id>
```

### Source Management

```bash
# List sources in a notebook
nlm sources <notebook-id>

# Add a source from URL
nlm add <notebook-id> https://example.com/article

# Add a source from file
nlm add <notebook-id> document.pdf

# Add source from stdin
echo "Some text" | nlm add <notebook-id> -

# Rename a source
nlm rename-source <source-id> "New Title"

# Remove a source
nlm rm-source <notebook-id> <source-id>
```

### Source Synchronization

For Google Docs sources that may become out of sync:

```bash
# Check if a single source needs synchronization
nlm check-source <notebook-id> <source-id>

# Manually refresh a single source
nlm refresh-source <notebook-id> <source-id>

# Batch sync all sources in a notebook
nlm batch-sync <notebook-id>

# Batch sync only Google Docs sources
nlm batch-sync <notebook-id> --google-docs-only

# Force sync all sources (skip freshness checks)
nlm batch-sync <notebook-id> --force

# Combine options: force sync only Google Docs
nlm batch-sync <notebook-id> --google-docs-only --force
```

#### Batch Sync Features

The `batch-sync` command provides comprehensive synchronization management:

- **Automatic Detection**: Identifies Google Docs sources that need synchronization
- **Progress Reporting**: Shows detailed results for each source processed
- **Filtering Options**:
  - `--google-docs-only`: Only process Google Docs sources
  - `--force`: Skip freshness checks and sync all sources
- **Status Tracking**: Reports sources as SYNCED, FAILED, SKIPPED, or NOT_NEEDED
- **Error Handling**: Provides recommendations for failed syncs

**Example Output:**
```
📊 Batch Sync Summary
═════════════════════
Total Sources: 5
✅ Synced: 3
❌ Failed: 0
⏭️  Skipped: 2

📋 Detailed Results
═══════════════════
STATUS    SOURCE              MESSAGE
──────    ──────              ───────
✅ SYNCED Research Paper.docx  Sync request sent successfully
✅ SYNCED Meeting Notes.docx   Sync request sent successfully
✅ SYNCED Project Plan.docx    Sync request sent successfully
⏭️ SKIPPED image.png          Not a Google Docs source
✓ NOT_NEEDED Data.xlsx        Source already synchronized
```

### Note Operations

```bash
# List notes in a notebook
nlm notes <notebook-id>

# Create a new note
nlm new-note <notebook-id> "Note Title"

# Edit a note
nlm edit-note <notebook-id> <note-id> "New content"

# Remove a note
nlm rm-note <note-id>
```

### Audio Overview

```bash
# Create an audio overview
nlm audio-create <notebook-id> "speak in a professional tone"

# Get audio overview status/content
nlm audio-get <notebook-id>

# Share audio overview (private)
nlm audio-share <notebook-id>

# Share audio overview (public)
nlm audio-share <notebook-id> --public
```

## Examples 📋

### Basic Workflow

Create a notebook and add some content:
```bash
# Create a new notebook
notebook_id=$(nlm create "Research Notes")

# Add some sources
nlm add $notebook_id https://example.com/research-paper
nlm add $notebook_id research-data.pdf

# Create an audio overview
nlm audio-create $notebook_id "summarize in a professional tone"

# Check the audio overview
nlm audio-get $notebook_id
```

### Daily Batch Sync Automation

Automatically sync all Google Docs sources in your notebooks:
```bash
# Get all notebook IDs
notebook_ids=$(nlm list | grep -o '^[a-f0-9-]*')

# Batch sync Google Docs sources for each notebook
for notebook_id in $notebook_ids; do
    echo "Syncing notebook: $notebook_id"
    nlm batch-sync $notebook_id --google-docs-only
done
```

### Source Management Workflow

```bash
# Create a new notebook for a project
project_id=$(nlm create "Project Documentation")

# Add various sources
nlm add $project_id project-spec.docx
nlm add $project_id https://github.com/company/project
nlm add $project_id meeting-notes.pdf

# List all sources to get IDs
nlm sources $project_id

# Check sync status of a Google Docs source
nlm check-source $project_id <source-id>

# If needed, manually refresh the source
nlm refresh-source $project_id <source-id>

# Or batch sync all sources at once
nlm batch-sync $project_id --google-docs-only

# Create notes based on the synced content
nlm new-note $project_id "Key Findings"

# Generate audio overview
nlm audio-create $project_id "create a technical summary for developers"
```

## Advanced Usage 🔧

### Debug Mode

Add `-debug` flag to see detailed API interactions:

```bash
nlm -debug list
```

### Environment Variables

- `NLM_AUTH_TOKEN`: Authentication token (stored in ~/.nlm/env)
- `NLM_COOKIES`: Authentication cookies (stored in ~/.nlm/env)
- `NLM_BROWSER_PROFILE`: Chrome profile to use for authentication (default: "Default")

These are typically managed by the `auth` command, but can be manually configured if needed.

### Batch Sync Best Practices

For optimal synchronization management:

**Daily Automation:**
```bash
# Create a daily sync script
#!/bin/bash
echo "Starting daily NotebookLM sync..."
for notebook_id in $(nlm list | grep -o '^[a-f0-9-]*'); do
    echo "Processing notebook: $notebook_id"
    nlm batch-sync $notebook_id --google-docs-only
done
echo "Sync completed!"
```

**Troubleshooting Sync Issues:**
```bash
# Check individual source status
nlm check-source <notebook-id> <source-id>

# Force refresh problematic sources
nlm refresh-source <notebook-id> <source-id>

# Use debug mode for detailed diagnostics
nlm -debug batch-sync <notebook-id> --force
```

**Performance Tips:**
- Use `--google-docs-only` to focus on sources that can become stale
- Run `batch-sync` without `--force` first to avoid unnecessary API calls
- Use `--force` only when you need to ensure all sources are refreshed
- Monitor sync results and address failed sources individually

**Integration with CI/CD:**
```bash
# Add to your build pipeline
nlm batch-sync $NOTEBOOK_ID --google-docs-only --force
if [ $? -eq 0 ]; then
    echo "Documentation sources synchronized successfully"
else
    echo "Warning: Some sources failed to sync"
fi
```

## Contributing 🤝

Contributions are welcome! Please feel free to submit a Pull Request.

## License 📄

MIT License - see [LICENSE](LICENSE) for details.
