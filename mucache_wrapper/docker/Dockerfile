# MuCache Wrapper Dockerfile
FROM elixir:1.15-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    build-base \
    git \
    zeromq-dev

ENV MIX_ENV=prod

WORKDIR /app

# Install dependencies
COPY mix.exs mix.lock ./
RUN mix local.hex --force && \
    mix local.rebar --force && \
    mix deps.get --only prod

# Build application  
COPY lib ./lib
COPY config ./config
RUN mix compile && mix release

# Runtime image
FROM alpine:3.18

# Runtime dependencies
RUN apk add --no-cache \
    openssl \
    ncurses-libs \
    zeromq

# Create app user
RUN adduser -D -S mucache

WORKDIR /opt/app
USER mucache

# Copy release
COPY --from=builder --chown=mucache:mucache /app/_build/prod/rel/mucache_wrapper ./

# Expose ports
EXPOSE 9090 9091

# Health check
HEALTHCHECK --interval=30s --timeout=3s \
  CMD curl -f http://localhost:9091/health || exit 1

ENTRYPOINT ["/opt/app/bin/mucache_wrapper"]
CMD ["start"]