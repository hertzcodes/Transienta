defmodule MucacheWrapper.Commands do
  @moduledoc """
  HTTP Command Server for Cache Manager communication.
  
  Receives Save and Invalidate commands from the Rust Cache Manager
  and applies them to the Dapr state store.
  """
  
  use GenServer
  require Logger
  
  # Client API
  
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end
  
  # Server Implementation
  
  @impl true
  def init(_opts) do
    port = String.to_integer(System.get_env("COMMANDS_PORT") || "9091")
    
    # Start HTTP server for Cache Manager commands
    {:ok, _} = Plug.Cowboy.http(MucacheWrapper.CommandsHandler, [], port: port)
    
    Logger.info("Started commands server on port #{port}")
    {:ok, %{port: port}}
  end
end

defmodule MucacheWrapper.CommandsHandler do
  @moduledoc """
  HTTP handler for Cache Manager commands (Save/Invalidate).
  """
  
  use Plug.Router
  require Logger
  
  alias MucacheWrapper.Dapr
  
  plug Plug.Logger
  plug :match
  plug Plug.Parsers,
    parsers: [:json],
    pass: ["application/json"],
    json_decoder: Jason
  plug :dispatch
  
  @doc """
  Health check endpoint.
  """
  get "/health" do
    health = MucacheWrapper.health()
    
    conn
    |> put_resp_content_type("application/json")
    |> send_resp(200, Jason.encode!(health))
  end
  
  @doc """
  Save command - Cache Manager tells us to store a result.
  """
  post "/save" do
    case conn.body_params do
      %{"call_args_hash" => hash, "result" => result} = params ->
        ttl = Map.get(params, "ttl_seconds", 3600)
        
        case Dapr.cache_set(hash, result, ttl) do
          :ok ->
            Logger.debug("Cached result for: #{hash}")
            send_json_response(conn, 200, %{status: "ok"})
            
          {:error, error} ->
            Logger.error("Failed to cache: #{inspect(error)}")
            send_json_response(conn, 500, %{error: "Cache failed"})
        end
        
      _ ->
        send_json_response(conn, 400, %{error: "Invalid save command"})
    end
  end
  
  @doc """
  Invalidate command - Cache Manager tells us to remove a cached item.
  """
  post "/invalidate" do
    case conn.body_params do
      %{"call_args_hash" => hash} ->
        case Dapr.cache_delete(hash) do
          :ok ->
            Logger.debug("Invalidated: #{hash}")
            send_json_response(conn, 200, %{status: "ok"})
            
          {:error, error} ->
            Logger.error("Failed to invalidate: #{inspect(error)}")
            send_json_response(conn, 500, %{error: "Invalidation failed"})
        end
        
      _ ->
        send_json_response(conn, 400, %{error: "Invalid invalidate command"})
    end
  end
  
  # Catch all
  match _ do
    send_json_response(conn, 404, %{error: "Not found"})
  end
  
  defp send_json_response(conn, status, data) do
    conn
    |> put_resp_content_type("application/json")
    |> send_resp(status, Jason.encode!(data))
  end
end