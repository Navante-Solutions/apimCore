package devportal

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/navantesolutions/apimcore/internal/store"
)

func TestDevPortal_APIs(t *testing.T) {
	s := store.NewStore()
	pid1 := s.CreateProduct(&store.ApiProduct{Name: "Prod1", Slug: "p1", Published: true})
	pid2 := s.CreateProduct(&store.ApiProduct{Name: "Prod2", Slug: "p2", Published: false})

	s.CreateDefinition(&store.ApiDefinition{ProductID: pid1, Name: "Api1", PathPrefix: "/a1"})
	s.CreateDefinition(&store.ApiDefinition{ProductID: pid2, Name: "Api2", PathPrefix: "/a2"})

	h := New(s, "/devportal")

	t.Run("ListProducts - only published", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/devportal/api/products", nil)
		rec := httptest.NewRecorder()
		h.listProducts(rec, req)

		var products []store.ApiProduct
		if err := json.Unmarshal(rec.Body.Bytes(), &products); err != nil {
			t.Fatal(err)
		}

		if len(products) != 1 || products[0].Name != "Prod1" {
			t.Errorf("expected 1 published product (Prod1), got %d: %v", len(products), products)
		}
	})

	t.Run("ListAPIs - filter by product", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/devportal/api/apis?product_id=1", nil)
		rec := httptest.NewRecorder()
		h.listAPIs(rec, req)

		var apis []store.ApiDefinition
		if err := json.Unmarshal(rec.Body.Bytes(), &apis); err != nil {
			t.Fatal(err)
		}

		if len(apis) != 1 || apis[0].Name != "Api1" {
			t.Errorf("expected Api1, got %v", apis)
		}
	})
}

func TestDevPortal_Usage(t *testing.T) {
	s := store.NewStore()
	h := New(s, "/devportal")

	// Record some usage
	s.RecordUsage(store.RequestUsage{
		Path:        "/a1",
		StatusCode:  200,
		RequestedAt: time.Now(),
	})

	req := httptest.NewRequest("GET", "/devportal/api/usage", nil)
	rec := httptest.NewRecorder()
	h.usage(rec, req)

	var result struct {
		Total  int            `json:"total"`
		ByPath map[string]int `json:"by_path"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}
	if result.ByPath["/a1"] != 1 {
		t.Errorf("expected path /a1 count 1, got %d", result.ByPath["/a1"])
	}
}
