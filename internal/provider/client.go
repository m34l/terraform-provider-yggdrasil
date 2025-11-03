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
	"strings"
	"time"

	"github.com/m34l/terraform-provider-yggdrasil/internal/utils"
)

type APIClient struct {
	baseURL     string
	hc          *http.Client
	token       string
	credKey     string // NEW: for write operations
	credSecret  string // NEW: for write operations
	apiVersion  string
	authHeader  string
	insecureTLS bool
}

func newClient(cfg Config) (*APIClient, error) {
	// TLS setup
	tlsCfg := &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify} //nolint:gosec

	// Custom CA
	if strings.TrimSpace(cfg.CACertPath) != "" {
		ca, err := os.ReadFile(filepath.Clean(cfg.CACertPath))
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		cp := x509.NewCertPool()
		if ok := cp.AppendCertsFromPEM(ca); !ok {
			return nil, fmt.Errorf("append CA cert failed")
		}
		tlsCfg.RootCAs = cp
	}

	// mTLS client cert
	if strings.TrimSpace(cfg.ClientCertPath) != "" && strings.TrimSpace(cfg.ClientKeyPath) != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCertPath, cfg.ClientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load client keypair: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	hc := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}

	apiVersion := strings.TrimSpace(cfg.APIVersion)
	if apiVersion == "" {
		apiVersion = "v2"
	}

	// Default header ke "token" (sesuai backend kamu)
	authHeader := strings.TrimSpace(cfg.AuthHeader)
	if authHeader == "" {
		authHeader = "token"
	}

	return &APIClient{
		baseURL:     strings.TrimRight(cfg.Endpoint, "/"),
		hc:          hc,
		token:       cfg.Token,
		credKey:     cfg.CredKey,    // NEW
		credSecret:  cfg.CredSecret, // NEW
		apiVersion:  apiVersion,
		authHeader:  authHeader,
		insecureTLS: cfg.InsecureSkipVerify,
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

func (c *APIClient) setAuth(req *http.Request) {
	if strings.TrimSpace(c.token) == "" {
		log.Printf("[WARN] Token is empty, request may fail authentication")
		return
	}
	// Yggdrasil expects: token: <value>
	req.Header.Set("token", c.token)
	log.Printf("[DEBUG] Setting auth header 'token' with value: %s", utils.RedactString(c.token))
}

// NEW: For write operations using key-secret pair
func (c *APIClient) setAuthKeySecret(req *http.Request) {
	if strings.TrimSpace(c.credKey) == "" || strings.TrimSpace(c.credSecret) == "" {
		log.Printf("[WARN] Credential key-secret is empty, falling back to token authentication")
		c.setAuth(req)
		return
	}

	// Yggdrasil expects separate headers: "key" and "secret"
	req.Header.Set("key", c.credKey)
	req.Header.Set("secret", c.credSecret)

	log.Printf("[DEBUG] Setting key-secret authentication")
	log.Printf("[DEBUG]   Key: %s (length: %d)", utils.RedactString(c.credKey), len(c.credKey))
	log.Printf("[DEBUG]   Secret length: %d", len(c.credSecret))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *APIClient) GetSecret(ns, key string, tag string) (*SecretResponse, error) {
	// GET /v2/configurations/:namespace/latest/all
	url := fmt.Sprintf("%s/%s/configurations/%s/latest/all", c.baseURL, c.apiVersion, ns)

	req, _ := http.NewRequest("GET", url, nil)
	c.setAuth(req)
	req.Header.Set("Accept", "application/json")

	log.Printf("[DEBUG] GET %s (fetching all tags)", utils.RedactURLQuery(url))
	if tag != "" {
		log.Printf("[DEBUG] Will extract value from tag: %s", tag)
	} else {
		log.Printf("[DEBUG] No tag specified, will try to get from any available tag")
	}
	logTLS(c)

	res, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	b, _ := io.ReadAll(res.Body)
	log.Printf("[DEBUG] Response status: %d", res.StatusCode)
	log.Printf("[DEBUG] Raw response body: %s", string(b))

	if res.StatusCode == http.StatusNotFound {
		log.Printf("[DEBUG] Namespace not found")
		return nil, nil
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("get secret failed (status %d): %s", res.StatusCode, string(utils.RedactBytesChain(b)))
	}

	var apiResp struct {
		Success          bool                      `json:"success"`
		Data             map[string]map[string]any `json:"data"` // tag -> configs
		APIVersion       string                    `json:"api_version"`
		NamespaceVersion string                    `json:"namespace_version"`
	}
	if err := json.Unmarshal(b, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w (body: %s)", err, string(b))
	}

	log.Printf("[DEBUG] Decoded response - Success: %v, Available tags: %v", apiResp.Success, getMapKeys(apiResp.Data))

	if !apiResp.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	// If tag is specified, get from that specific tag
	if tag != "" {
		tagConfigs, tagExists := apiResp.Data[tag]
		if !tagExists {
			log.Printf("[DEBUG] Tag '%s' not found. Available tags: %v", tag, getMapKeys(apiResp.Data))
			return nil, nil
		}

		rawValue, keyExists := tagConfigs[key]
		if !keyExists {
			log.Printf("[DEBUG] Key '%s' not found in tag '%s'. Available keys: %v", key, tag, getMapKeys(tagConfigs))
			return nil, nil
		}

		value := fmt.Sprintf("%v", rawValue)
		log.Printf("[DEBUG] Found key '%s' in tag '%s' with value: [REDACTED] (length: %d)", key, tag, len(value))

		return &SecretResponse{
			Namespace: ns,
			Key:       key,
			Value:     value,
			Version:   1,
			UpdatedAt: time.Now().Format(time.RFC3339),
		}, nil
	}

	// If no tag specified, try to find key in any available tag (for resource Read compatibility)
	for tagName, tagConfigs := range apiResp.Data {
		if rawValue, exists := tagConfigs[key]; exists {
			value := fmt.Sprintf("%v", rawValue)
			log.Printf("[DEBUG] Found key '%s' in tag '%s' (auto-detected) with value: [REDACTED] (length: %d)", key, tagName, len(value))

			return &SecretResponse{
				Namespace: ns,
				Key:       key,
				Value:     value,
				Version:   1,
				UpdatedAt: time.Now().Format(time.RFC3339),
			}, nil
		}
	}

	log.Printf("[DEBUG] Key '%s' not found in any tag. Available tags: %v", key, getMapKeys(apiResp.Data))
	return nil, nil
}

// Helper to get map keys for debugging
func getMapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (c *APIClient) UpsertSecret(p SecretPayload) (*SecretResponse, error) {
	ns := p.Namespace
	key := p.Key
	value := p.Value

	// If tags specified, use overrides endpoint
	if len(p.Tags) > 0 {
		return c.upsertSecretWithTags(ns, key, value, p.Tags)
	}

	// Otherwise, use base configuration endpoint
	return c.upsertSecretBase(ns, key, value)
}

// upsertSecretBase updates base configuration (no tags)
func (c *APIClient) upsertSecretBase(ns, key, value string) (*SecretResponse, error) {
	// Get existing configs to merge
	existingConfigs, err := c.getAllConfigs(ns)
	if err != nil {
		log.Printf("[WARN] Failed to get existing configs: %v, will create new", err)
		existingConfigs = make(map[string]any)
	}

	// Add/update our key
	existingConfigs[key] = value

	// PUT /v2/configurations/:namespace
	url := fmt.Sprintf("%s/%s/configurations/%s", c.baseURL, c.apiVersion, ns)

	// Build configurations - simple key-value pairs for base config
	configurations := make(map[string]map[string]string)
	for k, v := range existingConfigs {
		configurations[k] = map[string]string{
			"type": fmt.Sprintf("%v", v),
		}
	}

	payload := map[string]any{
		"search_tags":    []string{},
		"configurations": configurations,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PUT", url, bytes.NewReader(body))
	c.setAuthKeySecret(req) // CHANGED: use key-secret for write
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	log.Printf("[DEBUG] PUT %s", utils.RedactURLQuery(url))
	log.Printf("[DEBUG] Request body: %s", string(utils.RedactBytesChain(body)))
	logTLS(c)

	res, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, _ := io.ReadAll(res.Body)
	log.Printf("[DEBUG] Response status: %d", res.StatusCode)

	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("upsert secret failed (status %d): %s", res.StatusCode, string(utils.RedactBytesChain(respBody)))
	}

	return &SecretResponse{
		Namespace: ns,
		Key:       key,
		Value:     value,
		Version:   1,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// upsertSecretWithTags updates configuration with specific tags using overrides endpoint
func (c *APIClient) upsertSecretWithTags(ns, key, value string, tags map[string]string) (*SecretResponse, error) {
	// STEP 1: Ensure base configuration exists
	log.Printf("[DEBUG] Step 1: Checking if base config exists for key '%s'", key)
	existingConfigs, err := c.getAllConfigs(ns)
	if err != nil {
		log.Printf("[WARN] Failed to get existing configs: %v", err)
		existingConfigs = make(map[string]any)
	}

	// If key doesn't exist in base config, create it first
	if _, exists := existingConfigs[key]; !exists {
		log.Printf("[DEBUG] Key '%s' not found in base config, creating base configuration first", key)

		// Create base configuration with empty/placeholder value
		existingConfigs[key] = "" // Empty base value

		// PUT /v2/configurations/:namespace (base config)
		baseURL := fmt.Sprintf("%s/%s/configurations/%s", c.baseURL, c.apiVersion, ns)
		configurations := make(map[string]map[string]string)
		for k, v := range existingConfigs {
			configurations[k] = map[string]string{
				"type": fmt.Sprintf("%v", v),
			}
		}

		basePayload := map[string]any{
			"search_tags":    []string{},
			"configurations": configurations,
		}

		baseBody, _ := json.Marshal(basePayload)
		baseReq, _ := http.NewRequest("PUT", baseURL, bytes.NewReader(baseBody))
		c.setAuthKeySecret(baseReq)
		baseReq.Header.Set("Content-Type", "application/json")
		baseReq.Header.Set("Accept", "application/json")

		log.Printf("[DEBUG] Creating base config: PUT %s", utils.RedactURLQuery(baseURL))
		log.Printf("[DEBUG] Base config body: %s", string(utils.RedactBytesChain(baseBody)))

		baseRes, err := c.hc.Do(baseReq)
		if err != nil {
			return nil, fmt.Errorf("failed to create base config: %w", err)
		}
		defer baseRes.Body.Close()

		baseRespBody, _ := io.ReadAll(baseRes.Body)
		log.Printf("[DEBUG] Base config response status: %d", baseRes.StatusCode)
		log.Printf("[DEBUG] Base config response body: %s", string(utils.RedactBytesChain(baseRespBody)))

		if baseRes.StatusCode >= 300 {
			return nil, fmt.Errorf("create base config failed (status %d): %s", baseRes.StatusCode, string(utils.RedactBytesChain(baseRespBody)))
		}

		log.Printf("[DEBUG] Base config created successfully for key '%s'", key)
	} else {
		log.Printf("[DEBUG] Key '%s' already exists in base config", key)
	}

	// STEP 2: Now create the override for specific tag
	log.Printf("[DEBUG] Step 2: Creating override for tags: %v", getMapKeys(tags))
	url := fmt.Sprintf("%s/%s/configurations/%s/overrides", c.baseURL, c.apiVersion, ns)

	// Extract tag names (rels)
	rels := make([]string, 0, len(tags))
	for tagName := range tags {
		rels = append(rels, tagName)
	}

	// FIXED: Build overrides payload - value should be direct string, not nested object
	overrides := make(map[string]string)
	overrides[key] = value

	payload := map[string]any{
		"rels":      rels,
		"overrides": overrides,
	}

	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PUT", url, bytes.NewReader(body))
	c.setAuthKeySecret(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	log.Printf("[DEBUG] PUT %s", utils.RedactURLQuery(url))
	log.Printf("[DEBUG] Request body: %s", string(utils.RedactBytesChain(body)))
	logTLS(c)

	res, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, _ := io.ReadAll(res.Body)
	log.Printf("[DEBUG] Response status: %d", res.StatusCode)
	log.Printf("[DEBUG] Response body: %s", string(utils.RedactBytesChain(respBody)))

	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("upsert secret with tags failed (status %d): %s", res.StatusCode, string(utils.RedactBytesChain(respBody)))
	}

	return &SecretResponse{
		Namespace: ns,
		Key:       key,
		Value:     value,
		Version:   1,
		UpdatedAt: time.Now().Format(time.RFC3339),
		Tags:      tags,
	}, nil
}

func (c *APIClient) getAllConfigs(ns string) (map[string]any, error) {
	url := fmt.Sprintf("%s/%s/configurations/%s/latest", c.baseURL, c.apiVersion, ns)

	req, _ := http.NewRequest("GET", url, nil)
	c.setAuth(req)
	req.Header.Set("Accept", "application/json")

	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return make(map[string]any), nil
	}

	b, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d: %s", res.StatusCode, string(b))
	}

	var apiResp struct {
		Success bool              `json:"success"`
		Data    map[string]string `json:"data"` // CHANGED: direct string values
	}
	if err := json.Unmarshal(b, &apiResp); err != nil {
		return nil, err
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	// Convert to map[string]any for compatibility
	result := make(map[string]any, len(apiResp.Data))
	for k, v := range apiResp.Data {
		result[k] = v
	}

	return result, nil
}

func (c *APIClient) DeleteSecret(ns, key string) error {
	return c.deleteSecretWithRetry(ns, key, 3)
}

func (c *APIClient) deleteSecretWithRetry(ns, key string, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Printf("[DEBUG] Retrying delete after %v (attempt %d/%d)", backoff, attempt, maxRetries)
			time.Sleep(backoff)
		}

		err := c.deleteSecretOnce(ns, key)
		if err == nil {
			return nil
		}

		// Check if it's a lock error (409)
		if strings.Contains(err.Error(), "status 409") || strings.Contains(err.Error(), "lock") {
			log.Printf("[WARN] Lock contention detected, will retry: %v", err)
			lastErr = err
			continue
		}

		// Other errors, fail immediately
		return err
	}

	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (c *APIClient) deleteSecretOnce(ns, key string) error {
	// Get all existing configs
	existingConfigs, err := c.getAllConfigs(ns)
	if err != nil {
		return fmt.Errorf("failed to get existing configs: %w", err)
	}

	// Remove the key
	delete(existingConfigs, key)

	// Update with remaining configs
	url := fmt.Sprintf("%s/%s/configurations/%s", c.baseURL, c.apiVersion, ns)

	configurations := make(map[string]map[string]string)
	for k, v := range existingConfigs {
		configurations[k] = map[string]string{
			"type": fmt.Sprintf("%v", v),
		}
	}

	payload := map[string]any{
		"search_tags":    []string{},
		"configurations": configurations,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PUT", url, bytes.NewReader(body))
	c.setAuthKeySecret(req)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[DEBUG] PUT (delete) %s", utils.RedactURLQuery(url))

	res, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	b, _ := io.ReadAll(res.Body)
	log.Printf("[DEBUG] Response status: %d", res.StatusCode)

	if res.StatusCode == http.StatusNotFound {
		return nil
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("delete secret failed (status %d): %s", res.StatusCode, string(utils.RedactBytesChain(b)))
	}
	return nil
}

// --- helpers

func logTLS(c *APIClient) {
	tr, ok := c.hc.Transport.(*http.Transport)
	if !ok || tr.TLSClientConfig == nil {
		return
	}
	log.Printf("[DEBUG] TLS InsecureSkipVerify: %v", tr.TLSClientConfig.InsecureSkipVerify)
	log.Printf("[DEBUG] TLS Has Client Cert: %v", len(tr.TLSClientConfig.Certificates) > 0)
	log.Printf("[DEBUG] TLS Has RootCAs: %v", tr.TLSClientConfig.RootCAs != nil)
}
