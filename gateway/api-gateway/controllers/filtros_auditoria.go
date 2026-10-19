package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var auditTracePattern = regexp.MustCompile(`^[0-9a-f]{32}$`)
var auditServicePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,100}$`)

func auditAlias(c *fiber.Ctx, keys ...string) (string, error) {
	value := ""
	for _, key := range keys {
		if next := c.Query(key); next != "" {
			if value != "" && value != next {
				return "", fiber.NewError(400, "Filtros equivalentes con valores distintos")
			}
			value = next
		}
	}
	return strings.TrimSpace(value), nil
}
func filtrosAuditoria(c *fiber.Ctx) (map[string]any, error) {
	result := map[string]any{}
	for _, item := range []struct {
		field string
		keys  []string
	}{{"correlationId", []string{"correlationId", "idCorrelacion"}}, {"traceId", []string{"traceId"}}, {"servicio", []string{"servicio", "service"}}, {"nivel", []string{"nivel", "level", "severity"}}} {
		v, err := auditAlias(c, item.keys...)
		if err != nil {
			return nil, err
		}
		if item.field == "traceId" && c.Params("traceId") != "" {
			if v != "" && v != c.Params("traceId") {
				return nil, fiber.NewError(400, "traceId contradictorio")
			}
			v = c.Params("traceId")
		}
		if v == "" {
			continue
		}
		switch item.field {
		case "correlationId":
			id, e := uuid.Parse(v)
			if e != nil || id == uuid.Nil {
				return nil, fiber.NewError(400, "correlationId invalido")
			}
			result[item.field] = id
		case "traceId":
			v = strings.ToLower(v)
			if !auditTracePattern.MatchString(v) || v == strings.Repeat("0", 32) {
				return nil, fiber.NewError(400, "traceId invalido")
			}
			result[item.field] = v
		case "servicio":
			if !auditServicePattern.MatchString(v) {
				return nil, fiber.NewError(400, "servicio invalido")
			}
			result[item.field] = v
		case "nivel":
			v = strings.ToUpper(v)
			if v != "INFO" && v != "WARNING" && v != "ERROR" {
				return nil, fiber.NewError(400, "nivel invalido")
			}
			result[item.field] = v
		}
	}
	for _, item := range []struct {
		name               string
		fallback, min, max int
	}{{"limite", 50, 1, 100}, {"desplazamiento", 0, 0, 100000}} {
		value := item.fallback
		if raw := c.Query(item.name); raw != "" {
			parsed, e := strconv.Atoi(raw)
			if e != nil {
				return nil, fiber.NewError(400, item.name+" invalido")
			}
			value = parsed
		}
		if value < item.min || value > item.max {
			return nil, fiber.NewError(400, item.name+" fuera de rango")
		}
		result[item.name] = value
	}
	var since, until *time.Time
	for _, key := range []string{"desde", "hasta"} {
		if raw := c.Query(key); raw != "" {
			value, err := time.Parse(time.RFC3339Nano, raw)
			if err != nil {
				return nil, fiber.NewError(400, key+" debe usar RFC3339")
			}
			result[key] = value
			if key == "desde" {
				since = &value
			} else {
				until = &value
			}
		}
	}
	if since != nil && until != nil && !since.Before(*until) {
		return nil, fiber.NewError(400, "desde debe ser anterior a hasta")
	}
	return result, nil
}
