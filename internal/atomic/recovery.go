package atomic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type RecoveryResult struct {
	ResumedCommits int
	RolledBack     int
	Purged         int
	Errors         []error
}

func Recover(vaultRoot string, retention time.Duration) (*RecoveryResult, error) {
	txnRoot := filepath.Join(vaultRoot, ".txn")
	res := &RecoveryResult{}
	entries, err := os.ReadDir(txnRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return res, nil
		}
		return res, fmt.Errorf("read txn root: %w", err)
	}
	now := time.Now()
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(txnRoot, e.Name())
		jpath := filepath.Join(dir, "journal.json")
		data, err := os.ReadFile(jpath)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("read %s: %w", jpath, err))
			continue
		}
		var j Journal
		if err := json.Unmarshal(data, &j); err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("unmarshal %s: %w", jpath, err))
			continue
		}
		j.dir = dir
		j.vaultRoot = vaultRoot
		txn := &Transaction{j: &j}
		switch j.State {
		case StateCommitted:
			if retention > 0 && now.Sub(j.StartedAt) > retention {
				_ = os.RemoveAll(dir)
				res.Purged++
			}
		case StatePending, StateCommitting, StateFailed:
			if err := txn.Commit(); err != nil {
				if rerr := txn.Rollback(); rerr != nil {
					res.Errors = append(res.Errors, fmt.Errorf("rollback %s: %v (commit err: %v)", j.ID, rerr, err))
				} else {
					res.RolledBack++
				}
			} else {
				res.ResumedCommits++
			}
		case StateRollingBack:
			if err := txn.Rollback(); err != nil {
				res.Errors = append(res.Errors, fmt.Errorf("finish rollback %s: %v", j.ID, err))
			} else {
				res.RolledBack++
			}
		case StateRolledBack:
			if retention > 0 && now.Sub(j.StartedAt) > retention {
				_ = os.RemoveAll(dir)
				res.Purged++
			}
		}
	}
	return res, nil
}
