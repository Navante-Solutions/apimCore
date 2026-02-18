package store

import (
	"testing"

	"github.com/navantesolutions/apimcore/config"
)

func TestPopulateFromConfig(t *testing.T) {
	s := NewStore()
	cfg := &config.Config{
		Products: []config.ProductConfig{
			{
				Name: "Prod1",
				Slug: "slug1",
				Apis: []config.ApiConfig{
					{
						Name:       "Api1",
						PathPrefix: "/p1",
						BackendURL: "http://b1",
					},
				},
			},
		},
		Subscriptions: []config.SubscriptionConfig{
			{
				DeveloperID: "dev1",
				ProductSlug: "slug1",
				Keys: []config.KeyConfig{
					{Name: "k1", Value: "v1"},
				},
			},
		},
	}

	s.PopulateFromConfig(cfg)

	prods := s.ListProducts()
	if len(prods) != 1 || prods[0].Slug != "slug1" {
		t.Errorf("expected 1 product with slug1, got %v", prods)
	}

	defs := s.ListDefinitionsByProduct(prods[0].ID)
	if len(defs) != 1 || defs[0].Name != "Api1" {
		t.Errorf("expected 1 api definition for product, got %v", defs)
	}

	hash := hashKey("v1")
	key := s.GetKeyByHash(hash)
	if key == nil || key.Name != "k1" {
		t.Error("expected api key k1 for value v1 to be in store")
	}
}

func TestReset(t *testing.T) {
	s := NewStore()
	s.CreateProduct(&ApiProduct{Name: "P", Slug: "S", Published: true})

	if len(s.ListProducts()) == 0 {
		t.Fatal("expected product to be created")
	}

	s.Reset()

	if len(s.ListProducts()) != 0 {
		t.Error("expected store to be empty after reset")
	}
}
