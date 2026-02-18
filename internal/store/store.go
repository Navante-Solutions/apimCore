package store

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/navantesolutions/apimcore/config"
)

type ApiProduct struct {
	ID          int64
	Name        string
	Slug        string
	Description string
	TenantID    string
	Published   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ApiDefinition struct {
	ID             int64
	ProductID      int64
	Name           string
	PathPrefix     string
	BackendURL     string
	OpenAPISpecURL string
	Version        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Subscription struct {
	ID              int64
	ProductID       int64
	DeveloperID     string
	TenantID        string
	Plan            string
	RateLimitPerMin int
	Active          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ApiKey struct {
	ID             int64
	SubscriptionID int64
	KeyHash        string
	KeyPrefix      string
	Name           string
	Active         bool
	CreatedAt      time.Time
	LastUsedAt     time.Time
}

type RequestUsage struct {
	ID              int64
	SubscriptionID  int64
	ApiDefinitionID int64
	TenantID        string
	Method          string
	Path            string
	StatusCode      int
	ResponseTimeMs  int64
	RequestedAt     time.Time
}

type Store struct {
	mu            sync.RWMutex
	products      map[int64]*ApiProduct
	definitions   map[int64]*ApiDefinition
	subscriptions map[int64]*Subscription
	keysByHash    map[string]*ApiKey
	keysByPrefix  map[string]*ApiKey
	usage         []RequestUsage
	nextProduct   int64
	nextDef       int64
	nextSub       int64
	nextKey       int64
	nextUsage     int64
}

func NewStore() *Store {
	return &Store{
		products:      make(map[int64]*ApiProduct),
		definitions:   make(map[int64]*ApiDefinition),
		subscriptions: make(map[int64]*Subscription),
		keysByHash:    make(map[string]*ApiKey),
		keysByPrefix:  make(map[string]*ApiKey),
		usage:         make([]RequestUsage, 0, 10000),
		nextProduct:   1,
		nextDef:       1,
		nextSub:       1,
		nextKey:       1,
		nextUsage:     1,
	}
}

func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.products = make(map[int64]*ApiProduct)
	s.definitions = make(map[int64]*ApiDefinition)
	s.subscriptions = make(map[int64]*Subscription)
	s.keysByHash = make(map[string]*ApiKey)
	s.keysByPrefix = make(map[string]*ApiKey)
	s.nextProduct = 1
	s.nextDef = 1
	s.nextSub = 1
	s.nextKey = 1
}

func (s *Store) PopulateFromConfig(cfg *config.Config) {
	s.Reset()

	productSlugToID := make(map[string]int64)

	for _, pc := range cfg.Products {
		p := &ApiProduct{
			Name:        pc.Name,
			Slug:        pc.Slug,
			Description: pc.Description,
			Published:   true,
		}
		id := s.CreateProduct(p)
		productSlugToID[pc.Slug] = id

		for _, ac := range pc.Apis {
			d := &ApiDefinition{
				ProductID:      id,
				Name:           ac.Name,
				PathPrefix:     ac.PathPrefix,
				BackendURL:     ac.BackendURL,
				OpenAPISpecURL: ac.OpenAPISpecURL,
				Version:        ac.Version,
			}
			s.CreateDefinition(d)
		}
	}

	for _, sc := range cfg.Subscriptions {
		productID, ok := productSlugToID[sc.ProductSlug]
		if !ok {
			continue
		}
		sub := &Subscription{
			ProductID:   productID,
			DeveloperID: sc.DeveloperID,
			Plan:        sc.Plan,
			Active:      true,
		}
		subID := s.CreateSubscription(sub)

		for _, kc := range sc.Keys {
			hash := hashKey(kc.Value)
			prefix := kc.Value
			if len(prefix) > 8 {
				prefix = prefix[:8]
			}
			k := &ApiKey{
				SubscriptionID: subID,
				KeyHash:        hash,
				KeyPrefix:      prefix,
				Name:           kc.Name,
				Active:         true,
			}
			s.CreateApiKey(k)
		}
	}
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func (s *Store) CreateProduct(p *ApiProduct) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextProduct
	s.nextProduct++
	p.CreatedAt = time.Now()
	p.UpdatedAt = p.CreatedAt
	s.products[p.ID] = cloneProduct(p)
	return p.ID
}

func (s *Store) GetProduct(id int64) *ApiProduct {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneProduct(s.products[id])
}

func (s *Store) ListProducts() []ApiProduct {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ApiProduct, 0, len(s.products))
	for _, p := range s.products {
		out = append(out, *cloneProduct(p))
	}
	return out
}

func (s *Store) CreateDefinition(d *ApiDefinition) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	d.ID = s.nextDef
	s.nextDef++
	d.CreatedAt = time.Now()
	d.UpdatedAt = d.CreatedAt
	s.definitions[d.ID] = cloneDefinition(d)
	return d.ID
}

