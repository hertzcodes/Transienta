defmodule MucacheWrapper.MixProject do
  use Mix.Project

  def project do
    [
      app: :mucache_wrapper,
      version: "0.1.0",
      elixir: "~> 1.14",
      start_permanent: Mix.env() == :prod,
      deps: deps()
    ]
  end

  def application do
    [
      extra_applications: [:logger, :crypto],
      mod: {MucacheWrapper.Application, []}
    ]
  end

  defp deps do
    [
      # ZeroMQ for Cache Manager communication (as per paper)
      {:chumak, "~> 1.4"},
      
      # Dapr SDK for Kubernetes integration
      {:dapr, "~> 0.2"},
      
      # HTTP and JSON for Dapr communication
      {:plug, "~> 1.14"},
      {:plug_cowboy, "~> 2.6"},
      {:jason, "~> 1.4"},
      {:httpoison, "~> 2.0"},
      
      # Utilities
      {:uuid, "~> 1.1"}
    ]
  end
end