package comms

import (
	"testing"

	"github.com/hertzcodes/transienta/go-sdk/internal/fbs"
)

func TestStartRequest_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		request  *StartRequest
		expected struct {
			number fbs.RequestType
			args   uint32
		}
	}{
		{
			name: "basic start request",
			request: &StartRequest{
				Number: Start,
				Args:   42,
			},
			expected: struct {
				number fbs.RequestType
				args   uint32
			}{
				number: fbs.RequestTypeStart,
				args:   42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize the request
			serialized := tt.request.Serialize()
			if len(serialized) == 0 {
				t.Fatal("Serialized data should not be empty")
			}

			// Deserialize and verify
			deserialized := fbs.GetRootAsStartRequest(serialized, 0)
			if deserialized == nil {
				t.Fatal("Deserialized request should not be nil")
			}

			if deserialized.Number() != tt.expected.number {
				t.Errorf("Request type should match: expected %v, got %v", tt.expected.number, deserialized.Number())
			}
			if deserialized.Args() != tt.expected.args {
				t.Errorf("Args should match: expected %v, got %v", tt.expected.args, deserialized.Args())
			}
		})
	}
}

func TestEndRequest_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		request  *EndRequest
		expected struct {
			number fbs.RequestType
			args   uint32
			caller string
			deps   []string
			resp   []byte
		}
	}{
		{
			name: "basic end request",
			request: &EndRequest{
				Number: End,
				Args:   100,
				Caller: "test-caller",
				Deps:   []string{"dep1", "dep2"},
				Resp:   []byte("response data"),
			},
			expected: struct {
				number fbs.RequestType
				args   uint32
				caller string
				deps   []string
				resp   []byte
			}{
				number: fbs.RequestTypeEnd,
				args:   100,
				caller: "test-caller",
				deps:   []string{"dep1", "dep2"},
				resp:   []byte("response data"),
			},
		},
		{
			name: "end request with unicode strings",
			request: &EndRequest{
				Number: End,
				Args:   300,
				Caller: "测试调用者",
				Deps:   []string{"依赖1", "依赖2", "dependency-3"},
				Resp:   []byte("响应数据 with unicode"),
			},
			expected: struct {
				number fbs.RequestType
				args   uint32
				caller string
				deps   []string
				resp   []byte
			}{
				number: fbs.RequestTypeEnd,
				args:   300,
				caller: "测试调用者",
				deps:   []string{"依赖1", "依赖2", "dependency-3"},
				resp:   []byte("响应数据 with unicode"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize the request
			serialized := tt.request.Serialize()
			if len(serialized) == 0 {
				t.Fatal("Serialized data should not be empty")
			}

			// Deserialize and verify
			deserialized := fbs.GetRootAsEndRequest(serialized, 0)
			if deserialized == nil {
				t.Fatal("Deserialized request should not be nil")
			}

			if deserialized.Number() != tt.expected.number {
				t.Errorf("Request type should match: expected %v, got %v", tt.expected.number, deserialized.Number())
			}
			if deserialized.Args() != tt.expected.args {
				t.Errorf("Args should match: expected %v, got %v", tt.expected.args, deserialized.Args())
			}
			if string(deserialized.Caller()) != tt.expected.caller {
				t.Errorf("Caller should match: expected %v, got %v", tt.expected.caller, string(deserialized.Caller()))
			}

			// Check dependencies
			if deserialized.DepsLength() != len(tt.expected.deps) {
				t.Errorf("Deps length should match: expected %d, got %d", len(tt.expected.deps), deserialized.DepsLength())
			}
			for i := 0; i < deserialized.DepsLength(); i++ {
				if string(deserialized.Deps(i)) != tt.expected.deps[i] {
					t.Errorf("Dep %d should match: expected %v, got %v", i, tt.expected.deps[i], string(deserialized.Deps(i)))
				}
			}

			// Check response
			if deserialized.RespLength() != len(tt.expected.resp) {
				t.Errorf("Resp length should match: expected %d, got %d", len(tt.expected.resp), deserialized.RespLength())
			}
			actualResp := deserialized.RespBytes()
			if len(actualResp) != len(tt.expected.resp) {
				t.Errorf("Response data length should match: expected %d, got %d", len(tt.expected.resp), len(actualResp))
			}
			for i, b := range actualResp {
				if b != tt.expected.resp[i] {
					t.Errorf("Response data should match at index %d: expected %v, got %v", i, tt.expected.resp[i], b)
				}
			}
		})
	}
}

