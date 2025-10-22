defmodule MucacheWrapper.Dapr do
  @moduledoc """
  Dapr Client for state store and service invocation.
  
  Provides interface to Dapr services used by the wrapper:
  - State Store (Redis) for caching
  - Service Invocation for forwarding requests
  """
  
  use GenServer
  require Logger
  
  # Client API
  
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end
  
  @doc """
  Check if Dapr is available.
  """
  def available? do
    GenServer.call(__MODULE__, :available?)
  end
  
  @doc """
  Get cached response from Dapr state store.
  """
  def cache_get(key) do
    GenServer.call(__MODULE__, {:cache_get, key})
  end
  
  @doc """
  Store response in Dapr state store.
  """
  def cache_set(key, value, ttl_seconds \\ 3600) do
    GenServer.call(__MODULE__, {:cache_set, key, value, ttl_seconds})
  end
  
  @doc """
  Delete cached response from Dapr state store.
  """
  def cache_delete(key) do
    GenServer.call(__MODULE__, {:cache_delete, key})
  end
  
  @doc """
  Invoke target service via Dapr service invocation.
  """
  def invoke_service(method, path, body \\ "", headers \\ []) do
    GenServer.call(__MODULE__, {:invoke_service, method, path, body, headers}, 30_000)
  end
  
  # Server Implementation
  
  @impl true
  def init(_opts) do
    dapr_endpoint = System.get_env("DAPR_HTTP_ENDPOINT") || "http://localhost:3500"
    state_store = System.get_env("DAPR_STATE_STORE") || "mucache-redis"
    target_service = System.get_env("TARGET_SERVICE_NAME") || "app"
    
    state = %{
      dapr_endpoint: dapr_endpoint,
      state_store: state_store,
      target_service: target_service
    }
    
    Logger.info("Dapr client initialized: #{dapr_endpoint}")
    {:ok, state}
  end
  
  @impl true
  def handle_call(:available?, _from, %{dapr_endpoint: endpoint} = state) do
    available = case HTTPoison.get("#{endpoint}/v1.0/healthz", [], timeout: 2000) do
      {:ok, %{status_code: 204}} -> true
      _ -> false
    end
    
    {:reply, available, state}
  end
  
  @impl true
  def handle_call({:cache_get, key}, _from, %{dapr_endpoint: endpoint, state_store: store} = state) do
    url = "#{endpoint}/v1.0/state/#{store}/#{key}"
    
    case HTTPoison.get(url, [], timeout: 5000) do
      {:ok, %{status_code: 200, body: body}} when body != "" ->
        {:reply, {:ok, body}, state}
        
      {:ok, %{status_code: 204}} ->
        {:reply, {:error, :not_found}, state}
        
      {:error, error} ->
        Logger.error("Dapr state get failed: #{inspect(error)}")
        {:reply, {:error, error}, state}
    end
  end
  
  @impl true
  def handle_call({:cache_set, key, value, ttl}, _from, %{dapr_endpoint: endpoint, state_store: store} = state) do
    url = "#{endpoint}/v1.0/state/#{store}"
    
    # Dapr state format with TTL
    state_data = [%{
      "key" => key,
      "value" => value,
      "metadata" => %{
        "ttlInSeconds" => Integer.to_string(ttl)
      }
    }]
    
    headers = [{"Content-Type", "application/json"}]
    body = Jason.encode!(state_data)
    
    case HTTPoison.post(url, body, headers, timeout: 5000) do
      {:ok, %{status_code: status}} when status in 200..299 ->
        {:reply, :ok, state}
        
      {:error, error} ->
        Logger.error("Dapr state set failed: #{inspect(error)}")
        {:reply, {:error, error}, state}
    end
  end
  
  @impl true
  def handle_call({:cache_delete, key}, _from, %{dapr_endpoint: endpoint, state_store: store} = state) do
    url = "#{endpoint}/v1.0/state/#{store}/#{key}"
    
    case HTTPoison.delete(url, [], timeout: 5000) do
      {:ok, %{status_code: status}} when status in 200..299 ->
        {:reply, :ok, state}
        
      {:error, error} ->
        Logger.error("Dapr state delete failed: #{inspect(error)}")
        {:reply, {:error, error}, state}
    end
  end
  
  @impl true
  def handle_call({:invoke_service, method, path, body, headers}, _from, %{dapr_endpoint: endpoint, target_service: service} = state) do
    url = "#{endpoint}/v1.0/invoke/#{service}/method#{path}"
    
    http_method = case String.downcase(method) do
      "get" -> :get
      "post" -> :post
      "put" -> :put
      "patch" -> :patch
      "delete" -> :delete
      _ -> :get
    end
    
    case HTTPoison.request(http_method, url, body, headers, timeout: 30_000) do
      {:ok, response} ->
        result = %{
          status: response.status_code,
          headers: response.headers,
          body: response.body
        }
        {:reply, {:ok, result}, state}
        
      {:error, error} ->
        Logger.error("Dapr service invocation failed: #{inspect(error)}")
        {:reply, {:error, error}, state}
    end
  end
end