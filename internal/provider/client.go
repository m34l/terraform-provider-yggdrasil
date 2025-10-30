package provider

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/m34l/terraform-provider-yggdrasil/internal/utils"
)

type APIClient struct {
	baseURL    string
	hc         *http.Client
	token      string
	apiVersion string
}

func newClient(cfg Config) (*APIClient, error) {
	tlsCfg := &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify} //nolint:gosec

	// CA
	if cfg.CACertPath != "" {
		ca, err := os.ReadFile(filepath.Clean(cfg.CACertPath))
		if err != nil {
			return nil, err
		}
		cp := x509.NewCertPool()
		cp.AppendCertsFromPEM(ca)
		tlsCfg.RootCAs = cp
	}

	// mTLS
	if cfg.ClientCertPath != "" && cfg.ClientKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCertPath, cfg.ClientKeyPath)
		if err != nil {
			return nil, err
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	hc := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}

	apiVersion := cfg.APIVersion
	if apiVersion == "" {
		apiVersion = "v2" // default to v2
	}

	return &APIClient{
		baseURL:    cfg.Endpoint,
		hc:         hc,
		token:      cfg.Token,
		apiVersion: apiVersion,
	}, nil
}

type SecretPayload struct {
	Namespace string            `json:"namespace"`
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Tags      map[string]string `json:"tags,omitempty"`
}

type SecretResponse struct {
	Namespace string            `json:"namespace"`
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Version   int               `json:"version"`
	Tags      map[string]string `json:"tags,omitempty"`
	UpdatedAt string            `json:"updated_at"`
}

