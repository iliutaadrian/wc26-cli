package api

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	baseURL     = "https://api.football-data.org/v4"
	competition = "WC" // FIFA World Cup
)

// Client talks to football-data.org and caches responses on disk so the
// free-tier rate limit (10 req/min) is comfortably respected.
type Client struct {
	token    string
	http     *http.Client
	cacheDir string
	// cacheTTL controls how long a cached body is considered fresh.
	cacheTTL time.Duration
}

// NewClient builds a client. token is the football-data.org X-Auth-Token.
func NewClient(token string, cacheTTL time.Duration) *Client {
	dir, _ := os.UserCacheDir()
	cacheDir := filepath.Join(dir, "wc26")
	_ = os.MkdirAll(cacheDir, 0o755)
	return &Client{
		token:    token,
		http:     &http.Client{Timeout: 15 * time.Second},
		cacheDir: cacheDir,
		cacheTTL: cacheTTL,
	}
}

func (c *Client) cachePath(key string) string {
	sum := sha1.Sum([]byte(key))
	return filepath.Join(c.cacheDir, hex.EncodeToString(sum[:])+".json")
}

// get fetches path, using the on-disk cache when it is still fresh.
// forceFresh bypasses the freshness check (used for manual refresh / live data).
func (c *Client) get(path string, forceFresh bool) ([]byte, error) {
	cp := c.cachePath(path)

	if !forceFresh {
		if info, err := os.Stat(cp); err == nil {
			if time.Since(info.ModTime()) < c.cacheTTL {
				if data, err := os.ReadFile(cp); err == nil {
					return data, nil
				}
			}
		}
	}

	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("X-Auth-Token", c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		// Network error: fall back to any cached copy, even if stale.
		if data, rerr := os.ReadFile(cp); rerr == nil {
			return data, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		_ = os.WriteFile(cp, body, 0o644)
		return body, nil
	case http.StatusTooManyRequests:
		// Rate limited: serve stale cache if we have it.
		if data, rerr := os.ReadFile(cp); rerr == nil {
			return data, nil
		}
		return nil, fmt.Errorf("rate limited (free tier is 10 req/min) — try again shortly")
	case http.StatusForbidden, http.StatusUnauthorized:
		hint := "set a token via WC_API_TOKEN env var or --token (get one free at football-data.org)"
		return nil, fmt.Errorf("not authorized (%d): %s", resp.StatusCode, hint)
	default:
		msg := strings.TrimSpace(string(body))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, msg)
	}
}

// Matches returns all World Cup matches.
func (c *Client) Matches(forceFresh bool) (*MatchesResponse, error) {
	body, err := c.get("/competitions/"+competition+"/matches", forceFresh)
	if err != nil {
		return nil, err
	}
	var out MatchesResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding matches: %w", err)
	}
	return &out, nil
}

// Standings returns the group standings.
func (c *Client) Standings(forceFresh bool) (*StandingsResponse, error) {
	body, err := c.get("/competitions/"+competition+"/standings", forceFresh)
	if err != nil {
		return nil, err
	}
	var out StandingsResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding standings: %w", err)
	}
	return &out, nil
}

// Scorers returns the top scorers.
func (c *Client) Scorers(forceFresh bool) (*ScorersResponse, error) {
	body, err := c.get("/competitions/"+competition+"/scorers?limit=30", forceFresh)
	if err != nil {
		return nil, err
	}
	var out ScorersResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decoding scorers: %w", err)
	}
	return &out, nil
}
