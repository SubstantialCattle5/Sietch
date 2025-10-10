# Sietch Vault Templates

This directory contains template definitions for creating Sietch vaults with predefined configurations and directory structures.

## Template JSON Structure

Each template is defined in a JSON file with the following structure:

```json
{
  "name": "Template Display Name",
  "description": "Detailed description of what this template is for",
  "version": "1.0.0",
  "author": "Template Author",
  "tags": ["tag1", "tag2", "tag3"],
  "config": {
    "chunking_strategy": "fixed",
    "chunk_size": "8MB",
    "hash_algorithm": "sha256",
    "compression": "gzip",
    "sync_mode": "manual",
    "enable_dedup": true,
    "dedup_strategy": "content",
    "dedup_min_size": "1MB",
    "dedup_max_size": "64MB",
    "dedup_gc_threshold": 500,
    "dedup_index_enabled": true,
    "dedup_cross_file": true
  }
}
```

## Field Descriptions

### Basic Information
- **`name`**: Display name for the template (required)
- **`description`**: Human readable description of the template's purpose (required)
- **`version`**: Template version (required)
- **`author`**: Who created this template (required)
- **`tags`**: Array of tags for categorization and filtering (optional)

### Configuration (`config`)
Defines the default vault configuration that will be applied when using this template:

- **`chunking_strategy`**: How files are chunked (`"fixed"` or `"variable"`)
- **`chunk_size`**: Size of chunks (e.g., `"8MB"`, `"16MB"`)
- **`hash_algorithm`**: Hashing algorithm (`"sha256"`, `"sha512"`)
- **`compression`**: Compression method (`"gzip"`, `"lz4"`, `"none"`)
- **`sync_mode`**: Sync behavior (`"manual"`, `"auto"`)
- **`enable_dedup`**: Enable deduplication (`true`/`false`)
- **`dedup_strategy`**: Deduplication strategy (`"content"`, `"filename"`)
- **`dedup_min_size`**: Minimum file size for deduplication (e.g., `"1MB"`)
- **`dedup_max_size`**: Maximum file size for deduplication (e.g., `"64MB"`)
- **`dedup_gc_threshold`**: Garbage collection threshold (number)
- **`dedup_index_enabled`**: Enable deduplication index (`true`/`false`)
- **`dedup_cross_file`**: Allow cross file deduplication (`true`/`false`)

### Directory Structure (`directories`)
Array of directories to create in the vault. These are created relative to the vault root:

```json
"directories": [
  "photos/raw",        // Creates photos/raw/ directory
  "photos/edited",     // Creates photos/edited/ directory
  "photos/archives",   // Creates photos/archives/ directory
  "data",              // Creates data/ directory
  "metadata"           // Creates metadata/ directory
]
```

### Files (`files`)
Array of files to create in the vault with their content:

```json
"files": [
  {
    "path": "README.md",                    // File path relative to vault root
    "content": "# My Vault\n\nContent...",  // File content
    "mode": "0644"                          // File permissions (octal)
  }
]
```

**File Properties:**
- **`path`**: File path relative to vault root (required)
- **`content`**: File content as string (required)
- **`mode`**: File permissions in octal format (optional, defaults to `"0644"`)

## Why Directories and Files?

### `directories` Array
The `directories` array is crucial because:

1. **Predefined Structure**: Creates a logical folder hierarchy for the specific use case
2. **Consistency**: Ensures all vaults created from the same template have identical structure
3. **Organization**: Helps users organize their content from the start
4. **Best Practices**: Enforces good organizational patterns

**Example for Photo Vault:**
```json
"directories": [
  "photos/raw",      // For original, unprocessed photos
  "photos/edited",   // For processed/edited photos
  "photos/archives", // For old or backup photos
  "data",            // For metadata, databases, etc.
  "metadata"         // For EXIF data, thumbnails, etc.
]
```

### `files` Array
The `files` array is important because:

1. **Documentation**: Creates helpful README files and documentation
2. **Configuration**: Sets up initial configuration files
3. **Examples**: Provides example files or templates
4. **Guidance**: Gives users guidance on how to use the vault

**Example for Photo Vault:**
```json
"files": [
  {
    "path": "README.md",
    "content": "# Photo Vault\n\nThis vault is organized for photo storage...",
    "mode": "0644"
  },
  {
    "path": "photos/README.md",
    "content": "# Photos\n\n- raw/: Original photos\n- edited/: Processed photos\n- archives/: Old photos",
    "mode": "0644"
  }
]
```

## Creating Custom Templates

### Step 1: Create Template File
Create a new `.json` file in this directory:

```bash
# Example: Create a document vault template
touch template/documentVault.json
```

### Step 2: Define Template Structure
Edit the JSON file with your template definition:

```json
{
  "name": "Document Vault",
  "description": "A vault optimized for document storage and organization",
  "version": "1.0.0",
  "author": "Your Name",
  "tags": ["documents", "office", "storage"],
  "config": {
    "chunking_strategy": "fixed",
    "chunk_size": "4MB",
    "hash_algorithm": "sha256",
    "compression": "gzip",
    "sync_mode": "manual",
    "enable_dedup": true,
    "dedup_strategy": "content",
    "dedup_min_size": "1KB",
    "dedup_max_size": "32MB",
    "dedup_gc_threshold": 1000,
    "dedup_index_enabled": true,
    "dedup_cross_file": true
  },
  "directories": [
    "documents/personal",
    "documents/work",
    "documents/archives",
    "templates",
    "scans"
  ],
  "files": [
    {
      "path": "README.md",
      "content": "# Document Vault\n\nOrganize your documents here...",
      "mode": "0644"
    },
    {
      "path": "documents/README.md",
      "content": "# Documents\n\n- personal/: Personal documents\n- work/: Work documents\n- archives/: Old documents",
      "mode": "0644"
    }
  ]
}
```

