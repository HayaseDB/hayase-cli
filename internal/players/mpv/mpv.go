package mpv

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gen2brain/go-mpv"

	"github.com/hayasedb/hayase-cli/internal/models"
	"github.com/hayasedb/hayase-cli/internal/players"
)

type Player struct {
	name string
	mpv  *mpv.Mpv
}

func New() players.Player {
	return &Player{
		name: "MPV",
	}
}

func (p *Player) Name() string {
	return p.name
}

func (p *Player) IsAvailable() bool {
	return true
}

func (p *Player) Play(ctx context.Context, streamURL *models.StreamURL, title string) error {
	if streamURL.IsExpired() {
		return fmt.Errorf("stream URL has expired at %v (current time: %v)",
			streamURL.ExpiresAt, time.Now())
	}

	if streamURL.URL == "" {
		return fmt.Errorf("stream URL is empty")
	}

	if !strings.HasPrefix(streamURL.URL, "http") {
		return fmt.Errorf("invalid stream URL format: %s", streamURL.URL)
	}

	m := mpv.New()
	if m == nil {
		return fmt.Errorf("failed to create mpv instance")
	}
	defer m.TerminateDestroy()

	p.mpv = m

	if err := p.configureMPV(m, title); err != nil {
		return fmt.Errorf("failed to configure MPV: %w", err)
	}

	if err := m.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize MPV: %w", err)
	}

	log.Info("Starting playback", "title", title, "provider", streamURL.Provider, "quality", streamURL.Quality.String())

	if err := m.Command([]string{"loadfile", streamURL.URL}); err != nil {
		return fmt.Errorf("failed to load file: %w", err)
	}

	return p.eventLoop(ctx, m)
}

func (p *Player) configureMPV(m *mpv.Mpv, title string) error {
	if err := m.SetOptionString("user-agent", "Mozilla/5.0 (X11; Linux x86_64; rv:98.0) Gecko/20100101 Firefox/98.0"); err != nil {
		log.Debug("Failed to set user-agent", "error", err)
	}

	if err := m.SetOptionString("referrer", "https://aniworld.to"); err != nil {
		log.Debug("Failed to set referrer", "error", err)
	}

	if err := m.SetPropertyString("input-default-bindings", "yes"); err != nil {
		log.Debug("Failed to set input-default-bindings", "error", err)
	}

	if err := m.SetOptionString("input-vo-keyboard", "yes"); err != nil {
		log.Debug("Failed to set input-vo-keyboard", "error", err)
	}

	if err := m.SetOption("osc", mpv.FormatFlag, true); err != nil {
		log.Debug("Failed to enable OSC", "error", err)
	}

	if title != "" {
		if err := m.SetOptionString("force-media-title", title); err != nil {
			log.Debug("Failed to set media title", "error", err)
		}
	}

	if err := m.SetOption("fs", mpv.FormatFlag, true); err != nil {
		log.Debug("Failed to enable fullscreen", "error", err)
	}

	if err := m.SetOptionString("hwdec", "auto"); err != nil {
		log.Debug("Failed to set hardware decoding", "error", err)
	}

	if err := m.SetOptionString("vo", "gpu"); err != nil {
		log.Debug("Failed to set video output", "error", err)
	}

	switch runtime.GOOS {
	case "linux":
		if err := m.SetOptionString("ao", "pulse"); err != nil {
			log.Debug("Failed to set audio output to pulse", "error", err)
		}
	case "darwin":
		if err := m.SetOptionString("ao", "coreaudio"); err != nil {
			log.Debug("Failed to set audio output to coreaudio", "error", err)
		}
	}

	if err := m.SetOption("cache", mpv.FormatFlag, true); err != nil {
		log.Debug("Failed to enable cache", "error", err)
	}

	if err := m.SetOption("network-timeout", mpv.FormatInt64, int64(30)); err != nil {
		log.Debug("Failed to set network timeout", "error", err)
	}

	if err := m.SetOption("really-quiet", mpv.FormatFlag, true); err != nil {
		log.Debug("Failed to enable quiet mode", "error", err)
	}

	if err := m.SetOption("no-terminal", mpv.FormatFlag, true); err != nil {
		log.Debug("Failed to disable terminal", "error", err)
	}

	if err := m.RequestLogMessages("info"); err != nil {
		log.Debug("Failed to request log messages", "error", err)
	}

	if err := m.ObserveProperty(0, "pause", mpv.FormatFlag); err != nil {
		log.Debug("Failed to observe pause property", "error", err)
	}

	return nil
}

func (p *Player) eventLoop(ctx context.Context, m *mpv.Mpv) error {
	done := make(chan struct{})
	var playbackError error

	go func() {
		<-ctx.Done()
		log.Debug("Context cancelled, stopping MPV")
		if err := m.Command([]string{"quit"}); err != nil {
			log.Debug("Failed to quit MPV gracefully", "error", err)
		}
		close(done)
	}()

	for {
		select {
		case <-done:
			log.Debug("Event loop cancelled by context")
			return nil
		default:
			event := m.WaitEvent(1000)

			switch event.EventID {
			case mpv.EventPropertyChange:
				prop := event.Property()
				if prop.Name == "pause" {
					if paused, ok := prop.Data.(int); ok {
						log.Debug("Pause state changed", "paused", paused == 1)
					}
				}

			case mpv.EventFileLoaded:
				log.Debug("File loaded successfully")
				if p, err := m.GetProperty("media-title", mpv.FormatString); err == nil {
					if mediaTitle, ok := p.(string); ok {
						log.Debug("Media title", "title", mediaTitle)
					}
				}

			case mpv.EventStart:
				sf := event.StartFile()
				log.Debug("Playback started", "entry_id", sf.EntryID)

			case mpv.EventEnd:
				ef := event.EndFile()
				log.Debug("Playback ended", "entry_id", ef.EntryID, "reason", ef.Reason)

				if ef.Reason == mpv.EndFileEOF {
					log.Debug("Playback finished normally")
					return nil
				} else if ef.Reason == mpv.EndFileError {
					playbackError = fmt.Errorf("playback error: %s", ef.Error)
					log.Error("Playback error", "error", ef.Error)
					return playbackError
				} else if ef.Reason == mpv.EndFileQuit {
					log.Debug("User quit playback")
					return nil
				}

			case mpv.EventShutdown:
				log.Debug("MPV shutdown")
				return playbackError

			case mpv.EventLogMsg:
				msg := event.LogMessage()
				log.Debug("MPV log", "level", msg.Level, "text", strings.TrimSpace(msg.Text))

			case mpv.EventNone:
				continue

			default:
				log.Debug("Unhandled MPV event", "event_id", event.EventID)
			}

			if event.Error != nil {
				log.Debug("Event error", "error", event.Error)
			}
		}
	}
}
