## Servidor em GO

Para testar localmente:

1. Verficar se o arquivo compose.yml está de acordo
1. Copiar e colar as variaveis do .env.exemple para o .env
1. Iniciar o container
```bash
docker-compose up --build
```
1. Logar no pgAdmin com os dados que estão no arquivo compose.yml
1. Criar servidor:
    - Name: docker_db
    - Hostname: db
    - Port: 5432
    - Maintenance database: wsrs
    - Username: postgres
    - Password: 123456789
1. Verificar se o tern esta instalado
```bash
go install github.com/jackc/tern/v2@latest
```
1. Rodar configurações e migrações do tern
```bash
go run cmd/tools/terndotenv/main.go
```
1. Verificar se esta instalado a versão atualizada do sqlc
```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

sqlc generate -f ./internal/store/pgstore/sqlc.yaml
```

```bash
go mod tidy
```

```bash
go run cmd/wsrs/main.go
```