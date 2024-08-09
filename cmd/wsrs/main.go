package main

import (
	"context"   // Pacote para manipulação de contexto, que é útil para controlar cancelamentos e deadlines em operações.
	"errors"    // Pacote para manipulação de erros.
	"fmt"       // Pacote para formatação de strings.
	"net/http"  // Pacote para criação de servidores HTTP.
	"os"        // Pacote para interação com o sistema operacional, como leitura de variáveis de ambiente e manipulação de sinais.
	"os/signal" // Pacote para captura de sinais do sistema operacional, como interrupções.

	"github.com/jackc/pgx/v5/pgxpool"                         // Pacote para gerenciar um pool de conexões ao banco de dados PostgreSQL.
	"github.com/joao-ressel/go-server/internal/api"           // Pacote interno que contém o manipulador (handler) da API.
	"github.com/joao-ressel/go-server/internal/store/pgstore" // Pacote interno que gerencia a interação com o banco de dados.
	"github.com/joho/godotenv"                                // Pacote para carregar variáveis de ambiente de um arquivo .env.
)

func main() {
	// Carrega as variáveis de ambiente do arquivo .env.
	// Se houver erro durante o carregamento, o programa dispara um pânico.
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	// Cria um contexto de fundo (background) que pode ser utilizado para controlar a vida útil das operações.
	ctx := context.Background()

	// Cria uma nova pool de conexões com o banco de dados PostgreSQL usando as variáveis de ambiente.
	pool, err := pgxpool.New(ctx, fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s",
		os.Getenv("WSRS_DATABASE_USER"),
		os.Getenv("WSRS_DATABASE_PASSWORD"),
		os.Getenv("WSRS_DATABASE_HOST"),
		os.Getenv("WSRS_DATABASE_PORT"),
		os.Getenv("WSRS_DATABASE_NAME"),
	))
	// Se ocorrer um erro ao criar a pool de conexões, o programa dispara um pânico.
	if err != nil {
		panic(err)
	}

	// Garante que a pool de conexões será fechada quando o main terminar.
	defer pool.Close()

	// Verifica se a conexão com o banco de dados está ativa.
	// Se a verificação falhar, o programa dispara um pânico.
	if err := pool.Ping(ctx); err != nil {
		panic(err)
	}

	// Cria um novo handler da API utilizando a store de banco de dados criada (pgstore).
	handler := api.NewHandler(pgstore.New(pool))

	// Inicia o servidor HTTP em uma nova goroutine para escutar requisições na porta 8080.
	// Se o servidor falhar ao iniciar (exceto se for um erro de fechamento do servidor), o programa dispara um pânico.
	go func() {
		if err := http.ListenAndServe(":8080", handler); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()

	// Cria um canal que captura sinais do sistema operacional, como uma interrupção (Ctrl+C).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit // Bloqueia até que uma interrupção seja recebida.
}
