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

type UpdateResult struct {
	Version        int64
	RestartSession bool
	Events         []Event
}

type DocumentIdentity struct {
	TemplateID    string
	WorkDir       string
	DocumentKey   string
	MainTypstPath string
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

	return s.nextDocumentVersionLocked(identity)
}

func (s *Service) BeginUpdate(identity DocumentIdentity) UpdateResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	version, restartSession := s.nextDocumentVersionLocked(identity)
	event := s.eventLocked(EventStatus, nil, map[string]interface{}{
		"phase": "convert",
	})
	return UpdateResult{
		Version:        version,
		RestartSession: restartSession,
		Events:         []Event{event},
	}
}

func (s *Service) ConversionFailed(version int64, err error) Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	if version != s.documentVersion {
		return s.staleIgnoredLocked(map[string]interface{}{
			"rejectedDocumentVersion": version,
			"phase":                   "convert",
		})
	}

	return s.eventLocked(EventError, &ErrorInfo{
		Code:        "conversion_failed",
		Message:     "转换失败",
		Detail:      errorDetail(err),
		Recoverable: true,
	}, map[string]interface{}{
		"phase": "convert",
	})
}

func (s *Service) FallbackFailed(version int64, err error) Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	if version != s.documentVersion {
		return s.staleIgnoredLocked(map[string]interface{}{
			"rejectedDocumentVersion": version,
			"phase":                   "fallback",
		})
	}

	s.mode = ModeFallback
	return s.eventLocked(EventError, &ErrorInfo{
		Code:        "fallback_compile_failed",
		Message:     "兼容预览生成失败",
		Detail:      errorDetail(err),
		Recoverable: true,
	}, map[string]interface{}{
		"phase": "fallback",
	})
}

func (s *Service) TinymistUnavailable(reason string) Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.tinymistUnavailableLocked(reason)
}

func (s *Service) TinymistUnavailableForVersion(version int64, reason string) (Event, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if version != s.documentVersion {
		return s.staleIgnoredLocked(map[string]interface{}{
			"rejectedDocumentVersion": version,
			"phase":                   "fallback",
		}), false
	}

	return s.tinymistUnavailableLocked(reason), true
}

func (s *Service) tinymistUnavailableLocked(reason string) Event {
	s.mode = ModeFallback
	return s.eventLocked(EventFallback, &ErrorInfo{
		Code:        "tinymist_unavailable",
		Message:     "兼容预览",
		Detail:      reason,
		Recoverable: true,
	}, map[string]interface{}{
		"phase": "fallback",
	})
}

func (s *Service) CurrentSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.sessionID
}

func (s *Service) IsCurrentVersion(version int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return version == s.documentVersion
}

func (s *Service) StartSession(identity DocumentIdentity) Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.startSessionLocked(identity)
}

func (s *Service) StartSessionForVersion(version int64, identity DocumentIdentity) (Event, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if version != s.documentVersion {
		return s.staleIgnoredLocked(map[string]interface{}{
			"rejectedDocumentVersion": version,
			"lifecycle":               "start",
		}), false
	}

	return s.startSessionLocked(identity), true
}

func (s *Service) startSessionLocked(identity DocumentIdentity) Event {
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

	return s.applySessionEventLocked(sessionID, kind, dataPlaneURL, errInfo)
}

func (s *Service) ApplySessionEventForVersion(version int64, sessionID string, kind EventKind, dataPlaneURL string, errInfo *ErrorInfo) (Event, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if version != s.documentVersion {
		return s.staleIgnoredLocked(map[string]interface{}{
			"rejectedDocumentVersion": version,
			"rejectedSessionId":       sessionID,
		}), false
	}

	return s.applySessionEventLocked(sessionID, kind, dataPlaneURL, errInfo)
}

func (s *Service) applySessionEventLocked(sessionID string, kind EventKind, dataPlaneURL string, errInfo *ErrorInfo) (Event, bool) {
	if sessionID != s.sessionID {
		return s.staleIgnoredLocked(map[string]interface{}{
			"rejectedSessionId": sessionID,
		}), false
	}

	switch kind {
	case EventReady:
		s.mode = ModeEmbedded
	case EventRecover:
		// Tinymist recovery is availability only; the next edit/ready cycle owns switching back to embedded.
		if s.mode != ModeFallback {
			s.mode = ModeFallback
		}
	case EventFallback, EventRetry, EventError:
		s.mode = ModeFallback
	case EventTeardown:
		s.mode = ModeFallback
		s.identity = DocumentIdentity{}
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

func (s *Service) nextDocumentVersionLocked(identity DocumentIdentity) (version int64, restartSession bool) {
	restartSession = s.identity != identity
	s.documentVersion++
	s.identity = identity
	return s.documentVersion, restartSession
}

func errorDetail(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func clonePages(pages []Page) []Page {
	if pages == nil {
		return nil
	}
	out := make([]Page, len(pages))
	copy(out, pages)
	return out
}
