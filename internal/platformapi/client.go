package platformapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const maxErrorBodyBytes = 4096

type Client struct {
	baseURL   string
	accessKey string
	secretKey string
	client    *http.Client
}

type serviceEvent struct {
	HealthcheckUUID   string `json:"healthcheckUuid"`
	ReportedHealth    string `json:"reportedHealth"`
	ExternalTimestamp int64  `json:"externalTimestamp"`
}

func NewFromEnvironment() (*Client, error) {
	return New(
		os.Getenv("PLATFORM_URL"),
		os.Getenv("PLATFORM_ACCESS_KEY"),
		os.Getenv("PLATFORM_SECRET_KEY"),
	)
}

func New(baseURL, accessKey, secretKey string) (*Client, error) {
	if baseURL == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("PLATFORM_URL, PLATFORM_ACCESS_KEY, and PLATFORM_SECRET_KEY are required")
	}

	normalized, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:   normalized,
		accessKey: accessKey,
		secretKey: secretKey,
		client: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func normalizeBaseURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(raw, "/"))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid PLATFORM_URL")
	}

	path := strings.TrimRight(parsed.Path, "/")
	switch {
	case path == "":
		path = "/v2-beta"
	case strings.HasSuffix(path, "/v1"):
		path = strings.TrimSuffix(path, "/v1") + "/v2-beta"
	}
	parsed.Path = path
	return strings.TrimRight(parsed.String(), "/"), nil
}

func (c *Client) ReportStatus(ctx context.Context, healthcheckUUID, status string, timestamp int64) error {
	payload, err := json.Marshal(serviceEvent{
		HealthcheckUUID:   healthcheckUUID,
		ReportedHealth:    status,
		ExternalTimestamp: timestamp,
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/serviceevents", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.SetBasicAuth(c.accessKey, c.secretKey)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxErrorBodyBytes))
		return fmt.Errorf("service event API returned %s", response.Status)
	}
	return nil
}
