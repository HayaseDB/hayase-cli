package scraper

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/hayasedb/hayase/internal/models"
)

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func cleanText(text string) string {
	decoded := html.UnescapeString(text)
	decoded = strings.ReplaceAll(decoded, `\u2019`, "'")
	decoded = strings.ReplaceAll(decoded, `\u201c`, `"`)
	decoded = strings.ReplaceAll(decoded, `\u201d`, `"`)
	decoded = strings.ReplaceAll(decoded, `\u2026`, "…")
	cleaned := htmlTagRegex.ReplaceAllString(decoded, "")
	return strings.TrimSpace(cleaned)
}

func parseYear(s string) int {
	re := regexp.MustCompile(`\((\d{4})`)
	if matches := re.FindStringSubmatch(s); len(matches) > 1 {
		if year, err := strconv.Atoi(matches[1]); err == nil {
			return year
		}
	}
	return 0
}

func parseEpisodeURL(url string) (season, episode int, ok bool) {
	parts := strings.Split(url, "/")
	var seasonPart, episodePart string

	for _, part := range parts {
		if strings.HasPrefix(part, "staffel-") {
			seasonPart = strings.TrimPrefix(part, "staffel-")
		} else if strings.HasPrefix(part, "episode-") {
			episodePart = strings.TrimPrefix(part, "episode-")
		}
	}

	if seasonPart == "" || episodePart == "" {
		return 0, 0, false
	}

	season, err1 := strconv.Atoi(seasonPart)
	episode, err2 := strconv.Atoi(episodePart)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}

	return season, episode, true
}

func parseMovieURL(url string) (movieNum int, ok bool) {
	parts := strings.Split(url, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "film-") {
			if num, err := strconv.Atoi(strings.TrimPrefix(part, "film-")); err == nil {
				return num, true
			}
		}
	}
	return 0, false
}

func ParseAvailableSeasons(doc *goquery.Document) []int {
	var seasons []int
	seasonSet := make(map[int]bool)

	seasonContainer := doc.Find("ul").FilterFunction(func(_ int, s *goquery.Selection) bool {
		return s.Find("strong").FilterFunction(func(_ int, strong *goquery.Selection) bool {
			return strings.Contains(strings.ToLower(strong.Text()), "staffeln")
		}).Length() > 0
	})

	selector := seasonContainer
	if selector.Length() == 0 {
		selector = doc.Selection
	}

	selector.Find("a[href*='/staffel-']").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		for _, part := range strings.Split(href, "/") {
			if strings.HasPrefix(part, "staffel-") {
				if seasonNum, err := strconv.Atoi(strings.TrimPrefix(part, "staffel-")); err == nil && seasonNum > 0 {
					if !seasonSet[seasonNum] {
						seasonSet[seasonNum] = true
						seasons = append(seasons, seasonNum)
					}
				}
			}
		}
	})

	if doc.Find("a[href*='/filme']").Length() > 0 {
		if !seasonSet[0] {
			seasons = append(seasons, 0)
		}
	}

	return seasons
}

