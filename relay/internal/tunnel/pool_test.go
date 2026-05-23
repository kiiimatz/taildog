package tunnel

import (
	"testing"
)

func TestAllocateSpecific(t *testing.T) {
	p := NewPool(10000, 10010)

	port, err := p.Allocate(10000)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if port != 10000 {
		t.Fatalf("expected 10000, got %d", port)
	}
}

func TestAllocateConflictAutoAssign(t *testing.T) {
	p := NewPool(10000, 10010)
	_, _ = p.Allocate(10000)

	port, err := p.Allocate(10000)
	if err != nil {
		t.Fatalf("expected auto-assign, got: %v", err)
	}
	if port != 10001 {
		t.Fatalf("expected 10001, got %d", port)
	}
}

func TestAllocateAuto(t *testing.T) {
	p := NewPool(10000, 10010)

	port, err := p.Allocate(0)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if port < 10000 || port > 10010 {
		t.Fatalf("port %d out of range", port)
	}
}

func TestAllocateExhausted(t *testing.T) {
	p := NewPool(10000, 10002)
	for i := 10000; i <= 10002; i++ {
		if _, err := p.Allocate(i); err != nil {
			t.Fatalf("unexpected error allocating %d: %v", i, err)
		}
	}

	_, err := p.Allocate(0)
	if err == nil {
		t.Fatal("expected exhaustion error")
	}
}

func TestRelease(t *testing.T) {
	p := NewPool(10000, 10000)
	_, _ = p.Allocate(10000)

	if !p.IsAllocated(10000) {
		t.Fatal("expected 10000 to be allocated")
	}

	p.Release(10000)

	if p.IsAllocated(10000) {
		t.Fatal("expected 10000 to be released")
	}

	port, err := p.Allocate(10000)
	if err != nil || port != 10000 {
		t.Fatalf("expected 10000 after release, got %d err %v", port, err)
	}
}

func TestOutOfRangeAutoAssigns(t *testing.T) {
	// Requesting an out-of-range port falls through to auto-assign from the pool.
	p := NewPool(10000, 10010)
	port, err := p.Allocate(3000) // 3000 is outside [10000, 10010]
	if err != nil {
		t.Fatalf("expected auto-assign for out-of-range port, got: %v", err)
	}
	if port < 10000 || port > 10010 {
		t.Fatalf("expected auto-assigned port in range, got %d", port)
	}
}
