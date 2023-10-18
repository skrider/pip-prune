# syntax=docker/dockerfile:1

# Using the official golang image as a base
FROM golang:1.20.5-alpine3.18 AS base

# Enable BuildKit's features
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go env -w GOPROXY=https://proxy.golang.org,direct

# Set working directory
WORKDIR /src

# Copy go.mod and go.sum files for dependency caching
COPY go.mod go.sum ./

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY ./cmd/pip-prune ./cmd/pip-prune
COPY ./pkg ./pkg
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /pip-prune /src/cmd/pip-prune/main.go

FROM ubuntu AS test

RUN apt-get update && apt-get install -y \
    python3 python3-pip python3-venv

COPY --from=base /pip-prune /pip-prune

WORKDIR /src

COPY testdata/requirements.txt requirements.txt

RUN --mount=type=cache,target=/tmp \
    --mount=type=cache,target=/root/.cache/pip \
    sudo /pip-prune -requirements /src/requirements.txt -- -c 'import numpy as np; a = np.arr([1]); print(a * 2)'

