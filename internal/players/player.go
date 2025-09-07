package players

import (
	"context"
	"fmt"

	"github.com/hayasedb/hayase-cli/internal/models"
)

type Player interface {
	Name() string

	IsAvailable() bool

	Play(ctx context.Context, streamURL *models.StreamURL, title string) error
}

type Registry struct {
	players  map[string]Player
	default_ string
}

func NewRegistry() *Registry {
	return &Registry{
		players: make(map[string]Player),
	}
}

func (r *Registry) Register(name string, player Player) {
	r.players[name] = player
	if r.default_ == "" && player.IsAvailable() {
		r.default_ = name
	}
}

func (r *Registry) GetDefault() (Player, error) {
	if r.default_ == "" {
		return nil, fmt.Errorf("no default player available")
	}

	player, exists := r.players[r.default_]
	if !exists {
		return nil, fmt.Errorf("player '%s' not found", r.default_)
	}

	if !player.IsAvailable() {
		return nil, fmt.Errorf("player '%s' is not available", r.default_)
	}

	return player, nil
}
