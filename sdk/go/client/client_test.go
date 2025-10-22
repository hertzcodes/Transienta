package client

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// Test utilities and helper functions
func createTestConfig() ClientConfig {
	return ClientConfig{
		ManagerIP: "localhost:8080",
		Cache:     RedisConfig{},
	}
}

func resetSingleton() {
	instance = nil
	once = sync.Once{}
}

// Non-concurrent tests
func TestNew(t *testing.T) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	if client == nil {
		t.Fatal("New() returned nil client")
	}

	if client.config.ManagerIP != config.ManagerIP {
		t.Errorf("Expected ManagerIP %s, got %s", config.ManagerIP, client.config.ManagerIP)
	}

	if client.deps == nil {
		t.Error("Client deps map should be initialized")
	}

}

func TestNew_Singleton(t *testing.T) {
	resetSingleton()

	config1 := createTestConfig()
	config2 := createTestConfig()
	config2.ManagerIP = "different:8080"

	client1 := New(config1)
	client2 := New(config2)

	if client1 != client2 {
		t.Error("New() should return the same singleton instance")
	}

	// The first config should be used due to singleton pattern
	if client1.config.ManagerIP != config1.ManagerIP {
		t.Errorf("Expected first config to be used, got %s", client1.config.ManagerIP)
	}
}

func TestGetInstance_Success(t *testing.T) {
	resetSingleton()

	config := createTestConfig()
	New(config)

	client := GetInstance()
	if client == nil {
		t.Fatal("GetInstance() returned nil")
	}
}

func TestCtxCast(t *testing.T) {
	resetSingleton()
	config := createTestConfig()
	client := New(config)

	req := []byte("test request")
	caller := "test-caller"
	baseCtx := context.WithValue(context.Background(), "test", "1234")
	ctx := client.StartRequest(req, caller, baseCtx)

	func(c context.Context) {

		casted, ok := c.(*Ctx)
		if !ok {
			t.Errorf("expected context to be casted to Client Ctx")
		}

		if casted.id != ctx.id || casted.args != ctx.args || casted.caller != ctx.caller {
			t.Errorf("expected %v, %v to be same", ctx, casted)
		}

		if casted.Value("test") == nil {
			t.Error("expected casted to have value of 1234 not nil")
		}

	}(ctx)
}

func TestGetInstance_Panic(t *testing.T) {
	resetSingleton()

	defer func() {
		if r := recover(); r == nil {
			t.Error("GetInstance() should panic when no instance exists")
		}
	}()

	GetInstance()
}

func TestStartRequest(t *testing.T) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	req := []byte("test request")
	caller := "test-caller"
	baseCtx := context.Background()

	ctx := client.StartRequest(req, caller, baseCtx)

	if ctx == nil {
		t.Fatal("StartRequest returned nil context")
	}

	if ctx.caller != caller {
		t.Errorf("Expected caller %s, got %s", caller, ctx.caller)
	}

	if ctx.args == 0 {
		t.Error("Args should be hashed and non-zero")
	}

	// Check that the context was added to deps
	client.mu.RLock()
	if _, exists := client.deps[ctx.id]; !exists {
		t.Error("Context ID should be added to deps map")
	}
	client.mu.RUnlock()
}

func TestAddDependency_ValidContext(t *testing.T) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	req := []byte("test request")
	caller := "test-caller"
	baseCtx := context.Background()

	ctx := client.StartRequest(req, caller, baseCtx)

	key := "test-key"
	err := client.AddDependency(ctx, key)
	if err != nil {
		t.Errorf("AddDependency should not return error for valid context, got: %v", err)
	}

	if len(client.deps[ctx.id]) != 1 {
		t.Errorf("AddDependency should add the key to the context, got: %d", len(client.deps[ctx.id]))
	}
}

func TestAddDependency_InvalidContext(t *testing.T) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	// Create a context that's not in the deps map
	fakeCtx := &Ctx{
		id:      uuid.New(),
		caller:  "fake-caller",
		args:    123,
		Context: context.Background(),
	}

	key := "test-key"
	err := client.AddDependency(fakeCtx, key)

	if err == nil {
		t.Error("AddDependency should return error for invalid context")
	}

	expectedErr := "invalid ctx (id not found in requests)"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %s, got %s", expectedErr, err.Error())
	}
}

func TestEndRequest(t *testing.T) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	req := []byte("test request")
	caller := "test-caller"
	baseCtx := context.Background()

	ctx := client.StartRequest(req, caller, baseCtx)

	// Add some dependencies
	client.AddDependency(ctx, "key1")
	client.AddDependency(ctx, "key2")

	resp := "test response"

	// This should not panic
	client.EndRequest(ctx, resp)

	// Check that the context was removed from deps
	client.mu.RLock()
	if _, exists := client.deps[ctx.id]; exists {
		t.Error("Context should be removed from deps after EndRequest")
	}
	client.mu.RUnlock()
}

