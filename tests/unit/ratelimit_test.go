package unit

import (
	"context"
	"testing"
	"time"

	"github.com/Krive/ServiceNow-Toolkit/pkg/utils/ratelimit"
)

func TestTokenBucketLimiter(t *testing.T) {
	// Create limiter with 2 requests per second, burst of 5
	limiter := ratelimit.NewTokenBucketLimiter(2.0, 5)

	// Test immediate availability (burst)
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Errorf("Expected request %d to be allowed immediately", i+1)
		}
	}

	// 6th request should be denied (burst exhausted)
	if limiter.Allow() {
		t.Error("Expected 6th request to be denied")
	}

	// Test Wait functionality
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err := limiter.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Expected Wait to succeed, got error: %v", err)
	}

	// Should have waited approximately 500ms (1/2 requests per second)
	if elapsed < 400*time.Millisecond || elapsed > 600*time.Millisecond {
		t.Errorf("Expected wait time around 500ms, got %v", elapsed)
	}
}

func TestTokenBucketLimiterReservation(t *testing.T) {
	limiter := ratelimit.NewTokenBucketLimiter(1.0, 1)

	// Use up the burst
	if !limiter.Allow() {
		t.Error("Expected first request to be allowed")
	}

	// Reserve a token
	reservation := limiter.Reserve()
	if !reservation.OK() {
		t.Error("Expected reservation to be OK")
	}

	delay := reservation.Delay()
	if delay <= 0 {
		t.Error("Expected positive delay for reservation")
	}

	// Test cancellation
	reservation.Cancel()
	
	// After cancellation, should be able to make immediate request
	if !limiter.Allow() {
		t.Error("Expected request to be allowed after reservation cancellation")
	}
}

func TestServiceNowLimiterDefaultConfig(t *testing.T) {
	config := ratelimit.DefaultServiceNowConfig()

	if config.TableRequestsPerSecond != 5.0 {
		t.Errorf("Expected TableRequestsPerSecond to be 5.0, got %f", config.TableRequestsPerSecond)
	}

	if config.AttachmentRequestsPerSecond != 2.0 {
		t.Errorf("Expected AttachmentRequestsPerSecond to be 2.0, got %f", config.AttachmentRequestsPerSecond)
	}

	if config.ImportRequestsPerSecond != 1.0 {
		t.Errorf("Expected ImportRequestsPerSecond to be 1.0, got %f", config.ImportRequestsPerSecond)
	}

	if config.TableBurst != 10 {
		t.Errorf("Expected TableBurst to be 10, got %d", config.TableBurst)
	}
}

func TestServiceNowLimiter(t *testing.T) {
	config := ratelimit.ServiceNowLimiterConfig{
		TableRequestsPerSecond:      2.0,
		AttachmentRequestsPerSecond: 1.0,
		ImportRequestsPerSecond:     0.5,
		DefaultRequestsPerSecond:    1.5,
		TableBurst:                  3,
		AttachmentBurst:             2,
		ImportBurst:                 1,
		DefaultBurst:                2,
	}

	limiter := ratelimit.NewServiceNowLimiter(config)

	// Test table endpoint
	ctx := context.Background()
	
	// Should allow burst requests immediately
	for i := 0; i < 3; i++ {
		if !limiter.Allow(ratelimit.EndpointTypeTable) {
			t.Errorf("Expected table request %d to be allowed", i+1)
		}
	}

	// 4th request should require waiting
	if limiter.Allow(ratelimit.EndpointTypeTable) {
		t.Error("Expected 4th table request to be rate limited")
	}

	// Test different endpoint types have independent limits
	if !limiter.Allow(ratelimit.EndpointTypeAttachment) {
		t.Error("Expected attachment request to be allowed (different limit)")
	}

	// Test Wait functionality with context
	start := time.Now()
	err := limiter.Wait(ctx, ratelimit.EndpointTypeImport)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Expected Wait to succeed, got error: %v", err)
	}

	// For 0.5 req/sec, should wait ~2 seconds, but allow some tolerance
	if elapsed > 100*time.Millisecond {
		t.Logf("Wait time for import endpoint: %v (expected some delay)", elapsed)
	}
}

