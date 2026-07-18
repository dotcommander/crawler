package crawlers

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestDomainRateLimiter_DistinctDomainsDoNotBlockEachOther(t *testing.T) {
	t.Parallel()
	// 100ms spacing per domain. Two distinct domains' first requests both
	// pass immediately (burst=1), so the pair completes well under one delay.
	rl := NewDomainRateLimiter(100 * time.Millisecond)
	ctx := context.Background()

	start := time.Now()
	if err := rl.Wait(ctx, "https://a.example.com/x"); err != nil {
		t.Fatalf("domain a wait: %v", err)
	}
	if err := rl.Wait(ctx, "https://b.example.com/y"); err != nil {
		t.Fatalf("domain b wait: %v", err)
	}
	if elapsed := time.Since(start); elapsed >= 100*time.Millisecond {
		t.Fatalf("distinct domains blocked each other: elapsed=%v, want < 100ms", elapsed)
	}
}

func TestDomainRateLimiter_RespectsContextCancellation(t *testing.T) {
	t.Parallel()
	// 10s spacing: first request passes, second must wait ~10s — but a
	// cancelled ctx aborts it promptly instead of sleeping.
	rl := NewDomainRateLimiter(10 * time.Second)
	if err := rl.Wait(context.Background(), "https://slow.example.com/1"); err != nil {
		t.Fatalf("first wait: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if err := rl.Wait(ctx, "https://slow.example.com/2"); err == nil {
		t.Fatal("expected error from cancelled ctx, got nil")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("cancellation not respected: elapsed=%v, want < 1s", elapsed)
	}
}

func TestDomainRateLimiter_SerializesConcurrentFirstAccessToSameDomain(t *testing.T) {
	t.Parallel()

	const delay = 80 * time.Millisecond
	rl := NewDomainRateLimiter(delay)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	begin := time.Now()
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			<-start
			if err := rl.Wait(context.Background(), "https://same.example.com/path"); err != nil {
				t.Errorf("wait failed: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if elapsed := time.Since(begin); elapsed < delay/2 {
		t.Fatalf("same-domain first access was not serialized: elapsed=%v, want at least %v", elapsed, delay/2)
	}
}
