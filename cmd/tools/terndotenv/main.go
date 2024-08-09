package main

import (
	"os/exec" // Pacote para executar comandos externos

	"github.com/joho/godotenv" // Pacote para carregar variáveis de ambiente de um arquivo .env
)

func main() {
	// Carrega as variáveis de ambiente do arquivo .env
	// Se houver um erro durante o carregamento (por exemplo, se o arquivo .env não for encontrado),
	// o programa dispara um pânico (interrompe a execução com uma mensagem de erro).
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	// Cria um comando que executa o comando externo "tern migrate"
	// O comando tern é utilizado para gerenciar migrações de banco de dados
	// Os parâmetros passados são:
	// --migrations: Especifica o diretório onde estão as migrações
	// --config: Especifica o arquivo de configuração do tern
	cmd := exec.Command(
		"tern",
		"migrate",
		"--migrations",
		"./internal/store/pgstore/migrations",
		"--config",
		"./internal/store/pgstore/migrations/tern.conf",
	)

	// Executa o comando configurado anteriormente
	// Se houver um erro durante a execução do comando, o programa dispara um pânico
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
