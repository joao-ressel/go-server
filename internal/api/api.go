package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/joao-ressel/go-server/internal/store/pgstore"
)

// apiHandler é uma estrutura que lida com as requisições da API e gerencia WebSockets.
type apiHandler struct {
	q           *pgstore.Queries                                  // Consulta ao banco de dados
	r           *chi.Mux                                          // Roteador de rotas
	upgrader    websocket.Upgrader                                // Upgrader para WebSocket
	subscribers map[string]map[*websocket.Conn]context.CancelFunc // Mapeia conexões WebSocket por sala
	mu          *sync.Mutex                                       // Mutex para sincronização de acesso a subscribers
}

// ServeHTTP implementa a interface http.Handler para apiHandler.
func (h apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.r.ServeHTTP(w, r) // Encaminha a requisição para o roteador
}

// NewHandler cria uma nova instância de apiHandler e configura as rotas.
func NewHandler(q *pgstore.Queries) http.Handler {
	a := apiHandler{
		q:           q,
		upgrader:    websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		subscribers: make(map[string]map[*websocket.Conn]context.CancelFunc),
		mu:          &sync.Mutex{},
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, middleware.Logger) // Middleware para request ID, recuperação de panics e logging

	// Configuração do CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"}, // Permite todas as origens
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Rotas para WebSocket
	r.Get("/subscribe/{room_id}", a.handleSubscribe)

	// Rotas para a API principal
	r.Route("/api", func(r chi.Router) {
		r.Route("/rooms", func(r chi.Router) {
			r.Post("/", a.handleCreateRoom) // Criar nova sala
			r.Get("/", a.handleGetRooms)    // Listar salas

			r.Route("/{room_id}", func(r chi.Router) {
				r.Get("/", a.handleGetRoom) // Obter detalhes de uma sala

				r.Route("/messages", func(r chi.Router) {
					r.Post("/", a.handleCreateRoomMessage) // Criar mensagem na sala
					r.Get("/", a.handleGetRoomMessages)    // Listar mensagens da sala

					r.Route("/{message_id}", func(r chi.Router) {
						r.Get("/", a.handleGetRoomMessage)                 // Obter detalhes de uma mensagem
						r.Patch("/react", a.handleReactToMessage)          // Reagir a mensagem
						r.Delete("/react", a.handleRemoveReactFromMessage) // Remover reação de mensagem
						r.Patch("/answer", a.handleMarkMessageAsAnswered)  // Marcar mensagem como respondida
					})
				})
			})
		})
	})

	a.r = r
	return a
}

// Constantes para os tipos de mensagens enviadas aos clientes via WebSocket
const (
	MessageKindMessageCreated          = "message_created"
	MessageKindMessageRactionIncreased = "message_reaction_increased"
	MessageKindMessageRactionDecreased = "message_reaction_decreased"
	MessageKindMessageAnswered         = "message_answered"
)

// Estruturas para diferentes tipos de mensagens
type MessageMessageReactionIncreased struct {
	ID    string `json:"id"`
	Count int64  `json:"count"`
}

type MessageMessageReactionDecreased struct {
	ID    string `json:"id"`
	Count int64  `json:"count"`
}

type MessageMessageAnswered struct {
	ID string `json:"id"`
}

type MessageMessageCreated struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type Message struct {
	Kind   string `json:"kind"`
	Value  any    `json:"value"`
	RoomID string `json:"-"`
}

// notifyClients envia uma mensagem para todos os clientes assinantes da sala especificada.
func (h apiHandler) notifyClients(msg Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subscribers, ok := h.subscribers[msg.RoomID]
	if !ok || len(subscribers) == 0 {
		return // Se não houver assinantes para a sala, retorna
	}

	for conn, cancel := range subscribers {
		if err := conn.WriteJSON(msg); err != nil {
			slog.Error("failed to send message to client", "error", err)
			cancel() // Cancela a conexão se ocorrer um erro
		}
	}
}

