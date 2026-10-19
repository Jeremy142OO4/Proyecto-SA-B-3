package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/Proyecto-SA-B-3/api-gateway/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOperationalErrorOnlyLogsWithoutBroker(t *testing.T) {
	old := slog.Default()
	defer slog.SetDefault(old)
	var logs bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	app := fiber.New(fiber.Config{ErrorHandler: responderErrorHTTP})
	app.Use(middleware.Correlacion)
	app.Get("/test", func(c *fiber.Ctx) error { return errors.New("postgres://secret-password@db/private") })
	response, e := app.Test(httptest.NewRequest("GET", "/test?token=private", nil))
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	if response.StatusCode != 500 {
		t.Fatal(response.StatusCode)
	}
	var body struct {
		CorrelationID string `json:"correlationId"`
	}
	json.NewDecoder(response.Body).Decode(&body)
	if _, e = uuid.Parse(body.CorrelationID); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(logs.String(), "secret-password") || strings.Contains(logs.String(), "token=private") {
		t.Fatal("sensitive error data in infrastructure log")
	}
	if !strings.Contains(logs.String(), `"errorCode":"ERROR_TECNICO_HTTP"`) {
		t.Fatal("structured error missing")
	}
}
