package main

import (
	"sync"
)

type ConnectionLimiter struct {
	mu     sync.RWMutex
	active bool
}

func (cl *ConnectionLimiter) IsActive() bool {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.active
}

func (cl *ConnectionLimiter) SetActive(active bool) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.active = active
}
