package order

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateGetCancel(t *testing.T) {
	store := NewMemoryStore()
	h := NewHandler(store, NopPublisher{})
	mux := http.NewServeMux()
	h.Register(mux)

	body := `{"pickup_location":{"lat":12.97,"lng":77.59},"delivery_location":{"lat":12.98,"lng":77.60},"package_size":"M","priority":"high"}`
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: got %d %s", rr.Code, rr.Body.String())
	}

	var created Order
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Status != StatusCreated {
		t.Fatalf("unexpected order: %+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/orders/"+created.ID, nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get: got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/orders/"+created.ID+"/cancel", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("cancel: got %d %s", rr.Code, rr.Body.String())
	}
	var cancelled Order
	_ = json.Unmarshal(rr.Body.Bytes(), &cancelled)
	if cancelled.Status != StatusCancelled {
		t.Fatalf("expected CANCELLED, got %s", cancelled.Status)
	}
}
