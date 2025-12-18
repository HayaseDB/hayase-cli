package scraper

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/log"

	"github.com/hayasedb/hayase/internal/models"
)

type Scraper struct {
	client *Client
}

func New(opts ...ClientOption) *Scraper {
	return &Scraper{
		client: NewClient(opts...),
	}
}

func (s *Scraper) GetAnime(ctx context.Context, slug string) (*models.Anime, error) {
	path := "/anime/stream/" + slug

	doc, err := s.client.GetHTML(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("fetch anime page: %w", err)
	}

	seasons := ParseAvailableSeasons(doc)
	log.Debug("parsed seasons", "slug", slug, "seasons", seasons)

	if len(seasons) == 0 {
		seasons = []int{1}
	}

	var allEpisodes []models.Episode
	maxSeason := 0

	for _, season := range seasons {
		if season > maxSeason {
			maxSeason = season
		}

		var seasonPath string
		if season == 0 {
			seasonPath = "/anime/stream/" + slug + "/filme"
		} else {
			seasonPath = fmt.Sprintf("/anime/stream/%s/staffel-%d", slug, season)
		}

		seasonDoc, err := s.client.GetHTML(ctx, seasonPath)
		if err != nil {
			log.Warn("failed to fetch season", "slug", slug, "season", season, "error", err)
			continue
		}

		episodes := ParseSeasonEpisodes(seasonDoc, season)
		log.Debug("parsed episodes", "slug", slug, "season", season, "count", len(episodes))
		allEpisodes = append(allEpisodes, episodes...)
	}

	sort.Slice(allEpisodes, func(i, j int) bool {
		if allEpisodes[i].Season != allEpisodes[j].Season {
			return allEpisodes[i].Season < allEpisodes[j].Season
		}
		return allEpisodes[i].Number < allEpisodes[j].Number
	})

	title := doc.Find("h1.seriesTitle, .series-title h1, h1").First().Text()
	if title == "" {
		title = slug
	}

	desc := doc.Find(".seri_des, .series-description, p.description").First().Text()

	var genres []string
	doc.Find(".genre a, .genres a, a[href*='/genre/']").Each(func(_ int, s *goquery.Selection) {
		genre := strings.TrimSpace(s.Text())
		if genre != "" {
			genres = append(genres, genre)
		}
	})

	var year int
	startDateLink := doc.Find("span[itemprop='startDate'] a").First()
	if startDateLink.Length() > 0 {
		yearText := strings.TrimSpace(startDateLink.Text())
		if y, err := strconv.Atoi(yearText); err == nil && y >= 1900 && y <= 2100 {
			year = y
		} else {
			href := startDateLink.AttrOr("href", "")
			for _, part := range strings.Split(href, "/") {
				if len(part) == 4 {
					if y, err := strconv.Atoi(part); err == nil && y >= 1900 && y <= 2100 {
						year = y
						break
					}
				}
			}
		}
	}

	anime := &models.Anime{
		ID:       slug,
		Name:     cleanText(title),
		Desc:     cleanText(desc),
		Year:     year,
		Seasons:  maxSeason,
		Episodes: allEpisodes,
		Genres:   genres,
	}

	return anime, nil
}

type HomepageData struct {
	Popular     []models.Anime
	NewReleases []models.Anime
}

func (s *Scraper) GetHomepageData(ctx context.Context) (*HomepageData, error) {
	doc, err := s.client.GetHTML(ctx, "/")
	if err != nil {
		return nil, fmt.Errorf("fetch homepage: %w", err)
	}

	popular := ParseCurrentlyPopular(doc)
	log.Debug("parsed currently popular", "count", len(popular))

	newReleases := ParseLatestEpisodes(doc)
	log.Debug("parsed latest episodes", "count", len(newReleases))

	return &HomepageData{
		Popular:     popular,
		NewReleases: newReleases,
	}, nil
}
