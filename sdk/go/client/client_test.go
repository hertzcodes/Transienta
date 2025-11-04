package client

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.nanomsg.org/mangos/v3/protocol/rep"
	_ "go.nanomsg.org/mangos/v3/transport/all"
)

// Test utilities and helper functions
func createTestConfig() ClientConfig {
	return ClientConfig{
		ManagerIP: "localhost:8080",
		SocketURL: "tcp://127.0.0.1:5600",
		On: true,
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
	ctx := client.Start(req, caller, baseCtx)
	if _, ok := ctx.(*Ctx); !ok {
		t.Errorf("failed to cast")
	}
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

	ctx := client.Start(req, caller, baseCtx)

	if ctx == nil {
		t.Fatal("StartRequest returned nil context")
	}

	casted := ctx.(*Ctx)

	if casted.caller != caller {
		t.Errorf("Expected caller %s, got %s", caller, casted.caller)
	}

	if casted.args == 0 {
		t.Error("Args should be hashed and non-zero")
	}

	// Check that the context was added to deps
	client.mu.RLock()
	if _, exists := client.deps[casted.id]; !exists {
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

	ctx := client.Start(req, caller, baseCtx)

	key := "test-key"
	err := client.AddDependency(ctx, key)
	if err != nil {
		t.Errorf("AddDependency should not return error for valid context, got: %v", err)
	}
	casted := ctx.(*Ctx)
	if len(client.deps[casted.id]) != 1 {
		t.Errorf("AddDependency should add the key to the context, got: %d", len(client.deps[casted.id]))
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

	ctx := client.Start(req, caller, baseCtx)

	// Add some dependencies
	client.AddDependency(ctx, "key1")
	client.AddDependency(ctx, "key2")

	resp := "test response"

	// This should not panic
	client.Finish(ctx, resp)

	// Check that the context was removed from deps
	casted := ctx.(*Ctx)
	if _, exists := client.deps[casted.id]; exists {
		t.Error("Context should be removed from deps after EndRequest")
	}
}

// Concurrent tests
func TestConcurrentStartRequest(t *testing.T) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	const numGoroutines = 100
	var wg sync.WaitGroup
	contexts := make([]context.Context, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			req := []byte("test request")
			caller := "test-caller"
			baseCtx := context.Background()
			contexts[index] = client.Start(req, caller, baseCtx)
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
		casted := ctx.(*Ctx)
		if ids[casted.id] {
			t.Errorf("Duplicate context ID found: %s", casted.id)
		}
		ids[casted.id] = true
	}

	// Check that all contexts are in the deps map
	client.mu.RLock()
	for i, ctx := range contexts {
		casted := ctx.(*Ctx)
		if _, exists := client.deps[casted.id]; !exists {
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
	ctx := client.Start(req, caller, baseCtx)

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
			ctx := client.Start(req, caller, baseCtx)

			// Add some dependencies
			client.AddDependency(ctx, "key1")
			client.AddDependency(ctx, "key2")

			// End the request
			resp := "test response"
			client.Finish(ctx, resp)
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
			ctx := client.Start(req, caller, baseCtx)

			// Add dependencies
			client.AddDependency(ctx, "key1")
			client.AddDependency(ctx, "key2")

			// Invalidate some keys
			client.Invalidate("some-key")

			// End request
			resp := "test response"
			client.Finish(ctx, resp)
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
			ctx := client.Start(req, caller, baseCtx)

			// Rapidly add and remove dependencies
			for j := 0; j < 10; j++ {
				client.AddDependency(ctx, "key")
				client.Invalidate("key")
			}

			client.Finish(ctx, "response")
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
		ctx := client.Start(req, caller, baseCtx)
		client.Finish(ctx, "response")
	}
}

func BenchmarkAddDependency(b *testing.B) {
	resetSingleton()

	config := createTestConfig()
	client := New(config)

	req := []byte("test request")
	caller := "test-caller"
	baseCtx := context.Background()
	ctx := client.Start(req, caller, baseCtx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.AddDependency(ctx, "key")
	}

	client.Finish(ctx, "response")
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
			ctx := client.Start(req, caller, baseCtx)
			client.Finish(ctx, "response")
		}
	})
}

func TestStartRequestWithSocket(t *testing.T) {
    resetSingleton()
    config := createTestConfig()
    
    // Use a channel to signal when listener is ready
    ready := make(chan bool)
    
    go func() {
        sock, err := rep.NewSocket()
        if err != nil {
            panic(err)
        }
        if err = sock.Listen(config.SocketURL); err != nil {
            panic(err)
        }
        fmt.Println("NanoMsg listener connected and ready")
        
        // Signal that listener is ready
        close(ready)
        
        for {
            msg, err := sock.Recv()
            if err != nil {
                // Don't panic on normal errors, just log and continue
                fmt.Printf("Error receiving: %v\n", err)
                continue
            }
            
            fmt.Printf("Received message: %s\n", string(msg))
            time.Sleep(800 * time.Millisecond)
            // Send a response back
            err = sock.Send([]byte("1234"))
            if err != nil {
                fmt.Printf("Error sending response: %v\n", err)
            }
        }
    }()
    
    // Wait for listener to be ready
    <-ready
    time.Sleep(100 * time.Millisecond) // Small additional delay
    
    client := New(config)
    
    req := []byte("test request")
    caller := "test-caller"
    baseCtx := context.Background()
    result := client.Start(req, caller, baseCtx)
    if result == nil {
        t.Error("expected a result ctx")
    }
}