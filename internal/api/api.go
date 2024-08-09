package api

import (
	"net/http" // Pacote para manipulação de requisições e respostas HTTP.

	"github.com/go-chi/chi/v5"                                // Pacote para roteamento HTTP em Go.
	"github.com/joao-ressel/go-server/internal/store/pgstore" // Pacote interno para interação com o banco de dados.
)

// apiHandler é uma estrutura que armazena as dependências necessárias para lidar com requisições HTTP.
// q: É um ponteiro para pgstore.Queries, que contém métodos para interagir com o banco de dados.
// r: É um roteador HTTP da biblioteca chi, que define as rotas para as requisições.
type apiHandler struct {
	q *pgstore.Queries
	r *chi.Mux
}

// ServeHTTP é o método que implementa a interface http.Handler.
// Ele é responsável por redirecionar a requisição HTTP recebida para o roteador (r) definido na estrutura apiHandler.
func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.r.ServeHTTP(w, r)
}

// NewHandler cria e retorna uma nova instância de apiHandler configurada.
// q: Recebe um ponteiro para pgstore.Queries, que será armazenado no handler.
// Retorna um http.Handler que pode ser utilizado para lidar com requisições HTTP.
func NewHandler(q *pgstore.Queries) http.Handler {
	a := apiHandler{
		q: q, // Armazena o ponteiro para pgstore.Queries.
	}
	r := chi.NewRouter() // Cria um novo roteador usando a biblioteca chi.
	a.r = r              // Associa o roteador ao handler.
	return a             // Retorna o handler como um http.Handler.
}