func TestInvalidationRequest_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		request  *InvalidationRequest
		expected struct {
			number fbs.RequestType
			key    string
		}
	}{
		{
			name: "basic invalidation request",
			request: &InvalidationRequest{
				Number: Invalidation,
				Key:    "test-key",
			},
			expected: struct {
				number fbs.RequestType
				key    string
			}{
				number: fbs.RequestTypeInvalidation,
				key:    "test-key",
			},
		},
		{
			name: "empty key",
			request: &InvalidationRequest{
				Number: Invalidation,
				Key:    "",
			},
			expected: struct {
				number fbs.RequestType
				key    string
			}{
				number: fbs.RequestTypeInvalidation,
				key:    "",
			},
		},
		{
			name: "long key",
			request: &InvalidationRequest{
				Number: Invalidation,
				Key:    "very-long-key-with-many-characters-that-should-still-work-correctly",
			},
			expected: struct {
				number fbs.RequestType
				key    string
			}{
				number: fbs.RequestTypeInvalidation,
				key:    "very-long-key-with-many-characters-that-should-still-work-correctly",
			},
		},
		{
			name: "unicode key",
			request: &InvalidationRequest{
				Number: Invalidation,
				Key:    "测试键-测试",
			},
			expected: struct {
				number fbs.RequestType
				key    string
			}{
				number: fbs.RequestTypeInvalidation,
				key:    "测试键-测试",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize the request
			serialized := tt.request.Serialize()
			if len(serialized) == 0 {
				t.Fatal("Serialized data should not be empty")
			}

			// Deserialize and verify
			deserialized := fbs.GetRootAsInvalidationRequest(serialized, 0)
			if deserialized == nil {
				t.Fatal("Deserialized request should not be nil")
			}

			if deserialized.Number() != tt.expected.number {
				t.Errorf("Request type should match: expected %v, got %v", tt.expected.number, deserialized.Number())
			}
			if string(deserialized.Key()) != tt.expected.key {
				t.Errorf("Key should match: expected %v, got %v", tt.expected.key, string(deserialized.Key()))
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("nil slices in EndRequest", func(t *testing.T) {
		request := &EndRequest{
			Number: End,
			Args:   0,
			Caller: "",
			Deps:   nil,
			Resp:   nil,
		}

		serialized := request.Serialize()
		if len(serialized) == 0 {
			t.Fatal("Serialized data should not be empty")
		}

		deserialized := fbs.GetRootAsEndRequest(serialized, 0)
		if deserialized == nil {
			t.Fatal("Deserialized request should not be nil")
		}

		if deserialized.DepsLength() != 0 {
			t.Errorf("Deps length should be 0: got %d", deserialized.DepsLength())
		}
		if deserialized.RespLength() != 0 {
			t.Errorf("Resp length should be 0: got %d", deserialized.RespLength())
		}
	})

	t.Run("very large response data", func(t *testing.T) {
		// Create a large response data
		largeResp := make([]byte, 10000)
		for i := range largeResp {
			largeResp[i] = byte(i % 256)
		}

		request := &EndRequest{
			Number: End,
			Args:   0,
			Caller: "large-data-caller",
			Deps:   []string{"dep1"},
			Resp:   largeResp,
		}

		serialized := request.Serialize()
		if len(serialized) == 0 {
			t.Fatal("Serialized data should not be empty")
		}

		deserialized := fbs.GetRootAsEndRequest(serialized, 0)
		if deserialized == nil {
			t.Fatal("Deserialized request should not be nil")
		}

		actualResp := deserialized.RespBytes()
		if len(actualResp) != len(largeResp) {
			t.Errorf("Response length should match: expected %d, got %d", len(largeResp), len(actualResp))
		}
		for i, b := range actualResp {
			if b != largeResp[i] {
				t.Errorf("Response data should match at index %d: expected %v, got %v", i, largeResp[i], b)
			}
		}
	})
}
