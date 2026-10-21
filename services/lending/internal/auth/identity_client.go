package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"borrowhub/services/lending/internal/domain"
	"github.com/google/uuid"
)

type MemberInfo struct {
	IsMember       bool      `json:"is_member"`
	Role           string    `json:"role"`
	OrganizationID uuid.UUID `json:"organization_id"`
	UserID         uuid.UUID `json:"user_id"`
}

type IdentityClient interface {
	GetMember(ctx context.Context, orgID, userID uuid.UUID) (*MemberInfo, error)
}

type cachedMember struct {
	info      *MemberInfo
	expiresAt time.Time
}

const defaultMaxCacheEntries = 1000

type HTTPIdentityClient struct {
	baseURL       string
	internalToken string
	httpClient    *http.Client
	cacheTTL      time.Duration
	cacheMu       sync.RWMutex
	cache         map[string]cachedMember
	maxEntries    int
}

func NewHTTPIdentityClient(baseURL, internalToken string, cacheTTL time.Duration) *HTTPIdentityClient {
	if cacheTTL <= 0 {
		cacheTTL = 30 * time.Second
	}
	return &HTTPIdentityClient{
		baseURL:       baseURL,
		internalToken: internalToken,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
		cacheTTL:      cacheTTL,
		cache:         make(map[string]cachedMember),
		maxEntries:    defaultMaxCacheEntries,
	}
}

func (c *HTTPIdentityClient) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*MemberInfo, error) {
	cacheKey := fmt.Sprintf("%s:%s", orgID, userID)

	c.cacheMu.RLock()
	cached, found := c.cache[cacheKey]
	c.cacheMu.RUnlock()

	if found && time.Now().Before(cached.expiresAt) {
		return cached.info, nil
	}

	url := fmt.Sprintf("%s/api/v1/internal/organizations/%s/members/%s", c.baseURL, orgID, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating identity request: %w", err)
	}

	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling identity service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status from identity: %d", resp.StatusCode)
	}

	var response struct {
		Data MemberInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding identity response: %w", err)
	}

	c.cacheMu.Lock()
	now := time.Now()
	maxCap := c.maxEntries
	if maxCap <= 0 {
		maxCap = defaultMaxCacheEntries
	}
	if len(c.cache) >= maxCap {
		for k, v := range c.cache {
			if now.After(v.expiresAt) {
				delete(c.cache, k)
			}
		}
		if len(c.cache) >= maxCap {
			for k := range c.cache {
				delete(c.cache, k)
				break
			}
		}
	}
	c.cache[cacheKey] = cachedMember{
		info:      &response.Data,
		expiresAt: now.Add(c.cacheTTL),
	}
	c.cacheMu.Unlock()

	return &response.Data, nil
}