### Step 3: Test Your Template
Test your template by listing and using it:

```bash
# List available templates
sietch scaffold --list

# Use your template
sietch scaffold --template documentVault --name "My Documents"
```

## Template Management

### User Templates
Templates are automatically copied to `~/.config/sietch/templates/` when first used. You can:

- **Edit templates**: Modify files in `~/.config/sietch/templates/`
- **Add new templates**: Copy new `.json` files to `~/.config/sietch/templates/`
- **Remove templates**: Delete files from `~/.config/sietch/templates/`

### Template Validation
Templates are validated when loaded. Common issues:

- **Missing required fields**: Ensure `name`, `description`, `version`, and `author` are present
- **Invalid JSON**: Check JSON syntax
- **Invalid file paths**: Ensure file paths are relative to vault root
- **Invalid permissions**: Use octal format for file modes (e.g., `"0644"`)

## Best Practices

1. **Use descriptive names and descriptions**
2. **Include relevant tags for categorization**
3. **Create logical directory structures**
4. **Add helpful README files**
5. **Test templates before sharing**
6. **Version your templates**
7. **Document any special requirements**

## Available Templates

### Photos and Media
- **`photoVault`** - Photo storage with fixed chunking, strong dedup, high compression
- **`videoVault`** - Video storage with larger chunks, lighter compression, tuned hashing
- **`audioLibrary`** - Audio/podcast storage with balanced compression and indexing

### Documents and Knowledge
- **`documentsVault`** - Office/PDF documents with higher compression, content-dedup, index enabled
- **`codeVault`** - Code repositories/artifacts with fingerprint dedup, fast hashing, moderate chunks
- **`reporterVault`** - Journalism/reporting with secure defaults, manual sync, metadata emphasis

### Backups and Archives
- **`systemBackup`** - System backups with large chunks, parallel sync, conservative dedup
- **`coldArchive`** - Long-term archival with maximum compression, minimal write amplification

## Template Comparison Matrix

| Template | Chunk Size | Compression | Hash | Dedup Min/Max | GC Threshold | Index | Cross File Dedup | Use Case |
|----------|-----------|-------------|------|---------------|--------------|-------|------------------|----------|
| **photoVault** | 8MB | gzip | sha256 | 1MB / 64MB | 500 | ✓ | ✓ | Photos, RAW images, high-res media |
| **videoVault** | 32MB | lz4 | sha256 | 16MB / 256MB | 100 | ✓ | ✗ | Large video files, movies, recordings |
| **audioLibrary** | 8MB | gzip | sha256 | 2MB / 128MB | 300 | ✓ | ✓ | Music, podcasts, audio collections |
| **documentsVault** | 2MB | gzip | sha256 | 512KB / 32MB | 1500 | ✓ | ✓ | Office docs, PDFs, text files |
| **codeVault** | 4MB | gzip | sha256 | 256KB / 16MB | 2000 | ✓ | ✓ | Source code, repos, build artifacts |
| **reporterVault** | 4MB | gzip | sha256 | 1KB / 32MB | 1000 | ✓ | ✓ | Journalism, sensitive documents |
| **systemBackup** | 16MB | lz4 | sha256 | 8MB / 128MB | 500 | ✓ | ✓ | Full system backups, disaster recovery |
| **coldArchive** | 16MB | gzip | sha512 | 4MB / 256MB | 200 | ✓ | ✓ | Long term storage, archival data |

### Understanding the Settings

**Chunk Size:**
- **Small (2-4MB)**: Better dedup, slower for large files → Documents, Code
- **Medium (8MB)**: Balanced performance → Photos, Audio
- **Large (16-32MB)**: Faster processing, less dedup overhead → Videos, Backups

**Compression:**
- **gzip**: Higher compression ratio, slower → Documents, Photos, Archives
- **lz4**: Faster compression, lower ratio → Videos, Backups
- **none**: No compression overhead → Already compressed formats

**Hash Algorithm:**
- **sha256**: Standard security, fast
- **sha512**: Higher security, slightly slower → Cold Archives

**Deduplication:**
- **Min/Max Size**: Range of chunk sizes eligible for dedup
- **GC Threshold**: Number of unreferenced chunks before cleanup suggestion
- **Index**: Enables faster chunk lookups (recommended: enabled)
- **Cross File**: Dedup across different files (disable for independent files like videos)

**When to Choose Each Template:**

| If you're storing... | Use this template | Why |
|---------------------|-------------------|-----|
| Family photos, vacation pictures | `photoVault` | Optimized for JPEG/PNG with good dedup for similar images |
| Movies, screen recordings | `videoVault` | Large chunks, fast compression, minimal dedup overhead |
| Music library, podcasts | `audioLibrary` | Balanced for MP3/FLAC with moderate dedup |
| Word docs, PDFs, spreadsheets | `documentsVault` | Aggressive compression + dedup for text content |
| Git repos, npm packages | `codeVault` | Fine grained dedup for similar source files |
| Sensitive documents, sources | `reporterVault` | Security focused with manual sync control |
| Full system snapshots | `systemBackup` | Performance optimized for large backups |
| Old files, compliance data | `coldArchive` | Maximum compression for rarely accessed data |

## Examples

See the templates in this directory for reference implementations. Each template serves as an example of how to structure your own custom templates.
