package sms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAfroMessageSender(t *testing.T) {
	var gotTo, gotAuth, gotMessage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTo = r.URL.Query().Get("to")
		gotMessage = r.URL.Query().Get("message")
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"acknowledge":"success","response":{"status":"Send"}}`))
	}))
	defer server.Close()

	sender := NewAfroMessageSender(AfroMessageConfig{BaseURL: server.URL, Token: "tok", Sender: "Gebeya"})
	if err := sender.Send(context.Background(), "0911234567", "ሰላም"); err != nil {
		t.Fatal(err)
	}
	if gotTo != "251911234567" || gotAuth != "Bearer tok" || gotMessage != "ሰላም" {
		t.Errorf("unexpected request: to=%q auth=%q message=%q", gotTo, gotAuth, gotMessage)
	}

	if err := sender.Send(context.Background(), "12345", "x"); err == nil {
		t.Error("expected error for non-Ethiopian number")
	}
}

func TestAfroMessageSenderRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"acknowledge":"error","response":{"errors":["invalid token"]}}`))
	}))
	defer server.Close()

	sender := NewAfroMessageSender(AfroMessageConfig{BaseURL: server.URL, Token: "bad"})
	if err := sender.Send(context.Background(), "0911234567", "x"); err == nil {
		t.Error("expected error when provider rejects message")
	}
}
