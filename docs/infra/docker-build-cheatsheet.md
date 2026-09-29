1. Build and Run the Service Locally

Navigate to your service repo, e.g. comrade-bot:
```
cd comrade-bot
```
Build the image:
```
docker build -t comrade-bot:dev --target=dev .
```
or for production:
```
docker build -t comrade-bot:prod --target=production .
```
2. Run the Container

You need to provide any required environment variables (e.g., your Discord bot token, API URL).
You can use an .env file or just pass them via the docker run command.

Example (from your repo root):
```
docker run --rm \
  -e BOT_TOKEN=your_actual_discord_token \
  -e API_URL=http://localhost:8080 \
  comrade-bot:dev
```
    --rm: Clean up the container after it stops.

    -e VAR=val: Sets environment variables.

    comrade-bot:dev: The image you just built.

You can also mount code for hot reload, if your Dockerfile/dev target supports it:

```
docker run --rm \
  -e BOT_TOKEN=your_actual_discord_token \
  -e API_URL=http://localhost:8080 \
  -v $PWD/src:/app/src \
  comrade-bot:dev
```

This will mount your local src folder into the container for live code changes.
3. Using an .env File

If you want to use a local .env file:
```
docker run --rm \
  --env-file .env \
  comrade-bot:dev
```

Your .env file should look like:
```
BOT_TOKEN=your_actual_discord_token
API_URL=http://localhost:8080
```

4. Connecting to Other Services

    If you want your bot to call a live politburo backend, set API_URL to wherever it’s running.

    If testing only bot logic (not calling the API), you can set API_URL to anything valid or mock it.

5. Ports

    The Discord bot typically does not expose any ports, unless you add a status/healthcheck web endpoint. If it does, add -p <host>:<container> to your run command.

6. Logs

    All logs print to your terminal by default.

    You can check logs with docker logs <container-id> if you detach with -d.

7. Example: Run Production Build

```
docker build -t comrade-bot:prod --target=production .
docker run --rm --env-file .env comrade-bot:prod
```

8. No Compose Needed

    This is completely independent from labour-bureau or any compose file.

    The only thing you need is the repo for the service you want to run, plus its dependencies (like Node/npm for builds, or just Docker if you build everything in Docker).

| Step       | Command Example                                           | Notes                  |
| ---------- | --------------------------------------------------------- | ---------------------- |
| Build      | `docker build -t comrade-bot:dev --target=dev .`          | From repo root         |
| Run        | `docker run --rm --env-file .env comrade-bot:dev`         | Use env vars as needed |
| Hot reload | Add `-v $PWD/src:/app/src` (if Dockerfile supports)       | Dev only               |
| Logs       | Logs appear in terminal, or use `docker logs`             |                        |
| Stopping   | `CTRL+C` (if foreground), or `docker stop <container-id>` |                        |
