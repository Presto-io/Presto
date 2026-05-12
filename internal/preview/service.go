package preview

import (
	"fmt"
	"sync"
	"time"
)

type UpdateRequest struct {
	Markdown    string
	TemplateID  string
	WorkDir     string
	DocumentKey string
}

type DocumentIdentity struct {
	TemplateID    string
	WorkDir       string
	DocumentKey   string
	MainTypstPath string
}

type Page struct {
	Index int
	SVG   string
	Hash  string
}

type Service struct {
	mu              sync.Mutex
	documentVersion int64
	sessionID       string
	identity        DocumentIdentity
	mode            Mode
	lastFallback    []Page
	seq             int64
	sessionSeq      int64
}

func NewService() *Service {
	return &Service{mode: ModeFallback}
}

func (s *Service) NextDocumentVersion(identity DocumentIdentity) (version int64, restartSession bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	restartSession = s.identity != identity
	s.documentVersion++
	s.identity = identity
	return s.documentVersion, restartSession
}

func (s *Service) CurrentSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.sessionID
}

func (s *Service) StartSession(identity DocumentIdentity) Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessionSeq++
	s.sessionID = fmt.Sprintf("session-%d", s.sessionSeq)
	s.identity = identity
	s.mode = ModeStarting

	return s.eventLocked(EventStatus, nil, map[string]interface{}{
		"lifecycle": "start",
	})
}

func (s *Service) ApplySessionEvent(sessionID string, kind EventKind, dataPlaneURL string, errInfo *ErrorInfo) (Event, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID != s.sessionID {
		return s.staleIgnoredLocked(map[string]interface{}{
			"rejectedSessionId": sessionID,
		}), false
	}

	switch kind {
	case EventReady, EventRecover:
		s.mode = ModeEmbedded
	case EventFallback, EventRetry, EventError:
		s.mode = ModeFallback
	case EventTeardown:
		s.mode = ModeFallback
	}

	event := s.eventLocked(kind, errInfo, nil)
	event.DataPlaneURL = dataPlaneURL
	return event, true
}

func (s *Service) ApplyFallback(version int64, pages []Page) (Event, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if version != s.documentVersion {
		return s.staleIgnoredLocked(map[string]interface{}{
			"rejectedDocumentVersion": version,
		}), false
	}

	s.mode = ModeFallback
	s.lastFallback = clonePages(pages)
	event := s.eventLocked(EventFallback, nil, map[string]interface{}{
		"pageCount": len(pages),
	})
	return event, true
}

func (s *Service) LastFallback() []Page {
	s.mu.Lock()
	defer s.mu.Unlock()

	return clonePages(s.lastFallback)
}

func (s *Service) eventLocked(kind EventKind, errInfo *ErrorInfo, metadata map[string]interface{}) Event {
	s.seq++
	return Event{
		At:              time.Now().UTC(),
		Kind:            kind,
		Seq:             s.seq,
		SessionID:       s.sessionID,
		DocumentVersion: s.documentVersion,
		Mode:            s.mode,
		Error:           errInfo,
		Metadata:        metadata,
	}
}

func (s *Service) staleIgnoredLocked(metadata map[string]interface{}) Event {
	return s.eventLocked(EventStaleIgnored, nil, metadata)
}

func clonePages(pages []Page) []Page {
	if pages == nil {
		return nil
	}
	out := make([]Page, len(pages))
	copy(out, pages)
	return out
}
