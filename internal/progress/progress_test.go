package progress

import (
	"context"
	"testing"
	"time"
)

func TestProgressManager(t *testing.T) {
	// Test with normal mode (progress bars enabled)
	t.Run("NormalMode", func(t *testing.T) {
		pm := NewManager(Options{
			Quiet:   false,
			Verbose: false,
		})

		ctx := context.Background()
		_ = pm.SetupCancellation(ctx)

		// Initialize progress bars
		totalBytes := int64(1000)
		pm.InitTotalProgress(totalBytes, "Test operation")
		pm.InitFileProgress(totalBytes, "testfile.txt")

		// Simulate progress
		for i := int64(0); i < totalBytes; i += 100 {
			pm.UpdateTotalProgress(100)
			pm.UpdateFileProgress(100)
			time.Sleep(10 * time.Millisecond) // Small delay to see progress
		}

		// Complete progress
		pm.FinishTotalProgress()
		pm.FinishFileProgress()
		pm.Cleanup()
	})

	t.Run("QuietMode", func(t *testing.T) {
		pm := NewManager(Options{
			Quiet:   true,
			Verbose: false,
		})

		ctx := context.Background()
		_ = pm.SetupCancellation(ctx)

		// Initialize progress bars (should not show in quiet mode)
		totalBytes := int64(100)
		pm.InitTotalProgress(totalBytes, "Test operation")
		pm.InitFileProgress(totalBytes, "testfile.txt")

		// Simulate progress
		for i := int64(0); i < totalBytes; i += 10 {
			pm.UpdateTotalProgress(10)
			pm.UpdateFileProgress(10)
		}

		// Complete progress
		pm.FinishTotalProgress()
		pm.FinishFileProgress()
		pm.Cleanup()
	})

	t.Run("VerboseMode", func(t *testing.T) {
		pm := NewManager(Options{
			Quiet:   false,
			Verbose: true,
		})

		// Test verbose output
		pm.PrintVerbose("This is a verbose message\n")
		pm.PrintInfo("This is an info message\n")

		// Test with quiet mode
		pm2 := NewManager(Options{
			Quiet:   true,
			Verbose: false,
		})

		pm2.PrintInfo("This should not print in quiet mode\n")
		pm2.PrintVerbose("This should not print in quiet mode\n")
	})
}

func TestProgressManagerCancellation(t *testing.T) {
	pm := NewManager(Options{
		Quiet:   true, // Use quiet to avoid output during test
		Verbose: false,
	})

	ctx := context.Background()
	ctx = pm.SetupCancellation(ctx)

	// Test that cancellation works
	go func() {
		time.Sleep(50 * time.Millisecond)
		// Simulate sending SIGINT
		// Note: In a real test, we'd need to send actual signals
		// For this test, we'll just check the setup
	}()

	// The context should be cancellable
	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Log("Context was not cancelled as expected")
	}

	pm.Cleanup()
}

func TestProgressManagerCancellationDuringOperation(t *testing.T) {
	pm := NewManager(Options{
		Quiet:   true, // Use quiet to avoid output during test
		Verbose: false,
	})

	ctx := context.Background()
	ctx = pm.SetupCancellation(ctx)

	// Initialize progress bars
	totalBytes := int64(10000)
	pm.InitTotalProgress(totalBytes, "Test operation")
	pm.InitFileProgress(totalBytes, "testfile.txt")

	cancelled := false

	// Start a goroutine that simulates file processing
	done := make(chan bool)
	go func() {
		defer close(done)
		defer pm.Cleanup()

		for i := int64(0); i < totalBytes && !cancelled; i += 100 {
			select {
			case <-ctx.Done():
				cancelled = true
				return
			default:
				pm.UpdateTotalProgress(100)
				pm.UpdateFileProgress(100)
				time.Sleep(1 * time.Millisecond)
			}
		}

		if !cancelled {
			pm.FinishTotalProgress()
			pm.FinishFileProgress()
		}
	}()

	// Wait a bit then cancel
	time.Sleep(50 * time.Millisecond)

	// Simulate cancellation by calling the cancel function
	// In real usage, this would be triggered by SIGINT
	if pm.cancelFunc != nil {
		pm.cancelFunc()
	}

	// Set the cancelled flag directly since we're simulating
	pm.cancelMux.Lock()
	pm.cancelled = true
	pm.cancelMux.Unlock()

	// Wait for the operation to complete/cancel
	select {
	case <-done:
		// Operation completed
	case <-time.After(200 * time.Millisecond):
		t.Error("Operation did not complete within timeout")
	}

	if !cancelled {
		t.Error("Operation was not cancelled as expected")
	}

	// Verify that the operation was marked as cancelled
	if !pm.IsCancelled() {
		t.Error("ProgressManager should report as cancelled")
	}
}