func (s *Store) GetDefinition(id int64) *ApiDefinition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneDefinition(s.definitions[id])
}

func (s *Store) ListDefinitionsByProduct(productID int64) []ApiDefinition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []ApiDefinition
	for _, d := range s.definitions {
		if d.ProductID == productID {
			out = append(out, *cloneDefinition(d))
		}
	}
	return out
}

func (s *Store) CreateSubscription(sub *Subscription) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub.ID = s.nextSub
	s.nextSub++
	sub.CreatedAt = time.Now()
	sub.UpdatedAt = sub.CreatedAt
	s.subscriptions[sub.ID] = cloneSubscription(sub)
	return sub.ID
}

func (s *Store) GetSubscription(id int64) *Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSubscription(s.subscriptions[id])
}

func (s *Store) ListSubscriptionsByProduct(productID int64) []Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Subscription
	for _, sub := range s.subscriptions {
		if sub.ProductID == productID {
			out = append(out, *cloneSubscription(sub))
		}
	}
	return out
}

func (s *Store) CreateApiKey(k *ApiKey) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	k.ID = s.nextKey
	s.nextKey++
	k.CreatedAt = time.Now()
	k.LastUsedAt = k.CreatedAt
	s.keysByHash[k.KeyHash] = cloneKey(k)
	s.keysByPrefix[k.KeyPrefix] = cloneKey(k)
	return k.ID
}

func (s *Store) GetKeyByHash(hash string) *ApiKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneKey(s.keysByHash[hash])
}

func (s *Store) GetKeyByPrefix(prefix string) *ApiKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneKey(s.keysByPrefix[prefix])
}

func (s *Store) GetKeyByID(id int64) *ApiKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, k := range s.keysByHash {
		if k.ID == id {
			return cloneKey(k)
		}
	}
	return nil
}

func (s *Store) UpdateKeyLastUsed(id int64, t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range s.keysByHash {
		if k.ID == id {
			k.LastUsedAt = t
			return
		}
	}
}

func (s *Store) RecordUsage(u RequestUsage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u.ID = s.nextUsage
	s.nextUsage++
	u.RequestedAt = time.Now()
	s.usage = append(s.usage, u)
	if len(s.usage) > 100000 {
		s.usage = s.usage[len(s.usage)-50000:]
	}
}

func (s *Store) UsageSince(since time.Time) []RequestUsage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []RequestUsage
	for _, u := range s.usage {
		if !u.RequestedAt.Before(since) {
			out = append(out, u)
		}
	}
	return out
}

func (s *Store) UsageBySubscription(subID int64, since time.Time) []RequestUsage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []RequestUsage
	for _, u := range s.usage {
		if u.SubscriptionID == subID && !u.RequestedAt.Before(since) {
			out = append(out, u)
		}
	}
	return out
}

func (s *Store) UsageByApi(apiID int64, since time.Time) []RequestUsage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []RequestUsage
	for _, u := range s.usage {
		if u.ApiDefinitionID == apiID && !u.RequestedAt.Before(since) {
			out = append(out, u)
		}
	}
	return out
}

func cloneProduct(p *ApiProduct) *ApiProduct {
	if p == nil {
		return nil
	}
	q := *p
	return &q
}

func cloneDefinition(d *ApiDefinition) *ApiDefinition {
	if d == nil {
		return nil
	}
	c := *d
	return &c
}

func cloneSubscription(s *Subscription) *Subscription {
	if s == nil {
		return nil
	}
	c := *s
	return &c
}

func cloneKey(k *ApiKey) *ApiKey {
	if k == nil {
		return nil
	}
	c := *k
	return &c
}
