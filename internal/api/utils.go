package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/joao-ressel/go-server/internal/store/pgstore"
)

// readRoom obtém os detalhes de uma sala a partir do ID da sala na URL da requisição.
// Retorna a sala, o ID da sala como string, o ID da sala como uuid.UUID e um booleano indicando sucesso.
func (h apiHandler) readRoom(
	w http.ResponseWriter, // Resposta HTTP
	r *http.Request, // Requisição HTTP
) (room pgstore.Room, rawRoomID string, roomID uuid.UUID, ok bool) {
	// Obtém o ID da sala da URL da requisição
	rawRoomID = chi.URLParam(r, "room_id")

	// Converte o ID da sala de string para uuid.UUID
	roomID, err := uuid.Parse(rawRoomID)
	if err != nil {
		// Se o ID da sala for inválido, retorna um erro 400 Bad Request
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return pgstore.Room{}, "", uuid.UUID{}, false
	}

	// Obtém os detalhes da sala a partir do ID no banco de dados
	room, err = h.q.GetRoom(r.Context(), roomID)
	if err != nil {
		// Se a sala não for encontrada, retorna um erro 400 Bad Request
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "room not found", http.StatusBadRequest)
			return pgstore.Room{}, "", uuid.UUID{}, false
		}

		// Se ocorrer um erro ao buscar a sala, registra o erro e retorna um erro 500 Internal Server Error
		slog.Error("failed to get room", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return pgstore.Room{}, "", uuid.UUID{}, false
	}

	// Retorna os detalhes da sala, o ID da sala como string, o ID da sala como uuid.UUID e true indicando sucesso
	return room, rawRoomID, roomID, true
}

// sendJSON envia uma resposta JSON para o cliente.
// Converte o dado rawData para JSON e escreve no corpo da resposta HTTP.
func sendJSON(w http.ResponseWriter, rawData any) {
	// Converte o dado rawData para JSON
	data, _ := json.Marshal(rawData)

	// Define o cabeçalho da resposta como "application/json"
	w.Header().Set("Content-Type", "application/json")

	// Escreve os dados JSON no corpo da resposta
	_, _ = w.Write(data)
}
