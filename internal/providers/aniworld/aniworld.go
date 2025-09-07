package aniworld

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"github.com/hayasedb/hayase-cli/internal/models"
	"github.com/hayasedb/hayase-cli/internal/providers"
)

type Provider struct {
	client *Client
}

func New() providers.Provider {
	return &Provider{
		client: NewClient(),
	}
}

func (p *Provider) Name() string {
	return "AniWorld"
}

func (p *Provider) Search(ctx context.Context, query string) ([]*models.SearchResult, error) {
	results, err := p.client.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("API search failed: %w", err)
	}

	searchResults := make([]*models.SearchResult, 0, len(results))
	for _, result := range results {
		var year int
		if result.ProductionYear != "" {
			re := regexp.MustCompile(`\((\d{4})`)
			if matches := re.FindStringSubmatch(result.ProductionYear); len(matches) > 1 {
				if parsed, err := strconv.Atoi(matches[1]); err == nil {
					year = parsed
				}
			}
		}

		log.Debug("Original description from API", "description", result.Description)

		cleanTitle := cleanAndDecodeText(result.Name)
		cleanDesc := cleanAndDecodeText(result.Description)

		log.Debug("Cleaned title", "title", cleanTitle)
		log.Debug("Cleaned description", "description", cleanDesc)

		anime := &models.Anime{
			Title:       cleanTitle,
			Slug:        result.Link,
			Link:        fmt.Sprintf("%s/anime/stream/%s", BaseURL, result.Link),
			Description: cleanDesc,
			Year:        year,
			UpdatedAt:   time.Now(),
		}

		score := calculateMatchScore(query, cleanTitle)

		searchResults = append(searchResults, &models.SearchResult{
			Anime: anime,
			Score: score,
		})
	}

	sort.Slice(searchResults, func(i, j int) bool {
		return searchResults[i].Score > searchResults[j].Score
	})

	return searchResults, nil
}

func (p *Provider) GetEpisodes(ctx context.Context, anime *models.Anime) error {
	log.Debug("GetEpisodes starting", "anime", anime.Title, "slug", anime.Slug)

	log.Debug("Fetching anime page", "url", fmt.Sprintf("%s/anime/stream/%s", "https://aniworld.to", anime.Slug))
	doc, err := p.client.GetAnimePage(ctx, anime.Slug)
	if err != nil {
		log.Warn("Failed to fetch anime page", "slug", anime.Slug, "error", err)
		return fmt.Errorf("failed to fetch anime page: %w", err)
	}
	log.Debug("Anime page fetched successfully", "slug", anime.Slug)

	log.Debug("Parsing available seasons", "slug", anime.Slug)
	availableSeasons := p.client.ParseAvailableSeasons(doc)
	log.Debug("Available seasons parsed", "slug", anime.Slug, "seasons", availableSeasons)

	if len(availableSeasons) == 0 {
		log.Debug("No seasons found, using fallback approach", "slug", anime.Slug)
		episodes := p.client.ParseEpisodeLinks(doc)
		for _, ep := range episodes {
			ep.Anime = anime
		}
		anime.Episodes = convertToEpisodeSlice(episodes)
		log.Debug("Episodes loaded (fallback)", "slug", anime.Slug, "count", len(episodes))
		return nil
	}

	log.Debug("Fetching episodes for multiple seasons", "slug", anime.Slug, "seasons", len(availableSeasons))
	var allEpisodes []*models.Episode
	for _, season := range availableSeasons {
		log.Debug("Fetching episodes for season", "slug", anime.Slug, "season", season)
		seasonEpisodes, err := p.client.GetEpisodesForSeason(ctx, anime.Slug, season)
		if err != nil {
			log.Warn("Failed to get episodes for season", "slug", anime.Slug, "season", season, "error", err)
			continue
		}
		log.Debug("Episodes fetched for season", "slug", anime.Slug, "season", season, "count", len(seasonEpisodes))

		for _, ep := range seasonEpisodes {
			ep.Anime = anime
		}

		allEpisodes = append(allEpisodes, seasonEpisodes...)
	}

	sort.Slice(allEpisodes, func(i, j int) bool {
		if allEpisodes[i].Season != allEpisodes[j].Season {
			return allEpisodes[i].Season < allEpisodes[j].Season
		}
		return allEpisodes[i].Episode < allEpisodes[j].Episode
	})

	anime.Episodes = convertToEpisodeSlice(allEpisodes)

	log.Debug("Episodes loaded from all seasons",
		"slug", anime.Slug,
		"seasons", len(availableSeasons),
		"total_episodes", len(allEpisodes))

	return nil
}

func (p *Provider) GetEpisode(ctx context.Context, anime *models.Anime, season, episode int) (*models.Episode, error) {
	if len(anime.Episodes) == 0 {
		if err := p.GetEpisodes(ctx, anime); err != nil {
			return nil, err
		}
	}

	var targetEpisode *models.Episode
	for i := range anime.Episodes {
		if anime.Episodes[i].Season == season && anime.Episodes[i].Episode == episode {
			targetEpisode = &anime.Episodes[i]
			break
		}
	}

	if targetEpisode == nil {
		return nil, fmt.Errorf("episode S%02dE%02d not found", season, episode)
	}

	doc, err := p.client.GetEpisodePage(ctx, targetEpisode.Link)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch episode page: %w", err)
	}

	streamProviders := p.client.ParseProviders(doc)
	targetEpisode.Providers = streamProviders

	log.Debug("Episode providers loaded",
		"episode", targetEpisode.String(),
		"providers", len(streamProviders))

	return targetEpisode, nil
}

func (p *Provider) GetClient() *Client {
	return p.client
}

func calculateMatchScore(query, title string) float64 {
	queryLower := strings.ToLower(query)
	titleLower := strings.ToLower(title)

	if queryLower == titleLower {
		return 100.0
	}

	if strings.HasPrefix(titleLower, queryLower) {
		return 90.0
	}

	if strings.Contains(titleLower, queryLower) {
		return 70.0
	}

	queryWords := strings.Fields(queryLower)
	titleWords := strings.Fields(titleLower)

	matches := 0
	for _, qw := range queryWords {
		for _, tw := range titleWords {
			if strings.Contains(tw, qw) || strings.Contains(qw, tw) {
				matches++
				break
			}
		}
	}

	if len(queryWords) > 0 {
		return float64(matches) / float64(len(queryWords)) * 50.0
	}

	return 0.0
}

func cleanHTMLTags(text string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(text, "")
	return strings.TrimSpace(cleaned)
}

func cleanAndDecodeText(text string) string {
	decoded := html.UnescapeString(text)

	decoded = strings.ReplaceAll(decoded, `\u2019`, "'")
	decoded = strings.ReplaceAll(decoded, `\u201c`, `"`)
	decoded = strings.ReplaceAll(decoded, `\u201d`, `"`)
	decoded = strings.ReplaceAll(decoded, `\u2026`, "…")

	cleaned := cleanHTMLTags(decoded)

	return cleaned
}

func convertToEpisodeSlice(episodes []*models.Episode) []models.Episode {
	result := make([]models.Episode, len(episodes))
	for i, ep := range episodes {
		result[i] = *ep
	}
	return result
}
