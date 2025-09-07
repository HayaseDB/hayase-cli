package models

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Language int

const (
	GerSub Language = iota
	EngSub
	GerDub
)

func (l Language) String() string {
	switch l {
	case GerSub:
		return "ger-sub"
	case EngSub:
		return "eng-sub"
	case GerDub:
		return "ger-dub"
	default:
		return "unknown"
	}
}

func ParseLanguage(s string) (Language, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ger-sub", "german-sub", "deutsch-sub":
		return GerSub, nil
	case "eng-sub", "english-sub", "en-sub":
		return EngSub, nil
	case "ger-dub", "german-dub", "deutsch-dub":
		return GerDub, nil
	default:
		return GerSub, fmt.Errorf("unknown language: %s (valid: ger-sub, eng-sub, ger-dub)", s)
	}
}

type Quality int

const (
	Quality720p Quality = iota
	Quality1080p
	Quality1440p
	Quality2160p
)

func (q Quality) String() string {
	switch q {
	case Quality720p:
		return "720p"
	case Quality1080p:
		return "1080p"
	case Quality1440p:
		return "1440p"
	case Quality2160p:
		return "2160p"
	default:
		return "1080p"
	}
}

type StreamURL struct {
	URL       string
	Provider  string
	Quality   Quality
	ExpiresAt time.Time
}

func (s *StreamURL) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

type Extractor interface {
	Extract(ctx context.Context, embeddedURL string) (*StreamURL, error)

	CanHandle(url string) bool

	Name() string

	Priority() int
}
