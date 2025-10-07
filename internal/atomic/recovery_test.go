package atomic

import (
    "os"
    "path/filepath"
    "testing"
    "time"
)

func TestRecoveryPurgesOldCommitted(t *testing.T) {
    root := t.TempDir()
    txn, _ := Begin(root, nil)
    w, _ := txn.StageCreate("old.txt")
    w.Write([]byte("done"))
    w.Close()
    if err := txn.Commit(); err != nil { t.Fatalf("commit: %v", err) }
    // artificially age journal
    txn.j.StartedAt = time.Now().Add(-48 * time.Hour)
    _ = txn.j.persist()
    res, err := Recover(root, 24*time.Hour)
    if err != nil { t.Fatalf("recover: %v", err) }
    if res.Purged == 0 { t.Fatalf("expected purge of old committed transaction") }
    if _, err := os.Stat(filepath.Join(root, ".txn", txn.j.ID)); !os.IsNotExist(err) { t.Fatalf("txn dir should be removed") }
}