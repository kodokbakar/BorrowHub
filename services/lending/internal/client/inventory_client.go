package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"borrowhub/services/lending/internal/domain"
	"github.com/google/uuid"
)

type InventoryUnit struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	ItemID         uuid.UUID `json:"item_id"`
	SerialNumber   *string   `json:"serial_number,omitempty"`
	Barcode        *string   `json:"barcode,omitempty"`
	Status         string    `json:"status"`
	Condition      string    `json:"condition"`
}

type InventoryClient interface {
	GetEligibleUnits(ctx context.Context, itemID uuid.UUID) ([]InventoryUnit, error)
	TransitionUnitStatus(ctx context.Context, unitID uuid.UUID, fromStatus, toStatus string) (*InventoryUnit, error)
}

type HTTPInventoryClient struct {
	baseURL       string
	internalToken string
	httpClient    *http.Client
}

func NewHTTPInventoryClient(baseURL, internalToken string) *HTTPInventoryClient {
	return &HTTPInventoryClient{
		baseURL:       baseURL,
		internalToken: internalToken,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *HTTPInventoryClient) GetEligibleUnits(ctx context.Context, itemID uuid.UUID) ([]InventoryUnit, error) {
	url := fmt.Sprintf("%s/api/v1/internal/items/%s/eligible-units", c.baseURL, itemID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating eligible units request: %w", err)
	}
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling inventory service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inventory returned status %d", resp.StatusCode)
	}

	var response struct {
		Data []InventoryUnit `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding eligible units: %w", err)
	}

	return response.Data, nil
}

func (c *HTTPInventoryClient) TransitionUnitStatus(ctx context.Context, unitID uuid.UUID, fromStatus, toStatus string) (*InventoryUnit, error) {
	url := fmt.Sprintf("%s/api/v1/internal/units/%s/status", c.baseURL, unitID)
	payload := map[string]string{
		"from_status": fromStatus,
		"to_status":   toStatus,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling status request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating unit status request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling inventory service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrNotFound
	}
	if resp.StatusCode == http.StatusConflict {
		return nil, domain.ErrConflict
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inventory returned status %d", resp.StatusCode)
	}

	var response struct {
		Data InventoryUnit `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding unit response: %w", err)
	}

	return &response.Data, nil
}
