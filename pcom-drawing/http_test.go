package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIValidationAndExport(t *testing.T) {
	setupPresets(t)
	h := newHandler()
	body, _ := json.Marshal(defaultConfig())
	for _, path := range []string{"/api/preview", "/api/export/pptx", "/api/export/svg", "/api/export/json"} {
		r := httptest.NewRequest("POST", "http://127.0.0.1:8000"+path, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s %d %s", path, w.Code, w.Body.String())
		}
		if w.Body.Len() < 500 {
			t.Fatalf("empty export %s", path)
		}
	}
	bad := defaultConfig()
	bad.BitCode = "AF"
	body, _ = json.Marshal(bad)
	r := httptest.NewRequest("POST", "http://localhost/api/preview", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 422 {
		t.Fatal(w.Code)
	}
}
func TestOnlyLocalSameOriginRequests(t *testing.T) {
	setupPresets(t)
	h := newHandler()
	for _, tc := range []struct{ host, origin string }{{"attacker.invalid", ""}, {"127.0.0.1:8000", "https://unrelated.invalid"}} {
		r := httptest.NewRequest("GET", "http://"+tc.host+"/api/catalog", nil)
		if tc.origin != "" {
			r.Header.Set("Origin", tc.origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("foreignrequestaccepted: %d", w.Code)
		}
	}
}