func (c *APIClient) GetSecret(ns, key string) (*SecretResponse, error) {
	// GET /v2/configurations/:namespace/latest/all
	url := fmt.Sprintf("%s/%s/configurations/%s/latest/all", c.baseURL, c.apiVersion, ns)
	safeURL := utils.RedactURLQuery(url)
	log.Printf("[DEBUG] GET request to: %s", safeURL)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("token", c.token)

	// Log safe version of headers
	log.Printf("[DEBUG] Request headers: %v", utils.RedactHTTPHeaders(req.Header))

	res, err := c.hc.Do(req)
	if err != nil {
		log.Printf("[ERROR] HTTP request failed: %v", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	log.Printf("[DEBUG] Response status: %d", res.StatusCode)

	if res.StatusCode == 404 {
		return nil, nil
	}
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		safeBody := utils.RedactBytesChain(b)
		log.Printf("[ERROR] Get secret failed (status %d): %s", res.StatusCode, string(safeBody))
		return nil, fmt.Errorf("get secret failed (status %d): %s", res.StatusCode, string(b))
	}

	b, _ := io.ReadAll(res.Body)
	safeBody := utils.RedactBytesChain(b)
	log.Printf("[DEBUG] Response body: %s", string(safeBody))

	// Parse the response and extract the specific key
	var configs map[string]interface{}
	if err := json.Unmarshal(b, &configs); err != nil {
		log.Printf("[ERROR] Failed to decode JSON response: %v", err)
		return nil, fmt.Errorf("failed to decode response: %w (body: %s)", err, string(safeBody))
	}

	// Extract the specific key from configs
	if val, ok := configs[key]; ok {
		return &SecretResponse{
			Namespace: ns,
			Key:       key,
			Value:     fmt.Sprintf("%v", val),
			Version:   1, // Placeholder
			UpdatedAt: time.Now().Format(time.RFC3339),
		}, nil
	}

	return nil, nil
}

func (c *APIClient) UpsertSecret(p SecretPayload) (*SecretResponse, error) {
	// PUT /v2/configurations/:namespace
	url := fmt.Sprintf("%s/%s/configurations/%s", c.baseURL, c.apiVersion, p.Namespace)
	safeURL := utils.RedactURLQuery(url)
	log.Printf("[DEBUG] PUT request to: %s", safeURL)

	// Build the payload in the format Yggdrasil expects
	payload := map[string]interface{}{
		"configs": map[string]string{
			p.Key: p.Value,
		},
	}
	if len(p.Tags) > 0 {
		payload["tags"] = p.Tags
	}

	body, _ := json.Marshal(payload)
	safeBody := utils.RedactBytesChain(body)
	log.Printf("[DEBUG] Request body: %s", string(safeBody))

	req, _ := http.NewRequest("PUT", url, bytes.NewReader(body))
	req.Header.Set("token", c.token)
	req.Header.Set("Content-Type", "application/json")

	// Log safe version of headers
	safeHeaders := utils.RedactHTTPHeaders(req.Header)
	log.Printf("[DEBUG] Request headers: %v", safeHeaders)

	// Additional debug for token format
	if len(c.token) < 10 {
		log.Printf("[WARN] Token seems too short (length: %d), may be invalid", len(c.token))
	} else {
		log.Printf("[DEBUG] Token length: %d, preview: %s...", len(c.token), c.token[:min(8, len(c.token))])
	}

	res, err := c.hc.Do(req)
	if err != nil {
		log.Printf("[ERROR] HTTP request failed: %v", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	log.Printf("[DEBUG] Response status: %d", res.StatusCode)
	log.Printf("[DEBUG] Response headers: %v", utils.RedactHTTPHeaders(res.Header))

	b, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		log.Printf("[ERROR] Failed to read response body: %v", readErr)
	} else {
		safeRespBody := utils.RedactBytesChain(b)
		log.Printf("[DEBUG] Response body: %s", string(safeRespBody))
	}

	if res.StatusCode >= 300 {
		if readErr != nil {
			return nil, fmt.Errorf("upsert secret failed (status %d): unable to read response body: %w", res.StatusCode, readErr)
		}
		if len(b) == 0 {
			return nil, fmt.Errorf("upsert secret failed (status %d): empty response body", res.StatusCode)
		}

		// Special handling for 401
		if res.StatusCode == 401 {
			log.Printf("[ERROR] Authentication failed - check token validity and permissions")
			log.Printf("[DEBUG] Endpoint: %s", c.baseURL)
			log.Printf("[DEBUG] API Version: %s", c.apiVersion)
		}

		return nil, fmt.Errorf("upsert secret failed (status %d): %s", res.StatusCode, string(b))
	}

	if readErr != nil {
		return nil, fmt.Errorf("failed to read response body: %w", readErr)
	}

	// Return a success response
	out := &SecretResponse{
		Namespace: p.Namespace,
		Key:       p.Key,
		Value:     p.Value,
		Version:   1,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	log.Printf("[DEBUG] Successfully upserted secret")
	return out, nil
}

func (c *APIClient) DeleteSecret(ns, key string) error {
	// To delete a specific key, we need to update the namespace without that key
	// Or use the appropriate Yggdrasil API endpoint
	url := fmt.Sprintf("%s/%s/configurations/%s", c.baseURL, c.apiVersion, ns)
	safeURL := utils.RedactURLQuery(url)
	log.Printf("[DEBUG] PUT (delete) request to: %s", safeURL)

	// Send an empty value or use DELETE endpoint if available
	payload := map[string]interface{}{
		"configs": map[string]interface{}{
			key: nil, // or empty string to remove
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(body))
	req.Header.Set("token", c.token)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[DEBUG] Request headers: %v", utils.RedactHTTPHeaders(req.Header))

	res, err := c.hc.Do(req)
	if err != nil {
		log.Printf("[ERROR] HTTP request failed: %v", err)
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	log.Printf("[DEBUG] Response status: %d", res.StatusCode)

	if res.StatusCode == 404 {
		return nil
	}
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		safeBody := utils.RedactBytesChain(b)
		log.Printf("[ERROR] Delete secret failed (status %d): %s", res.StatusCode, string(safeBody))
		return fmt.Errorf("delete secret failed (status %d): %s", res.StatusCode, string(b))
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
