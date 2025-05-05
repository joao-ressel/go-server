# Etapa base com Go
FROM golang:1.24

# Define o diretório de trabalho
WORKDIR /app

# Copia os arquivos do projeto para o container
COPY . .

# Baixa as dependências
RUN go mod download

# Instala o tern e sqlc
RUN go install github.com/jackc/tern/v2@latest && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Expõe a porta da aplicação (se necessário)
EXPOSE 8080

# O comando principal será definido no docker-compose.yml
