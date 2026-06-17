package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// TripClient is a typed HTTP client for the Trip Service internal endpoint.
type TripClient struct {
	baseURL string
	http    *http.Client
}

func NewTripClient(addr string) *TripClient {
	return &TripClient{
		baseURL: "http://" + addr,
		http: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 20,
			},
		},
	}
}

// AssignDriver calls the Trip Service internal endpoint POST /internal/trips/{id}/assign
func (c *TripClient) AssignDriver(ctx context.Context, tripID, driverID uuid.UUID) error {
	body, _ := json.Marshal(map[string]string{"driver_id": driverID.String()})
	url := fmt.Sprintf("%s/internal/trips/%s/assign", c.baseURL, tripID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("assign driver request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("assign driver: unexpected status %d", resp.StatusCode)
	}
	return nil
}
