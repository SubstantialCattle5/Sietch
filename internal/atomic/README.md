# Atomic Transactions

This package provides a lightweight, journaled transaction layer for file operations so commands like add and delete never leave the vault in a half-written state.

At a glance:

- Staging happens under a per-transaction journal directory: `.txn/<id>/`
- New or replacement files go to `new/`; deleted originals go to `trash/`
- Commit promotes staged files with atomic renames and clears trash
- Rollback deletes staged files and restores trash
- Recover scans `.txn/` and completes or rolls back interrupted transactions

## Directory layout

- `.txn/<id>/` — transaction journal root
  - `new/` — staged new files or replacements
  - `trash/` — backups of originals scheduled for deletion/replacement
  - `state` — current state marker (e.g., pending, committing, ...)

## States

- `pending` → work is staged
- `committing` → promotion in progress
- `committed` → finished successfully
- `rolling_back` → rollback in progress
- `rolled_back` → rollback finished
- `failed` → unrecoverable error recorded

## Usage

Example (simplified) flow for a command that creates, replaces, and deletes files atomically:

```go
// import "github.com/substantialcattle5/sietch/internal/atomic"

tx := atomic.NewTransaction(vaultRoot)
defer tx.Cleanup() // removes empty journal dirs after commit/rollback

// Stage operations
if err := tx.StageCreate("manifests/new.json", newBytes); err != nil { /* handle */ }
if err := tx.StageReplace("manifests/old.json", replacementBytes); err != nil { /* handle */ }
if err := tx.StageDelete("chunks/orphaned.bin"); err != nil { /* handle */ }

// Commit atomically
if err := tx.Commit(); err != nil {
    // If commit fails, roll back to restore previous state
    _ = tx.Rollback()
    return err
}
```

## Recovery

If a process crashes mid-transaction, the journal remains. On startup (or via an explicit command), call recover to resolve any incomplete transactions:

```go
// import "github.com/substantialcattle5/sietch/internal/atomic"

err := atomic.Recover(vaultRoot, time.Hour*24) // purge completed journals older than 24h
if err != nil { /* handle */ }
```

Recovery scans `.txn/` and, per journal state, either resumes commit or rolls back to a consistent state. Completed journals older than the retention window are cleaned up.

## Integration notes

- Use `StageCreate` for brand-new files; `StageReplace` to swap existing ones; `StageDelete` to remove files safely
- Prefer small batches per transaction to limit blast radius and improve recoverability
- Logging around commit/rollback helps post-mortem debugging

## Testing

The package includes unit tests that simulate interruptions and verify both commit and rollback paths. See `transaction_test.go` and `recovery_test.go`.
