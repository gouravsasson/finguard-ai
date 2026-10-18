package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSPARoutesDoNotRedirect(t *testing.T) {
	s := &Server{}
	for _, path := range []string{"/", "/login", "/workspace", "/cases/example", "/index.html"} {
		t.Run(path, func(t *testing.T) {
			res := httptest.NewRecorder()
			s.static(res, httptest.NewRequest(http.MethodGet, path, nil))
			if res.Code != http.StatusOK || res.Header().Get("Location") != "" || !strings.Contains(res.Body.String(), `<div id="root">`) {
				t.Fatalf("SPA route %s: status=%d location=%q", path, res.Code, res.Header().Get("Location"))
			}
		})
	}
}
func TestMarketPageIsNotTheSignInApp(t *testing.T) {
	s := &Server{}
	res := httptest.NewRecorder()
	s.static(res, httptest.NewRequest(http.MethodGet, "/market/index.html?brand=mark", nil))
	if res.Code == http.StatusNotFound {
		t.Skip("market page is copied into the image by the frontend build")
	}
	if res.Code != http.StatusOK || res.Header().Get("Location") != "" || strings.Contains(res.Body.String(), `<div id="root">`) {
		t.Fatalf("market page: status=%d location=%q", res.Code, res.Header().Get("Location"))
	}
}

func TestMissingAssetsAreNotHTML(t *testing.T) {
	s := &Server{}
	for _, path := range []string{"/assets/missing.js", "/images/missing.jpg"} {
		res := httptest.NewRecorder()
		s.static(res, httptest.NewRequest(http.MethodGet, path, nil))
		if res.Code != http.StatusNotFound {
			t.Fatalf("%s: %d", path, res.Code)
		}
	}
}
