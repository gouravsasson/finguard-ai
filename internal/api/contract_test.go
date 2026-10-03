package api

import (
	"strings"
	"testing"
)

func TestOpenAPICoversRoutes(t *testing.T) {
	spec := string(openAPIYAML)
	s := &Server{}
	for _, pattern := range s.Patterns() {
		_, path, ok := strings.Cut(pattern, " ")
		if !ok {
			t.Fatal(pattern)
		}
		if !strings.Contains(spec, "\n  "+path+":") && !strings.Contains(spec, "\n"+path+":") {
			t.Errorf("openapi missing %s", path)
		}
	}
}
