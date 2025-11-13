package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/MuhamadAgungGumelar/whatsapp-bot-saas-be/internal/repositories"
)

type KBHandler struct {
	kbRepo repositories.KBRepo
}

func NewKBHandler(repo repositories.KBRepo) *KBHandler {
	return &KBHandler{kbRepo: repo}
}

// GET /knowledge-base/{client_id}
func (h *KBHandler) GetKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "client_id required", http.StatusBadRequest)
		return
	}

	kb, err := h.kbRepo.GetKnowledgeBase(clientID)
	if err != nil {
		http.Error(w, "failed to fetch KB", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kb)
}

// POST /knowledge-base
func (h *KBHandler) AddKnowledgeItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientID string  `json:"client_id"`
		Type     string  `json:"type"` // faq / product
		Question string  `json:"question,omitempty"`
		Answer   string  `json:"answer,omitempty"`
		Name     string  `json:"name,omitempty"`
		Price    float64 `json:"price,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// TODO: insert ke table knowledge_base
	// Sementara return success dulu
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
