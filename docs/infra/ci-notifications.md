# CI/CD Discord notifications

Pipeline results are posted to a Discord channel via webhook.

## Setup

1. Discord → target channel → **Integrations** → **Webhooks** → create webhook → copy URL.
2. GitHub → **Settings** → **Environments** → **`production`** → add **Environment secret** (preferred for deploy/CD):
   - Name: `DISCORD_WEBHOOK_URL` (or `WEBHOOK_URL`)
   - Value: webhook URL

   Environment **variables** with the same names are also supported if you prefer not to store the URL as a secret.

3. For **PR CI** failure alerts (jobs that run on `ubuntu-latest` without the prod runner), either:
   - allow the **`production`** environment on all branches (no deployment-branch restriction), or
   - duplicate the webhook as a **repository** secret `DISCORD_WEBHOOK_URL` / `WEBHOOK_URL` (repository **Settings** → **Secrets and variables** → **Actions**).

Workflows resolve the URL in this order: `DISCORD_WEBHOOK_URL` secret → `WEBHOOK_URL` secret → same names as environment/repository **variables**.

Jobs that post to Discord use `environment: production` so environment-scoped secrets and variables are visible. Notify-only jobs use `continue-on-error: true` so a missing webhook or Discord outage does not fail the pipeline.

## Behavior

| Event | Discord |
|-------|---------|
| Any workflow job **failure** (PR or `main`) | Notify (when webhook is configured) |
| **`main`** `production-image` success | Notify (image pushed to GHCR) |
| **`main`** `deploy` success / failure | Notify |
| PR success only | Silent |

Implementation: [`.github/actions/discord-notify`](../../.github/actions/discord-notify).

## Rotate webhook

Create a new webhook in Discord, update `DISCORD_WEBHOOK_URL`, delete the old webhook in Discord.
