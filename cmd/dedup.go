/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package cmd

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/deduplication"
	"github.com/substantialcattle5/sietch/internal/fs"
	"github.com/substantialcattle5/sietch/util"
)

// dedupCmd represents the dedup command
var dedupCmd = &cobra.Command{
	Use:   "dedup",
	Short: "Manage deduplication in your Sietch vault",
	Long: `Manage deduplication settings and operations in your Sietch vault.

This command provides subcommands for:
- Getting deduplication statistics
- Running garbage collection
- Optimizing storage

You can also configure deduplication settings interactively using the --setup flag.

Example:
  sietch dedup --setup   # Configure deduplication settings interactively
  sietch dedup stats     # Show deduplication statistics
  sietch dedup gc        # Run garbage collection
  sietch dedup optimize  # Optimize storage
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if --setup flag is set
		setup, _ := cmd.Flags().GetBool("setup")
		if !setup {
			// If no flag is set, show help
			return cmd.Help()
		}

		vaultRoot, err := fs.FindVaultRoot()
		if err != nil {
			return fmt.Errorf("not inside a vault: %v", err)
		}

		// Check if vault is initialized
		if !fs.IsVaultInitialized(vaultRoot) {
			return fmt.Errorf("vault not initialized, run 'sietch init' first")
		}

		// Load vault configuration
		vaultConfig, err := config.LoadVaultConfig(vaultRoot)
		if err != nil {
			return fmt.Errorf("failed to load vault configuration: %v", err)
		}

		// Display current settings
		fmt.Println("🔧 Configure Deduplication Settings")
		fmt.Println("===================================")
		fmt.Println()

		if vaultConfig.Deduplication.Enabled {
			fmt.Println("Current settings:")
			fmt.Printf("  Enabled: %v\n", vaultConfig.Deduplication.Enabled)
			fmt.Printf("  Strategy: %s\n", vaultConfig.Deduplication.Strategy)
			fmt.Printf("  Min chunk size: %s\n", vaultConfig.Deduplication.MinChunkSize)
			fmt.Printf("  Max chunk size: %s\n", vaultConfig.Deduplication.MaxChunkSize)
			fmt.Printf("  GC threshold: %d\n", vaultConfig.Deduplication.GCThreshold)
			fmt.Printf("  Index enabled: %v\n", vaultConfig.Deduplication.IndexEnabled)
			fmt.Println()
		} else {
			fmt.Println("Deduplication is currently disabled.")
			fmt.Println()
		}

		// Prompt for deduplication configuration
		if err := deduplication.PromptDeduplicationConfig(vaultConfig); err != nil {
			return fmt.Errorf("configuration failed: %v", err)
		}

		// Display summary
		fmt.Println()
		fmt.Println("📋 New Configuration Summary")
		fmt.Println("===========================")
		fmt.Printf("  Enabled: %v\n", vaultConfig.Deduplication.Enabled)

		if vaultConfig.Deduplication.Enabled {
			fmt.Printf("  Strategy: %s\n", vaultConfig.Deduplication.Strategy)
			fmt.Printf("  Min chunk size: %s\n", vaultConfig.Deduplication.MinChunkSize)
			fmt.Printf("  Max chunk size: %s\n", vaultConfig.Deduplication.MaxChunkSize)
			fmt.Printf("  GC threshold: %d\n", vaultConfig.Deduplication.GCThreshold)
			fmt.Printf("  Index enabled: %v\n", vaultConfig.Deduplication.IndexEnabled)
		}
		fmt.Println()

		// Confirm before saving
		confirmPrompt := promptui.Prompt{
			Label:     "Save these settings",
			IsConfirm: true,
			Default:   "y",
		}

		_, err = confirmPrompt.Run()
		if err != nil {
			if err == promptui.ErrAbort {
				fmt.Println("Configuration cancelled.")
				return nil
			}
			return fmt.Errorf("confirmation failed: %v", err)
		}

		// Save updated configuration
		if err := config.SaveVaultConfig(vaultRoot, vaultConfig); err != nil {
			return fmt.Errorf("failed to save configuration: %v", err)
		}

		fmt.Println("✓ Deduplication configuration saved successfully!")

		if vaultConfig.Deduplication.Enabled {
			fmt.Println("\n💡 Note: Deduplication will apply to new files added to the vault.")
			fmt.Println("   Existing files will not be automatically deduplicated.")
		}

		return nil
	},
}

// dedupStatsCmd shows deduplication statistics
var dedupStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show deduplication statistics",
	Long: `Display detailed statistics about deduplication in your vault.

This includes:
- Total number of chunks
- Total storage size
- Space saved through deduplication
- Number of unreferenced chunks

Example:
  sietch dedup stats
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultRoot, err := fs.FindVaultRoot()
		if err != nil {
			return fmt.Errorf("not inside a vault: %v", err)
		}

		// Check if vault is initialized
		if !fs.IsVaultInitialized(vaultRoot) {
			return fmt.Errorf("vault not initialized, run 'sietch init' first")
		}

		// Load vault configuration
		vaultConfig, err := config.LoadVaultConfig(vaultRoot)
		if err != nil {
			return fmt.Errorf("failed to load vault configuration: %v", err)
		}

		// Initialize deduplication manager
		dedupManager, err := deduplication.NewManager(vaultRoot, vaultConfig.Deduplication)
		if err != nil {
			return fmt.Errorf("failed to initialize deduplication manager: %v", err)
		}

		// Get statistics
		stats := dedupManager.GetStats()

		// Display statistics
		fmt.Printf("\nDeduplication Statistics:\n")
		fmt.Printf("========================\n")
		fmt.Printf("Deduplication enabled: %v\n", vaultConfig.Deduplication.Enabled)
		fmt.Printf("Total chunks: %d\n", stats.TotalChunks)
		fmt.Printf("Total size: %s\n", util.HumanReadableSize(stats.TotalSize))
		fmt.Printf("Space saved: %s\n", util.HumanReadableSize(stats.SavedSpace))
		fmt.Printf("Unreferenced chunks: %d\n", stats.UnreferencedChunks)

		if stats.TotalSize > 0 {
			percentage := float64(stats.SavedSpace) / float64(stats.TotalSize+stats.SavedSpace) * 100
			fmt.Printf("Deduplication ratio: %.2f%%\n", percentage)
		}

		if stats.UnreferencedChunks > 0 {
			fmt.Printf("\n⚠️  You have %d unreferenced chunks. Consider running 'sietch dedup gc' to clean them up.\n", stats.UnreferencedChunks)
		}

		return nil
	},
}

