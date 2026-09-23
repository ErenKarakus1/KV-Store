package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ErenKarakus1/KV-Store/internal/store"
	"github.com/gin-gonic/gin"
)

func newTestRouter(s *store.Store) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/kv/:key/exists", ExistsHandler(s))
	r.POST("/kv/:key/increment", IncrementHandler(s))
	r.POST("/kv/:key/setnx", SetNXHandler(s))
	r.GET("/kv/:key", GetHandler(s))
	r.PUT("/kv/:key", SetHandler(s))
	r.DELETE("/kv/:key", DeleteHandler(s))

	return r
}

func performRequest(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandlersSetGetExistsDelete(t *testing.T) {
	r := newTestRouter(store.NewStore(0))

	w := performRequest(r, http.MethodPut, "/kv/name", `{"value":"eren"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d; want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	w = performRequest(r, http.MethodGet, "/kv/name", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d; want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"value":"eren"`) {
		t.Fatalf("GET body = %s; want value eren", w.Body.String())
	}

	w = performRequest(r, http.MethodGet, "/kv/name/exists", "")
	if w.Code != http.StatusOK {
		t.Fatalf("EXISTS status = %d; want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"exists":true`) {
		t.Fatalf("EXISTS body = %s; want exists true", w.Body.String())
	}

	w = performRequest(r, http.MethodDelete, "/kv/name", "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d; want %d; body=%s", w.Code, http.StatusNoContent, w.Body.String())
	}

	w = performRequest(r, http.MethodGet, "/kv/name", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET deleted status = %d; want %d; body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestSetHandlerRejectsInvalidRequests(t *testing.T) {
	r := newTestRouter(store.NewStore(0))

	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "missing value", body: `{"ttl":"1s"}`},
		{name: "invalid ttl", body: `{"value":"x","ttl":"soon"}`},
		{name: "zero ttl", body: `{"value":"x","ttl":"0s"}`},
		{name: "negative ttl", body: `{"value":"x","ttl":"-1s"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest(r, http.MethodPut, "/kv/bad", tt.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d; want %d; body=%s", w.Code, http.StatusBadRequest, w.Body.String())
			}
		})
	}
}

func TestHandlersTTLExpiration(t *testing.T) {
	r := newTestRouter(store.NewStore(0))

	w := performRequest(r, http.MethodPut, "/kv/temp", `{"value":"short","ttl":"5ms"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT ttl status = %d; want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	time.Sleep(20 * time.Millisecond)

	w = performRequest(r, http.MethodGet, "/kv/temp", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET expired status = %d; want %d; body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}

	w = performRequest(r, http.MethodGet, "/kv/temp/exists", "")
	if w.Code != http.StatusOK {
		t.Fatalf("EXISTS expired status = %d; want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"exists":false`) {
		t.Fatalf("EXISTS expired body = %s; want exists false", w.Body.String())
	}
}

func TestDeleteMissingKey(t *testing.T) {
	r := newTestRouter(store.NewStore(0))

	w := performRequest(r, http.MethodDelete, "/kv/missing", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("DELETE missing status = %d; want %d; body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestIncrementHandler(t *testing.T) {
	r := newTestRouter(store.NewStore(0))

	w := performRequest(r, http.MethodPost, "/kv/counter/increment", "")
	if w.Code != http.StatusOK {
		t.Fatalf("first increment status = %d; want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"value":"1"`) {
		t.Fatalf("first increment body = %s; want value 1", w.Body.String())
	}

	w = performRequest(r, http.MethodPost, "/kv/counter/increment", "")
	if w.Code != http.StatusOK {
		t.Fatalf("second increment status = %d; want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"value":"2"`) {
		t.Fatalf("second increment body = %s; want value 2", w.Body.String())
	}
}

func TestIncrementHandlerRejectsNonIntegerValue(t *testing.T) {
	r := newTestRouter(store.NewStore(0))

	w := performRequest(r, http.MethodPut, "/kv/name", `{"value":"eren"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d; want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	w = performRequest(r, http.MethodPost, "/kv/name/increment", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("increment non-integer status = %d; want %d; body=%s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestSetNXHandler(t *testing.T) {
	r := newTestRouter(store.NewStore(0))

	w := performRequest(r, http.MethodPost, "/kv/name/setnx", `{"value":"eren"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("first setnx status = %d; want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	w = performRequest(r, http.MethodGet, "/kv/name", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET after setnx status = %d; want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"value":"eren"`) {
		t.Fatalf("GET after setnx body = %s; want value eren", w.Body.String())
	}

	w = performRequest(r, http.MethodPost, "/kv/name/setnx", `{"value":"other"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("second setnx status = %d; want %d; body=%s", w.Code, http.StatusConflict, w.Body.String())
	}

	w = performRequest(r, http.MethodGet, "/kv/name", "")
	if !strings.Contains(w.Body.String(), `"value":"eren"`) {
		t.Fatalf("GET after conflicting setnx body = %s; want original value eren", w.Body.String())
	}
}

func TestSetNXHandlerRejectsInvalidRequests(t *testing.T) {
	r := newTestRouter(store.NewStore(0))

	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "missing value", body: `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest(r, http.MethodPost, "/kv/bad/setnx", tt.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d; want %d; body=%s", w.Code, http.StatusBadRequest, w.Body.String())
			}
		})
	}
}
