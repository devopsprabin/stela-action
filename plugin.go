package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/appleboy/drone-template-lib/template"
)

const (
	// DefaultBaseURL is the Stela API used when only webhook-id and webhook-token are set.
	DefaultBaseURL = "https://api-stela.ktmbees.dev"
	// DefaultIconURL is the actor avatar used when no commit author avatar is known.
	DefaultIconURL = "https://raw.githubusercontent.com/devopsprabin/stela-action/main/images/logo.png"
)

type (
	// Repo information.
	Repo struct {
		FullName  string
		Namespace string
		Name      string
	}

	// Commit information.
	Commit struct {
		Sha     string
		Ref     string
		Branch  string
		Link    string
		Author  string
		Avatar  string
		Email   string
		Message string
	}

	// Build information.
	Build struct {
		Tag      string
		Event    string
		Number   int
		Status   string
		Link     string
		Started  int64
		Finished int64
		PR       string
		DeployTo string
	}

	// Config for the plugin. The secret-bearing fields are unexported so
	// they don't leak through debug dumps or templates.
	Config struct {
		webhookURL     string
		WebhookID      string
		webhookToken   string
		BaseURL        string
		Title          string
		Description    string
		Status         string
		Color          string
		SourceURL      string
		ActorName      string
		ActorAvatarURL string
		Debug          bool
	}

	// Payload is Stela's generic webhook body.
	Payload struct {
		Title          string `json:"title"`
		Description    string `json:"description,omitempty"`
		Status         string `json:"status,omitempty"`
		Color          string `json:"color,omitempty"`
		SourceURL      string `json:"sourceUrl,omitempty"`
		ActorName      string `json:"actorName,omitempty"`
		ActorAvatarURL string `json:"actorAvatarUrl,omitempty"`
	}

	// Plugin values.
	Plugin struct {
		Repo       Repo
		Build      Build
		Config     Config
		Commit     Commit
		httpClient *http.Client
	}
)

func (c *Config) validate() error {
	if c.webhookURL != "" {
		u, err := url.Parse(c.webhookURL)
		if err != nil {
			return fmt.Errorf("invalid webhook url: %w", err)
		}
		if u.Scheme != "https" && u.Scheme != "http" {
			return errors.New("invalid webhook url: scheme must be http or https")
		}
		return nil
	}

	var missingFields []string
	if c.WebhookID == "" {
		missingFields = append(missingFields, "WebhookID")
	}
	if c.webhookToken == "" {
		missingFields = append(missingFields, "WebhookToken")
	}
	if len(missingFields) > 0 {
		return fmt.Errorf("missing stela config: %s", strings.Join(missingFields, ", "))
	}
	return nil
}

// GetWebhookURL returns the configured URL, or builds one from base url, id and token.
func (c *Config) GetWebhookURL() string {
	if c.webhookURL != "" {
		return c.webhookURL
	}
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	return fmt.Sprintf("%s/webhooks/%s/%s",
		strings.TrimRight(base, "/"),
		url.PathEscape(c.WebhookID),
		url.PathEscape(c.webhookToken),
	)
}

func templateMessage(t string, plugin Plugin) (string, error) {
	return template.RenderTrim(t, plugin)
}

// Exec executes the plugin.
func (p *Plugin) Exec(ctx context.Context) error {
	if p.httpClient == nil {
		p.httpClient = &http.Client{
			Timeout: 15 * time.Second,
		}
	}

	if err := p.Config.validate(); err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}

	payload, err := p.BuildPayload()
	if err != nil {
		return err
	}

	return p.Send(ctx, payload)
}

