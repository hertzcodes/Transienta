## Transienta

Transienta is a version-aware invalidation and caching coordination layer for microservices. It tracks request dependencies across services and invalidates cached results when underlying data changes, using a lightweight wire protocol over NanoMsg and a simple Go SDK for instrumentation.

### Why

- **Consistency**: Cached responses are only reused if none of their dependent data changed since the request started.
- **Simplicity**: Instrument code with a few calls: start a request, add dependencies as you read, invalidate on writes, finish the request.
- **Performance**: Uses a monotonic timestamped version store for constant-time conflict checks; messaging uses FlatBuffers for zero-copy serialization.

## Architecture

- **Manager (Rust)**: A daemon that receives events over NanoMsg, maintains a dependency graph and a per-key version store, and decides whether requests are valid to cache.
  - Versioning logic: `VersionedHistoryStorage` increments a global timestamp on writes/invalidations and validates requests by comparing dependency versions with the request start timestamp.
  - Transport: NanoMsg Pair; schema encoded via FlatBuffers.
- **SDK (Go)**: A client library to instrument services.
  - Exposes `Start`, `AddDependency`, `Invalidate`, `Finish`.

Flow:
1. Service should run Start() before a request is being processed.
2. Service reads data and calls `AddDependency(...)` per dependency path (e.g., `users/123`).
3. On writes / updates, service invokes `Invalidate(key)` to notify the cache that a dependency has changed.
4. Service sends `Finish(resp)`; manager validates: request is valid if none of the `deps` changed after the start timestamp. If valid, it records dependency graph and sends the save request to save the response in the `caller` cache.

## Repository layout

- `src/`
  - `main.rs`, `app.rs`: wiring and setup
  - `manager/manager.rs`: message loop and request handling
  - `adapters/storage/version_storage.rs`: version store, validation, dependency graph
  - `adapters/comms/`: NanoMsg + FlatBuffers helpers
  - `cache/`: cache provider abstraction (Redis stub)
  - `config/`: YAML config loader/types
- `sdk/go/`: Go SDK (client and internal comms/fbs)
- `fbs/requests.fbs`: FlatBuffers schema
- `build/`: Dockerfiles (alpine, scratch)

## Build (Manager)

Prerequisites: Rust toolchain (edition 2024), NanoMsg (mangos for Go SDK; Rust uses `nng` crate).

```bash
cargo build --release
```

Run the manager (default config path is `/etc/transienta/config.yaml` unless overridden):

```bash
export TRANSIENTA_CONFIG_PATH=/path/to/config.yaml
cargo run --release
```
## Configuration

Provide a YAML file; see `example-config.yaml`:

```yaml
manager:
    name: shard-1
    port: 5532
    shard_id: 0
    neighbours:
        shard-2: "127.0.0.1"

cache:
    type: "redis"
    config:
        tls: false
        port: 6379
        db: 0
        host: localhost
        username: ""
        password: ""
```

If `TRANSIENTA_CONFIG_PATH` is not set, the manager reads `/etc/transienta/config.yaml`.

## Go SDK quickstart

```go
import (
    "context"
    client "github.com/hertzcodes/transienta/go-sdk/client"
)

cfg := client.ClientConfig{
    ManagerIP: "127.0.0.1:5532", // informational; used in hashing
    SocketURL: "tcp://127.0.0.1:5600", // your NanoMsg endpoint
    On:        true,                     // enable/disable SDK
}
c := client.New(cfg)

base := context.Background()
ctx := c.Start([]byte("request-bytes"), "my-service", base)
defer c.Finish(ctx, nil)

// When reading data or calling other services:
_ = c.AddDependency(ctx, "users/123")
//
//
//      Request processes here
//
_ = c.AddDependency(ctx, "orders/987")

// When writing/updating data:
c.Invalidate("users/123")
// request does not cache since users/123 is invalidated.
```

Notes:
- `Start` creates a contextual ID and sends `StartRequest`.
- `AddDependency` accumulates dependency paths for the request.
- `Invalidate` emits `InvalidationRequest` and persists to an outbox on failure.
- `Finish` sends `EndRequest` with the dependency list and request args hash.

## How validation works

Inside the manager (`VersionedHistoryStorage`):
- Every invalidation/write increments a global `current_timestamp` and updates the version for the impacted dependency path.
- When a request starts, its `call_id` is mapped to the current timestamp.
- On `EndRequest`, the manager verifies all dependency paths have a version <= the start timestamp; if so, the request is valid to cache and the dependency and subscriber maps are updated.

This ensures cached results are only reused when none of the dependencies were modified after the request began.

## Running with Docker

Two Dockerfiles are provided in `build/` (`Dockerfile-alpine`, `Dockerfile-scratch`). Build and run as desired, mounting your config path to the container and exposing the NanoMsg/manager ports you use.

## Testing and benchmarks

- Rust unit tests: `cargo test` cover validation edge cases.
- Go SDK tests/benchmarks: `sdk/go/client/client_test.go` includes concurrency tests; run `go test ./...` (use `-race` for race detection).

## License
See `LICENSE`.
