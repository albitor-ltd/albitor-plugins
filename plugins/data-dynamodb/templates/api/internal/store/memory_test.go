package store_test

import (
	"testing"

	"webapp/internal/store"
	"webapp/internal/store/storetest"
)

// The in-memory adapter is what a local run and the app's specs use, so it
// must pass the same suite as the deployed adapter.
func TestMemoryConformance(t *testing.T) {
	storetest.Run(t, func(t *testing.T) store.Store { return store.NewMemory() })
}