// BuildPayload renders the configured templates, falling back to build defaults.
func (p *Plugin) BuildPayload() (Payload, error) {
	title := p.defaultTitle()
	if p.Config.Title != "" {
		txt, err := templateMessage(p.Config.Title, *p)
		if err != nil {
			return Payload{}, fmt.Errorf("failed to render title template: %w", err)
		}
		title = txt
	}

	description := p.defaultDescription()
	if p.Config.Description != "" {
		txt, err := templateMessage(p.Config.Description, *p)
		if err != nil {
			return Payload{}, fmt.Errorf("failed to render description template: %w", err)
		}
		description = txt
	}

	status := firstNonEmpty(p.Config.Status, p.Build.Status)

	return Payload{
		Title:          title,
		Description:    description,
		Status:         status,
		Color:          p.color(status),
		SourceURL:      p.sourceURL(),
		ActorName:      firstNonEmpty(p.Config.ActorName, p.Commit.Author),
		ActorAvatarURL: firstNonEmpty(p.Config.ActorAvatarURL, p.Commit.Avatar, DefaultIconURL),
	}, nil
}

// Send posts the payload to the Stela webhook.
func (p *Plugin) Send(ctx context.Context, payload Payload) error {
	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(payload); err != nil {
		return fmt.Errorf("failed to encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Config.GetWebhookURL(), b)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		// Strip the URL from transport errors so the token never hits CI logs.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var jsonResponse struct {
			Message any    `json:"message"`
			Error   string `json:"error"`
		}
		if err := json.Unmarshal(bodyBytes, &jsonResponse); err != nil || jsonResponse.Message == nil {
			return fmt.Errorf("failed to send message, status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
		}
		return fmt.Errorf("failed to send message, status code: %d, error: %v", resp.StatusCode, jsonResponse.Message)
	}

	return nil
}

// sourceURL returns the message link; "none" sends the message without one.
func (p *Plugin) sourceURL() string {
	if strings.EqualFold(strings.TrimSpace(p.Config.SourceURL), "none") {
		return ""
	}
	return firstNonEmpty(p.Config.SourceURL, p.Build.Link, p.Commit.Link)
}

func (p *Plugin) defaultTitle() string {
	if title, _, _ := strings.Cut(strings.TrimSpace(p.Commit.Message), "\n"); title != "" {
		return title
	}
	return fmt.Sprintf("%s build #%d %s", p.repoName(), p.Build.Number, p.Build.Status)
}

func (p *Plugin) defaultDescription() string {
	var action string
	switch p.Build.Event {
	case "pull_request":
		action = fmt.Sprintf("%s updated pull request #%s", p.Commit.Author, p.Build.PR)
	case "tag":
		action = fmt.Sprintf("%s pushed tag %s", p.Commit.Author, firstNonEmpty(p.Build.Tag, p.Commit.Branch))
	case "promote", "rollback", "deployment":
		action = fmt.Sprintf("%s %s to %s", p.Commit.Author, p.Build.Event, p.Build.DeployTo)
	default:
		action = fmt.Sprintf("%s pushed to %s", p.Commit.Author, p.Commit.Branch)
	}

	parts := []string{action}
	if repo := p.repoName(); repo != "" {
		parts = append(parts, fmt.Sprintf("%s build #%d: %s", repo, p.Build.Number, p.Build.Status))
	}
	if p.Commit.Sha != "" {
		parts = append(parts, "commit "+truncate(p.Commit.Sha, 8))
	}
	return strings.Join(parts, " · ")
}

func (p *Plugin) repoName() string {
	if p.Repo.FullName != "" {
		return p.Repo.FullName
	}
	if p.Repo.Namespace != "" && p.Repo.Name != "" {
		return p.Repo.Namespace + "/" + p.Repo.Name
	}
	return p.Repo.Name
}

// color returns the configured color normalized to #rrggbb, or one derived from the status.
func (p *Plugin) color(status string) string {
	if c := strings.TrimPrefix(p.Config.Color, "#"); c != "" {
		if _, err := strconv.ParseUint(c, 16, 32); err == nil && len(c) == 6 {
			return "#" + strings.ToLower(c)
		}
	}

	switch status {
	case "success":
		return "#1ac600"
	case "failure", "error", "killed":
		return "#ff3232"
	default:
		return "#ffd930"
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
