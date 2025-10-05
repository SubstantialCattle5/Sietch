/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/internal/gc"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sietch",
	Short: "Sietch - A secure, nomadic file system",
	Long: `Sietch is a secure, decentralized file which allows users to securely synchronize 
encrypted data across machines, even with limited connectivity.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start GC manager in background if in a vault
	var gcCancel context.CancelFunc
	if shouldStartGC() {
		gcCtx, gcCancelFunc := context.WithCancel(ctx)
		gcCancel = gcCancelFunc

		go func() {
			if err := gc.StartGlobalGC(gcCtx); err != nil {
				// Log error but don't fail execution
				fmt.Printf("Warning: Failed to start automatic GC: %v\n", err)
			}
		}()
	}

	// Handle shutdown
	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal %v, shutting down...\n", sig)

		// Cancel GC manager
		if gcCancel != nil {
			gcCancel()
			time.Sleep(100 * time.Millisecond) // Give GC time to stop
		}

		cancel()
		os.Exit(0)
	}()

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// shouldStartGC determines if GC should be started
func shouldStartGC() bool {
	// Check if we're in a vault directory
	vaultRoot, err := fs.FindVaultRoot()
	if err != nil {
		return false
	}

	// Check if vault is initialized
	if !fs.IsVaultInitialized(vaultRoot) {
		return false
	}

	return true
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sietch.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Disable progress bars and reduce output")
}
