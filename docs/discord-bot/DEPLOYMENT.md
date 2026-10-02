# Command Deployment Guide

This guide explains how to deploy slash commands for Comrade Bot.

## Quick Reference

| Scenario | Command | When to Use |
|----------|---------|-------------|
| **Dev - TypeScript** | `npm run deploy:dev:local` | Local development with TS |
| **Prod - JavaScript** | `npm run deploy:local` | After `npm run build` |
| **Shell Script** | `./deploy.sh local` | Works in dev and prod |
| **Docker Container** | `docker exec comrade-bot /app/deploy.sh local` | Running in Docker |
| **Discord Command** | `/rollout mode:local` | Quick updates via Discord (god-mode only) |
| **Production (k3s CI)** | `[sync-cmds]` in merge commit or Actions workflow | Opt-in global sync after deploy (see below) |

## Overview

Comrade Bot supports two deployment modes:

1. **Local (Guild-Specific)** - Instant updates, great for development
2. **Global** - Updates all servers, takes ~1 hour to propagate

---

## Initial Setup: Deploy the `/rollout` Command

Before you can use the `/rollout` command, you need to deploy it first using npm scripts.

### For Local Development (Single Server)

```bash
# Make sure GUILD_ID is set in .env
npm run deploy:local
```

This deploys commands **only** to the server specified in the `GUILD_ID` environment variable. Commands are available **immediately**.

### For Production (All Servers)

```bash
npm run deploy:global
```

This deploys commands to **all servers** where the bot is installed. Commands take **up to 1 hour** to propagate.

---

## Using the `/rollout` Command (After Initial Deployment)

Once deployed, god-mode users can use the `/rollout` command directly in Discord for future updates.

### Local Rollout (Quick Updates)

```
/rollout mode:🏠 Local (this server only - instant)
```

- ✅ Updates commands for the current server only
- ✅ Changes are **immediate**
- ✅ Perfect for testing and development
- ⚠️ Requires god-mode access

### Global Rollout (Production Updates)

```
/rollout mode:🌍 Global (all servers - takes ~1 hour)
```

- ✅ Updates commands for **all servers**
- ⏳ Changes take **up to 1 hour** to propagate
- ✅ Use for production releases
- ⚠️ Requires god-mode access

---

## God-Mode Access

The `/rollout` command is restricted to god-mode users only.

### Setting Up God-Mode

1. Add your Discord user ID to `PLATFORM_OPERATOR_DISCORD_IDS` in Politburo config (comma-separated if multiple operators).

2. The bot verifies god-mode access via the backend API endpoint:
   - `GET /api/v1/admin/verify-god`

3. Only Discord users listed in `PLATFORM_OPERATOR_DISCORD_IDS` can use `/rollout`

---

## Deployment Methods

### Method 1: NPM Scripts (Local Development)

For development on your local machine with TypeScript:

| Script | Description | Speed |
|--------|-------------|-------|
| `npm run deploy:dev` | Auto-detect mode (local if GUILD_ID set, else global) | Varies |
| `npm run deploy:dev:local` | Deploy to single guild (requires GUILD_ID) | Instant |
| `npm run deploy:dev:global` | Deploy to all servers globally | ~1 hour |

### Method 2: NPM Scripts (Production Build)

For deployment with compiled JavaScript:

| Script | Description | Speed |
|--------|-------------|-------|
| `npm run deploy` | Auto-detect mode (local if GUILD_ID set, else global) | Varies |
| `npm run deploy:local` | Deploy to single guild (requires GUILD_ID) | Instant |
| `npm run deploy:global` | Deploy to all servers globally | ~1 hour |

**Important**: Production scripts require `npm run build` first!

```bash
npm run build           # Compile TypeScript to JavaScript
npm run deploy:local    # Deploy using compiled JS
```

### Method 3: Shell Script (Works Everywhere)

The `deploy.sh` script automatically detects whether to use TypeScript or JavaScript:

```bash
# Auto-detect mode
./deploy.sh

# Local deployment
./deploy.sh local

# Global deployment
./deploy.sh global
```

### Method 4: Docker Container

For deployment inside a Docker container:

```bash
# Enter the running container
docker exec -it comrade-bot /bin/bash

# Inside container - use shell script (production mode)
cd /app
./deploy.sh local

# Or use npm scripts (production mode)
npm run deploy:local
```

---

## Production (k3s / GitHub Actions)

Rolling out the comrade-bot **container** (`.github/workflows/discord-bot.yml` deploy job) does **not** update Discord slash command definitions. Command sync is **opt-in**.

### Opt in on merge to `main`

Include `[sync-cmds]` in the **merge commit message**. When the workflow runs (bot-related path filters), it will:

1. Build and push the image (on push to `main`)
2. Roll out the deployment on the self-hosted prod runner
3. Run the **sync-discord-commands** job: a short-lived Kubernetes Job in `ie-apps` that runs `node dist/deploy-commands.js global` using the same GHCR image tag as the build and credentials from the `comrade-bot-env` secret

