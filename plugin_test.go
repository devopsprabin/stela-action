package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, status int, got *Payload, path *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.Header.Get("Content-Type"), "application/json")
		if path != nil {
			*path = r.URL.Path
		}
		if got != nil {
			require.NoError(t, json.NewDecoder(r.Body).Decode(got))
		}
		w.WriteHeader(status)
		if status >= 400 {
			_, _ = w.Write([]byte(`{"message":"Invalid webhook","error":"Unauthorized","statusCode":401}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestMissingConfig(t *testing.T) {
	plugin := Plugin{}

	err := plugin.Exec(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing stela config: WebhookID, WebhookSecret")
}

func TestInvalidWebhookURL(t *testing.T) {
	plugin := Plugin{Config: Config{webhookURL: "ftp://example.com/x"}}

	err := plugin.Exec(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "scheme must be http or https")
}

func TestGetWebhookURLFromIDAndSecret(t *testing.T) {
	c := Config{WebhookID: "abc", webhookSecret: "s3cr3t", BaseURL: "https://example.com/"}
	assert.Equal(t, "https://example.com/webhooks/abc/s3cr3t", c.GetWebhookURL())

	c.BaseURL = ""
	assert.Equal(t, DefaultBaseURL+"/webhooks/abc/s3cr3t", c.GetWebhookURL())
}

func TestSendDefaultPayload(t *testing.T) {
	var got Payload
	var path string
	srv := newTestServer(t, http.StatusOK, &got, &path)

	plugin := Plugin{
		Repo: Repo{FullName: "ktmbees/stela", Namespace: "ktmbees", Name: "stela"},
		Commit: Commit{
			Sha:     "e7c4f0a63ceeb42a39ac7806f7b51f3f0d204fd2",
			Branch:  "main",
			Author:  "pstha",
			Avatar:  "https://example.com/avatar.png",
			Message: "feat: add webhook\n\nlonger body",
		},
		Build: Build{Number: 42, Status: "failure", Event: "push", Link: "https://ci.example.com/42"},
		Config: Config{
			WebhookID:     "hook-id",
			webhookSecret: "hook-secret",
			BaseURL:       srv.URL,
		},
	}

	require.NoError(t, plugin.Exec(context.Background()))

	assert.Equal(t, "/webhooks/hook-id/hook-secret", path)
	assert.Equal(t, Payload{
		Title:          "feat: add webhook",
		Description:    "pstha pushed to main · ktmbees/stela build #42: failure · commit e7c4f0a6",
		Status:         "failure",
		Color:          "#ff3232",
		SourceURL:      "https://ci.example.com/42",
		ActorName:      "pstha",
		ActorAvatarURL: "https://example.com/avatar.png",
	}, got)
}

func TestSendTemplatedPayload(t *testing.T) {
	var got Payload
	srv := newTestServer(t, http.StatusNoContent, &got, nil)

	plugin := Plugin{
		Repo:   Repo{FullName: "ktmbees/stela"},
		Commit: Commit{Author: "pstha"},
		Build:  Build{Number: 7, Status: "success"},
		Config: Config{
			webhookURL:  srv.URL + "/webhooks/id/secret",
			Title:       "Deploy {{repo.fullName}} #{{build.number}}",
			Description: "{{#success build.status}}all good{{else}}broken{{/success}}",
			Color:       "#ABCDEF",
			ActorName:   "Drone",
		},
	}

	require.NoError(t, plugin.Exec(context.Background()))

	assert.Equal(t, "Deploy ktmbees/stela #7", got.Title)
	assert.Equal(t, "all good", got.Description)
	assert.Equal(t, "success", got.Status)
	assert.Equal(t, "#abcdef", got.Color)
	assert.Equal(t, "Drone", got.ActorName)
}

func TestSendErrorResponse(t *testing.T) {
	srv := newTestServer(t, http.StatusUnauthorized, nil, nil)

	plugin := Plugin{Config: Config{webhookURL: srv.URL + "/webhooks/id/bad"}}

	err := plugin.Exec(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 401")
	assert.Contains(t, err.Error(), "Invalid webhook")
}

func TestTransportErrorHidesSecret(t *testing.T) {
	plugin := Plugin{Config: Config{webhookURL: "http://127.0.0.1:1/webhooks/id/topsecret"}}

	err := plugin.Exec(context.Background())

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "topsecret")
}

// TestLiveWebhook sends a real message when WEBHOOK_URL is set.
func TestLiveWebhook(t *testing.T) {
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		t.Skip("WEBHOOK_URL not set")
	}

	plugin := Plugin{
		Repo:   Repo{FullName: "devopsprabin/stela-webhook"},
		Commit: Commit{Author: "stela-webhook", Branch: "main", Message: "stela-webhook live test"},
		Build:  Build{Number: 1, Status: "success", Event: "push"},
		Config: Config{webhookURL: webhookURL},
	}

	assert.NoError(t, plugin.Exec(context.Background()))
}
