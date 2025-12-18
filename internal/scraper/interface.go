package scraper

import (
	"context"

	"github.com/hayasedb/hayase/internal/models"
)

type Interface interface {
	GetAnime(ctx context.Context, slug string) (*models.Anime, error)
	GetHomepageData(ctx context.Context) (*HomepageData, error)
	Search(ctx context.Context, query string) ([]models.Anime, error)
}
