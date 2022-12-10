package restyss_test

import (
	"net/http"
	"testing"

	"github.com/ccmonky/restyss"
)

func TestR(t *testing.T) {
	r := restyss.R()
	if r == nil {
		t.Fatal("should not nil")
	}
	r = restyss.R("")
	if r == nil {
		t.Fatal("should not nil")
	}
	var sr *http.Request
	r = restyss.R("", sr)
	if r == nil {
		t.Fatal("should not nil")
	}
}