// handleSubscribe lida com conexões WebSocket para uma sala específica.
func (h apiHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	_, rawRoomID, _, ok := h.readRoom(w, r) // Obtém o ID da sala a partir da requisição
	if !ok {
		return
	}

	c, err := h.upgrader.Upgrade(w, r, nil) // Faz o upgrade da conexão para WebSocket
	if err != nil {
		slog.Warn("failed to upgrade connection", "error", err)
		http.Error(w, "failed to upgrade to ws connection", http.StatusBadRequest)
		return
	}

	defer c.Close() // Garante que a conexão será fechada quando a função terminar

	ctx, cancel := context.WithCancel(r.Context())

	h.mu.Lock()
	if _, ok := h.subscribers[rawRoomID]; !ok {
		h.subscribers[rawRoomID] = make(map[*websocket.Conn]context.CancelFunc)
	}
	slog.Info("new client connected", "room_id", rawRoomID, "client_ip", r.RemoteAddr)
	h.subscribers[rawRoomID][c] = cancel
	h.mu.Unlock()

	<-ctx.Done() // Aguarda até que o contexto seja cancelado

	h.mu.Lock()
	delete(h.subscribers[rawRoomID], c) // Remove o cliente da lista de assinantes quando o contexto for cancelado
	h.mu.Unlock()
}

// handleCreateRoom cria uma nova sala com base no corpo da requisição.
func (h apiHandler) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	type _body struct {
		Theme string `json:"theme"`
	}
	var body _body
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	roomID, err := h.q.InsertRoom(r.Context(), body.Theme) // Insere a sala no banco de dados
	if err != nil {
		slog.Error("failed to insert room", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	type response struct {
		ID string `json:"id"`
	}

	sendJSON(w, response{ID: roomID.String()}) // Envia o ID da nova sala como resposta
}

// handleGetRooms lista todas as salas existentes.
func (h apiHandler) handleGetRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.q.GetRooms(r.Context()) // Obtém as salas do banco de dados
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		slog.Error("failed to get rooms", "error", err)
		return
	}

	if rooms == nil {
		rooms = []pgstore.Room{}
	}

	sendJSON(w, rooms) // Envia a lista de salas como resposta
}

// handleGetRoom obtém os detalhes de uma sala específica.
func (h apiHandler) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	room, _, _, ok := h.readRoom(w, r) // Obtém os detalhes da sala
	if !ok {
		return
	}

	sendJSON(w, room) // Envia os detalhes da sala como resposta
}

// handleCreateRoomMessage cria uma nova mensagem em uma sala.
func (h apiHandler) handleCreateRoomMessage(w http.ResponseWriter, r *http.Request) {
	_, rawRoomID, roomID, ok := h.readRoom(w, r) // Obtém o ID da sala
	if !ok {
		return
	}

	type _body struct {
		Message string `json:"message"`
	}
	var body _body
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	messageID, err := h.q.InsertMessage(r.Context(), pgstore.InsertMessageParams{RoomID: roomID, Message: body.Message}) // Insere a mensagem no banco de dados
	if err != nil {
		slog.Error("failed to insert message", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	type response struct {
		ID string `json:"id"`
	}

	sendJSON(w, response{ID: messageID.String()}) // Envia o ID da nova mensagem como resposta

	// Notifica os clientes assinantes da sala sobre a nova mensagem
	go h.notifyClients(Message{
		Kind:   MessageKindMessageCreated,
		RoomID: rawRoomID,
		Value: MessageMessageCreated{
			ID:      messageID.String(),
			Message: body.Message,
		},
	})
}

// handleGetRoomMessages lista todas as mensagens de uma sala específica.
func (h apiHandler) handleGetRoomMessages(w http.ResponseWriter, r *http.Request) {
	_, _, roomID, ok := h.readRoom(w, r) // Obtém o ID da sala
	if !ok {
		return
	}

	messages, err := h.q.GetRoomMessages(r.Context(), roomID) // Obtém as mensagens da sala
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		slog.Error("failed to get room messages", "error", err)
		return
	}

	if messages == nil {
		messages = []pgstore.Message{}
	}

	sendJSON(w, messages) // Envia a lista de mensagens como resposta
}

// handleGetRoomMessage obtém os detalhes de uma mensagem específica.
func (h apiHandler) handleGetRoomMessage(w http.ResponseWriter, r *http.Request) {
	_, _, _, ok := h.readRoom(w, r) // Obtém o ID da sala
	if !ok {
		return
	}

	rawMessageID := chi.URLParam(r, "message_id") // Obtém o ID da mensagem da URL
	messageID, err := uuid.Parse(rawMessageID)
	if err != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	messages, err := h.q.GetMessage(r.Context(), messageID) // Obtém os detalhes da mensagem
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "message not found", http.StatusBadRequest)
			return
		}

		slog.Error("failed to get message", "error", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	sendJSON(w, messages) // Envia os detalhes da mensagem como resposta
}

