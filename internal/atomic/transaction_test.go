package atomic

import (
    "os"
    "path/filepath"
    "testing"
    "time"
)

func TestTransactionCommit(t *testing.T) {
    root := t.TempDir()
    txn, err := Begin(root, map[string]any{"test":"commit"})
    if err != nil { t.Fatalf("begin: %v", err) }
    w, err := txn.StageCreate("data/file.txt")
    if err != nil { t.Fatalf("stage: %v", err) }
    if _, err := w.Write([]byte("hello")); err != nil { t.Fatalf("write: %v", err) }
    if err := w.Close(); err != nil { t.Fatalf("close: %v", err) }
    if err := txn.Commit(); err != nil { t.Fatalf("commit: %v", err) }
    if _, err := os.Stat(filepath.Join(root, "data", "file.txt")); err != nil { t.Fatalf("expected promoted file: %v", err) }
}

func TestTransactionRollback(t *testing.T) {
    root := t.TempDir()
    txn, err := Begin(root, map[string]any{"test":"rollback"})
    if err != nil { t.Fatalf("begin: %v", err) }
    w, err := txn.StageCreate("data/file.txt")
    if err != nil { t.Fatalf("stage: %v", err) }
    if _, err := w.Write([]byte("temp")); err != nil { t.Fatalf("write: %v", err) }
    _ = w.Close()
    if err := txn.Rollback(); err != nil { t.Fatalf("rollback: %v", err) }
    if _, err := os.Stat(filepath.Join(root, "data", "file.txt")); !os.IsNotExist(err) { t.Fatalf("file should not exist after rollback") }
}

func TestRecoveryRollsBackOrCommitsPending(t *testing.T) {
    root := t.TempDir()
    txn, _ := Begin(root, nil)
    w, _ := txn.StageCreate("X.txt")
    w.Write([]byte("data"))
    w.Close() // no commit
    res, err := Recover(root, 0)
    if err != nil { t.Fatalf("recover: %v", err) }
    if res.RolledBack == 0 && res.ResumedCommits == 0 { t.Fatalf("expected rollback or resume") }
    _, statErr := os.Stat(filepath.Join(root, "X.txt"))
    if res.ResumedCommits > 0 && statErr != nil { t.Fatalf("expected file promoted on resumed commit: %v", statErr) }
    if res.RolledBack > 0 && !os.IsNotExist(statErr) { t.Fatalf("expected file absent after rollback") }
}

func TestStageDeleteCommit(t *testing.T) {
    root := t.TempDir()
    // create file to delete
    targetPath := filepath.Join(root, "victim.txt")
    if err := os.WriteFile(targetPath, []byte("data"), 0o644); err != nil { t.Fatalf("prep: %v", err) }
    txn, err := Begin(root, nil); if err != nil { t.Fatalf("begin: %v", err) }
    if err := txn.StageDelete("victim.txt"); err != nil { t.Fatalf("stage delete: %v", err) }
    if _, err := os.Stat(targetPath); !os.IsNotExist(err) { t.Fatalf("file should be moved to trash before commit") }
    if err := txn.Commit(); err != nil { t.Fatalf("commit: %v", err) }
    if _, err := os.Stat(targetPath); !os.IsNotExist(err) { t.Fatalf("file should remain deleted after commit") }
}

func TestStageDeleteRollback(t *testing.T) {
    root := t.TempDir()
    targetPath := filepath.Join(root, "victim.txt")
    os.WriteFile(targetPath, []byte("data"), 0o644)
    txn, _ := Begin(root, nil)
    if err := txn.StageDelete("victim.txt"); err != nil { t.Fatalf("stage delete: %v", err) }
    if err := txn.Rollback(); err != nil { t.Fatalf("rollback: %v", err) }
    if _, err := os.Stat(targetPath); err != nil { t.Fatalf("file should be restored after rollback: %v", err) }
}

func TestStageReplaceCommit(t *testing.T) {
    root := t.TempDir()
    path := filepath.Join(root, "file.txt")
    os.WriteFile(path, []byte("old"), 0o644)
    txn, _ := Begin(root, nil)
    w, err := txn.StageReplace("file.txt"); if err != nil { t.Fatalf("stage replace: %v", err) }
    w.Write([]byte("new"))
    w.Close()
    if err := txn.Commit(); err != nil { t.Fatalf("commit: %v", err) }
    data, _ := os.ReadFile(path)
    if string(data) != "new" { t.Fatalf("expected replaced content, got %s", string(data)) }
}

func TestStageReplaceRollback(t *testing.T) {
    root := t.TempDir()
    path := filepath.Join(root, "file.txt")
    os.WriteFile(path, []byte("old"), 0o644)
    txn, _ := Begin(root, nil)
    w, _ := txn.StageReplace("file.txt")
    w.Write([]byte("new"))
    w.Close()
    if err := txn.Rollback(); err != nil { t.Fatalf("rollback: %v", err) }
    data, _ := os.ReadFile(path)
    if string(data) != "old" { t.Fatalf("expected original content, got %s", string(data)) }
}

func TestDeleteNonexistent(t *testing.T) {
    root := t.TempDir()
    txn, _ := Begin(root, nil)
    if err := txn.StageDelete("missing.txt"); err != nil { t.Fatalf("stage delete missing: %v", err) }
    if err := txn.Commit(); err != nil { t.Fatalf("commit: %v", err) }
}

func TestChecksumRecorded(t *testing.T) {
    root := t.TempDir()
    txn, _ := Begin(root, nil)
    w, _ := txn.StageCreate("a.bin")
    w.Write([]byte("abc"))
    w.Close()
    if len(txn.j.Entries) != 1 { t.Fatalf("expected 1 entry, got %d", len(txn.j.Entries)) }
    if txn.j.Entries[0].Checksum == "" { t.Fatalf("checksum not recorded") }
}

func TestIdempotentRollback(t *testing.T) {
    root := t.TempDir()
    txn, _ := Begin(root, nil)
    w, _ := txn.StageCreate("b.txt")
    w.Write([]byte("data"))
    w.Close()
    _ = txn.Rollback()
    // second rollback should be no-op
    if err := txn.Rollback(); err != nil { t.Fatalf("second rollback should not error: %v", err) }
}

func TestCommitStateGuard(t *testing.T) {
    root := t.TempDir()
    txn, _ := Begin(root, nil)
    txn.j.State = StateCommitted
    if err := txn.Commit(); err == nil { t.Fatalf("expected error committing already committed txn") }
}

func TestRecoveryResumesCommit(t *testing.T) {
    root := t.TempDir()
    txn, _ := Begin(root, nil)
    w, _ := txn.StageCreate("promote.txt")
    w.Write([]byte("x"))
    w.Close()
    // simulate partially set state to pending (already is) and run recovery
    res, err := Recover(root, time.Hour)
    if err != nil { t.Fatalf("recover: %v", err) }
    if res.ResumedCommits == 0 && res.RolledBack == 0 { t.Fatalf("expected resume or rollback action") }
}