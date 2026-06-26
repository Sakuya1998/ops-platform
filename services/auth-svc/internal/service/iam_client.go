package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type IAMClient struct {
	baseURL    string
	httpClient *http.Client
}

type IAMRole struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type apiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func NewIAMClient(baseURL string) *IAMClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://localhost:8081"
	}
	return &IAMClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *IAMClient) GetUserRoleCodes(ctx context.Context, userID string) ([]string, error) {
	endpoint := c.baseURL + "/internal/v1/users/" + url.PathEscape(userID) + "/roles"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("iam returned status %d", resp.StatusCode)
	}
	var out apiResponse[[]IAMRole]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	roles := make([]string, 0, len(out.Data))
	for _, role := range out.Data {
		if role.Code != "" {
			roles = append(roles, role.Code)
		}
	}
	return roles, nil
}
