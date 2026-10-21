package interbank

import "testing"

func TestSandboxPartnerRequiresExplicitIsolatedLoopback(t *testing.T) {
	t.Setenv("INTERBANK_TEST_MODE", "true")
	t.Setenv("ENTORNO", "interbank-sandbox")
	if !validSandboxPartnerURL("http://127.0.0.1:8099/interbank/v1") {
		t.Fatal("loopback demo denied")
	}
	for _, raw := range []string{"http://real-bank.example/interbank/v1", "http://127.0.0.1.evil/interbank/v1", "http://127.0.0.1:8099/interbank/v1?x=1", "http://u@localhost/interbank/v1"} {
		if validSandboxPartnerURL(raw) {
			t.Fatal(raw)
		}
	}
	t.Setenv("ENTORNO", "dev")
	if validSandboxPartnerURL("http://127.0.0.1/interbank/v1") {
		t.Fatal("dev must remain HTTPS-only")
	}
}
