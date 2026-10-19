package models

import (
	"errors"
	"github.com/google/uuid"
	"strings"
	"time"
)

type SLOAlert struct {
	AlertID       string        `json:"idAlerta"`
	Service       string        `json:"servicio"`
	Severity      EventSeverity `json:"nivel"`
	Summary       string        `json:"resumen"`
	StartedAt     time.Time     `json:"inicio"`
	CorrelationID *uuid.UUID    `json:"correlationId,omitempty"`
	TraceID       string        `json:"traceId,omitempty"`
}

func (a *SLOAlert) Normalize() error {
	a.AlertID = strings.TrimSpace(a.AlertID)
	a.Service = strings.TrimSpace(a.Service)
	a.Summary = strings.TrimSpace(a.Summary)
	a.TraceID = strings.ToLower(strings.TrimSpace(a.TraceID))
	a.Severity = EventSeverity(strings.ToUpper(strings.TrimSpace(string(a.Severity))))
	if a.Severity == "" {
		a.Severity = EventWarning
	}
	if len(a.AlertID) < 1 || len(a.AlertID) > 200 || !servicePattern.MatchString(a.Service) || len(a.Summary) < 1 || len(a.Summary) > 1000 || a.StartedAt.IsZero() {
		return errors.New("indique idAlerta, servicio, resumen e inicio RFC3339 validos")
	}
	if a.Severity != EventWarning && a.Severity != EventError {
		return errors.New("una alerta SLO debe ser WARNING o ERROR")
	}
	if a.TraceID != "" && !ValidTraceID(a.TraceID) {
		return errors.New("traceId invalido")
	}
	if a.CorrelationID != nil && *a.CorrelationID == uuid.Nil {
		return errors.New("correlationId invalido")
	}
	return nil
}
func (a SLOAlert) EventID() uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("SLOAlertTriggered:"+a.Service+":"+a.AlertID+":"+string(a.Severity)+":"+a.StartedAt.UTC().Format(time.RFC3339Nano)))
}
