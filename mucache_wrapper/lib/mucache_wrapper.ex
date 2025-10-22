defmodule MucacheWrapper do
  @moduledoc """
  MuCache Wrapper - Dapr Middleware Implementation
  
  This module implements the wrapper/interceptor component from the MuCache research paper
  as a Dapr middleware for Kubernetes microservices.
  
  ## Architecture
  
  The wrapper sits between Dapr and your microservice, intercepting HTTP requests:
  
  ```
  Client → Dapr → MuCache Wrapper → Your Service
                      ↓ ZeroMQ
                 Cache Manager (Rust)
  ```
  
  ## Key Features
  
  - **Dapr Middleware Integration**: Registers as HTTP middleware in Dapr pipeline
  - **ZeroMQ Communication**: High-performance async messaging with Cache Manager  
  - **Cache Hit/Miss Logic**: Implements exact MuCache protocol from paper
  - **Kubernetes Native**: Designed for cloud-native microservice deployments
  
  ## Protocol Implementation
  
  ### Cache Miss Flow
  1. Dapr routes request to wrapper middleware
  2. Wrapper checks Dapr state store (Redis) - MISS
  3. Wrapper sends Start(call_args) via ZMQ to Cache Manager (async)
  4. Wrapper forwards request to service via Dapr service invocation
  5. Service responds with result and readset  
  6. Wrapper sends End(call_args, readset, result) via ZMQ (async)
  7. Cache Manager processes and sends Save command back via HTTP
  8. Wrapper caches result in Dapr state store
  
  ### Cache Hit Flow  
  1. Dapr routes request to wrapper middleware
  2. Wrapper checks Dapr state store (Redis) - HIT
  3. Wrapper returns cached result immediately (no service call)
  
  ### Invalidation Flow
  1. Write request forwarded to service
  2. Wrapper sends Inv(key) via ZMQ to Cache Manager (async)
  3. Cache Manager propagates invalidations to upstream wrappers
  4. Upstream wrappers receive HTTP invalidate commands and clear cache
  """
  
  @doc """
  Get wrapper health status.
  """
  def health do
    %{
      status: "healthy",
      zmq_connected: MucacheWrapper.ZMQ.connected?(),
      dapr_available: MucacheWrapper.Dapr.available?()
    }
  end
end