func TestServiceNowLimiterUpdateConfig(t *testing.T) {
	initialConfig := ratelimit.ServiceNowLimiterConfig{
		TableRequestsPerSecond: 1.0,
		TableBurst:            1,
		AttachmentRequestsPerSecond: 1.0,
		AttachmentBurst: 1,
		ImportRequestsPerSecond: 1.0,
		ImportBurst: 1,
		DefaultRequestsPerSecond: 1.0,
		DefaultBurst: 1,
	}

	limiter := ratelimit.NewServiceNowLimiter(initialConfig)

	// Use up initial burst
	if !limiter.Allow(ratelimit.EndpointTypeTable) {
		t.Error("Expected initial table request to be allowed")
	}

	// Should be rate limited now
	if limiter.Allow(ratelimit.EndpointTypeTable) {
		t.Error("Expected second table request to be rate limited")
	}

	// Update to more permissive config
	newConfig := ratelimit.ServiceNowLimiterConfig{
		TableRequestsPerSecond: 10.0,
		TableBurst:            10,
		AttachmentRequestsPerSecond: 10.0,
		AttachmentBurst: 10,
		ImportRequestsPerSecond: 10.0,
		ImportBurst: 10,
		DefaultRequestsPerSecond: 10.0,
		DefaultBurst: 10,
	}

	limiter.UpdateConfig(newConfig)

	// Should now allow requests
	if !limiter.Allow(ratelimit.EndpointTypeTable) {
		t.Error("Expected table request to be allowed after config update")
	}
}

func TestDetectEndpointType(t *testing.T) {
	tests := []struct {
		path     string
		expected ratelimit.EndpointType
	}{
		{"/table/incident", ratelimit.EndpointTypeTable},
		{"/now/table/sys_user", ratelimit.EndpointTypeTable},
		{"/attachment/upload", ratelimit.EndpointTypeAttachment},
		{"/now/attachment", ratelimit.EndpointTypeAttachment},
		{"/import/staging_table", ratelimit.EndpointTypeImport},
		{"/now/import/transform", ratelimit.EndpointTypeImport},
		{"/stats", ratelimit.EndpointTypeDefault},
		{"/oauth_token.do", ratelimit.EndpointTypeDefault},
		{"", ratelimit.EndpointTypeDefault},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			result := ratelimit.DetectEndpointType(test.path)
			if result != test.expected {
				t.Errorf("Expected endpoint type %s for path %s, got %s", 
					test.expected, test.path, result)
			}
		})
	}
}

func TestRateLimitError(t *testing.T) {
	retryAfter := 30 * time.Second
	err := ratelimit.NewRateLimitError("Too many requests", retryAfter)

	expectedMsg := "rate limit exceeded: Too many requests (retry after 30s)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}

	if err.RetryAfter != retryAfter {
		t.Errorf("Expected RetryAfter to be %v, got %v", retryAfter, err.RetryAfter)
	}
}

func TestServiceNowLimiterWithContext(t *testing.T) {
	// Create very restrictive limiter
	config := ratelimit.ServiceNowLimiterConfig{
		TableRequestsPerSecond:      0.1, // Very slow - 1 request per 10 seconds
		TableBurst:                  0,   // No burst
		AttachmentRequestsPerSecond: 1.0,
		AttachmentBurst:            1,
		ImportRequestsPerSecond:     1.0,
		ImportBurst:                1,
		DefaultRequestsPerSecond:    1.0,
		DefaultBurst:               1,
	}

	limiter := ratelimit.NewServiceNowLimiter(config)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should timeout due to very slow rate limit
	err := limiter.Wait(ctx, ratelimit.EndpointTypeTable)
	if err == nil {
		t.Error("Expected timeout error for very restrictive rate limit")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func BenchmarkTokenBucketLimiter(b *testing.B) {
	limiter := ratelimit.NewTokenBucketLimiter(1000.0, 100)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow()
		}
	})
}

func BenchmarkServiceNowLimiter(b *testing.B) {
	config := ratelimit.DefaultServiceNowConfig()
	limiter := ratelimit.NewServiceNowLimiter(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow(ratelimit.EndpointTypeTable)
		}
	})
}