func ParseSeasonEpisodes(doc *goquery.Document, season int) []models.Episode {
	var episodes []models.Episode
	episodeSet := make(map[string]bool)

	if season == 0 {
		doc.Find("a[href*='/film-']").Each(func(_ int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists {
				return
			}

			if movieNum, ok := parseMovieURL(href); ok {
				key := fmt.Sprintf("s0e%d", movieNum)
				if !episodeSet[key] {
					episodeSet[key] = true
					title := strings.TrimSpace(s.Text())
					if title == "" {
						title = fmt.Sprintf("Movie %d", movieNum)
					}
					episodes = append(episodes, models.Episode{
						Season:   0,
						Number:   movieNum,
						Title:    title,
						Duration: "",
					})
				}
			}
		})
		return episodes
	}

	doc.Find("table.seasonEpisodesList tbody tr").Each(func(_ int, s *goquery.Selection) {
		episodeLinkCell := s.Find("td[class*='EpisodeID'] a").First()
		href := episodeLinkCell.AttrOr("href", "")

		if href == "" || !strings.Contains(href, "/episode-") {
			return
		}

		if episodeSeason, episodeNum, ok := parseEpisodeURL(href); ok && episodeSeason == season {
			key := fmt.Sprintf("s%de%d", episodeSeason, episodeNum)
			if !episodeSet[key] {
				episodeSet[key] = true

				titleCell := s.Find("td.seasonEpisodeTitle")
				var title string

				if titleCell.Length() > 0 {
					titleLink := titleCell.Find("a").First()
					title = strings.TrimSpace(titleLink.Find("strong").Text())
					if title == "" {
						title = strings.TrimSpace(titleLink.Text())
					}
				}

				if title == "" {
					title = fmt.Sprintf("Episode %d", episodeNum)
				}

				episodes = append(episodes, models.Episode{
					Season:   episodeSeason,
					Number:   episodeNum,
					Title:    title,
					Duration: "24m",
				})
			}
		}
	})

	if len(episodes) == 0 {
		doc.Find("a[href*='/episode-']").Each(func(_ int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists {
				return
			}

			if episodeSeason, episodeNum, ok := parseEpisodeURL(href); ok && episodeSeason == season {
				key := fmt.Sprintf("s%de%d", episodeSeason, episodeNum)
				if !episodeSet[key] {
					episodeSet[key] = true
					title := fmt.Sprintf("Episode %d", episodeNum)
					episodes = append(episodes, models.Episode{
						Season:   episodeSeason,
						Number:   episodeNum,
						Title:    title,
						Duration: "24m",
					})
				}
			}
		})
	}

	return episodes
}

func ParseCurrentlyPopular(doc *goquery.Document) []models.Anime {
	var animes []models.Anime
	seen := make(map[string]bool)

	doc.Find(".carousel").Each(func(_ int, carousel *goquery.Selection) {
		if !strings.Contains(carousel.Find("h2").Text(), "Derzeit beliebt") {
			return
		}

		carousel.Find(".coverListItem a[href*='/anime/stream/']").Each(func(_ int, s *goquery.Selection) {
			href := s.AttrOr("href", "")
			if href == "" || strings.Contains(href, "/staffel-") || strings.Contains(href, "/episode-") {
				return
			}

			slug := strings.TrimPrefix(href, "/anime/stream/")
			slug = strings.Split(slug, "/")[0]
			if slug == "" || seen[slug] {
				return
			}
			seen[slug] = true

			h3 := s.Find("h3").Clone()
			h3.Find("span").Remove()
			title := strings.TrimSpace(h3.Text())

			if title != "" {
				animes = append(animes, models.Anime{
					ID:   slug,
					Name: cleanText(title),
				})
			}
		})
	})

	return animes
}

func ParseLatestEpisodes(doc *goquery.Document) []models.Anime {
	var animes []models.Anime
	seen := make(map[string]bool)

	doc.Find(".newEpisodeList a[href*='/anime/stream/']").Each(func(_ int, s *goquery.Selection) {
		href := s.AttrOr("href", "")
		if href == "" || !strings.Contains(href, "/staffel-") {
			return
		}

		pathParts := strings.Split(strings.TrimPrefix(href, "/anime/stream/"), "/")
		if len(pathParts) == 0 || pathParts[0] == "" {
			return
		}
		slug := pathParts[0]

		if seen[slug] {
			return
		}
		seen[slug] = true

		title := strings.TrimSpace(s.Find("strong").Text())

		if title != "" {
			animes = append(animes, models.Anime{
				ID:   slug,
				Name: cleanText(title),
			})
		}
	})

	return animes
}

type SearchResult struct {
	Name           string `json:"name"`
	Link           string `json:"link"`
	Description    string `json:"description"`
	Cover          string `json:"cover"`
	ProductionYear string `json:"productionYear"`
}

func (r SearchResult) ToAnime() models.Anime {
	return models.Anime{
		ID:      r.Link,
		Name:    cleanText(r.Name),
		Desc:    cleanText(r.Description),
		Year:    parseYear(r.ProductionYear),
		Seasons: 0,
		Genres:  nil,
	}
}
