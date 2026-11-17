package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	"pr4_minenko/internal/domain"
	"pr4_minenko/internal/repository"
)

func newTestServer() *httptest.Server {
	r := chi.NewRouter()
	r.Mount("/api/v1/devices", NewDeviceHandler(repository.NewMemoryDeviceRepository()).Routes())
	return httptest.NewServer(r)
}

func do(t *testing.T, method, url, body string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, b
}

func TestDeviceLifecycle(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	base := srv.URL + "/api/v1/devices"

	resp, body := do(t, http.MethodPost, base,
		`{"name":"Main Thermal Sensor","ip_address":"192.168.1.50","type":"sensor","status":"online"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST: want 201, got %d: %s", resp.StatusCode, body)
	}
	var created domain.Device
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatal(err)
	}
	if created.ID != "dev-001" || created.UpdatedAt.IsZero() {
		t.Fatalf("unexpected created device: %+v", created)
	}

	resp, body = do(t, http.MethodGet, base, "")
	var list []domain.Device
	_ = json.Unmarshal(body, &list)
	if resp.StatusCode != http.StatusOK || len(list) != 1 {
		t.Fatalf("GET list: want 200 with 1 item, got %d: %s", resp.StatusCode, body)
	}

	resp, _ = do(t, http.MethodGet, base+"/dev-001", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET by id: want 200, got %d", resp.StatusCode)
	}

	resp, body = do(t, http.MethodPut, base+"/dev-001",
		`{"name":"Main Thermal Sensor v2","ip_address":"192.168.1.51","type":"sensor","status":"offline"}`)
	var updated domain.Device
	_ = json.Unmarshal(body, &updated)
	if resp.StatusCode != http.StatusOK || updated.Name != "Main Thermal Sensor v2" ||
		updated.Status != "offline" || !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("PUT: unexpected %d: %s", resp.StatusCode, body)
	}

	resp, body = do(t, http.MethodDelete, base+"/dev-001", "")
	if resp.StatusCode != http.StatusNoContent || len(body) != 0 {
		t.Fatalf("DELETE: want 204 without body, got %d: %q", resp.StatusCode, body)
	}

	resp, _ = do(t, http.MethodGet, base+"/dev-001", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET after delete: want 404, got %d", resp.StatusCode)
	}
}

func TestErrors(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	base := srv.URL + "/api/v1/devices"

	cases := []struct {
		name, method, url, body string
		want                    int
	}{
		{"invalid json", http.MethodPost, base, `{bad json`, 400},
		{"missing fields", http.MethodPost, base, `{"name":"X"}`, 400},
		{"bad status", http.MethodPost, base, `{"name":"X","ip_address":"1.1.1.1","type":"sensor","status":"up"}`, 400},
		{"get unknown", http.MethodGet, base + "/dev-999", "", 404},
		{"put unknown", http.MethodPut, base + "/dev-999", `{"name":"X","ip_address":"1.1.1.1","type":"sensor"}`, 404},
		{"delete unknown", http.MethodDelete, base + "/dev-999", "", 404},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, body := do(t, c.method, c.url, c.body)
			if resp.StatusCode != c.want {
				t.Fatalf("want %d, got %d: %s", c.want, resp.StatusCode, body)
			}
			if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
				t.Fatalf("want JSON content type, got %q", ct)
			}
		})
	}
}

func TestConcurrentCreate(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()
	base := srv.URL + "/api/v1/devices"

	const n = 50
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			do(t, http.MethodPost, base, fmt.Sprintf(`{"name":"node-%d","ip_address":"10.0.0.%d","type":"sensor"}`, i, i))
			do(t, http.MethodGet, base, "")
		}(i)
	}
	wg.Wait()

	_, body := do(t, http.MethodGet, base, "")
	var list []domain.Device
	_ = json.Unmarshal(body, &list)
	if len(list) != n {
		t.Fatalf("want %d devices, got %d", n, len(list))
	}
}
