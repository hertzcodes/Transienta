defmodule MucacheWrapper.ZMQ do
  @moduledoc """
  ZeroMQ Communication with Cache Manager.
  
  Implements the high-performance, ordered message queue communication
  between wrapper and Cache Manager as specified in the MuCache paper.
  
  Uses PUSH socket for sending events (non-blocking).
  """
  
  use GenServer
  require Logger
  
  # Client API
  
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end
  
  @doc """
  Send Start message to Cache Manager (non-blocking).
  """
  def send_start(request_id, call_args) do
    message = {:start_request, request_id, call_args, System.system_time(:microsecond)}
    GenServer.cast(__MODULE__, {:send, message})
  end
  
  @doc """
  Send End message to Cache Manager (non-blocking).
  """
  def send_end(request_id, call_args, readset, result) do
    message = {:end_request, request_id, call_args, readset, result, System.system_time(:microsecond)}
    GenServer.cast(__MODULE__, {:send, message})
  end
  
  @doc """
  Send Invalidation message to Cache Manager (non-blocking).
  """
  def send_invalidation(key) do
    message = {:invalidation, key, System.system_time(:microsecond)}
    GenServer.cast(__MODULE__, {:send, message})
  end
  
  @doc """
  Check if ZMQ connection is healthy.
  """
  def connected? do
    GenServer.call(__MODULE__, :connected?)
  end
  
  # Server Implementation
  
  @impl true
  def init(_opts) do
    # ZMQ configuration - Cache Manager endpoint
    cm_endpoint = System.get_env("CACHE_MANAGER_ZMQ") || "tcp://cache-manager:5555"
    
    Logger.info("Connecting to Cache Manager via ZMQ: #{cm_endpoint}")
    
    case setup_zmq(cm_endpoint) do
      {:ok, socket} ->
        Logger.info("ZMQ connected to Cache Manager")
        {:ok, %{socket: socket, endpoint: cm_endpoint}}
        
      {:error, error} ->
        Logger.error("Failed to connect ZMQ: #{inspect(error)}")
        {:stop, error}
    end
  end
  
  @impl true
  def handle_cast({:send, message}, %{socket: socket} = state) do
    # Serialize and send message via ZMQ (non-blocking)
    binary = :erlang.term_to_binary(message)
    
    case :chumak.send(socket, binary) do
      :ok ->
        Logger.debug("Sent ZMQ message: #{elem(message, 0)}")
        
      {:error, error} ->
        Logger.error("ZMQ send failed: #{inspect(error)}")
    end
    
    {:noreply, state}
  end
  
  @impl true
  def handle_call(:connected?, _from, %{socket: socket} = state) do
    connected = is_pid(socket) and Process.alive?(socket)
    {:reply, connected, state}
  end
  
  # Private Functions
  
  defp setup_zmq(endpoint) do
    with {:ok, ctx} <- :chumak.start_link(),
         {:ok, socket} <- :chumak.socket(ctx, :push),
         :ok <- :chumak.connect(socket, endpoint) do
      
      # Set high water mark for performance
      :chumak.setsockopt(socket, :sndhwm, 1000)
      
      {:ok, socket}
    else
      error -> {:error, error}
    end
  end
end