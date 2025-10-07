package cmd

import (
    "fmt"
    "time"

    "github.com/spf13/cobra"

    "github.com/substantialcattle5/sietch/internal/atomic"
    "github.com/substantialcattle5/sietch/internal/fs"
)

var recoverCmd = &cobra.Command{
    Use:   "recover",
    Short: "Recover incomplete or failed vault transactions",
    RunE: func(cmd *cobra.Command, args []string) error {
        vaultRoot, err := fs.FindVaultRoot(); if err != nil { return fmt.Errorf("not inside a vault: %v", err) }
        retention, _ := cmd.Flags().GetDuration("retention")
        res, err := atomic.Recover(vaultRoot, retention); if err != nil { return err }
        fmt.Fprintf(cmd.OutOrStdout(), "Recovery complete. Resumed=%d RolledBack=%d Purged=%d Errors=%d\n", res.ResumedCommits, res.RolledBack, res.Purged, len(res.Errors))
        for _, e := range res.Errors { fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", e) }
        return nil
    },
}

func init() {
    recoverCmd.Flags().Duration("retention", 24*time.Hour, "Retention window before purging finished transaction journals")
    rootCmd.AddCommand(recoverCmd)
}
