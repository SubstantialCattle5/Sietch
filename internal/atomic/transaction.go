package atomic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type State string

const (
	StatePending     State = "pending"
	StateCommitting  State = "committing"
	StateCommitted   State = "committed"
	StateRollingBack State = "rolling_back"
	StateRolledBack  State = "rolled_back"
	StateFailed      State = "failed"
)

type EntryType string

const (
	EntryCreate  EntryType = "create"
	EntryDelete  EntryType = "delete"
	EntryReplace EntryType = "replace"
)

type JournalEntry struct {
	Type               EntryType `json:"type"`
	FinalPath          string    `json:"finalPath"`
	StagedPath         string    `json:"stagedPath,omitempty"`
	OriginalBackupPath string    `json:"originalBackupPath,omitempty"`
	Size               int64     `json:"size,omitempty"`
	Checksum           string    `json:"checksum,omitempty"`
}

type Journal struct {
	Version   int            `json:"version"`
	ID        string         `json:"id"`
	StartedAt time.Time      `json:"startedAt"`
	State     State          `json:"state"`
	Entries   []JournalEntry `json:"entries"`
	Metadata  map[string]any `json:"metadata,omitempty"`

	dir       string
	vaultRoot string
	mu        sync.Mutex
}

type Transaction struct{ j *Journal }

var (
	ErrTxnConflict = errors.New("transaction conflict")
	ErrTxnCorrupt  = errors.New("transaction journal corrupt")
)

func Begin(vaultRoot string, metadata map[string]any) (*Transaction, error) {
	txnRoot := filepath.Join(vaultRoot, ".txn")
	if err := os.MkdirAll(txnRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create txn root: %w", err)
	}
	id := time.Now().UTC().Format("20060102T150405Z") + fmt.Sprintf("-%06d", time.Now().Nanosecond())
	dir := filepath.Join(txnRoot, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create txn dir: %w", err)
	}
	j := &Journal{Version: 1, ID: id, StartedAt: time.Now().UTC(), State: StatePending, Entries: []JournalEntry{}, Metadata: metadata, dir: dir, vaultRoot: vaultRoot}
	if err := j.persist(); err != nil {
		return nil, err
	}
	_ = os.MkdirAll(filepath.Join(dir, "new"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "trash"), 0o755)
	return &Transaction{j: j}, nil
}

func (t *Transaction) StageCreate(finalRelPath string) (io.WriteCloser, error) {
	t.j.mu.Lock()
	defer t.j.mu.Unlock()
	staged := filepath.Join(t.j.dir, "new", filepath.FromSlash(finalRelPath))
	if err := os.MkdirAll(filepath.Dir(staged), 0o755); err != nil {
		return nil, fmt.Errorf("stage create mkdir: %w", err)
	}
	f, err := os.Create(staged)
	if err != nil {
		return nil, fmt.Errorf("stage create open: %w", err)
	}
	h := sha256.New()
	w := &createWriter{multi: io.MultiWriter(f, h), f: f, t: t, staged: staged, rel: filepath.ToSlash(finalRelPath), hsh: h}
	return w, nil
}

type createWriter struct {
	multi  io.Writer
	f      *os.File
	t      *Transaction
	staged string
	rel    string
	hsh    interface{ Sum([]byte) []byte }
}

func (cw *createWriter) Write(b []byte) (int, error) { return cw.multi.Write(b) }
func (cw *createWriter) Close() error {
	if err := cw.f.Close(); err != nil {
		return err
	}
	fi, err := os.Stat(cw.staged)
	if err != nil {
		return err
	}
	sum := cw.hsh.Sum(nil)
	cw.t.j.mu.Lock()
	defer cw.t.j.mu.Unlock()
	cw.t.j.Entries = append(cw.t.j.Entries, JournalEntry{Type: EntryCreate, FinalPath: cw.rel, StagedPath: cw.staged, Size: fi.Size(), Checksum: "sha256:" + hex.EncodeToString(sum)})
	return cw.t.j.persistLocked()
}

func (t *Transaction) StageDelete(finalRelPath string) error {
	t.j.mu.Lock()
	defer t.j.mu.Unlock()
	abs := filepath.Join(t.j.vaultRoot, filepath.FromSlash(finalRelPath))
	if _, err := os.Stat(abs); err != nil {
		if os.IsNotExist(err) {
			t.j.Entries = append(t.j.Entries, JournalEntry{Type: EntryDelete, FinalPath: filepath.ToSlash(finalRelPath)})
			return t.j.persistLocked()
		}
		return fmt.Errorf("stage delete stat: %w", err)
	}
	trash := filepath.Join(t.j.dir, "trash", filepath.FromSlash(finalRelPath))
	if err := os.MkdirAll(filepath.Dir(trash), 0o755); err != nil {
		return fmt.Errorf("stage delete mkdir: %w", err)
	}
	if err := os.Rename(abs, trash); err != nil {
		return fmt.Errorf("stage delete move: %w", err)
	}
	t.j.Entries = append(t.j.Entries, JournalEntry{Type: EntryDelete, FinalPath: filepath.ToSlash(finalRelPath), OriginalBackupPath: trash})
	return t.j.persistLocked()
}

