# x-digest

CLI tool that reads your X (Twitter) feed and shows you the highlights — sorted by engagement, styled in the terminal. No API keys, no cost.

## How it works

1. Reads your `auth_token` and `ct0` cookies from Firefox
2. Hits X's internal GraphQL API (same endpoints the website uses)
3. Sorts tweets by engagement (likes + retweets)
4. Renders with [Lip Gloss](https://github.com/charmbracelet/lipgloss)

Your cookies never leave your machine. The only network calls go to `x.com`.

## Project structure

```
x-digest/
  cmd/x-digest/          # entry point — flag parsing, wiring
  internal/
    twitter/             # API client, response parsing, types
    cookies/             # Firefox cookie extraction
    display/             # lipgloss rendering, formatting
  Makefile
  README.md
```

## Install

```
go install github.com/pscyk/x-digest/cmd/x-digest@latest
```

Or build from source:

```
git clone https://github.com/pscyk/x-digest.git
cd x-digest
make build
```

### Prerequisites

- **Go 1.21+**
- **Firefox** with an active X/Twitter session (must be logged in)

## Usage

```bash
# Top 20 from your Following feed
x-digest

# Top 10
x-digest --count 10

# For You feed
x-digest --timeline foryou

# Specific user's tweets
x-digest --user elonmusk
x-digest --user @Polymarket --count 5

# Search your bookmarks
x-digest --bookmarks "claude code"
x-digest --bookmarks all --count 10
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--count` | `20` | Number of top tweets to show |
| `--timeline` | `following` | `following` or `foryou` |
| `--user` | | Fetch tweets from a specific account |
| `--bookmarks` | | Search bookmarks (keyword, or `all`) |
| `--profile` | | Firefox profile name override |
| `--query-id` | | Override GraphQL query ID (if default breaks) |

## About the bearer token

The bearer token in `client.go` is X's **public** web-client token — identical for every user who visits x.com. It's embedded in their JavaScript bundle and is used by every open-source X scraper. It is not a secret.

## Query ID rotation

X periodically rotates GraphQL query IDs when they deploy new client bundles. If you get a `400` or `Query not found` error, the IDs are stale. To fix:

1. Visit `view-source:https://x.com` and find the main JS bundle URL
2. Search the bundle for `operationName:"HomeLatestTimeline"` to find the new query ID
3. Pass it with `--query-id <new-id>`, or update the constants in `internal/twitter/client.go`

## Style

This project follows [Go-Tiger-Style](https://github.com/Predixus/Go-Tiger-Style):

- Explicit slice/map capacity allocation
- Table-driven tests + fuzz tests
- Transparent error handling — no panics in release builds
- `internal/` packages to prevent external import of internals
