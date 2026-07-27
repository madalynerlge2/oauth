package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TokenResponse represents the response from the token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// PollDeviceToken polls the token endpoint for the device flow.
func PollDeviceToken(ctx context.Context, client *http.Client, tokenURL string, clientID string, deviceCode string, interval time.Duration) (*TokenResponse, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			resp, err := pollTokenEndpoint(ctx, client, tokenURL, clientID, deviceCode)
			if err != nil {
				// Check if context was canceled during the request
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				default:
				}

				if err.Error() == "authorization_pending" {
					continue
				}
				if err.Error() == "slow_down" {
					interval += 5 * time.Second
					ticker.Reset(interval)
					continue
				}
				return nil, err
			}
			if resp != nil && resp.AccessToken != "" {
				return resp, nil
			}
		}
	}
}

func pollTokenEndpoint(ctx context.Context, client *http.Client, tokenURL string, clientID string, deviceCode string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errRes struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errRes); err == nil {
			if errRes.Error != "" {
				return nil, errors.New(errRes.Error)
			}
		}
		return nil, errors.New("unknown error")
	}

	var tokenRes TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return nil, err
	}
	return &tokenRes, nil
}