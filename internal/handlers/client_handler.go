package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/repositories"
)

type ClientHandler struct {
	clientRepo repositories.ClientRepo
}

func NewClientHandler(repo repositories.ClientRepo) *ClientHandler {
	return &ClientHandler{clientRepo: repo}
}

// GET /clients
func (h *ClientHandler) GetActiveClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.clientRepo.GetActiveClients()
	if err != nil {
		http.Error(w, "failed to fetch clients", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}
