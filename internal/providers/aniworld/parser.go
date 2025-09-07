package aniworld

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/log"

	"github.com/hayasedb/hayase-cli/internal/models"
)

func (c *Client) ParseEpisodeLinks(doc *goquery.Document) []*models.Episode {
	var episodes []*models.Episode
	episodeSet := make(map[string]bool)

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		if strings.Contains(href, "/episode-") {
			if season, episode, ok := parseEpisodeURL(href); ok {
				key := fmt.Sprintf("s%de%d", season, episode)
				if !episodeSet[key] {
					episodeSet[key] = true

					fullURL := href
					if !strings.HasPrefix(href, "http") {
						fullURL = BaseURL + href
					}

					episodes = append(episodes, &models.Episode{
						Season:    season,
						Episode:   episode,
						Link:      fullURL,
						Providers: make(map[string]map[models.Language]string),
						UpdatedAt: time.Now(),
					})
				}
			}
		}

		if strings.Contains(href, "/film-") {
			if movieNum, ok := parseMovieURL(href); ok {
				key := fmt.Sprintf("s0e%d", movieNum)
				if !episodeSet[key] {
					episodeSet[key] = true

					fullURL := href
					if !strings.HasPrefix(href, "http") {
						fullURL = BaseURL + href
					}

					episodes = append(episodes, &models.Episode{
						Season:    0,
						Episode:   movieNum,
						Link:      fullURL,
						Providers: make(map[string]map[models.Language]string),
						UpdatedAt: time.Now(),
					})
				}
			}
		}
	})

	return episodes
}

func (c *Client) GetEpisodesForSeason(ctx context.Context, animeSlug string, season int) ([]*models.Episode, error) {
	var seasonURL string
	if season == 0 {
		seasonURL = fmt.Sprintf("%s/anime/stream/%s/filme", BaseURL, animeSlug)
	} else {
		seasonURL = fmt.Sprintf("%s/anime/stream/%s/staffel-%d", BaseURL, animeSlug, season)
	}

	doc, err := c.getPage(ctx, seasonURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch season page: %w", err)
	}

	var episodes []*models.Episode
	episodeSet := make(map[string]bool)

	if season == 0 {
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists {
				return
			}

			if strings.Contains(href, "/film-") {
				if movieNum, ok := parseMovieURL(href); ok {
					key := fmt.Sprintf("s0e%d", movieNum)
					if !episodeSet[key] {
						episodeSet[key] = true

						fullURL := href
						if !strings.HasPrefix(href, "http") {
							fullURL = BaseURL + href
						}

						title := strings.TrimSpace(s.Text())
						if title == "" {
							title = fmt.Sprintf("Movie %d", movieNum)
						}

						episodes = append(episodes, &models.Episode{
							Season:    0,
							Episode:   movieNum,
							Title:     title,
							Link:      fullURL,
							Providers: make(map[string]map[models.Language]string),
							UpdatedAt: time.Now(),
						})
					}
				}
			}
		})
	} else {
		doc.Find("table.seasonEpisodesList tbody tr").Each(func(i int, s *goquery.Selection) {
			episodeLinkCell := s.Find("td.season1EpisodeID a, td[class*='EpisodeID'] a").First()
			href := episodeLinkCell.AttrOr("href", "")

			if href == "" || !strings.Contains(href, "/episode-") {
				return
			}

			if episodeSeason, episodeNum, ok := parseEpisodeURL(href); ok && episodeSeason == season {
				key := fmt.Sprintf("s%de%d", episodeSeason, episodeNum)
				if !episodeSet[key] {
					episodeSet[key] = true

					fullURL := href
					if !strings.HasPrefix(href, "http") {
						fullURL = BaseURL + href
					}

					titleCell := s.Find("td.seasonEpisodeTitle")
					var episodeTitle string

					if titleCell.Length() > 0 {
						titleLink := titleCell.Find("a").First()
						germanTitle := strings.TrimSpace(titleLink.Find("strong").Text())
						englishTitle := strings.TrimSpace(titleLink.Find("span").Text())

						if germanTitle != "" && englishTitle != "" {
							episodeTitle = fmt.Sprintf("%s - %s", germanTitle, englishTitle)
						} else if germanTitle != "" {
							episodeTitle = germanTitle
						} else if englishTitle != "" {
							episodeTitle = englishTitle
						} else {
							episodeTitle = strings.TrimSpace(titleLink.Text())
						}
					}

					if episodeTitle == "" {
						episodeTitle = fmt.Sprintf("Episode %d", episodeNum)
					}

					episodes = append(episodes, &models.Episode{
						Season:    episodeSeason,
						Episode:   episodeNum,
						Title:     episodeTitle,
						Link:      fullURL,
						Providers: make(map[string]map[models.Language]string),
						UpdatedAt: time.Now(),
					})
				}
			}
		})

		if len(episodes) == 0 {
			doc.Find("a").Each(func(i int, s *goquery.Selection) {
				href, exists := s.Attr("href")
				if !exists {
					return
				}

				if strings.Contains(href, "/episode-") {
					if episodeSeason, episodeNum, ok := parseEpisodeURL(href); ok && episodeSeason == season {
						key := fmt.Sprintf("s%de%d", episodeSeason, episodeNum)
						if !episodeSet[key] {
							episodeSet[key] = true

							fullURL := href
							if !strings.HasPrefix(href, "http") {
								fullURL = BaseURL + href
							}

							episodes = append(episodes, &models.Episode{
								Season:    episodeSeason,
								Episode:   episodeNum,
								Title:     fmt.Sprintf("Episode %d", episodeNum),
								Link:      fullURL,
								Providers: make(map[string]map[models.Language]string),
								UpdatedAt: time.Now(),
							})
						}
					}
				}
			})
		}
	}

	return episodes, nil
}

func (c *Client) ParseProviders(doc *goquery.Document) map[string]map[models.Language]string {
	providers := make(map[string]map[models.Language]string)

	doc.Find("li[class*='episodeLink']").Each(func(i int, s *goquery.Selection) {
		providerName := strings.TrimSpace(s.Find("h4").Text())
		if providerName == "" {
			return
		}

		watchLink := s.Find("a.watchEpisode")
		if watchLink.Length() == 0 {
			return
		}

		href, exists := watchLink.Attr("href")
		if !exists {
			return
		}

		langKey, exists := s.Attr("data-lang-key")
		if !exists {
			return
		}

		langCode, err := strconv.Atoi(langKey)
		if err != nil {
			return
		}

		var language models.Language
		switch langCode {
		case 1:
			language = models.GerDub
		case 2:
			language = models.EngSub
		case 3:
			language = models.GerSub
		default:
			return
		}

		fullURL := href
		if !strings.HasPrefix(href, "http") {
			fullURL = BaseURL + href
		}

		if providers[providerName] == nil {
			providers[providerName] = make(map[models.Language]string)
		}

		providers[providerName][language] = fullURL
	})

	return providers
}

func (c *Client) getPage(ctx context.Context, url string) (*goquery.Document, error) {
	resp, err := c.doRequest(ctx, "GET", url)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug("Failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("page returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return doc, nil
}
