defmodule MucacheWrapper.Middleware do
  @moduledoc """
  Dapr Middleware HTTP Server.
  
  This module implements the HTTP server that integrates with Dapr's
  middleware pipeline to intercept all requests and apply MuCache logic.
  
  Dapr will route requests to this server, which then applies caching
  logic and forwards requests to the target service.
  """
  
  use GenServer
  require Logger
  
  alias MucacheWrapper.{Dapr, ZMQ}
  
  # Client API
  
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end
  
  # Server Implementation
  
  @impl true
  def init(_opts) do
    port = String.to_integer(System.get_env("MIDDLEWARE_PORT") || "9090")
    
    # Start HTTP server for Dapr middleware
    {:ok, _} = Plug.Cowboy.http(MucacheWrapper.MiddlewareHandler, [], port: port)
    
    Logger.info("Started MuCache middleware server on port #{port}")
    {:ok, %{port: port}}
  end
end

defmodule MucacheWrapper.MiddlewareHandler do
  @moduledoc """
  HTTP request handler for Dapr middleware integration.
  
  This implements the core MuCache wrapper logic:
  - Intercepts ALL HTTP requests from Dapr
  - Implements cache hit/miss logic  
  - Forwards requests to target service
  - Sends ZMQ messages to Cache Manager
  """
  
  use Plug.Router
  require Logger
  
  alias MucacheWrapper.{Dapr, ZMQ}
  
  plug Plug.Logger
  plug :match
  plug Plug.Parsers,
    parsers: [:urlencoded, :json],
    pass: ["*/*"],
    json_decoder: Jason,
    body_reader: {__MODULE__, :cache_body_reader, []}
  plug :dispatch
  
  # Health check for Dapr
  get "/health" do
    health = MucacheWrapper.health()
    
    conn
    |> put_resp_content_type("application/json")
    |> send_resp(200, Jason.encode!(health))
  end
  
  # Intercept ALL other requests (Dapr middleware entry point)
  match _ do
    process_request(conn)
  end
  
  # Cache request body for forwarding
  def cache_body_reader(conn, opts) do
    {:ok, body, conn} = Plug.Conn.read_body(conn, opts)
    conn = update_in(conn.assigns[:raw_body], &(&1 || body))
    {:ok, body, conn}
  end
  
  # Core MuCache Logic
  
  defp process_request(conn) do
    method = String.downcase(conn.method)
    path = conn.request_path
    query = conn.query_string  
    headers = get_headers(conn)
    body = conn.assigns[:raw_body] || ""
    
    case method do
      "get" ->
        # Read-only request - apply caching
        handle_cacheable_request(conn, method, path, query, headers)
        
      _ ->
        # Write request - forward and invalidate  
        handle_write_request(conn, method, path, query, headers, body)
    end
  end
  
  defp handle_cacheable_request(conn, method, path, query, headers) do
    # Generate cache key
    cache_key = generate_cache_key(method, path, query, headers)
    
    case Dapr.cache_get(cache_key) do
      {:ok, cached_response} ->
        # Cache HIT - return immediately (< 1ms)
        Logger.debug("Cache HIT: #{cache_key}")
        send_cached_response(conn, cached_response)
        
      {:error, :not_found} ->
        # Cache MISS - forward request and track
        Logger.debug("Cache MISS: #{cache_key}")
        handle_cache_miss(conn, method, path, query, headers, cache_key)
        
      {:error, error} ->
        Logger.error("Cache get error: #{inspect(error)}")
        # Fallback to forwarding request
        forward_and_respond(conn, method, path, query, headers, "")
    end
  end
  
  defp handle_cache_miss(conn, method, path, query, headers, cache_key) do
    # Generate request ID for tracking
    request_id = UUID.uuid4()
    
    # Prepare call arguments
    call_args = %{
      method: method,
      path: path,
      query: query,
      headers: filter_cacheable_headers(headers)
    }
    
    # Send Start message to Cache Manager via ZMQ (async, < 0.1ms overhead)
    ZMQ.send_start(request_id, call_args)
    
    # Forward request to target service
    case Dapr.invoke_service(method, build_full_path(path, query), "", headers) do
      {:ok, response} ->
        # Extract readset from response headers (if service provides it)
        readset = extract_readset_from_headers(response.headers)
        
        # Send End message to Cache Manager via ZMQ (async)
        result_data = serialize_response(response)
        ZMQ.send_end(request_id, call_args, readset, result_data)
        
        # Return response to client
        send_response(conn, response)
        
      {:error, error} ->
        Logger.error("Service invocation failed: #{inspect(error)}")
        send_resp(conn, 502, "Service unavailable")
    end
  end
  
  defp handle_write_request(conn, method, path, query, headers, body) do
    # Forward write request to target service
    case Dapr.invoke_service(method, build_full_path(path, query), body, headers) do
      {:ok, response} ->
        # If write successful, send invalidation
        if response.status in 200..299 do
          invalidation_key = generate_invalidation_key(method, path, query)
          ZMQ.send_invalidation(invalidation_key)
          
          Logger.debug("Invalidation sent for: #{invalidation_key}")
        end
        
        send_response(conn, response)
        
      {:error, error} ->
        Logger.error("Write request failed: #{inspect(error)}")
        send_resp(conn, 502, "Service unavailable")
    end
  end
  
  defp forward_and_respond(conn, method, path, query, headers, body) do
    case Dapr.invoke_service(method, build_full_path(path, query), body, headers) do
      {:ok, response} -> send_response(conn, response)
      {:error, _} -> send_resp(conn, 502, "Service unavailable")
    end
  end
  
  # Helper Functions
  
  defp generate_cache_key(method, path, query, headers) do
    # Create deterministic hash for caching
    canonical_headers = canonicalize_headers(headers)
    data = "#{method}:#{path}:#{query}:#{canonical_headers}"
    
    :crypto.hash(:sha256, data) |> Base.encode16(case: :lower)
  end
  
  defp canonicalize_headers(headers) do
    headers
    |> Enum.reject(fn {key, _} ->
      # Filter non-deterministic headers
      String.downcase(key) in [
        "date", "x-request-id", "x-trace-id", 
        "authorization", "cookie"
      ]
    end)
    |> Enum.sort()
    |> Enum.map(fn {k, v} -> "#{String.downcase(k)}:#{v}" end)
    |> Enum.join("|")
  end
  
  defp generate_invalidation_key(method, path, _query) do
    # Extract resource ID for invalidation
    # This should be configurable per service
    case extract_resource_id(path) do
      nil -> path
      resource_id -> resource_id  
    end
  end
  
  defp extract_resource_id(path) do
    cond do
      Regex.match?(~r/\/users\/(\d+)/, path) ->
        case Regex.run(~r/\/users\/(\d+)/, path) do
          [_, id] -> "user:#{id}"
          _ -> nil
        end
        
      Regex.match?(~r/\/posts\/(\d+)/, path) ->
        case Regex.run(~r/\/posts\/(\d+)/, path) do
          [_, id] -> "post:#{id}" 
          _ -> nil
        end
        
      true -> nil
    end
  end
  
  defp extract_readset_from_headers(headers) do
    # Look for readset in response headers
    case Enum.find(headers, fn {key, _} -> 
      String.downcase(key) == "x-mucache-readset" 
    end) do
      {_, readset_json} ->
        case Jason.decode(readset_json) do
          {:ok, readset} -> readset
          _ -> []
        end
      nil -> []
    end
  end
  
  defp serialize_response(response) do
    Jason.encode!(%{
      status: response.status,
      headers: response.headers,
      body: response.body,
      timestamp: System.system_time(:millisecond)
    })
  end
  
  defp send_cached_response(conn, cached_data) do
    case Jason.decode(cached_data) do
      {:ok, %{"status" => status, "headers" => headers, "body" => body}} ->
        conn
        |> put_response_headers(headers)
        |> put_resp_header("x-mucache-hit", "true")
        |> send_resp(status, body)
        
      _ ->
        send_resp(conn, 500, "Cache corruption")
    end
  end
  
  defp send_response(conn, response) do
    conn
    |> put_response_headers(response.headers)
    |> put_resp_header("x-mucache-hit", "false")
    |> send_resp(response.status, response.body)
  end
  
  defp put_response_headers(conn, headers) do
    Enum.reduce(headers, conn, fn {key, value}, acc ->
      put_resp_header(acc, key, value)
    end)
  end
  
  defp filter_cacheable_headers(headers) do
    # Remove headers that shouldn't be cached
    Enum.reject(headers, fn {key, _} ->
      String.downcase(key) in [
        "authorization", "cookie", "x-api-key"
      ]
    end)
  end
  
  defp get_headers(conn) do
    conn.req_headers
  end
  
  defp build_full_path(path, "") do
    path
  end
  
  defp build_full_path(path, query) do
    "#{path}?#{query}"
  end
end