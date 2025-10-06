package progress

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/schollz/progressbar/v3"
)

// Options configures progress bar behavior
type Options struct {
	Quiet   bool
	Verbose bool
}

// Manager handles progress bars and cancellation
type Manager struct {
	options    Options
	totalBar   *progressbar.ProgressBar
	fileBar    *progressbar.ProgressBar
	cancelFunc context.CancelFunc
	cancelled  bool
	cancelMux  sync.Mutex
	signalChan chan os.Signal
}

// NewManager creates a new progress manager
func NewManager(options Options) *Manager {
	return &Manager{
		options:    options,
		signalChan: make(chan os.Signal, 1),
	}
}

// SetupCancellation sets up signal handling for cancellation
func (pm *Manager) SetupCancellation(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	pm.cancelFunc = cancel

	signal.Notify(pm.signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-pm.signalChan:
			pm.cancelMux.Lock()
			pm.cancelled = true
			pm.cancelMux.Unlock()
			// #nosec G104 - cancellation message is not critical for functionality
			fmt.Println("\nOperation cancelled by user")
			cancel()
		case <-ctx.Done():
			// Context already cancelled
		}
	}()

	return ctx
}

// IsCancelled checks if the operation was cancelled
func (pm *Manager) IsCancelled() bool {
	pm.cancelMux.Lock()
	defer pm.cancelMux.Unlock()
	return pm.cancelled
}

// Cleanup removes signal handlers
func (pm *Manager) Cleanup() {
	signal.Stop(pm.signalChan)
	if pm.cancelFunc != nil {
		pm.cancelFunc()
	}
}

// InitTotalProgress initializes the total progress bar
func (pm *Manager) InitTotalProgress(totalBytes int64, description string) {
	if pm.options.Quiet {
		return
	}

	pm.totalBar = progressbar.NewOptions64(totalBytes,
		progressbar.OptionSetDescription(fmt.Sprintf("%s [total]", description)),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(65),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			// #nosec G104 - progress bar completion message is not critical
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
	)
}

// InitFileProgress initializes the per-file progress bar
func (pm *Manager) InitFileProgress(totalBytes int64, filename string) {
	if pm.options.Quiet {
		return
	}

	pm.fileBar = progressbar.NewOptions64(totalBytes,
		progressbar.OptionSetDescription(fmt.Sprintf("Processing %s", filename)),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(65),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			// #nosec G104 - progress bar completion message is not critical
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
	)
}

// UpdateTotalProgress updates the total progress bar
func (pm *Manager) UpdateTotalProgress(bytes int64) {
	if pm.options.Quiet || pm.totalBar == nil {
		return
	}
	// #nosec G104 - progress bar errors are not critical for functionality
	pm.totalBar.Add64(bytes)
}

// UpdateFileProgress updates the per-file progress bar
func (pm *Manager) UpdateFileProgress(bytes int64) {
	if pm.options.Quiet || pm.fileBar == nil {
		return
	}
	// #nosec G104 - progress bar errors are not critical for functionality
	pm.fileBar.Add64(bytes)
}

// FinishTotalProgress marks the total progress as complete
func (pm *Manager) FinishTotalProgress() {
	if pm.options.Quiet || pm.totalBar == nil {
		return
	}
	pm.totalBar.Finish()
}

// FinishFileProgress marks the file progress as complete
func (pm *Manager) FinishFileProgress() {
	if pm.options.Quiet || pm.fileBar == nil {
		return
	}
	// #nosec G104 - progress bar errors are not critical for functionality
	pm.fileBar.Finish()
}

// PrintVerbose prints verbose information if verbose mode is enabled
func (pm *Manager) PrintVerbose(format string, args ...interface{}) {
	if pm.options.Verbose {
		// Clear the progress bar before printing to avoid line breaks
		if pm.totalBar != nil {
			// #nosec G104 - progress bar clear is not critical for functionality
			pm.totalBar.Clear()
		}

		// #nosec G104 - verbose output errors are not critical for functionality
		fmt.Printf(format, args...)
		// Ensure output ends with newline if not already present
		if len(format) == 0 || format[len(format)-1] != '\n' {
			// #nosec G104 - newline output is not critical for functionality
			fmt.Println()
		}
	}
}

// PrintInfo prints informational messages (unless quiet mode)
func (pm *Manager) PrintInfo(format string, args ...interface{}) {
	if !pm.options.Quiet {
		// Clear the progress bar before printing to avoid line breaks
		if pm.totalBar != nil {
			// #nosec G104 - progress bar clear is not critical for functionality
			pm.totalBar.Clear()
		}

		// #nosec G104 - info output errors are not critical for functionality
		fmt.Printf(format, args...)
	}
}
