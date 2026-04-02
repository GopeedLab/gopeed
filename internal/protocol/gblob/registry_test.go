package gblob

import (
	"errors"
	"testing"
)

func TestRegistry_RevokeOpenWritableStreamAbortsAndCleansUpAfterUnpin(t *testing.T) {
	registry := NewRegistry(t.TempDir())
	url, err := registry.CreateWritableStream(&CreateWritableStreamOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Pin(url); err != nil {
		t.Fatal(err)
	}

	src, err := registry.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if state := src.Snapshot().State; state != SourceStateOpen {
		t.Fatalf("expected initial open state, got %s", state)
	}

	if err := registry.Revoke(url); err != nil {
		t.Fatal(err)
	}

	src, err = registry.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := src.Snapshot()
	if snapshot.State != SourceStateAborted {
		t.Fatalf("expected revoked open source to become aborted, got %s", snapshot.State)
	}
	if !errors.Is(snapshot.Err, ErrSourceRevoked) {
		t.Fatalf("expected revoke error, got %v", snapshot.Err)
	}

	registry.Unpin(url)
	if _, err := registry.Get(url); !errors.Is(err, ErrSourceNotFound) {
		t.Fatalf("expected source cleanup after unpin, got %v", err)
	}
}
