# Stela Action for GitHub Actions

<img src="images/logo.svg" alt="logo" width="96">

[![Lint and Testing](https://github.com/devopsprabin/stela-action/actions/workflows/testing.yml/badge.svg)](https://github.com/devopsprabin/stela-action/actions/workflows/testing.yml)
[![Docker Image](https://github.com/devopsprabin/stela-action/actions/workflows/docker.yml/badge.svg)](https://github.com/devopsprabin/stela-action/actions/workflows/docker.yml)

GitHub Action for sending a build notification to a Stela group. It also works as a Drone and Woodpecker plugin.

**Important:** Only supports Linux Docker containers.

## Features

- [x] Default build summary (commit, author, branch, build number, status)
- [x] Custom title and description with templates
- [x] Status-based colors (green for success, red for failure, yellow otherwise)
- [x] Custom actor name and avatar
- [x] Works on GitHub Actions, Drone and Woodpecker

## Usage

Create a Stela group webhook with provider `GENERIC` and save its URL as the repository secret `WEBHOOK_URL`, or save its ID and token as `WEBHOOK_ID` and `WEBHOOK_TOKEN`. The token is secret, so never commit it.

Send a custom message as shown below:

```yaml
name: stela message
on: [push]
jobs:
  build:
    name: Build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: send custom message
        uses: devopsprabin/stela-action@v1
        with:
          webhook_url: ${{ secrets.WEBHOOK_URL }}
          title: ${{ github.repository }} build
          description: The ${{ github.event_name }} event triggered first step.
```

## Input variables

- `webhook_url`: Webhook URL of the Stela group.
- `webhook_id`: Webhook ID of the Stela group.
- `webhook_token`: Webhook token of the Stela group.
- `base_url`: (Optional) Stela API base URL. Available in the Stela app.
- `title`: (Optional) Message title. Default: first line of the commit message.
- `description`: (Optional) Message description. Default: a build summary.
- `status`: (Optional) Build status, usually `${{ job.status }}`. Default: `success`.
- `color`: (Optional) Hex color code. Default: derived from the status.
- `source_url`: (Optional) Link attached to the message. Default: the workflow run URL. Set to `none` to send no link.
- `actor_name`: (Optional) Override the actor name. Default: the commit author.
- `actor_avatar_url`: (Optional) Override the actor avatar. Default: the commit author's GitHub avatar, or the Stela Action logo.
- `debug`: (Optional) Enable debug mode.

## Example

Send a custom message using `webhook_url`:

```yaml
- name: send message
  uses: devopsprabin/stela-action@v1
  with:
    webhook_url: ${{ secrets.WEBHOOK_URL }}
    description: The ${{ github.event_name }} event triggered first step.
```

Send the default message:

```yaml
- name: send message
  uses: devopsprabin/stela-action@v1
  with:
    webhook_url: ${{ secrets.WEBHOOK_URL }}
```

Report the job result, even when earlier steps fail:

```yaml
- name: send message
  if: always()
  uses: devopsprabin/stela-action@v1
  with:
    webhook_url: ${{ secrets.WEBHOOK_URL }}
    status: ${{ job.status }}
```

Send the message with a custom color and actor name:

```yaml
- name: send message
  uses: devopsprabin/stela-action@v1
  with:
    webhook_id: ${{ secrets.WEBHOOK_ID }}
    webhook_token: ${{ secrets.WEBHOOK_TOKEN }}
    color: "#48f442"
    actor_name: "GitHub Bot"
    description: "A new commit has been pushed with custom color."
```

Use a template in the description:

```yaml
- name: send message
  uses: devopsprabin/stela-action@v1
  with:
    webhook_url: ${{ secrets.WEBHOOK_URL }}
    status: ${{ job.status }}
    description: >
      {{#success build.status}}{{repo.fullName}} #{{build.number}} passed{{else}}{{repo.fullName}} #{{build.number}} failed{{/success}}
```

Templates can use `repo.*`, `commit.*` (`sha`, `branch`, `author`, `message`, `link`) and `build.*` (`number`, `status`, `event`, `link`, `tag`, `started`, `finished`), plus helpers such as `success`, `failure`, `truncate`, `datetime` and `since`. Field names are camelCase, e.g. `{{repo.fullName}}`.

## Drone and Woodpecker

```yaml
- name: notify stela
  image: devopsprabin/stela-action
  settings:
    webhook_url:
      from_secret: webhook_url
  when:
    status: [success, failure]
```

Every input above is also a plugin setting with the same name. On Drone, the status, commit message, author and build link are filled in automatically.

On Woodpecker 3.x, `build.status` is always `success`. Use separate steps with `when.status: [success]` and `when.status: [failure]` to send different messages.

## Development

```sh
make test                                                  # unit tests, no network
cp .env.example .env                                       # add your real URL
make build && PLUGIN_ENV_FILE=.env ./bin/stela-action     # send a message locally
make docker                                                # build linux/amd64 image
```