// handleReactToMessage adiciona uma reação a uma mensagem.
func (h apiHandler) handleReactToMessage(w http.ResponseWriter, r *http.Request) {
	_, rawRoomID, _, ok := h.readRoom(w, r) // Obtém o ID da sala
	if !ok {
		return
	}

	rawID := chi.URLParam(r, "message_id") // Obtém o ID da mensagem da URL
	id, err := uuid.Parse(rawID)
	if err != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	count, err := h.q.ReactToMessage(r.Context(), id) // Adiciona uma reação à mensagem
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		slog.Error("failed to react to message", "error", err)
		return
	}

	type response struct {
		Count int64 `json:"count"`
	}

	sendJSON(w, response{Count: count}) // Envia a contagem atualizada de reações como resposta

	// Notifica os clientes assinantes da sala sobre a reação aumentada
	go h.notifyClients(Message{
		Kind:   MessageKindMessageRactionIncreased,
		RoomID: rawRoomID,
		Value: MessageMessageReactionIncreased{
			ID:    rawID,
			Count: count,
		},
	})
}

// handleRemoveReactFromMessage remove uma reação de uma mensagem.
func (h apiHandler) handleRemoveReactFromMessage(w http.ResponseWriter, r *http.Request) {
	_, rawRoomID, _, ok := h.readRoom(w, r) // Obtém o ID da sala
	if !ok {
		return
	}

	rawID := chi.URLParam(r, "message_id") // Obtém o ID da mensagem da URL
	id, err := uuid.Parse(rawID)
	if err != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	count, err := h.q.RemoveReactionFromMessage(r.Context(), id) // Remove uma reação da mensagem
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		slog.Error("failed to react to message", "error", err)
		return
	}

	type response struct {
		Count int64 `json:"count"`
	}

	sendJSON(w, response{Count: count}) // Envia a contagem atualizada de reações como resposta

	// Notifica os clientes assinantes da sala sobre a reação diminuída
	go h.notifyClients(Message{
		Kind:   MessageKindMessageRactionDecreased,
		RoomID: rawRoomID,
		Value: MessageMessageReactionDecreased{
			ID:    rawID,
			Count: count,
		},
	})
}

// handleMarkMessageAsAnswered marca uma mensagem como respondida.
func (h apiHandler) handleMarkMessageAsAnswered(w http.ResponseWriter, r *http.Request) {
	_, rawRoomID, _, ok := h.readRoom(w, r) // Obtém o ID da sala
	if !ok {
		return
	}

	rawID := chi.URLParam(r, "message_id") // Obtém o ID da mensagem da URL
	id, err := uuid.Parse(rawID)
	if err != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	err = h.q.MarkMessageAsAnswered(r.Context(), id) // Marca a mensagem como respondida
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		slog.Error("failed to react to message", "error", err)
		return
	}

	w.WriteHeader(http.StatusOK) // Envia status 200 OK

	// Notifica os clientes assinantes da sala sobre a mensagem respondida
	go h.notifyClients(Message{
		Kind:   MessageKindMessageAnswered,
		RoomID: rawRoomID,
		Value: MessageMessageAnswered{
			ID: rawID,
		},
	})
}
