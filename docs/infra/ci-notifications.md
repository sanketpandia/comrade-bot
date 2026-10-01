# CI/CD Discord notifications

Pipeline results are posted to a Discord channel via webhook.

## Setup

1. Discord → target channel → **Integrations** → **Webhooks** → create webhook → copy URL.
2. GitHub → repository **Settings** → **Secrets and variables** → **Actions** → **New repository secret**:
   - Name: `DISCORD_WEBHOOK_URL`
   - Value: webhook URL

Optional: scope the same secret to the **`production`** environment for deploy-only notifications.

## Behavior

| Event | Discord |
|-------|---------|
| Any workflow job **failure** (PR or `main`) | Notify |
| **`main`** `production-image` success | Notify (image pushed to GHCR) |
| **`main`** `deploy` success / failure | Notify |
| PR success only | Silent |

Implementation: [`.github/actions/discord-notify`](../../.github/actions/discord-notify).

## Rotate webhook

Create a new webhook in Discord, update `DISCORD_WEBHOOK_URL`, delete the old webhook in Discord.
