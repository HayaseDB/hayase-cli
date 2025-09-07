package providers

import (
	"context"
	"fmt"

	"github.com/hayasedb/hayase-cli/internal/models"
)

type Provider interface {
	Name() string

	Search(ctx context.Context, query string) ([]*models.SearchResult, error)

	GetEpisodes(ctx context.Context, anime *models.Anime) error

	GetEpisode(ctx context.Context, anime *models.Anime, season, episode int) (*models.Episode, error)
}

type Registry struct {
	providers map[string]Provider
	default_  string
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

func (r *Registry) Register(name string, provider Provider) {
	r.providers[name] = provider
	if r.default_ == "" {
		r.default_ = name
	}
}

func (r *Registry) Get(name string) (Provider, error) {
	if name == "" {
		name = r.default_
	}

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}

	return provider, nil
}

func (r *Registry) GetDefault() (Provider, error) {
	return r.Get(r.default_)
}
