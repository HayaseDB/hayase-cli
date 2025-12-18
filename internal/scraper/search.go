package scraper

import (
	"context"
	"net/url"

	"github.com/hayasedb/hayase/internal/models"
)

func (s *Scraper) Search(ctx context.Context, query string) ([]models.Anime, error) {
	path := "/ajax/seriesSearch?keyword=" + url.QueryEscape(query)

	var results []SearchResult
	if err := s.client.GetJSON(ctx, path, &results); err != nil {
		return nil, err
	}

	animes := make([]models.Anime, len(results))
	for i, r := range results {
		animes[i] = r.ToAnime()
	}

	return animes, nil
}
