package driver

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/iyertrisha/fleetflow/pkg/httpx"
)

type Handler struct {
	store *Store
	// SafeAssign controls whether assign uses atomic SQL (Step 6) or naive check-then-set (Step 5).
	SafeAssign bool
}

func NewHandler(store *Store) *Handler {
	safe := os.Getenv("SAFE_ASSIGN") != "false"
	return &Handler{store: store, SafeAssign: safe}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /drivers", h.create)
	mux.HandleFunc("GET /drivers/{id}", h.get)
	mux.HandleFunc("PATCH /drivers/{id}/status", h.patchStatus)
	mux.HandleFunc("GET /drivers", h.listAvailable)
	mux.HandleFunc("POST /drivers/{id}/assign", h.assign)
	mux.HandleFunc("POST /drivers/{id}/location", h.location)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	d, err := h.store.Create(req)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, d)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	d, err := h.store.Get(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "driver not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

func (h *Handler) patchStatus(w http.ResponseWriter, r *http.Request) {
	var req StatusRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	d, err := h.store.UpdateStatus(r.PathValue("id"), req.Status)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "driver not found")
			return
		}
		if errors.Is(err, ErrInvalidStatus) {
			httpx.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

func (h *Handler) listAvailable(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("status") != "" && r.URL.Query().Get("status") != StatusAvailable {
		httpx.Error(w, http.StatusBadRequest, "only status=AVAILABLE supported")
		return
	}
	list, err := h.store.ListAvailable()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []Driver{}
	}
	httpx.JSON(w, http.StatusOK, list)
}

func (h *Handler) assign(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var d *Driver
	var err error
	if h.SafeAssign {
		d, err = h.store.TryAssign(id)
	} else {
		d, err = h.store.NaiveAssign(id)
	}
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "driver not found")
			return
		}
		if errors.Is(err, ErrNotAvailable) {
			httpx.Error(w, http.StatusConflict, "driver not available")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

func (h *Handler) location(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.store.UpdateLocation(r.PathValue("id"), body.Lat, body.Lng); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "driver not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
