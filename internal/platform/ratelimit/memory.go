package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Memory is an in-memory fixed-window limiter. Not suitable for
// multi-instance deployments; see R22 in docs/decisions.
type Memory struct {
	mu      sync.Mutex
	buckets map[string][]time.Time
	cfg     Config
}

func NewMemory(cfg Config) *Memory {
	m := &Memory{
		buckets: make(map[string][]time.Time),
		cfg:     cfg,
	}
	go m.sweepLoop()
	return m
}

// Allow trims the bucket to the current window and checks the count.
func (m *Memory) Allow(_ context.Context, key string) (bool, error) {
	now := time.Now()
	cutoff := now.Add(-m.cfg.Window)

	m.mu.Lock()
	defer m.mu.Unlock()

	events := m.buckets[key]
	// Trim expired events.
	trimmed := events[:0]
	for _, t := range events {
		if t.After(cutoff) {
			trimmed = append(trimmed, t)
		}
	}
	if len(trimmed) >= m.cfg.Limit {
		m.buckets[key] = trimmed
		return false, nil
	}
	m.buckets[key] = append(trimmed, now)
	return true, nil
}

// sweepLoop evicts empty buckets periodically to bound memory.
func (m *Memory) sweepLoop() {
	t := time.NewTicker(m.cfg.Window)
	defer t.Stop()
	for range t.C {
		m.sweep()
	}
}

func (m *Memory) sweep() {
	cutoff := time.Now().Add(-m.cfg.Window)
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, events := range m.buckets {
		trimmed := events[:0]
		for _, t := range events {
			if t.After(cutoff) {
				trimmed = append(trimmed, t)
			}
		}
		if len(trimmed) == 0 {
			delete(m.buckets, k)
		} else {
			m.buckets[k] = trimmed
		}
	}
}

var _ Limiter = (*Memory)(nil)