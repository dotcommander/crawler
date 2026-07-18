package crawlers

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

const (
	MobileViewportWidth  = 390
	MobileViewportHeight = 844
)

// QueueItem represents an item in the crawl queue.
type QueueItem struct {
	URL   string
	Depth int
}

// DomainCircuitBreaker manages circuit breakers per domain.
type DomainCircuitBreaker struct {
	mu       sync.RWMutex
	breakers map[string]*gobreaker.CircuitBreaker
	settings gobreaker.Settings
}

func NewDomainCircuitBreaker(maxRequests uint32, timeout time.Duration) *DomainCircuitBreaker {
	return &DomainCircuitBreaker{
		breakers: make(map[string]*gobreaker.CircuitBreaker),
		settings: gobreaker.Settings{
			MaxRequests: maxRequests,
			Interval:    timeout,
			Timeout:     timeout,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= maxRequests
			},
		},
	}
}

func (dcb *DomainCircuitBreaker) getBreaker(domain string) *gobreaker.CircuitBreaker {
	dcb.mu.RLock()
	breaker, exists := dcb.breakers[domain]
	dcb.mu.RUnlock()
	if exists {
		return breaker
	}

	dcb.mu.Lock()
	defer dcb.mu.Unlock()
	if breaker, exists := dcb.breakers[domain]; exists {
		return breaker
	}

	settings := dcb.settings
	settings.Name = domain
	breaker = gobreaker.NewCircuitBreaker(settings)
	dcb.breakers[domain] = breaker
	return breaker
}

func (dcb *DomainCircuitBreaker) IsBlocked(domain string) bool {
	return dcb.getBreaker(domain).State() == gobreaker.StateOpen
}

func (dcb *DomainCircuitBreaker) Execute(domain string, req func() error) error {
	_, err := dcb.getBreaker(domain).Execute(func() (interface{}, error) {
		return nil, req()
	})
	return err
}

// DomainRateLimiter implements per-domain rate limiting.
type DomainRateLimiter struct {
	mu          sync.RWMutex
	limiters    map[string]*rate.Limiter
	defaultRate rate.Limit
}

func NewDomainRateLimiter(defaultDelay time.Duration) *DomainRateLimiter {
	return &DomainRateLimiter{
		limiters:    make(map[string]*rate.Limiter),
		defaultRate: rate.Every(defaultDelay),
	}
}

func (d *DomainRateLimiter) SetDomainDelay(domain string, delay time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.limiters[domain] = rate.NewLimiter(rate.Every(delay), 1)
}

func (d *DomainRateLimiter) Wait(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	domain := u.Hostname()
	d.mu.RLock()
	limiter, exists := d.limiters[domain]
	d.mu.RUnlock()
	if !exists {
		d.mu.Lock()
		if limiter, exists = d.limiters[domain]; !exists {
			limiter = rate.NewLimiter(d.defaultRate, 1)
			d.limiters[domain] = limiter
		}
		d.mu.Unlock()
	}

	return limiter.Wait(ctx)
}
