package chapa

import (
	"encoding/json"
	"testing"
)

func TestParseAmountCents(t *testing.T) {
	cases := map[string]int64{
		"2650":    265000,
		"2650.5":  265050,
		"2650.50": 265050,
		"0.07":    7,
		"10.100":  1010,
	}
	for in, want := range cases {
		got, err := ParseAmountCents(in)
		if err != nil {
			t.Fatalf("ParseAmountCents(%q) error: %v", in, err)
		}
		if got != want {
			t.Errorf("ParseAmountCents(%q) = %d, want %d", in, got, want)
		}
	}

	for _, bad := range []string{"", "abc", "1.234", "-5"} {
		if _, err := ParseAmountCents(bad); err == nil {
			t.Errorf("ParseAmountCents(%q) expected error", bad)
		}
	}
}

func TestFormatAmount(t *testing.T) {
	if got := FormatAmount(265007); got != "2650.07" {
		t.Errorf("FormatAmount = %q", got)
	}
	if got := FormatAmount(5); got != "0.05" {
		t.Errorf("FormatAmount = %q", got)
	}
}

func TestVerifySignature(t *testing.T) {
	secret := "whsec"
	payload := []byte(`{"event":"charge.success","tx_ref":"GEBEYA-1"}`)

	if !VerifySignature(secret, payload, Sign(secret, payload)) {
		t.Error("payload signature should verify")
	}
	if !VerifySignature(secret, payload, "", Sign(secret, []byte(secret))) {
		t.Error("secret signature should verify")
	}
	if VerifySignature(secret, payload, Sign("other", payload)) {
		t.Error("signature with wrong secret must not verify")
	}
	if VerifySignature("", payload, Sign("", payload)) {
		t.Error("empty secret must never verify")
	}
}

func TestWebhookPayloadShapes(t *testing.T) {
	var flat WebhookPayload
	if err := json.Unmarshal([]byte(`{"event":"charge.success","tx_ref":"A","status":"success","amount":"100.00"}`), &flat); err != nil {
		t.Fatal(err)
	}
	if flat.Reference() != "A" || flat.PaymentStatus() != "success" {
		t.Errorf("flat payload parsed as %q/%q", flat.Reference(), flat.PaymentStatus())
	}

	var nested WebhookPayload
	if err := json.Unmarshal([]byte(`{"event":"charge.success","data":{"tx_ref":"B","status":"success","amount":"5"}}`), &nested); err != nil {
		t.Fatal(err)
	}
	if nested.Reference() != "B" || nested.PaymentStatus() != "success" {
		t.Errorf("nested payload parsed as %q/%q", nested.Reference(), nested.PaymentStatus())
	}
}
