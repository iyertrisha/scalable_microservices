package order

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

// Handler serves HTTP endpoints for orders.
type Handler struct {
	store     Store
	publisher Publisher
}

func NewHandler(store Store, publisher Publisher) *Handler {
	if publisher == nil {
		publisher = NopPublisher{}
	}
	return &Handler{store: store, publisher: publisher}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /orders", h.create)
	mux.HandleFunc("GET /orders/{id}", h.get)
	mux.HandleFunc("POST /orders/{id}/cancel", h.cancel)
	mux.HandleFunc("POST /orders/{id}/assign", h.AssignDriver)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.PackageSize == "" {
		req.PackageSize = "M"
	}
	if req.Priority == "" {
		req.Priority = "normal"
	}
	o, err := h.store.Create(req)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Publish after commit so consumers never see uncommitted orders.
	if err := h.publisher.PublishOrderCreated(r.Context(), o); err != nil {
		log.Printf("publish order.created failed for %s: %v", o.ID, err)
	}
	writeJSON(w, http.StatusCreated, o)
}

// AssignDriver is used by Dispatch (internal HTTP).
func (h *Handler) AssignDriver(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		DriverID string `json:"driver_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.DriverID == "" {
		writeErr(w, http.StatusBadRequest, "driver_id required")
		return
	}
	o, err := h.store.AssignDriver(id, body.DriverID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, "order not found")
			return
		}
		if errors.Is(err, ErrInvalidStatus) {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	o, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, "order not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	o, err := h.store.Cancel(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, "order not found")
			return
		}
		if errors.Is(err, ErrInvalidStatus) {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": strings.TrimSpace(msg)})
}
