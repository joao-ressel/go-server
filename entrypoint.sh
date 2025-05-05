#!/bin/bash
set -e

echo "==> Executando go generate (tern + sqlc)..."
go generate ./...

echo "==> Rodando servidor Go..."
go run cmd/wsrs/main.go
