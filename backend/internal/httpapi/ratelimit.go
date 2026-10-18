package httpapi

import (
	"sync"
	"time"
)

// rateLimiter 是一个按 key 计数的固定窗口限流器。
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	lastGC  time.Time
}

type rateBucket struct {
	count int
	reset time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		buckets: make(map[string]*rateBucket),
		lastGC:  time.Now(),
	}
}

// allow 返回该 key 在窗口内是否还有配额。
func (l *rateLimiter) allow(key string, limit int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	if now.Sub(l.lastGC) > time.Minute {
		for k, bucket := range l.buckets {
			if now.After(bucket.reset) {
				delete(l.buckets, k)
			}
		}
		l.lastGC = now
	}

	bucket, ok := l.buckets[key]
	if !ok || now.After(bucket.reset) {
		l.buckets[key] = &rateBucket{count: 1, reset: now.Add(window)}
		return true
	}

	if bucket.count >= limit {
		return false
	}

	bucket.count++
	return true
}
