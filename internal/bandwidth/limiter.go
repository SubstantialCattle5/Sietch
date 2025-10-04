package bandwidth

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"

	"github.com/substantialcattle5/sietch/util"
)

// Limiter provides bandwidth limiting functionality using a token bucket algorithm
type Limiter struct {
	rateLimiter *rate.Limiter
	limit       string // Original limit string for display purposes
}

// NewLimiter creates a new bandwidth limiter from a limit string
// Examples: "1M", "100K", "500KB", "2MB"
func NewLimiter(limitStr string) (*Limiter, error) {
	if limitStr == "" {
		return nil, nil // No limiting if empty
	}

	// Parse the limit using existing utility function
	bytesPerSecond, err := util.ParseChunkSize(limitStr)
	if err != nil {
		return nil, fmt.Errorf("invalid bandwidth limit '%s': %w", limitStr, err)
	}

	if bytesPerSecond <= 0 {
		return nil, fmt.Errorf("bandwidth limit must be positive, got %d bytes/second", bytesPerSecond)
	}

	// Create rate limiter with burst capacity equal to 1 second of data
	// This allows for smooth transfer while maintaining the overall rate
	burst := int(bytesPerSecond)
	if burst < 1024 {
		burst = 1024 // Minimum burst of 1KB for small limits
	}

	limiter := rate.NewLimiter(rate.Limit(bytesPerSecond), burst)

	return &Limiter{
		rateLimiter: limiter,
		limit:       limitStr,
	}, nil
}

// WaitN waits for n bytes to be available according to the rate limit
func (l *Limiter) WaitN(ctx context.Context, n int) error {
	if l == nil || l.rateLimiter == nil {
		return nil // No limiting if limiter is nil
	}

	return l.rateLimiter.WaitN(ctx, n)
}

// AllowN checks if n bytes can be transferred immediately without waiting
func (l *Limiter) AllowN(n int) bool {
	if l == nil || l.rateLimiter == nil {
		return true // No limiting if limiter is nil
	}

	return l.rateLimiter.AllowN(time.Now(), n)
}

// Limit returns the original limit string
func (l *Limiter) Limit() string {
	if l == nil {
		return ""
	}
	return l.limit
}

// Rate returns the current rate limit in bytes per second
func (l *Limiter) Rate() float64 {
	if l == nil || l.rateLimiter == nil {
		return 0
	}
	return float64(l.rateLimiter.Limit())
}

// Burst returns the current burst capacity
func (l *Limiter) Burst() int {
	if l == nil || l.rateLimiter == nil {
		return 0
	}
	return l.rateLimiter.Burst()
}
