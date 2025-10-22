defmodule MucacheWrapper.Application do
  @moduledoc """
  MuCache Wrapper Application for Dapr Middleware.
  
  Minimal wrapper implementation that integrates with Dapr as middleware
  and communicates with the Rust Cache Manager via ZeroMQ.
  """

  use Application
  require Logger

  @impl true
  def start(_type, _args) do
    Logger.info("Starting MuCache Wrapper for Dapr...")
    
    children = [
      # ZeroMQ connection to Cache Manager (as per research paper)
      {MucacheWrapper.ZMQ, []},
      
      # Dapr client for state store and service invocation
      {MucacheWrapper.Dapr, []},
      
      # HTTP server for Dapr middleware integration
      {MucacheWrapper.Middleware, []},
      
      # HTTP server for receiving Cache Manager commands
      {MucacheWrapper.Commands, []}
    ]

    opts = [strategy: :one_for_one, name: MucacheWrapper.Supervisor]
    Supervisor.start_link(children, opts)
  end
end