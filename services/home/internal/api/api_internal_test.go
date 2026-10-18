package api

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http/httptest"
	"testing"

	"github.com/codedogapp/bakhyard/services/home/internal/screen"
)

// TestEmptyHome checks that a Home with no items still sends them as [], since the spec requires
// an array. The demo data is never empty, so this case can't be reached through Register yet.
func TestEmptyHome(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := GetHome200JSONResponse(toHome(screen.Home{})).VisitGetHomeResponse(rec); err != nil {
		t.Fatalf("encode: %v", err)
	}

	var got struct {
		Items jsontext.Value `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if string(got.Items) != "[]" {
		t.Errorf("items = %s, want []", got.Items)
	}
}
