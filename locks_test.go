package kemu

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestKeyedMutex_LockUnlock(t *testing.T) {
	km := New()

	key := "testKey"
	km.Lock(key)
	km.Unlock(key)

	if _, ok := km.locks[key]; ok {
		t.Errorf("Expected mutex for key %s to be removed", key)
	}
}

func TestKeyedMutex_TryLocked(t *testing.T) {
	km := New()

	key := "testKey"
	if !km.TryLock(key) {
		t.Errorf("Expected TryLock to succeed for key %s", key)
	}

	if km.TryLock(key) {
		t.Errorf("Expected TryLock to fail for key %s", key)
	}

	km.Unlock(key)
	if !km.TryLock(key) {
		t.Errorf("Expected TryLock to succeed for key %s after unlock", key)
	}
}

func TestKeyedMutex_ConcurrentAccess(t *testing.T) {
	km := New()
	key := "testKey"
	var wg sync.WaitGroup

	itr := 1000
	j := 0

	for i := 0; i < itr; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			km.Lock(key)
			j++
			km.Unlock(key)
		}()
	}

	wg.Wait()

	if j != itr {
		t.Errorf("Expected j to be %d, got %d", itr, j)
	}
}

func TestKeyedMutex_Locked(t *testing.T) {
	km := New()

	key := "testKey"
	if km.Locked(key) {
		t.Errorf("Expected key %s to be initially unlocked", key)
	}

	km.Lock(key)
	if !km.Locked(key) {
		t.Errorf("Expected key %s to be locked", key)
	}

	km.Unlock(key)
	if km.Locked(key) {
		t.Errorf("Expected key %s to be unlocked after unlock", key)
	}
}

func TestKeyedMutex_HighConcurrency(t *testing.T) {
	km := New()
	const numKeys = 100
	const numGoroutines = 1000
	const iterations = 50

	keys := make([]string, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
	}

	var wg sync.WaitGroup
	counters := make([]int, numKeys)

	// Launch goroutines that randomly lock/unlock different keys
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(id)))

			for j := 0; j < iterations; j++ {
				keyIdx := r.Intn(numKeys)
				key := keys[keyIdx]

				// Try both regular lock and trylock
				if r.Intn(2) == 0 {
					km.Lock(key)
					counters[keyIdx]++
					km.Unlock(key)
				} else {
					if km.TryLock(key) {
						counters[keyIdx]++
						km.Unlock(key)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all locks are released
	for _, key := range keys {
		if km.Locked(key) {
			t.Errorf("Key %s still locked after test completion", key)
		}
	}
}

func TestKeyedMutex_DeadlockDetection(t *testing.T) {
	km := New()
	key := "testKey"

	// Lock once
	km.Lock(key)

	// Set up a channel to detect if we're deadlocked
	done := make(chan bool)
	go func() {
		// Try to acquire the same lock again - this shouldn't deadlock
		// because the locks aren't reentrant, but it should fail
		success := km.TryLock(key)
		if success {
			t.Errorf("TryLock succeeded on an already locked key")
			km.Unlock(key) // Clean up if it unexpectedly succeeded
		}
		done <- true
	}()

	// Wait with timeout to detect deadlock
	select {
	case <-done:
		// Good, no deadlock
	case <-time.After(time.Second):
		t.Fatalf("Deadlock detected")
	}

	km.Unlock(key)
}

func TestKeyedMutex_StressTest(t *testing.T) {
	km := New()
	const numKeys = 10
	const numOps = 100000

	var wg sync.WaitGroup

	// Track lock state for verification
	lockState := make([]int32, numKeys)

	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(opNum int) {
			defer wg.Done()

			keyIdx := opNum % numKeys
			key := fmt.Sprintf("key-%d", keyIdx)

			// Randomly choose between Lock and TryLock
			if opNum%3 == 0 {
				if km.TryLock(key) {
					// Verify no other goroutine thinks it has the lock
					if atomic.AddInt32(&lockState[keyIdx], 1) != 1 {
						t.Errorf("Lock collision detected on key %s", key)
					}

					// Small sleep to increase chance of race conditions
					time.Sleep(time.Microsecond)

					atomic.AddInt32(&lockState[keyIdx], -1)
					km.Unlock(key)
				}
			} else {
				km.Lock(key)

				// Verify no other goroutine thinks it has the lock
				if atomic.AddInt32(&lockState[keyIdx], 1) != 1 {
					t.Errorf("Lock collision detected on key %s", key)
				}

				// Small sleep to increase chance of race conditions
				time.Sleep(time.Microsecond)

				atomic.AddInt32(&lockState[keyIdx], -1)
				km.Unlock(key)
			}
		}(i)
	}

	wg.Wait()

	// Verify all locks are released
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		if km.Locked(key) {
			t.Errorf("Key %s still locked after test completion", key)
		}
	}
}

func TestKeyedMutex_UnlockNonExistentKey(t *testing.T) {
	km := New()
	key := "nonExistentKey"

	// Test that unlocking a non-existent key panics
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Unlocking a non-existent key should panic")
		}
	}()

	km.Unlock(key)
}

func TestKeyedMutex_LockedAfterPanic(t *testing.T) {
	km := New()
	key := "testKey"

	// Function that will lock, panic, and recover
	func() {
		defer func() {
			recover() // Recover from the panic
		}()

		km.Lock(key)
		panic("deliberate panic") // This should not leave the key locked
	}()

	// The key should still be locked after a panic
	if !km.Locked(key) {
		t.Errorf("Key should remain locked after panic")
	}

	// Clean up
	km.Unlock(key)
}

func TestKeyedMutex_ConcurrentDifferentKeys(t *testing.T) {
	km := New()
	const numKeys = 100

	// Should be able to lock different keys concurrently
	var wg sync.WaitGroup

	for i := 0; i < numKeys; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", idx)

			km.Lock(key)
			// Simulate work
			time.Sleep(10 * time.Millisecond)
			km.Unlock(key)
		}(i)
	}

	wg.Wait()

	// All keys should be unlocked
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		if km.Locked(key) {
			t.Errorf("Key %s should be unlocked", key)
		}
	}
}

func TestKeyedMutex_MemoryLeak(t *testing.T) {
	km := New()
	const numKeys = 10000

	// Lock and unlock many keys
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		km.Lock(key)
		km.Unlock(key)
	}

	// Check that the internal map doesn't retain entries
	// This is a white-box test that depends on implementation details
	mapSize := len(km.locks)
	if mapSize > 0 {
		t.Errorf("Expected empty map after all locks released, but found %d entries", mapSize)
	}
}
