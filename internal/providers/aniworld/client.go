package aniworld

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/log"

	"github.com/hayasedb/hayase-cli/internal/extractors"
	"github.com/hayasedb/hayase-cli/internal/models"
)

const (
	BaseURL   = "https://aniworld.to"
	SearchURL = "https://aniworld.to/ajax/seriesSearch"
)

type Client struct {
	httpClient *http.Client
	userAgent  string
	timeout    time.Duration
}

func NewClient() *Client {
	timeout := 10 * time.Second
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		userAgent: "Mozilla/5.0 (X11; Linux x86_64; rv:98.0) Gecko/20100101 Firefox/98.0",
		timeout:   timeout,
	}
}

func (c *Client) doRequest(ctx context.Context, method, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

type SearchResponse struct {
	Name           string `json:"name"`
	Link           string `json:"link"`
	Description    string `json:"description"`
	Cover          string `json:"cover"`
	ProductionYear string `json:"productionYear"`
}

func (c *Client) Search(ctx context.Context, query string) ([]*SearchResponse, error) {
	searchURL := fmt.Sprintf("%s?keyword=%s", SearchURL, url.QueryEscape(query))
	log.Debug("Making search request", "url", searchURL)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug("Failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var results []*SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Debug("Search completed", "count", len(results), "query", query)
	return results, nil
}

func (c *Client) GetAnimePage(ctx context.Context, slug string) (*goquery.Document, error) {
	animeURL := fmt.Sprintf("%s/anime/stream/%s", BaseURL, slug)
	log.Debug("Making HTTP request", "url", animeURL)

	resp, err := c.doRequest(ctx, "GET", animeURL)
	if err != nil {
		log.Warn("HTTP request failed", "url", animeURL, "error", err)
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug("Failed to close response body", "error", err)
		}
	}()

	log.Debug("HTTP response received", "url", animeURL, "status", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		log.Warn("Bad HTTP status", "url", animeURL, "status", resp.StatusCode)
		return nil, fmt.Errorf("page returned status %d", resp.StatusCode)
	}

	log.Debug("Parsing HTML document", "url", animeURL)
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Warn("HTML parsing failed", "url", animeURL, "error", err)
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	log.Debug("HTML document parsed successfully", "url", animeURL)
	return doc, nil
}

func (c *Client) GetEpisodePage(ctx context.Context, episodeURL string) (*goquery.Document, error) {
	if !strings.HasPrefix(episodeURL, "http") {
		episodeURL = BaseURL + episodeURL
	}

	resp, err := c.doRequest(ctx, "GET", episodeURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug("Failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("episode page returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse episode HTML: %w", err)
	}

	return doc, nil
}

func (c *Client) FollowRedirect(ctx context.Context, redirectURL string) (string, error) {
	if !strings.HasPrefix(redirectURL, "http") {
		redirectURL = BaseURL + redirectURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", redirectURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	originalCheckRedirect := c.httpClient.CheckRedirect
	c.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	defer func() {
		c.httpClient.CheckRedirect = originalCheckRedirect
	}()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug("Failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		if location != "" {
			return location, nil
		}
	}

	return redirectURL, nil
}

func (c *Client) ParseAvailableSeasons(doc *goquery.Document) []int {
	var seasons []int
	seasonSet := make(map[int]bool)

	seasonContainer := doc.Find("ul").FilterFunction(func(i int, s *goquery.Selection) bool {
		return s.Find("strong").FilterFunction(func(j int, strong *goquery.Selection) bool {
			return strings.Contains(strings.ToLower(strong.Text()), "staffeln")
		}).Length() > 0
	})

	if seasonContainer.Length() > 0 {
		seasonContainer.Find("a[href*='/staffel-']").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists {
				return
			}

			parts := strings.Split(href, "/")
			for _, part := range parts {
				if strings.HasPrefix(part, "staffel-") {
					if seasonNum, err := strconv.Atoi(strings.TrimPrefix(part, "staffel-")); err == nil && seasonNum > 0 {
						if !seasonSet[seasonNum] {
							seasonSet[seasonNum] = true
							seasons = append(seasons, seasonNum)
						}
					}
				}
			}
		})
	} else {
		doc.Find("a[href*='/staffel-']").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists {
				return
			}

			parts := strings.Split(href, "/")
			for _, part := range parts {
				if strings.HasPrefix(part, "staffel-") {
					if seasonNum, err := strconv.Atoi(strings.TrimPrefix(part, "staffel-")); err == nil && seasonNum > 0 {
						if !seasonSet[seasonNum] {
							seasonSet[seasonNum] = true
							seasons = append(seasons, seasonNum)
						}
					}
				}
			}
		})
	}

	doc.Find("a[href*='/filme/']").Each(func(i int, s *goquery.Selection) {
		if !seasonSet[0] {
			seasonSet[0] = true
			seasons = append(seasons, 0)
		}
	})

	return seasons
}

func parseEpisodeURL(url string) (season, episode int, ok bool) {
	parts := strings.Split(url, "/")
	if len(parts) < 3 {
		return 0, 0, false
	}

	seasonPart := ""
	episodePart := ""

	for _, part := range parts {
		if strings.HasPrefix(part, "staffel-") {
			seasonPart = strings.TrimPrefix(part, "staffel-")
		} else if strings.HasPrefix(part, "episode-") {
			episodePart = strings.TrimPrefix(part, "episode-")
		}
	}

	if seasonPart == "" || episodePart == "" {
		return 0, 0, false
	}

	season, err1 := strconv.Atoi(seasonPart)
	episode, err2 := strconv.Atoi(episodePart)

	if err1 != nil || err2 != nil {
		return 0, 0, false
	}

	return season, episode, true
}

func parseMovieURL(url string) (movieNum int, ok bool) {
	parts := strings.Split(url, "/")

	for _, part := range parts {
		if strings.HasPrefix(part, "film-") {
			numStr := strings.TrimPrefix(part, "film-")
			if num, err := strconv.Atoi(numStr); err == nil {
				return num, true
			}
		}
	}

	return 0, false
}

func (c *Client) ExtractStreamURL(ctx context.Context, redirectURL string) (*models.StreamURL, error) {
	extractorSystem := extractors.NewSystem(nil)

	embedURL, err := c.FollowRedirect(ctx, redirectURL)
	if err != nil {
		return nil, fmt.Errorf("failed to follow redirect: %w", err)
	}

	log.Debug("Following redirect", "redirect_url", redirectURL, "embed_url", embedURL)

	streamURL, err := extractorSystem.Extract(ctx, embedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract stream URL: %w", err)
	}

	log.Info("Successfully extracted stream URL", "provider", streamURL.Provider, "quality", streamURL.Quality.String())

	return streamURL, nil
}
