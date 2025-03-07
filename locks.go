package kemu

import (
	"sync"
)

// Mutex provides a simple, coarse-grained keyed mutex (lockmap)
// implementation. It is intended to be used by components that need to
// lock on a specific key, but do not need to lock on multiple keys
// simultaneously. The locks are not reentrant.
type Mutex struct {
	mu    sync.Mutex
	locks map[string]*lockEntry
}

type lockEntry struct {
	mu       sync.Mutex
	refCount int
}

// New is a Keyed Mutex implementation with a simple, coarse-grained
// locking strategy. It is intended to be used by components that need
// to lock on a specific key, but do not need to lock on multiple keys
// simultaneously. The locks are not reentrant.
func New() *Mutex {
	sl := &Mutex{
		locks: make(map[string]*lockEntry),
	}
	return sl
}

// Locked returns true if the key is currently locked, false otherwise
func (km *Mutex) Locked(key string) bool {
	km.mu.Lock()
	defer km.mu.Unlock()
	le, exists := km.locks[key]
	return exists && le.refCount > 0
}

// TryLock returns true if the lock was acquired, false otherwise
func (km *Mutex) TryLock(key string) bool {
	km.mu.Lock()
	defer km.mu.Unlock()
	le, exists := km.locks[key]
	if exists && le.refCount > 0 {
		return false
	}
	if !exists {
		le = &lockEntry{}
		km.locks[key] = le
	}
	le.refCount++
	le.mu.Lock()
	return true
}

// Lock acquires a lock on the key
func (km *Mutex) Lock(key string) {
	km.mu.Lock()
	le, exists := km.locks[key]
	if !exists {
		le = &lockEntry{}
		km.locks[key] = le
	}
	le.refCount++
	km.mu.Unlock()

	le.mu.Lock()
}

// Unlock releases a lock on the key
func (km *Mutex) Unlock(key string) {
	km.mu.Lock()
	defer km.mu.Unlock()

	le, exists := km.locks[key]
	if !exists {
		panic("unlock of unlocked lock")
	}
	le.refCount--
	if le.refCount == 0 {
		delete(km.locks, key)
	}
	le.mu.Unlock()
}
