import Config

# MuCache Wrapper Configuration for Dapr/Kubernetes

config :mucache_wrapper,
  # Target service (your microservice's Dapr app-id)
  target_service_name: System.get_env("TARGET_SERVICE_NAME") || "app",
  
  # Dapr configuration  
  dapr_http_endpoint: System.get_env("DAPR_HTTP_ENDPOINT") || "http://localhost:3500",
  dapr_state_store: System.get_env("DAPR_STATE_STORE") || "mucache-redis",
  
  # ZeroMQ configuration (Cache Manager connection)
  cache_manager_zmq: System.get_env("CACHE_MANAGER_ZMQ") || "tcp://cache-manager:5555",
  
  # HTTP server ports
  middleware_port: String.to_integer(System.get_env("MIDDLEWARE_PORT") || "9090"),
  commands_port: String.to_integer(System.get_env("COMMANDS_PORT") || "9091"),
  
  # Kubernetes metadata
  service_name: System.get_env("SERVICE_NAME") || "unknown",
  pod_name: System.get_env("POD_NAME") || "unknown"

# Logging
config :logger, :console,
  format: "[$level] $time [$metadata] $message\n",
  metadata: [:module, :function, :pod_name, :service_name]

# Environment specific
case config_env() do
  :prod ->
    config :logger, level: :info
    
  :dev ->
    config :logger, level: :debug
    
  :test ->
    config :logger, level: :warning
    
    config :mucache_wrapper,
      middleware_port: 19090,
      commands_port: 19091
end