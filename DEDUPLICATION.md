```markdown
# 🔁 Deduplication in Sietch Vault

## Overview

Sietch Vault uses **content-defined chunking** and **cryptographic deduplication** to minimize redundant data storage.  
Instead of storing an entire file whenever it changes, Sietch breaks each file into small, fixed- or variable-sized chunks (default: 4 MB), computes unique hashes for each chunk, and only stores chunks that haven't been stored before.

This makes syncing and storage highly efficient — identical data across files, folders, or even vaults are stored only once, reducing disk space and sync time.

---

## How Deduplication Works

Deduplication in Sietch happens at **chunk level**, not file level.

1. **Chunking**
   - Every file added to a vault is split into smaller data chunks.
   - Chunk size is configurable (`--chunk-size` flag, default 4 MB).
   - Chunk boundaries can be static or computed using a **rolling hash**, which helps detect identical regions even when a file shifts slightly.

2. **Hashing**
   - Each chunk is assigned two hashes:
     - **Content Hash:** A cryptographic hash (e.g., SHA-256) of the unencrypted chunk.
     - **Storage Hash:** A hash of the encrypted chunk (post-encryption).

3. **Deduplication Check**
   - When a new chunk is processed, Sietch checks if the **content hash** already exists in the vault index.
   - If found, it reuses the existing chunk reference instead of storing a duplicate.

4. **Encryption**
   - Chunks are encrypted **after** deduplication check.
   - This ensures that identical plaintext chunks yield identical content hashes (so dedup works), while the **storage hash** maintains uniqueness in encrypted storage.
   - Supported encryption modes:
     - **AES-256-GCM**
     - **ChaCha20-Poly1305**

5. **Manifest Tracking**
   - Each file's metadata (its list of chunks, sizes, hashes) is stored in a **manifest**.
   - The manifest maps logical files to their deduplicated chunks in the storage backend.
   - Manifests are versioned, so rolling back or verifying integrity is easy.

---

## Content Hash vs Storage Hash

| Type | Definition | Purpose | Scope |
|------|-------------|----------|--------|
| **Content Hash** | Hash of the raw (unencrypted) chunk data. | Used to identify identical content for deduplication before encryption. | Computed once during ingestion. |
| **Storage Hash** | Hash of the encrypted chunk data. | Used to verify integrity and locate stored encrypted blobs. | Used internally during sync and retrieval. |

**In short:**
- **Content Hash = Dedup identity**
- **Storage Hash = Integrity + storage mapping**

By separating these two, Sietch maintains **efficient deduplication** while ensuring **secure encryption** and **data integrity**.

---

## Migration Guide — Enabling Dedup on an Existing Vault

If you created a vault before deduplication was enabled, follow this step-by-step process to migrate safely.

### 🧩 Step 1: Backup Existing Vault
Before any operation:
```bash
sietch backup --output ./vault-backup
```
This ensures you can roll back if migration fails.

### ⚙️ Step 2: Enable Deduplication
Enable dedup in the configuration:

```bash
sietch config set dedup.enabled true
sietch config set dedup.chunk-size 4MB
```
Or manually in the config file (~/.sietch/config.json):

```json
{
  "dedup": {
    "enabled": true,
    "chunk_size": "4MB"
  }
}
```

### 🔍 Step 3: Re-index Existing Files
Run the re-indexing tool to compute chunk hashes for existing data:

```bash
sietch dedup reindex
```
This step scans all files, computes content hashes, and builds a deduplication index.

### 🧼 Step 4: Garbage Collect Old Chunks
Once the dedup index is ready:

```bash
sietch dedup gc
```
Removes orphaned or redundant chunks not referenced in any manifest.

### 🧠 Step 5: Optimize Storage Layout
To finalize:

```bash
sietch dedup optimize
```
Reorganizes chunks and manifests for better read/write performance.

**Note:** For very large vaults, perform these steps on a local copy or use the `--dry-run` flag first to estimate changes.

---

## Performance Tuning & Chunk Size Recommendations

Chunk size directly affects both storage efficiency and CPU performance:

| Chunk Size | Use Case | Storage Efficiency | CPU Cost |
|------------|----------|-------------------|----------|
| 1 MB | Rapidly changing files, e.g., source code, logs | High | High |
| 4 MB (default) | Balanced general purpose | Medium | Medium |
| 8–16 MB | Large static files (media, backups) | Lower dedup gain | Low |

**Tips:**
- Smaller chunks → better deduplication but slower processing.
- Larger chunks → faster sync and less overhead but fewer dedup hits.
- For mixed workloads, keep 4 MB or use `--adaptive-chunking` (if available).

---

## Best Practices

- Run `sietch dedup stats` regularly to monitor chunk reuse and storage savings.
- Avoid changing chunk size after initial vault creation — this can break dedup references.
- Use `sietch dedup optimize` monthly to defragment storage.
- Keep your manifests backed up; they're critical for mapping files to chunks.
- Use `--dry-run` with dedup operations before running them in production.

---

## Example Workflow

```bash
# Initialize vault with ChaCha20 encryption
sietch init --name research --key-type chacha20

# Add files
sietch add ./datasets ./vault/data

# Check dedup stats
sietch dedup stats

# Clean up unreferenced chunks
sietch dedup gc

# Optimize layout
sietch dedup optimize
```

Output might look like:

```yaml
Deduplication Statistics
------------------------
Total Chunks: 12,843
Unique Chunks: 9,557
Space Saved: 4.21 GB (32%)
Garbage Collected: 152 chunks
Optimization Complete: OK
```

---

## Future Improvements (Planned)

- **Adaptive Chunking** — variable chunk sizes based on content entropy.
- **Cross-Vault Dedup** — share dedup indices securely across multiple vaults.
- **Dedup Metrics API** — expose storage savings via REST/CLI metrics.
```