If you merge without `[sync-cmds]`, only the bot image is updated; run sync manually when command schemas change.

### Manual sync (Actions)

**Actions → Discord bot CI → Run workflow**

| Input | Purpose |
|-------|---------|
| **Sync Discord commands** | Run global command deploy (default on) |
| **Commands only** | Skip test, image build, and app deploy; only sync |
| **Image tag** | GHCR tag when **commands only** (e.g. `main` or a commit SHA) |

Use **commands only** to push slash definitions from an already-published image without redeploying pods.

On the server, the runner invokes [`infra/prod/scripts/k8s-sync-discord-commands.sh`](../../infra/prod/scripts/k8s-sync-discord-commands.sh).

### Break-glass

If CI is unavailable, operators can still use `/rollout mode:global` (god-mode) or run the sync script locally with `kubectl` access and `IMAGE` set.

---

## Deployment Flow

### First-Time Setup

```bash
# 1. Set environment variables in .env
GUILD_ID=your_test_server_id
# Politburo: PLATFORM_OPERATOR_DISCORD_IDS for /rollout

# 2. Deploy commands locally for testing
npm run deploy:local

# 3. Test the bot and commands in your server

# 4. When ready for production, deploy globally
npm run deploy:global
```

### Ongoing Development

```bash
# 1. Make changes to commands

# 2. Test locally on your dev server
npm run deploy:local

# 3. Test in Discord

# 4. For production k3s: merge with [sync-cmds] when slash schemas changed
#    Or use Actions → Discord bot CI (see Production section above)
#    Dev break-glass: /rollout mode:local or mode:global (god-mode)
```

---

## Important Notes

### Command Cache Behavior

- **Local deployments**: Discord updates the command cache **immediately**
- **Global deployments**: Discord propagates changes over **up to 1 hour**

### When to Use Each Mode

**Use Local (`npm run deploy:local` or `/rollout mode:local`):**
- During active development
- Testing new commands
- Quick iterations
- Single-server updates

**Use Global (`npm run deploy:global` or `/rollout mode:global`):**
- Production releases
- Deploying to all servers
- Final verified changes
- When you're confident the changes are stable

### Rollout Command Deployment

The `/rollout` command itself must be deployed using npm scripts first. After that, you can use it for all future deployments directly in Discord.

**Chicken-and-egg problem solution:**
```bash
# Initial deployment (includes /rollout command)
npm run deploy:local

# Now you can use /rollout for all future updates!
/rollout mode:local
```

---

## Troubleshooting

### Commands not appearing in Discord

1. **Local deployment**: Commands should appear within seconds
   - Check if `GUILD_ID` matches your server
   - Verify bot has permission in the server

2. **Global deployment**: Wait up to 1 hour
   - Discord caches global commands
   - Be patient, they will appear

### `/rollout` command not working

1. Verify god-mode is configured (`PLATFORM_OPERATOR_DISCORD_IDS` in Politburo includes your Discord user ID)

2. Check bot logs for authentication errors

3. Ensure the bot is running and connected

### Permission Denied Error

The `/rollout` command requires god-mode access. Only the Discord user specified in the `GOD_MODE` environment variable can use it.

---

## Example Workflows

### Local Development Workflow

```bash
# Day 1: Setup
npm run deploy:dev:local      # Deploy to dev server
# Test /rollout command works

# Day 2: Add new /status command
# Edit src/commands/status.ts
npm run deploy:dev:local      # Quick test
# Verify in Discord

# Day 3: Ready for production
/rollout mode:global          # Deploy to all servers using Discord
# Wait 1 hour for propagation

# Day 4: Hot fix needed
/rollout mode:local           # Fix in dev server first
# Test thoroughly
/rollout mode:global          # Deploy to production
```

### Docker Production Workflow

```bash
# Day 1: Initial deployment on server
docker exec -it comrade-bot /bin/bash
cd /app
./deploy.sh local             # Deploy to your server
exit
# Test /rollout command works

# Day 2: Code changes pushed to production
# After rebuilding container
docker exec comrade-bot /app/deploy.sh local

# Day 3: Use /rollout for convenience
/rollout mode:local           # Quick updates via Discord
# No need to enter container!

# Day 4: Global rollout
/rollout mode:global          # Deploy to all servers
# Wait 1 hour for propagation
```

### One-Liner Docker Deployments

From your server host machine (outside container):

```bash
# Local deployment
docker exec comrade-bot /app/deploy.sh local

# Global deployment
docker exec comrade-bot /app/deploy.sh global

# Or using npm
docker exec comrade-bot npm run deploy:local
docker exec comrade-bot npm run deploy:global
```

---

## Architecture

The `/rollout` command uses **CommandDeploymentService**, which runs inside the bot process:

```
Discord /rollout command
    ↓
Verify god-mode (backend API)
    ↓
DeploymentService.deployToGuild() or .deployGlobally()
    ↓
Discord REST API
    ↓
Commands deployed!
```

No separate container or process needed - everything runs in the bot!
