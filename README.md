# kemu - Keyed Mutex for Go

`kemu` is a simple, efficient implementation of a keyed mutex (lock map) for Go. It provides a way to lock operations based on string keys, allowing concurrent access to different keys while serializing access to the same key.

## Features

- Lock operations by string keys
- Non-blocking TryLock operation
- Clean memory management (no leaks from unused locks)
- Thread-safe implementation
- Simple, easy-to-use API

## Installation

```bash
go get github.com/modfin/kemu
```

## Usage

```go
package main

import (
    "fmt"
    "github.com/modfin/kemu"
)

func main() {
    // Create a new keyed mutex
    km := kemu.New()
    
    // Lock based on a key
    key := "user-123"
    km.Lock(key)
    
    // Critical section for this key
    // ... perform operations that need exclusive access to this key ...
    
    // Unlock when done
    km.Unlock(key)
    
    // Try to acquire a lock without blocking
    if km.TryLock(key) {
        // Lock acquired
        // ... do something ...
        km.Unlock(key)
    } else {
        // Lock not acquired, do something else
        fmt.Println("Could not acquire lock for", key)
    }
    
    // Check if a key is currently locked
    if km.Locked(key) {
        fmt.Println("Key is locked")
    }
}
```

## API Reference

### `New() *Mutex`

Creates a new keyed mutex instance.

### `Lock(key string)`

Acquires a lock for the specified key. If the key is already locked, this will block until the lock becomes available.

### `TryLock(key string) bool`

Attempts to acquire a lock for the specified key without blocking. Returns `true` if the lock was acquired, `false` otherwise.

### `Unlock(key string)`

Releases a lock for the specified key. Panics if the key is not locked.

### `Locked(key string) bool`

Returns `true` if the key is currently locked, `false` otherwise.

## Implementation Details

- The keyed mutex uses a map of string keys to lock entries
- Each lock entry contains a mutex and a reference count
- The implementation is not reentrant (the same goroutine cannot lock the same key multiple times)
- Locks are automatically cleaned up when the last reference is released, preventing memory leaks

## Concurrency Safety

`kemu` is designed for high-concurrency environments and has been thoroughly tested with concurrent access patterns. It is safe to use from multiple goroutines simultaneously.

## License

[MIT License](LICENSE.md)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.