package ratelimit

import (
	"sync"
	"time"
)

type tokenBucket struct {
	tokens float64
	last   time.Time
}

type Limiter struct {
	mu      sync.RWMutex
	buckets map[string]*tokenBucket
	rate    float64
	burst   float64
}

func New(rate float64, burst int) *Limiter {
	return &Limiter{
		buckets: make(map[string]*tokenBucket),
		rate:    rate,
		burst:   float64(burst),
	}
}

func (b *tokenBucket) allow(now time.Time, rate, burst float64) (bool, time.Duration) {
	elapsed := now.Sub(b.last).Seconds()
	drippedTokens := rate * elapsed
	totalTokens := min(b.tokens+drippedTokens, burst)
	b.tokens = totalTokens
	b.last = now

	if totalTokens >= 1 {
		b.tokens -= 1
		return true, 0
	}
	missingTokens := 1 - totalTokens
	wait := time.Duration((missingTokens / rate) * float64(time.Second))
	return false, wait
}

// Allow is safe for multiple goroutines at once. The function returns the current possibility of using a rate limit token and the duration till 1 token is available for usage respectively
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()

	b, ok := l.buckets[key]
	if !ok {
		b = &tokenBucket{
			tokens: l.burst,
			last:   now,
		}
		l.buckets[key] = b
	}

	return b.allow(now, l.rate, l.burst)
}
