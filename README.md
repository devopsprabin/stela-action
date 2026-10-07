# stela-webhook

Drone/Woodpecker/GitHub Actions plugin that posts build notifications to a Stela group through a **Generic** webhook. Modeled on [appleboy/drone-discord](https://github.com/appleboy/drone-discord).

It POSTs Stela's generic payload to `https://api-stela.ktmbees.dev/webhooks/<id>/<secret>`:

```json
{ "title": "...", "description": "...", "status": "success", "color": "#1ac600",
  "sourceUrl": "<build link>", "actorName": "<commit author>", "actorAvatarUrl": "<avatar>" }
```

## Setup

1. In Stela, create a group webhook with provider `GENERIC` (`POST /groups/{groupId}/webhooks`). Copy the URL, because the secret is shown only once.
2. Store the full URL as a CI secret, e.g. `stela_webhook_url`. Don't commit it.

## Drone usage

```yaml
- name: notify stela
  image: devopsprabin/stela-webhook
  settings:
    webhook_url:
      from_secret: stela_webhook_url
  when:
    status: [success, failure]
```

With custom templates:

```yaml
- name: notify stela
  image: devopsprabin/stela-webhook
  settings:
    webhook_url:
      from_secret: stela_webhook_url
    title: "{{repo.fullName}} #{{build.number}}"
    description: >
      {{#success build.status}}Deployed {{commit.branch}} by {{commit.author}}{{else}}Build failed on {{commit.branch}}{{/success}}
    actor_name: Drone
  when:
    status: [success, failure]
```

## Settings

| Setting | Env fallbacks | Default |
|---|---|---|
| `webhook_url` | `STELA_WEBHOOK_URL`, `WEBHOOK_URL` | required, unless id + secret are set |
| `webhook_id` + `webhook_secret` | `STELA_WEBHOOK_ID`, `STELA_WEBHOOK_SECRET` | alternative to `webhook_url` |
| `base_url` | `STELA_BASE_URL` | `https://api-stela.ktmbees.dev` |
| `title` | | first line of the commit message |
| `description` (alias `message`) | | build summary (author, branch, repo, build #, sha) |
| `status` | | build status |
| `color` | | from status: green success, red failure/error/killed, yellow otherwise |
| `source_url` | | build link |
| `actor_name` (alias `username`) | | commit author |
| `actor_avatar_url` (alias `avatar_url`) | | commit author avatar |
| `debug` | | `false` |

`title` and `description` are Handlebars templates rendered with [drone-template-lib](https://github.com/appleboy/drone-template-lib). They can use `repo.*`, `commit.*` (`sha`, `branch`, `author`, `message`, `link`), `build.*` (`number`, `status`, `event`, `link`, `tag`, `started`, `finished`) and helpers such as `success`, `failure`, `truncate`, `datetime` and `since`. Field names are camelCase, e.g. `{{repo.fullName}}`.

On Woodpecker 3.x, `build.status` is always `success`. Use separate steps with `when.status` for success and failure, as described in drone-discord's docs.

## Development

```sh
make test                         # unit tests (httptest, no network)
cp .env.example .env              # add your real URL
STELA_WEBHOOK_URL=... go test -run TestLiveWebhook ./...   # sends one real message
make build && PLUGIN_ENV_FILE=.env ./bin/stela-webhook       # run locally
make docker                       # builds linux/amd64 image tagged stela-webhook
```
