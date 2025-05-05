package main

import (
	"fmt"
	"os/exec"

	"github.com/joho/godotenv"
)

func main() {
	// Carrega variáveis de ambiente do .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("Erro ao carregar o arquivo .env:", err)
		return
	}

	// Comando tern migrate
	cmd := exec.Command(
		"tern",
		"migrate",
		"--migrations",
		"./internal/store/pgstore/migrations",
		"--config",
		"./internal/store/pgstore/migrations/tern.conf",
	)

	// Executa o comando e captura a saída
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Erro ao executar o comando: %s\n", err)
		fmt.Printf("Saída do comando: %s\n", output)
		return
	}

	fmt.Println("Migrações executadas com sucesso:")
	fmt.Println(string(output))
}