func (t *Transaction) StageReplace(finalRelPath string) (io.WriteCloser, error) {
	t.j.mu.Lock()
	defer t.j.mu.Unlock()
	abs := filepath.Join(t.j.vaultRoot, filepath.FromSlash(finalRelPath))
	trash := filepath.Join(t.j.dir, "trash", filepath.FromSlash(finalRelPath))
	if _, err := os.Stat(abs); err == nil {
		if err := os.MkdirAll(filepath.Dir(trash), 0o755); err != nil {
			return nil, fmt.Errorf("stage replace mkdir trash: %w", err)
		}
		if err := os.Rename(abs, trash); err != nil {
			return nil, fmt.Errorf("stage replace move: %w", err)
		}
	}
	staged := filepath.Join(t.j.dir, "new", filepath.FromSlash(finalRelPath))
	if err := os.MkdirAll(filepath.Dir(staged), 0o755); err != nil {
		return nil, fmt.Errorf("stage replace mkdir new: %w", err)
	}
	f, err := os.Create(staged)
	if err != nil {
		return nil, fmt.Errorf("stage replace open: %w", err)
	}
	h := sha256.New()
	w := &replaceWriter{multi: io.MultiWriter(f, h), f: f, t: t, staged: staged, rel: filepath.ToSlash(finalRelPath), trash: trash, hsh: h}
	return w, nil
}

type replaceWriter struct {
	multi  io.Writer
	f      *os.File
	t      *Transaction
	staged string
	rel    string
	trash  string
	hsh    interface{ Sum([]byte) []byte }
}

func (rw *replaceWriter) Write(b []byte) (int, error) { return rw.multi.Write(b) }
func (rw *replaceWriter) Close() error {
	if err := rw.f.Close(); err != nil {
		return err
	}
	fi, err := os.Stat(rw.staged)
	if err != nil {
		return err
	}
	sum := rw.hsh.Sum(nil)
	rw.t.j.mu.Lock()
	defer rw.t.j.mu.Unlock()
	rw.t.j.Entries = append(rw.t.j.Entries, JournalEntry{Type: EntryReplace, FinalPath: rw.rel, StagedPath: rw.staged, OriginalBackupPath: rw.trash, Size: fi.Size(), Checksum: "sha256:" + hex.EncodeToString(sum)})
	return rw.t.j.persistLocked()
}

func (t *Transaction) Commit() error {
	t.j.mu.Lock()
	if t.j.State != StatePending {
		t.j.mu.Unlock()
		return fmt.Errorf("cannot commit in state %s", t.j.State)
	}
	t.j.State = StateCommitting
	if err := t.j.persistLocked(); err != nil {
		t.j.mu.Unlock()
		return err
	}
	entries := append([]JournalEntry(nil), t.j.Entries...)
	t.j.mu.Unlock()
	for _, e := range entries {
		if e.Type == EntryCreate || e.Type == EntryReplace {
			if e.StagedPath == "" {
				return t.fail(fmt.Errorf("missing staged path for %s", e.FinalPath))
			}
			finalAbs := filepath.Join(t.j.vaultRoot, filepath.FromSlash(e.FinalPath))
			if err := os.MkdirAll(filepath.Dir(finalAbs), 0o755); err != nil {
				return t.fail(fmt.Errorf("commit mkdir: %w", err))
			}
			if err := os.Rename(e.StagedPath, finalAbs); err != nil {
				return t.fail(fmt.Errorf("commit promote %s: %w", e.FinalPath, err))
			}
		}
	}
	for _, e := range entries {
		if (e.Type == EntryDelete || e.Type == EntryReplace) && e.OriginalBackupPath != "" {
			_ = os.Remove(e.OriginalBackupPath)
		}
	}
	t.j.mu.Lock()
	t.j.State = StateCommitted
	err := t.j.persistLocked()
	t.j.mu.Unlock()
	return err
}

func (t *Transaction) Rollback() error {
	t.j.mu.Lock()
	if t.j.State != StatePending && t.j.State != StateCommitting && t.j.State != StateFailed {
		t.j.mu.Unlock()
		return nil
	}
	t.j.State = StateRollingBack
	if err := t.j.persistLocked(); err != nil {
		t.j.mu.Unlock()
		return err
	}
	entries := append([]JournalEntry(nil), t.j.Entries...)
	t.j.mu.Unlock()
	for _, e := range entries {
		if (e.Type == EntryCreate || e.Type == EntryReplace) && e.StagedPath != "" {
			_ = os.Remove(e.StagedPath)
		}
	}
	for _, e := range entries {
		if (e.Type == EntryDelete || e.Type == EntryReplace) && e.OriginalBackupPath != "" {
			finalAbs := filepath.Join(t.j.vaultRoot, filepath.FromSlash(e.FinalPath))
			if _, err := os.Stat(finalAbs); err == nil {
				continue
			}
			_ = os.MkdirAll(filepath.Dir(finalAbs), 0o755)
			_ = os.Rename(e.OriginalBackupPath, finalAbs)
		}
	}
	t.j.mu.Lock()
	t.j.State = StateRolledBack
	err := t.j.persistLocked()
	t.j.mu.Unlock()
	return err
}

func (t *Transaction) fail(err error) error {
	t.j.mu.Lock()
	t.j.State = StateFailed
	_ = t.j.persistLocked()
	t.j.mu.Unlock()
	_ = t.Rollback()
	return err
}

func (j *Journal) persist() error { j.mu.Lock(); defer j.mu.Unlock(); return j.persistLocked() }
func (j *Journal) persistLocked() error {
	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(j.dir, "journal.json.tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(j.dir, "journal.json"))
}
