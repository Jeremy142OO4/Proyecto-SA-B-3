package controllers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const controlJWT = "sandbox-jwt-secret-for-tests-only-12345"

type testControlTransport func(*http.Request) (*http.Response, error)

func (f testControlTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCloudRelayUsesOnlyPrivateFixedTargetAfterAdminAuthorization(t *testing.T) {
	previous := http.DefaultTransport
	defer func() { http.DefaultTransport = previous }()
	calls := 0
	http.DefaultTransport = testControlTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "http://interbank-scenarios:8080/api/administracion/pruebas-interbancarias/" || r.Header.Get("Authorization") != "Bearer "+controlToken("ADMIN") {
			t.Fatal("unexpected cloud relay request", r.URL)
		}
		if r.Header.Get("X-Api-Key") != "" || r.Header.Get("X-Mock-Control-Key") != "" {
			t.Fatal("bank key forwarded")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"enabled":true,"target":"MOCK"}`)), Header: make(http.Header)}, nil
	})
	for _, role := range []string{"", "CLIENTE", "TELLER"} {
		controlCall(t, TestControl{CloudRelay: true}, "GET", role, "")
	}
	if calls != 0 {
		t.Fatal("unauthorized user reached cloud")
	}
	code, body := controlCall(t, TestControl{CloudRelay: true}, "GET", "ADMIN", "")
	if code != 200 || calls != 1 || !strings.Contains(body, `"target":"MOCK"`) {
		t.Fatal(code, body, calls)
	}
}

func controlToken(role string) string {
	enc := func(v any) string { b, _ := json.Marshal(v); return base64.RawURLEncoding.EncodeToString(b) }
	unsigned := enc(map[string]string{"alg": "HS256"}) + "." + enc(map[string]any{"sub": "00000000-0000-4000-8000-000000000001", "role": role, "exp": time.Now().Add(time.Hour).Unix()})
	mac := hmac.New(sha256.New, []byte(controlJWT))
	mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func controlCall(t *testing.T, g TestControl, method, role, body string) (int, string) {
	t.Helper()
	g.JWT = controlJWT
	app := fiber.New()
	g.Register(app)
	r := httptest.NewRequest(method, "/api/administracion/pruebas-interbancarias/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if role != "" {
		r.Header.Set("Authorization", "Bearer "+controlToken(role))
	}
	response, e := app.Test(r, 5000)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	b, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(b)
}
func TestTestControlPermissionsAndDisabled(t *testing.T) {
	for _, tc := range []struct {
		role string
		code int
	}{{"", 401}, {"CLIENTE", 403}, {"TELLER", 403}, {"ADMIN", 200}} {
		code, body := controlCall(t, TestControl{}, "GET", tc.role, "")
		if code != tc.code {
			t.Fatalf("role %s: %d %s", tc.role, code, body)
		}
	}
	code, _ := controlCall(t, TestControl{}, "PUT", "ADMIN", `{"scenario":"reject"}`)
	if code != 503 {
		t.Fatal(code)
	}
}
func TestTestControlFailClosed(t *testing.T) {
	for _, base := range []string{"https://bank14.example", "http://10.20.0.22", "http://127.0.0.1.evil", "http://user@127.0.0.1", "http://127.0.0.1/other", "http://127.0.0.1?redirect=1"} {
		g := TestControl{Enabled: true, Environment: "interbank-sandbox", BaseURL: base, Key: "local-control-secret-key", PartnerURL: base + "/interbank/v1"}
		if g.ready() {
			t.Fatal("unsafe base accepted:", base)
		}
	}
}
func TestTestControlForwardsOnlyScenarioAndActor(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/__mock/control" || r.Header.Get("Authorization") != "" || r.Header.Get("X-Mock-Control-Key") != "local-control-secret-key" || r.Header.Get("X-Mock-Actor") == "" {
			t.Error("invalid private request")
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["scenario"] != "reject" {
			t.Error(body)
		}
		io.WriteString(w, `{"target":"MOCK","scenario":"reject","apiKey":"must-not-leak"}`)
	}))
	defer upstream.Close()
	g := TestControl{Enabled: true, Environment: "interbank-sandbox", BaseURL: upstream.URL, Key: "local-control-secret-key", PartnerURL: upstream.URL + "/interbank/v1"}
	for _, bad := range []string{`{"scenario":"oops"}`, `{"scenario":"accept","url":"bad"}`, `{"scenario":"accept"} {}`} {
		code, _ := controlCall(t, g, "PUT", "ADMIN", bad)
		if code != 400 {
			t.Fatal(code)
		}
	}
	if calls != 0 {
		t.Fatal("invalid request reached mock")
	}
	code, body := controlCall(t, g, "PUT", "ADMIN", `{"scenario":"reject"}`)
	if code != 200 || !strings.Contains(body, `"scenario":"reject"`) || strings.Contains(body, "must-not-leak") {
		t.Fatal(code, body)
	}
	g.Environment = "dev"
	if g.ready() {
		t.Fatal("dev must not enable sandbox control")
	}
	g.Environment = "interbank-sandbox"
	g.PartnerURL = "https://api.pitagoricos.tech/interbank/v1"
	if g.ready() {
		t.Fatal("real bank configured")
	}
}
