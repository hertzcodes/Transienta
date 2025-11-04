package comms

import (
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/uuid"
	"github.com/hertzcodes/transienta/go-sdk/internal/fbs"
)

func TestStartRequest_Serialize(t *testing.T) {
	id := uuid.NewString()
	tests := []struct {
		name     string
		request  *StartRequest
		expected struct {
			ID string
		}
	}{
		{
			name: "basic start request",
			request: &StartRequest{
				ID: id,
			},
			expected: struct {
				ID string
			}{
				ID: id,
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

			// Deserialize the Request wrapper first
			reqWrapper := fbs.GetRootAsRequest(serialized, 0)
			if reqWrapper == nil {
				t.Fatal("Deserialized request wrapper should not be nil")
			}

			// Check the request type
			if reqWrapper.RequestType() != fbs.RequestUnionStartRequest {
				t.Fatalf("Expected RequestUnionStartRequest, got %v", reqWrapper.RequestType())
			}

			// Extract the union and get the StartRequest
			var unionTable flatbuffers.Table
			if !reqWrapper.Request(&unionTable) {
				t.Fatal("Failed to get union table")
			}

			deserialized := &fbs.StartRequest{}
			deserialized.Init(unionTable.Bytes, unionTable.Pos)

			if string(deserialized.Id()) != tt.expected.ID {
				t.Errorf("ID should match: expected %v, got %v", tt.expected.ID, string(deserialized.Id()))
			}
		})
	}
}

func TestEndRequest_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		request  *EndRequest
		expected struct {
			args   uint32
			caller string
			deps   []string
			resp   []byte
		}
	}{
		{
			name: "basic end request",
			request: &EndRequest{
				Args:   100,
				Caller: "test-caller",
				Deps:   []string{"dep1", "dep2"},
				Resp:   []byte("response data"),
			},
			expected: struct {
				args   uint32
				caller string
				deps   []string
				resp   []byte
			}{
				args:   100,
				caller: "test-caller",
				deps:   []string{"dep1", "dep2"},
				resp:   []byte("response data"),
			},
		},
		{
			name: "end request with unicode strings",
			request: &EndRequest{
				Args:   300,
				Caller: "测试调用者",
				Deps:   []string{"依赖1", "依赖2", "dependency-3"},
				Resp:   []byte("响应数据 with unicode"),
			},
			expected: struct {
				args   uint32
				caller string
				deps   []string
				resp   []byte
			}{
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

			// Deserialize the Request wrapper first
			reqWrapper := fbs.GetRootAsRequest(serialized, 0)
			if reqWrapper == nil {
				t.Fatal("Deserialized request wrapper should not be nil")
			}

			// Check the request type
			if reqWrapper.RequestType() != fbs.RequestUnionEndRequest {
				t.Fatalf("Expected RequestUnionEndRequest, got %v", reqWrapper.RequestType())
			}

			// Extract the union and get the EndRequest
			var unionTable flatbuffers.Table
			if !reqWrapper.Request(&unionTable) {
				t.Fatal("Failed to get union table")
			}

			deserialized := &fbs.EndRequest{}
			deserialized.Init(unionTable.Bytes, unionTable.Pos)

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
			key string
		}
	}{
		{
			name: "basic invalidation request",
			request: &InvalidationRequest{
				Key: "test-key",
			},
			expected: struct {
				key string
			}{
				key: "test-key",
			},
		},
		{
			name: "empty key",
			request: &InvalidationRequest{
				Key: "",
			},
			expected: struct {
				key string
			}{
				key: "",
			},
		},
		{
			name: "long key",
			request: &InvalidationRequest{
				Key: "very-long-key-with-many-characters-that-should-still-work-correctly",
			},
			expected: struct {
				key string
			}{
				key: "very-long-key-with-many-characters-that-should-still-work-correctly",
			},
		},
		{
			name: "unicode key",
			request: &InvalidationRequest{
				Key: "测试键-测试",
			},
			expected: struct {
				key string
			}{
				key: "测试键-测试",
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

			// Deserialize the Request wrapper first
			reqWrapper := fbs.GetRootAsRequest(serialized, 0)
			if reqWrapper == nil {
				t.Fatal("Deserialized request wrapper should not be nil")
			}

			// Check the request type
			if reqWrapper.RequestType() != fbs.RequestUnionInvalidationRequest {
				t.Fatalf("Expected RequestUnionInvalidationRequest, got %v", reqWrapper.RequestType())
			}

			// Extract the union and get the InvalidationRequest
			var unionTable flatbuffers.Table
			if !reqWrapper.Request(&unionTable) {
				t.Fatal("Failed to get union table")
			}

			deserialized := &fbs.InvalidationRequest{}
			deserialized.Init(unionTable.Bytes, unionTable.Pos)

			if string(deserialized.Key()) != tt.expected.key {
				t.Errorf("Key should match: expected %v, got %v", tt.expected.key, string(deserialized.Key()))
			}
		})
	}
}
