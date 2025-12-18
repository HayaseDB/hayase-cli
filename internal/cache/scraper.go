package cache

import (
	"context"
	"time"

	"github.com/charmbracelet/log"

	"github.com/hayasedb/hayase/internal/models"
	"github.com/hayasedb/hayase/internal/scraper"
)

const (
	AnimeTTL    = 24 * time.Hour
	HomepageTTL = 2 * time.Hour
	SearchTTL   = 12 * time.Hour
)

type Scraper struct {
	scraper *scraper.Scraper
	cache   *Cache
}

func NewScraper(s *scraper.Scraper, noCache bool) *Scraper {
	return &Scraper{
		scraper: s,
		cache:   New(noCache),
	}
}

func (s *Scraper) GetAnime(ctx context.Context, slug string) (*models.Anime, error) {
	key := "anime/" + slug

	if anime, ok := Get[models.Anime](s.cache, key); ok {
		return &anime, nil
	}

	log.Debug("cache miss, fetching anime", "slug", slug)
	anime, err := s.scraper.GetAnime(ctx, slug)
	if err != nil {
		return nil, err
	}

	Set(s.cache, key, *anime, AnimeTTL)
	return anime, nil
}

func (s *Scraper) GetHomepageData(ctx context.Context) (*scraper.HomepageData, error) {
	key := "homepage"

	if data, ok := Get[scraper.HomepageData](s.cache, key); ok {
		return &data, nil
	}

	log.Debug("cache miss, fetching homepage")
	data, err := s.scraper.GetHomepageData(ctx)
	if err != nil {
		return nil, err
	}

	Set(s.cache, key, *data, HomepageTTL)
	return data, nil
}

func (s *Scraper) Search(ctx context.Context, query string) ([]models.Anime, error) {
	key := "search/" + HashKey(query)

	if results, ok := Get[[]models.Anime](s.cache, key); ok {
		return results, nil
	}

	log.Debug("cache miss, searching", "query", query)
	results, err := s.scraper.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	Set(s.cache, key, results, SearchTTL)
	return results, nil
}
