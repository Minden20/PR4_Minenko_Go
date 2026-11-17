package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"

	"pr4_minenko/internal/domain"
	"pr4_minenko/internal/repository"
)

type DeviceHandler struct {
	repo    repository.DeviceRepository
	counter atomic.Uint64
}

func NewDeviceHandler(repo repository.DeviceRepository) *DeviceHandler {
	return &DeviceHandler{repo: repo}
}

func (h *DeviceHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
	})
	return r
}

type deviceInput struct {
	Name      string `json:"name"`
	IPAddress string `json:"ip_address"`
	Type      string `json:"type"`
	Status    string `json:"status"`
}

func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.repo.GetAll())
}

func (h *DeviceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	device, ok := h.repo.GetByID(id)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("device with ID '%s' not found", id))
		return
	}
	writeJSON(w, http.StatusOK, device)
}

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeAndValidate(w, r)
	if !ok {
		return
	}

	device := domain.Device{
		Name:      in.Name,
		IPAddress: in.IPAddress,
		Type:      in.Type,
		Status:    in.Status,
		UpdatedAt: time.Now().UTC(),
	}
	for {
		device.ID = fmt.Sprintf("dev-%03d", h.counter.Add(1))
		if err := h.repo.Create(device); err == nil {
			break
		}
	}

	w.Header().Set("Location", "/api/v1/devices/"+device.ID)
	writeJSON(w, http.StatusCreated, device)
}

func (h *DeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	in, ok := decodeAndValidate(w, r)
	if !ok {
		return
	}

	device := domain.Device{
		ID:        id,
		Name:      in.Name,
		IPAddress: in.IPAddress,
		Type:      in.Type,
		Status:    in.Status,
		UpdatedAt: time.Now().UTC(),
	}
	if !h.repo.Update(id, device) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("device with ID '%s' not found", id))
		return
	}
	writeJSON(w, http.StatusOK, device)
}

func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.repo.Delete(id) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("device with ID '%s' not found", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeAndValidate(w http.ResponseWriter, r *http.Request) (deviceInput, bool) {
	var in deviceInput
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return in, false
	}

	in.Name = strings.TrimSpace(in.Name)
	in.IPAddress = strings.TrimSpace(in.IPAddress)
	in.Type = strings.TrimSpace(in.Type)
	in.Status = strings.ToLower(strings.TrimSpace(in.Status))

	var missing []string
	if in.Name == "" {
		missing = append(missing, "name")
	}
	if in.IPAddress == "" {
		missing = append(missing, "ip_address")
	}
	if in.Type == "" {
		missing = append(missing, "type")
	}
	if len(missing) > 0 {
		writeError(w, http.StatusBadRequest, "missing required fields: "+strings.Join(missing, ", "))
		return in, false
	}

	switch in.Status {
	case "":
		in.Status = domain.StatusOffline
	case domain.StatusOnline, domain.StatusOffline:
	default:
		writeError(w, http.StatusBadRequest, "status must be 'online' or 'offline'")
		return in, false
	}
	return in, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
