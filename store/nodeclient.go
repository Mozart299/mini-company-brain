// NodeClient implements Store by making HTTP calls to a remote NodeServer.
// The ReplicatedStore holds one NodeClient per store node in the cluster.
package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"company-brain/pkg/types"
)

type NodeClient struct {
	baseURL string
	http    *http.Client
}

func NewNodeClient(addr string) *NodeClient {
	return &NodeClient{
		baseURL: "http://" + addr,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *NodeClient) Set(ctx context.Context, fact types.Fact) error {
	data, err := json.Marshal(fact)
	if err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+"/facts/"+url.PathEscape(fact.Key), bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("set %s: %w", fact.Key, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set %s: status %d: %s", fact.Key, resp.StatusCode, body)
	}
	return nil
}

func (c *NodeClient) Get(ctx context.Context, key string) (types.Fact, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/facts/"+url.PathEscape(key), nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return types.Fact{}, fmt.Errorf("get %s: %w", key, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return types.Fact{}, fmt.Errorf("key not found: %s", key)
	}
	var fact types.Fact
	if err := json.NewDecoder(resp.Body).Decode(&fact); err != nil {
		return types.Fact{}, err
	}
	return fact, nil
}

func (c *NodeClient) Delete(ctx context.Context, key string) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete,
		c.baseURL+"/facts/"+url.PathEscape(key), nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	resp.Body.Close()
	return nil
}

func (c *NodeClient) List(ctx context.Context, prefix string) ([]types.Fact, error) {
	u := c.baseURL + "/facts"
	if prefix != "" {
		u += "?prefix=" + url.QueryEscape(prefix)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	defer resp.Body.Close()
	var facts []types.Fact
	if err := json.NewDecoder(resp.Body).Decode(&facts); err != nil {
		return nil, err
	}
	return facts, nil
}
