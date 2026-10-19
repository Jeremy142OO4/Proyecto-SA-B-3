package models

import (
	"errors"
	"github.com/google/uuid"
	"regexp"
	"strings"
	"time"
)

var traceIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)
var servicePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,100}$`)

func ValidTraceID(id string) bool {
	return traceIDPattern.MatchString(id) && id != "00000000000000000000000000000000"
}

type AuditFilter struct {
	CorrelationID *uuid.UUID    `json:"correlationId,omitempty"`
	TraceID       string        `json:"traceId,omitempty"`
	Service       string        `json:"servicio,omitempty"`
	Severity      EventSeverity `json:"nivel,omitempty"`
	Since         *time.Time    `json:"desde,omitempty"`
	Until         *time.Time    `json:"hasta,omitempty"`
	Limit         int           `json:"limite"`
	Offset        int           `json:"desplazamiento"`
}

func (f *AuditFilter) Normalize() error {
	f.TraceID = strings.ToLower(strings.TrimSpace(f.TraceID))
	f.Service = strings.TrimSpace(f.Service)
	f.Severity = EventSeverity(strings.ToUpper(strings.TrimSpace(string(f.Severity))))
	if f.Limit == 0 {
		f.Limit = 50
	}
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 || f.Offset > 100000 {
		return errors.New("limite de 1 a 100 y desplazamiento de 0 a 100000")
	}
	if f.CorrelationID != nil && *f.CorrelationID == uuid.Nil {
		return errors.New("correlationId invalido")
	}
	if f.TraceID != "" && !ValidTraceID(f.TraceID) {
		return errors.New("traceId debe contener 32 caracteres hexadecimales y no ser cero")
	}
	if f.Service != "" && !servicePattern.MatchString(f.Service) {
		return errors.New("servicio invalido")
	}
	if f.Severity != "" && f.Severity != EventInfo && f.Severity != EventWarning && f.Severity != EventError {
		return errors.New("nivel debe ser INFO, WARNING o ERROR")
	}
	if f.Since != nil && f.Until != nil && !f.Since.Before(*f.Until) {
		return errors.New("desde debe ser anterior a hasta")
	}
	return nil
}
