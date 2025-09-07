package extractors

import (
	"context"
	"fmt"
	"sort"

	"github.com/charmbracelet/log"

	"github.com/hayasedb/hayase-cli/internal/extractors/voe"
	"github.com/hayasedb/hayase-cli/internal/models"
	"github.com/hayasedb/hayase-cli/internal/storage"
)

type System struct {
	extractors []models.Extractor
	config     *storage.Config
}

func NewSystem(config *storage.Config) *System {
	system := &System{
		config: config,
	}

	system.registerExtractors()

	sort.Slice(system.extractors, func(i, j int) bool {
		return system.extractors[i].Priority() > system.extractors[j].Priority()
	})

	log.Debug("Initialized extractor system", "count", len(system.extractors))

	return system
}

func (s *System) registerExtractors() {
	s.extractors = append(s.extractors, voe.New())
}

func (s *System) Extract(ctx context.Context, embeddedURL string) (*models.StreamURL, error) {
	log.Debug("Starting extraction", "url", embeddedURL)

	var candidates []models.Extractor
	for _, extractor := range s.extractors {
		if extractor.CanHandle(embeddedURL) {
			candidates = append(candidates, extractor)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no extractor can handle URL: %s", embeddedURL)
	}

	log.Debug("Found candidate extractors", "candidates", len(candidates))

	var lastErr error
	for _, extractor := range candidates {
		log.Debug("Trying extractor", "extractor", extractor.Name())

		streamURL, err := extractor.Extract(ctx, embeddedURL)
		if err != nil {
			log.Debug("Extractor failed", "extractor", extractor.Name(), "error", err)
			lastErr = err
			continue
		}

		if streamURL != nil && !streamURL.IsExpired() {
			log.Info("Successfully extracted stream URL",
				"extractor", extractor.Name(),
				"provider", streamURL.Provider,
				"quality", streamURL.Quality.String())
			return streamURL, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all extractors failed, last error: %w", lastErr)
	}

	return nil, fmt.Errorf("no valid stream URL found")
}

func (s *System) GetExtractors() []models.Extractor {
	return s.extractors
}
