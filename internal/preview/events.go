package preview

import "time"

type Mode string

const (
	ModeStarting Mode = "starting"
	ModeEmbedded Mode = "embedded"
	ModeFallback Mode = "fallback"
)

type EventKind string

const (
	EventStatus       EventKind = "status"
	EventReady        EventKind = "ready"
	EventFallback     EventKind = "fallback"
	EventRetry        EventKind = "retry"
	EventRecover      EventKind = "recover"
	EventTeardown     EventKind = "teardown"
	EventPage         EventKind = "page"
	EventError        EventKind = "error"
	EventDiagnostic   EventKind = "diagnostic"
	EventStaleIgnored EventKind = "stale_ignored"
)

type Event struct {
	At              time.Time              `json:"at"`
	Kind            EventKind              `json:"kind"`
	Seq             int64                  `json:"seq"`
	SessionID       string                 `json:"sessionId,omitempty"`
	DocumentVersion int64                  `json:"documentVersion,omitempty"`
	Mode            Mode                   `json:"mode,omitempty"`
	DataPlaneURL    string                 `json:"dataPlaneUrl,omitempty"`
	Page            int                    `json:"page,omitempty"`
	SVG             string                 `json:"svg,omitempty"`
	PageHash        string                 `json:"pageHash,omitempty"`
	Error           *ErrorInfo             `json:"error,omitempty"`
	Diagnostics     []Diagnostic           `json:"diagnostics,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type ErrorInfo struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Detail      string `json:"detail,omitempty"`
	Recoverable bool   `json:"recoverable"`
}

type Diagnostic struct {
	Severity          string `json:"severity"`
	Message           string `json:"message"`
	Source            string `json:"source,omitempty"`
	Line              int    `json:"line,omitempty"`
	Column            int    `json:"column,omitempty"`
	MappingConfidence string `json:"mappingConfidence,omitempty"`
}
