package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/types"
	"github.com/Krive/ServiceNow-Toolkit/pkg/utils/retry"
)

// Simple retry tests without core dependencies
func TestRetryBasicConfig(t *testing.T) {
	config := retry.DefaultConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts to be 3, got %d", config.MaxAttempts)
	}

	if config.BaseDelay != 100*time.Millisecond {
		t.Errorf("Expected BaseDelay to be 100ms, got %v", config.BaseDelay)
	}

	if config.Multiplier != 2.0 {
		t.Errorf("Expected Multiplier to be 2.0, got %f", config.Multiplier)
	}

	if !config.Jitter {
		t.Error("Expected Jitter to be true")
	}
}

func TestRetryBasicSuccess(t *testing.T) {
	config := retry.Config{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
		Jitter:      false,
		RetryOn:     []types.ErrorType{}, // Empty for basic test
	}

	attemptCount := 0
	fn := func() error {
		attemptCount++
		if attemptCount < 2 {
			return errors.New("temporary error")
		}
		return nil
	}

	ctx := context.Background()
	
	// This will fail because we don't have error classification without core
	// But it tests the basic retry structure
	err := retry.Do(ctx, config, fn)
	
	// With no retry-on types, should only attempt once
	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt with empty RetryOn, got %d", attemptCount)
	}

	// Should get the error since no retries
	if err == nil {
		t.Error("Expected error since no retry-on types specified")
	}
}

func TestRetryContextCancellation(t *testing.T) {
	config := retry.Config{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
		Jitter:      false,
		RetryOn:     []types.ErrorType{},
	}

	fn := func() error {
		return errors.New("always fails")
	}

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := retry.Do(ctx, config, fn)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected error")
	}

	// Should complete quickly due to context cancellation
	if elapsed > 50*time.Millisecond {
		t.Errorf("Expected quick completion due to context cancellation, took %v", elapsed)
	}
}

func TestRetryWithResultBasic(t *testing.T) {
	config := retry.Config{
		MaxAttempts: 2,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
		Jitter:      false,
		RetryOn:     []types.ErrorType{},
	}

	fn := func() (string, error) {
		return "result", errors.New("error")
	}

	ctx := context.Background()
	result, err := retry.DoWithResult(ctx, config, fn)

	if err == nil {
		t.Error("Expected error")
	}

	if result != "result" {
		t.Errorf("Expected result 'result', got: %s", result)
	}
}