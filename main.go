package main

import (
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
	"github.com/yassinebenaid/godump"
)

// Version set at compile-time
var Version string

func main() {
	// Load env-file if it exists first
	if filename, found := os.LookupEnv("PLUGIN_ENV_FILE"); found {
		_ = godotenv.Load(filename)
	}

	if _, err := os.Stat("/run/drone/env"); err == nil {
		_ = godotenv.Overload("/run/drone/env")
	}

	app := cli.NewApp()
	app.Name = "Stela Action"
	app.Usage = "Send build notifications to a Stela group using a webhook"
	app.Copyright = "Copyright (c) " + strconv.Itoa(time.Now().Year()) + " ktmbees"
	app.Action = run
	app.Version = Version
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "webhook-url",
			Usage:   "The full Stela webhook URL (https://<host>/webhooks/<id>/<secret>).",
			EnvVars: []string{"PLUGIN_WEBHOOK_URL", "WEBHOOK_URL", "INPUT_WEBHOOK_URL"},
		},
		&cli.StringFlag{
			Name:    "webhook-id",
			Usage:   "The Stela webhook ID (alternative to webhook-url).",
			EnvVars: []string{"PLUGIN_WEBHOOK_ID", "WEBHOOK_ID", "INPUT_WEBHOOK_ID"},
		},
		&cli.StringFlag{
			Name:    "webhook-secret",
			Usage:   "The Stela webhook secret (alternative to webhook-url).",
			EnvVars: []string{"PLUGIN_WEBHOOK_SECRET", "WEBHOOK_SECRET", "INPUT_WEBHOOK_SECRET"},
		},
		&cli.StringFlag{
			Name:    "base-url",
			Value:   DefaultBaseURL,
			Usage:   "The Stela API base URL, used with webhook-id and webhook-secret.",
			EnvVars: []string{"PLUGIN_BASE_URL", "BASE_URL", "INPUT_BASE_URL"},
		},
		&cli.StringFlag{
			Name:    "title",
			Usage:   "Message title (template). Defaults to the commit message.",
			EnvVars: []string{"PLUGIN_TITLE", "INPUT_TITLE"},
		},
		&cli.StringFlag{
			Name:    "description",
			Usage:   "Message description (template). Defaults to a build summary.",
			EnvVars: []string{"PLUGIN_DESCRIPTION", "PLUGIN_MESSAGE", "INPUT_DESCRIPTION"},
		},
		&cli.StringFlag{
			Name:    "status",
			Usage:   "Status sent to Stela. Defaults to the build status.",
			EnvVars: []string{"PLUGIN_STATUS", "INPUT_STATUS"},
		},
		&cli.StringFlag{
			Name:    "color",
			Usage:   "Hex color (e.g. #1ac600). Defaults to a color derived from the status.",
			EnvVars: []string{"PLUGIN_COLOR", "INPUT_COLOR"},
		},
		&cli.StringFlag{
			Name:    "source-url",
			Usage:   "Link attached to the message. Defaults to the build link.",
			EnvVars: []string{"PLUGIN_SOURCE_URL", "INPUT_SOURCE_URL"},
		},
		&cli.StringFlag{
			Name:    "actor-name",
			Usage:   "Override the actor name. Defaults to the commit author.",
			EnvVars: []string{"PLUGIN_ACTOR_NAME", "PLUGIN_USERNAME", "INPUT_ACTOR_NAME"},
		},
		&cli.StringFlag{
			Name:    "actor-avatar-url",
			Usage:   "Override the actor avatar. Defaults to the commit author avatar.",
			EnvVars: []string{"PLUGIN_ACTOR_AVATAR_URL", "PLUGIN_AVATAR_URL", "INPUT_ACTOR_AVATAR_URL"},
		},
		&cli.StringFlag{
			Name:    "repo",
			Usage:   "The repository owner and repository name.",
			EnvVars: []string{"DRONE_REPO", "CI_REPO", "GITHUB_REPOSITORY"},
		},
		&cli.StringFlag{
			Name:    "repo.namespace",
			Usage:   "The repository namespace.",
			EnvVars: []string{"DRONE_REPO_OWNER", "DRONE_REPO_NAMESPACE", "CI_REPO_OWNER", "GITHUB_ACTOR"},
		},
		&cli.StringFlag{
			Name:    "repo.name",
			Usage:   "The repository name.",
			EnvVars: []string{"DRONE_REPO_NAME", "CI_REPO_NAME"},
		},
		&cli.StringFlag{
			Name:    "commit.sha",
			Usage:   "The Git commit SHA.",
			EnvVars: []string{"DRONE_COMMIT_SHA", "CI_COMMIT_SHA", "GITHUB_SHA"},
		},
		&cli.StringFlag{
			Name:    "commit.ref",
			Usage:   "The Git commit reference.",
			EnvVars: []string{"DRONE_COMMIT_REF", "CI_COMMIT_REF", "GITHUB_REF"},
		},
		&cli.StringFlag{
			Name:    "commit.branch",
			Value:   "main",
			Usage:   "The Git commit branch.",
			EnvVars: []string{"DRONE_COMMIT_BRANCH", "CI_COMMIT_BRANCH", "GITHUB_REF_NAME"},
		},
		&cli.StringFlag{
			Name:    "commit.link",
			Usage:   "The link to the Git commit.",
			EnvVars: []string{"DRONE_COMMIT_LINK", "CI_PIPELINE_FORGE_URL"},
		},
		&cli.StringFlag{
			Name:    "commit.author",
			Usage:   "The name of the Git commit author.",
			EnvVars: []string{"DRONE_COMMIT_AUTHOR", "CI_COMMIT_AUTHOR", "GITHUB_ACTOR"},
		},
		&cli.StringFlag{
			Name:    "commit.author.email",
			Usage:   "The email of the Git commit author.",
			EnvVars: []string{"DRONE_COMMIT_AUTHOR_EMAIL", "CI_COMMIT_AUTHOR_EMAIL"},
		},
		&cli.StringFlag{
			Name:    "commit.author.avatar",
			Usage:   "The avatar URL of the Git commit author.",
			EnvVars: []string{"DRONE_COMMIT_AUTHOR_AVATAR", "CI_COMMIT_AUTHOR_AVATAR"},
		},
		&cli.StringFlag{
			Name:    "commit.message",
			Usage:   "The Git commit message.",
			EnvVars: []string{"DRONE_COMMIT_MESSAGE", "CI_COMMIT_MESSAGE"},
		},
		&cli.StringFlag{
			Name:    "build.event",
			Value:   "push",
			Usage:   "The build event type.",
			EnvVars: []string{"DRONE_BUILD_EVENT", "CI_PIPELINE_EVENT", "GITHUB_EVENT_NAME"},
		},
		&cli.IntFlag{
			Name:    "build.number",
			Usage:   "The build number.",
			EnvVars: []string{"DRONE_BUILD_NUMBER", "CI_PIPELINE_NUMBER", "GITHUB_RUN_NUMBER"},
		},
		&cli.StringFlag{
			Name:    "build.status",
			Usage:   "The build status.",
			Value:   "success",
			EnvVars: []string{"DRONE_BUILD_STATUS", "CI_PIPELINE_STATUS"},
		},
		&cli.StringFlag{
			Name:    "build.link",
			Usage:   "The link to the build.",
			EnvVars: []string{"DRONE_BUILD_LINK", "CI_PIPELINE_URL"},
		},
		&cli.StringFlag{
			Name:    "build.tag",
			Usage:   "The build tag.",
			EnvVars: []string{"DRONE_TAG", "CI_COMMIT_TAG"},
		},
		&cli.StringFlag{
			Name:    "pull.request",
			Usage:   "The pull request number.",
			EnvVars: []string{"DRONE_PULL_REQUEST", "CI_COMMIT_PULL_REQUEST"},
		},
		&cli.Int64Flag{
			Name:    "build.started",
			Usage:   "The timestamp when the build started.",
			EnvVars: []string{"DRONE_BUILD_STARTED", "CI_PIPELINE_STARTED"},
		},
		&cli.Int64Flag{
			Name:    "build.finished",
			Usage:   "The timestamp when the build finished.",
			EnvVars: []string{"DRONE_BUILD_FINISHED", "CI_PIPELINE_FINISHED"},
		},
		&cli.StringFlag{
			Name:    "deploy.to",
			Usage:   "The target deployment environment for promotion and rollback pipelines.",
			EnvVars: []string{"DRONE_DEPLOY_TO", "CI_PIPELINE_DEPLOY_TARGET"},
		},
		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "Enable debug mode.",
			EnvVars: []string{"PLUGIN_DEBUG", "INPUT_DEBUG", "DEBUG"},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	plugin := Plugin{
		Repo: Repo{
			FullName:  c.String("repo"),
			Namespace: c.String("repo.namespace"),
			Name:      c.String("repo.name"),
		},
		Commit: Commit{
			Sha:     c.String("commit.sha"),
			Ref:     c.String("commit.ref"),
			Branch:  c.String("commit.branch"),
			Link:    c.String("commit.link"),
			Author:  c.String("commit.author"),
			Email:   c.String("commit.author.email"),
			Avatar:  authorAvatar(c.String("commit.author.avatar")),
			Message: c.String("commit.message"),
		},
		Build: Build{
			Tag:      c.String("build.tag"),
			Number:   c.Int("build.number"),
			Event:    c.String("build.event"),
			Status:   c.String("build.status"),
			Link:     buildLink(c.String("build.link")),
			Started:  c.Int64("build.started"),
			Finished: c.Int64("build.finished"),
			PR:       c.String("pull.request"),
			DeployTo: c.String("deploy.to"),
		},
		Config: Config{
			webhookURL:     c.String("webhook-url"),
			WebhookID:      c.String("webhook-id"),
			webhookSecret:  c.String("webhook-secret"),
			BaseURL:        c.String("base-url"),
			Title:          c.String("title"),
			Description:    c.String("description"),
			Status:         c.String("status"),
			Color:          c.String("color"),
			SourceURL:      c.String("source-url"),
			ActorName:      c.String("actor-name"),
			ActorAvatarURL: c.String("actor-avatar-url"),
			Debug:          c.Bool("debug"),
		},
	}

	if plugin.Config.Debug {
		_ = godump.Dump(plugin)
	}

	return plugin.Exec(c.Context)
}

// buildLink falls back to the GitHub Actions run URL when no CI link is set.
func buildLink(link string) string {
	if link != "" {
		return link
	}
	server, repo, runID := os.Getenv("GITHUB_SERVER_URL"), os.Getenv("GITHUB_REPOSITORY"), os.Getenv("GITHUB_RUN_ID")
	if server == "" || repo == "" || runID == "" {
		return ""
	}
	return server + "/" + repo + "/actions/runs/" + runID
}

// authorAvatar falls back to the GitHub avatar of the actor on GitHub Actions.
func authorAvatar(avatar string) string {
	if avatar != "" {
		return avatar
	}
	if actor := os.Getenv("GITHUB_ACTOR"); actor != "" && os.Getenv("GITHUB_ACTIONS") == "true" {
		return "https://github.com/" + url.PathEscape(actor) + ".png"
	}
	return ""
}