// dedupGcCmd runs garbage collection
var dedupGcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Run garbage collection on unreferenced chunks",
	Long: `Remove chunks that are no longer referenced by any files.

This command will:
- Identify chunks that are not referenced by any file manifests
- Remove these chunks from storage
- Update the deduplication index

Example:
  sietch dedup gc
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultRoot, err := fs.FindVaultRoot()
		if err != nil {
			return fmt.Errorf("not inside a vault: %v", err)
		}

		// Check if vault is initialized
		if !fs.IsVaultInitialized(vaultRoot) {
			return fmt.Errorf("vault not initialized, run 'sietch init' first")
		}

		// Load vault configuration
		vaultConfig, err := config.LoadVaultConfig(vaultRoot)
		if err != nil {
			return fmt.Errorf("failed to load vault configuration: %v", err)
		}

		if !vaultConfig.Deduplication.Enabled {
			return fmt.Errorf("deduplication is not enabled in this vault")
		}

		// Initialize deduplication manager
		dedupManager, err := deduplication.NewManager(vaultRoot, vaultConfig.Deduplication)
		if err != nil {
			return fmt.Errorf("failed to initialize deduplication manager: %v", err)
		}

		fmt.Println("Running garbage collection...")

		// Run garbage collection
		removedChunks, err := dedupManager.GarbageCollect()
		if err != nil {
			return fmt.Errorf("garbage collection failed: %v", err)
		}

		// Save the updated index
		if err := dedupManager.Save(); err != nil {
			return fmt.Errorf("failed to save updated index: %v", err)
		}

		fmt.Printf("✓ Garbage collection completed\n")
		fmt.Printf("✓ Removed %d unreferenced chunks\n", removedChunks)

		return nil
	},
}

// dedupOptimizeCmd optimizes storage
var dedupOptimizeCmd = &cobra.Command{
	Use:   "optimize",
	Short: "Optimize vault storage",
	Long: `Perform comprehensive storage optimization.

This command will:
- Run garbage collection to remove unreferenced chunks
- Update and optimize the deduplication index
- Display optimization results

Example:
  sietch dedup optimize
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultRoot, err := fs.FindVaultRoot()
		if err != nil {
			return fmt.Errorf("not inside a vault: %v", err)
		}

		// Check if vault is initialized
		if !fs.IsVaultInitialized(vaultRoot) {
			return fmt.Errorf("vault not initialized, run 'sietch init' first")
		}

		// Load vault configuration
		vaultConfig, err := config.LoadVaultConfig(vaultRoot)
		if err != nil {
			return fmt.Errorf("failed to load vault configuration: %v", err)
		}

		if !vaultConfig.Deduplication.Enabled {
			return fmt.Errorf("deduplication is not enabled in this vault")
		}

		// Initialize deduplication manager
		dedupManager, err := deduplication.NewManager(vaultRoot, vaultConfig.Deduplication)
		if err != nil {
			return fmt.Errorf("failed to initialize deduplication manager: %v", err)
		}

		fmt.Println("Optimizing vault storage...")

		// Run optimization
		result, err := dedupManager.OptimizeStorage()
		if err != nil {
			return fmt.Errorf("optimization failed: %v", err)
		}

		// Display results
		fmt.Printf("\nOptimization Results:\n")
		fmt.Printf("====================\n")
		fmt.Printf("✓ Total chunks: %d\n", result.TotalChunks)
		fmt.Printf("✓ Removed chunks: %d\n", result.RemovedChunks)
		fmt.Printf("✓ Space saved: %s\n", util.HumanReadableSize(result.SavedSpace))
		fmt.Printf("✓ Remaining unreferenced chunks: %d\n", result.UnreferencedChunks)

		if result.RemovedChunks > 0 {
			fmt.Printf("\n✓ Storage optimization completed successfully\n")
		} else {
			fmt.Printf("\n✓ Storage is already optimized\n")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dedupCmd)

	// Add --setup flag for interactive configuration
	dedupCmd.Flags().BoolP("setup", "s", false, "Configure deduplication settings interactively")

	// Add subcommands
	dedupCmd.AddCommand(dedupStatsCmd)
	dedupCmd.AddCommand(dedupGcCmd)
	dedupCmd.AddCommand(dedupOptimizeCmd)
}