// Concurrent tests
func TestConcurrentStartRequest(t *testing.T) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	const numGoroutines = 100
	var wg sync.WaitGroup
	contexts := make([]*Ctx, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			req := []byte("test request")
			caller := "test-caller"
			baseCtx := context.Background()
			contexts[index] = client.StartRequest(req, caller, baseCtx)
		}(i)
	}

	wg.Wait()

	// Check that all contexts were created and have unique IDs
	ids := make(map[uuid.UUID]bool)
	for i, ctx := range contexts {
		if ctx == nil {
			t.Errorf("Context %d is nil", i)
			continue
		}

		if ids[ctx.id] {
			t.Errorf("Duplicate context ID found: %s", ctx.id)
		}
		ids[ctx.id] = true
	}

	// Check that all contexts are in the deps map
	client.mu.RLock()
	for i, ctx := range contexts {
		if _, exists := client.deps[ctx.id]; !exists {
			t.Errorf("Context %d not found in deps map", i)
		}
	}
	client.mu.RUnlock()
}

func TestConcurrentAddDependency(t *testing.T) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	// Create a context first
	req := []byte("test request")
	caller := "test-caller"
	baseCtx := context.Background()
	ctx := client.StartRequest(req, caller, baseCtx)

	const numGoroutines = 100
	var wg sync.WaitGroup
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			key := "test-key"
			errors[index] = client.AddDependency(ctx, key)
		}(i)
	}

	wg.Wait()

	// All operations should succeed
	for i, err := range errors {
		if err != nil {
			t.Errorf("Goroutine %d got error: %v", i, err)
		}
	}
}

func TestConcurrentEndRequest(t *testing.T) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	const numGoroutines = 100
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Create a new context for each goroutine
			req := []byte("test request")
			caller := "test-caller"
			baseCtx := context.Background()
			ctx := client.StartRequest(req, caller, baseCtx)

			// Add some dependencies
			client.AddDependency(ctx, "key1")
			client.AddDependency(ctx, "key2")

			// End the request
			resp := "test response"
			client.EndRequest(ctx, resp)
		}(i)
	}

	wg.Wait()

	// All contexts should be cleaned up
	client.mu.RLock()
	if len(client.deps) != 0 {
		t.Errorf("Expected empty deps map, got %d entries", len(client.deps))
	}
	client.mu.RUnlock()
}

func TestConcurrentMixedOperations(t *testing.T) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	const numGoroutines = 50
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Mix of operations
			req := []byte("test request")
			caller := "test-caller"
			baseCtx := context.Background()
			ctx := client.StartRequest(req, caller, baseCtx)

			// Add dependencies
			client.AddDependency(ctx, "key1")
			client.AddDependency(ctx, "key2")

			// Invalidate some keys
			client.Invalidate("some-key")

			// End request
			resp := "test response"
			client.EndRequest(ctx, resp)
		}(i)
	}

	wg.Wait()

	// All contexts should be cleaned up
	client.mu.RLock()
	if len(client.deps) != 0 {
		t.Errorf("Expected empty deps map, got %d entries", len(client.deps))
	}
	client.mu.RUnlock()
}

func TestRaceConditionDetection(t *testing.T) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	// This test is designed to help detect race conditions
	// Run with: go test -race

	const numGoroutines = 1000
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			req := []byte("test request")
			caller := "test-caller"
			baseCtx := context.Background()
			ctx := client.StartRequest(req, caller, baseCtx)

			// Rapidly add and remove dependencies
			for j := 0; j < 10; j++ {
				client.AddDependency(ctx, "key")
				client.Invalidate("key")
			}

			client.EndRequest(ctx, "response")
		}(i)
	}

	wg.Wait()
}

// Benchmark tests
func BenchmarkStartRequest(b *testing.B) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	req := []byte("test request")
	caller := "test-caller"
	baseCtx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := client.StartRequest(req, caller, baseCtx)
		client.EndRequest(ctx, "response")
	}
}

func BenchmarkAddDependency(b *testing.B) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	req := []byte("test request")
	caller := "test-caller"
	baseCtx := context.Background()
	ctx := client.StartRequest(req, caller, baseCtx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.AddDependency(ctx, "key")
	}

	client.EndRequest(ctx, "response")
}

func BenchmarkConcurrentStartRequest(b *testing.B) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	req := []byte("test request")
	caller := "test-caller"
	baseCtx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx := client.StartRequest(req, caller, baseCtx)
			client.EndRequest(ctx, "response")
		}
	})
}
