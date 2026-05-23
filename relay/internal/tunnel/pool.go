// Package tunnel manages port allocation and the tunnel/client registry.
package tunnel

import (
	"fmt"
	"sync"
)

// Pool is a thread-safe port allocation pool over the range [min, max].
type Pool struct {
	mu        sync.Mutex
	min, max  int
	allocated map[int]struct{}
	next      int // next candidate for sequential scan
}

// NewPool creates a Pool over the inclusive range [min, max].
func NewPool(min, max int) *Pool {
	return &Pool{
		min:       min,
		max:       max,
		allocated: make(map[int]struct{}),
		next:      min,
	}
}

// Allocate returns a port to use for a new tunnel.
//
//   - If requested is non-zero and the port is free, it is allocated and returned.
//   - If requested is zero or the requested port is already taken, the pool
//     finds the next free port in the range and returns it.
//
// Returns an error when the pool is exhausted.
func (p *Pool) Allocate(requested int) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if requested != 0 {
		if !p.inRange(requested) {
			return 0, fmt.Errorf("pool: port %d out of range [%d, %d]", requested, p.min, p.max)
		}
		if _, taken := p.allocated[requested]; !taken {
			p.allocated[requested] = struct{}{}
			return requested, nil
		}
		// Conflict — fall through to auto-assign
	}

	// Sequential scan from p.next, wrapping once.
	for i := 0; i < (p.max - p.min + 1); i++ {
		candidate := p.next
		p.advance()
		if _, taken := p.allocated[candidate]; !taken {
			p.allocated[candidate] = struct{}{}
			return candidate, nil
		}
	}
	return 0, fmt.Errorf("pool: no free ports in range %d–%d", p.min, p.max)
}

// Release returns a port to the free pool.
func (p *Pool) Release(port int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.allocated, port)
}

// IsAllocated reports whether port is currently in use.
func (p *Pool) IsAllocated(port int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.allocated[port]
	return ok
}

func (p *Pool) inRange(port int) bool {
	return port >= p.min && port <= p.max
}

func (p *Pool) advance() {
	p.next++
	if p.next > p.max {
		p.next = p.min
	}
}
