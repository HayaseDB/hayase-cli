package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/log"
	"golang.org/x/time/rate"
)

const (
	defaultBaseURL    = "https://aniworld.to"
	defaultUserAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	defaultTimeout    = 15 * time.Second
	defaultMaxRetries = 3
	defaultRateLimit  = 5.0
)

type ClientOption func(*Client)

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

func WithMaxRetries(retries int) ClientOption {
	return func(c *Client) {
		c.maxRetries = retries
	}
}

func WithRateLimit(rps float64) ClientOption {
	return func(c *Client) {
		c.rateLimiter = rate.NewLimiter(rate.Limit(rps), 1)
	}
}

func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

type Client struct {
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	baseURL     string
	userAgent   string
	maxRetries  int
}

func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		rateLimiter: rate.NewLimiter(rate.Limit(defaultRateLimit), 1),
		baseURL:     defaultBaseURL,
		userAgent:   defaultUserAgent,
		maxRetries:  defaultMaxRetries,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	url := c.baseURL + path

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			log.Debug("retrying request", "attempt", attempt, "backoff", backoff, "url", url)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9,de;q=0.8")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = &NetworkError{URL: url, Err: err}
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			_ = resp.Body.Close()
			lastErr = ErrRateLimited
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			_ = resp.Body.Close()
			return nil, ErrNotFound
		}

		if resp.StatusCode >= 500 {
			_ = resp.Body.Close()
			lastErr = &NetworkError{URL: url, StatusCode: resp.StatusCode}
			continue
		}

		if resp.StatusCode >= 400 {
			_ = resp.Body.Close()
			return nil, &NetworkError{URL: url, StatusCode: resp.StatusCode}
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, &NetworkError{URL: url, Err: fmt.Errorf("max retries exceeded")}
}

func (c *Client) GetJSON(ctx context.Context, path string, v any) error {
	url := c.baseURL + path

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &NetworkError{URL: url, Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return &NetworkError{URL: url, StatusCode: resp.StatusCode}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if err := json.Unmarshal(body, v); err != nil {
		return &ParseError{Context: "json decode", Err: err}
	}

	return nil
}

func (c *Client) GetHTML(ctx context.Context, path string) (*goquery.Document, error) {
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, &ParseError{Context: "html parse", Err: err}
	}

	return doc, nil
}
