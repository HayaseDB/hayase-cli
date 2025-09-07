package voe

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/log"

	"github.com/hayasedb/hayase-cli/internal/models"
)

type Extractor struct {
	name       string
	priority   int
	httpClient *http.Client

	redirectPattern *regexp.Regexp
	b64Pattern      *regexp.Regexp
	hlsPattern      *regexp.Regexp
	junkParts       []string
}

func New() models.Extractor {
	return &Extractor{
		name:     "VOE",
		priority: 5,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
			},
		},

		redirectPattern: regexp.MustCompile(`https?://[^'"<>]+`),
		b64Pattern:      regexp.MustCompile(`var a168c='([^']+)'`),
		hlsPattern:      regexp.MustCompile(`'hls': '([^']+)'`),

		junkParts: []string{"@$", "^^", "~@", "%?", "*~", "!!", "#&"},
	}
}

func (e *Extractor) Name() string {
	return e.name
}

func (e *Extractor) Priority() int {
	return e.priority
}

func (e *Extractor) CanHandle(embedURL string) bool {
	return strings.Contains(embedURL, "voe.sx") ||
		strings.Contains(embedURL, "voe.to") ||
		strings.Contains(embedURL, "voe.com")
}

func (e *Extractor) Extract(ctx context.Context, embeddedURL string) (*models.StreamURL, error) {
	log.Debug("Extracting VOE stream", "url", embeddedURL)

	req, err := http.NewRequestWithContext(ctx, "GET", embeddedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial page: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Debug("Initial page returned error status", "status", resp.StatusCode)
		return nil, fmt.Errorf("initial page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read initial page: %w", err)
	}

	responseText := string(body)
	log.Debug("Got initial VOE response", "response_length", len(responseText))

	redirectMatch := e.redirectPattern.FindString(responseText)
	if redirectMatch == "" {
		log.Debug("No redirect URL found in response", "response_preview", responseText[:min(200, len(responseText))])
		return nil, fmt.Errorf("no redirect URL found")
	}

	redirectURL := redirectMatch
	log.Debug("Found redirect URL", "redirect_url", redirectURL)

	finalReq, err := http.NewRequestWithContext(ctx, "GET", redirectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create redirect request: %w", err)
	}

	finalReq.Header.Set("User-Agent", "Mozilla/5.0")

	finalResp, err := e.httpClient.Do(finalReq)
	if err != nil {
		log.Debug("Failed to follow redirect", "redirect_url", redirectURL, "error", err)
		return nil, fmt.Errorf("failed to follow redirect: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(finalResp.Body)

	finalBody, err := io.ReadAll(finalResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read redirect response: %w", err)
	}

	html := string(finalBody)
	log.Debug("Got final HTML from redirect", "final_html_length", len(html))

	if streamURL := e.extractFromScript(html); streamURL != "" {
		log.Debug("Successfully extracted using method 1", "method", "script_tag")
		return e.createStreamURL(streamURL), nil
	}

	if streamURL := e.extractFromB64Variable(html); streamURL != "" {
		log.Debug("Successfully extracted using method 2", "method", "b64_variable")
		return e.createStreamURL(streamURL), nil
	}

	if streamURL := e.extractHLSSource(html); streamURL != "" {
		log.Debug("Successfully extracted using method 3", "method", "hls_pattern")
		return e.createStreamURL(streamURL), nil
	}

	log.Debug("All extraction methods failed")
	return nil, fmt.Errorf("failed to extract stream URL using all methods")
}

func (e *Extractor) extractFromScript(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Debug("Failed to parse HTML for script extraction", "error", err)
		return ""
	}

	var result string
	doc.Find("script[type='application/json']").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if len(text) > 4 {
			scriptContent := text[2 : len(text)-2]

			if decoded, err := e.decodeVOEString(scriptContent); err == nil {
				if source, ok := decoded["source"].(string); ok && source != "" {
					result = source
					return
				}
			}
		}
	})

	return result
}

func (e *Extractor) extractFromB64Variable(html string) string {
	matches := e.b64Pattern.FindStringSubmatch(html)
	if len(matches) < 2 {
		return ""
	}

	encoded := matches[1]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		log.Debug("Failed to decode base64", "error", err)
		return ""
	}

	reversed := reverseString(string(decoded))

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(reversed), &data); err != nil {
		log.Debug("Failed to parse JSON from b64 method", "error", err)
		return ""
	}

	if source, ok := data["source"].(string); ok && source != "" {
		return source
	}

	return ""
}

func (e *Extractor) extractHLSSource(html string) string {
	matches := e.hlsPattern.FindStringSubmatch(html)
	if len(matches) < 2 {
		return ""
	}

	encoded := matches[1]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		log.Debug("Failed to decode HLS base64", "error", err)
		return ""
	}

	result := string(decoded)
	if result == "" {
		return ""
	}

	return result
}

func (e *Extractor) decodeVOEString(encoded string) (map[string]interface{}, error) {
	step1 := e.shiftLetters(encoded)

	step2 := e.replaceJunk(step1)
	step2 = strings.ReplaceAll(step2, "_", "")

	step3Bytes, err := base64.StdEncoding.DecodeString(step2)
	if err != nil {
		return nil, fmt.Errorf("step 3 decode failed: %w", err)
	}
	step3 := string(step3Bytes)

	step4 := e.shiftBack(step3, 3)

	step4Reversed := reverseString(step4)
	step5Bytes, err := base64.StdEncoding.DecodeString(step4Reversed)
	if err != nil {
		return nil, fmt.Errorf("step 5 decode failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(step5Bytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

func (e *Extractor) shiftLetters(input string) string {
	result := make([]byte, len(input))
	for i, c := range []byte(input) {
		if c >= 'A' && c <= 'Z' {
			result[i] = ((c - 'A' + 13) % 26) + 'A'
		} else if c >= 'a' && c <= 'z' {
			result[i] = ((c - 'a' + 13) % 26) + 'a'
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func (e *Extractor) replaceJunk(input string) string {
	result := input
	for _, part := range e.junkParts {
		result = strings.ReplaceAll(result, part, "_")
	}
	return result
}

func (e *Extractor) shiftBack(input string, n int) string {
	result := make([]byte, len(input))
	for i, c := range []byte(input) {
		result[i] = c - byte(n)
	}
	return string(result)
}

func (e *Extractor) createStreamURL(url string) *models.StreamURL {
	quality := models.Quality1080p

	if strings.Contains(url, "720") {
		quality = models.Quality720p
	} else if strings.Contains(url, "1080") {
		quality = models.Quality1080p
	} else if strings.Contains(url, "1440") {
		quality = models.Quality1440p
	} else if strings.Contains(url, "2160") {
		quality = models.Quality2160p
	}

	expiresAt := time.Now().Add(2 * time.Hour)

	return &models.StreamURL{
		URL:       url,
		Quality:   quality,
		Provider:  e.Name(),
		ExpiresAt: expiresAt,
	}
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
