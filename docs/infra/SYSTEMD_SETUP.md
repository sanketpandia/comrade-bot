# Systemd Service Setup for Labour Bureau Services

This guide explains how to set up the Labour Bureau services to start automatically on system boot using systemd.

## Container Restart Behavior

✅ **Containers will automatically restart on crash!**

All containers are created with the `--restart unless-stopped` flag, which means:
- ✅ Containers automatically restart if they crash
- ✅ Containers restart after system reboot (if the systemd service is enabled)
- ⚠️ Containers do NOT restart if you manually stop them (`podman stop`)
- ✅ Containers restart if the systemd service is restarted

This is configured in each individual start script (e.g., `scripts/start-db.sh`, `scripts/start-politburo.sh`).

## Installation Steps

### 1. Determine Podman User

First, check if you're running podman as root or as a regular user:

```bash
# If this works without sudo, you're using rootless podman (use your username)
podman ps

# If you need sudo, you're using rootful podman (use root)
sudo podman ps
```

### 2. Update Service File

Edit `labour-bureau-services.service` and set the correct `User` and `Group`:

```bash
nano labour-bureau-services.service
```

- **For rootless podman**: Set `User=your-username` and `Group=your-username`
- **For rootful podman**: Set `User=root` and `Group=root`

Also verify the `WorkingDirectory` path matches your actual installation path.

### 3. Install the Service

Copy the service file to systemd directory:

```bash
# For user-level service (rootless podman)
cp labour-bureau-services.service ~/.config/systemd/user/

# For system-level service (rootful podman)
sudo cp labour-bureau-services.service /etc/systemd/system/
```

### 4. Reload Systemd

```bash
# For user-level service
systemctl --user daemon-reload

# For system-level service
sudo systemctl daemon-reload
```

### 5. Enable the Service

This makes the service start automatically on boot:

```bash
# For user-level service
systemctl --user enable labour-bureau-services.service

# For system-level service
sudo systemctl enable labour-bureau-services.service
```

### 6. Start the Service

```bash
# For user-level service
systemctl --user start labour-bureau-services.service

# For system-level service
sudo systemctl start labour-bureau-services.service
```

## Managing the Service

### Check Status

```bash
# For user-level service
systemctl --user status labour-bureau-services.service

# For system-level service
sudo systemctl status labour-bureau-services.service
```

### View Logs

```bash
# For user-level service
journalctl --user -u labour-bureau-services.service -f

# For system-level service
sudo journalctl -u labour-bureau-services.service -f
```

### Stop the Service

```bash
# For user-level service
systemctl --user stop labour-bureau-services.service

# For system-level service
sudo systemctl stop labour-bureau-services.service
```

### Restart the Service

```bash
# For user-level service
systemctl --user restart labour-bureau-services.service

# For system-level service
sudo systemctl restart labour-bureau-services.service
```

### Disable Auto-Start

```bash
# For user-level service
systemctl --user disable labour-bureau-services.service

# For system-level service
sudo systemctl disable labour-bureau-services.service
```

## How It Works

1. **On Boot**: Systemd starts the service after network and podman are ready
2. **Service Start**: The `start-services.sh` script runs, which:
   - Creates the podman network if needed
   - Creates volumes if needed
   - Starts each container (or starts existing containers)
3. **Container Restart**: Each container has `--restart unless-stopped`, so they auto-restart on crash
4. **On Shutdown**: Systemd calls `stop-services.sh` to gracefully stop all containers

## Troubleshooting

### Service Fails to Start

1. Check the logs:
   ```bash
   journalctl --user -u labour-bureau-services.service -n 50
   ```

2. Verify podman is working:
   ```bash
   podman ps
   ```

3. Test the script manually:
   ```bash
   cd /home/odin/projects/infinite-experiment/labour-bureau/prod
   ./start-services.sh
   ```

### Containers Not Starting

1. Check if containers exist:
   ```bash
   podman ps -a
   ```

2. Check container logs:
   ```bash
   podman logs <container-name>
   ```

3. Verify environment files exist:
   ```bash
   ls -la prod/env/*.env
   ```

### Permission Issues

If you get permission errors:

1. For rootless podman, ensure the service file uses your username
2. For rootful podman, ensure the service file uses `root`
3. Check file permissions on the scripts:
   ```bash
   chmod +x start-services.sh stop-services.sh
   chmod +x scripts/*.sh
   ```

### Service Starts but Containers Don't

The service uses `Type=oneshot` with `RemainAfterExit=yes`, which means:
- The service reports as "active" once the script completes
- If containers crash, they restart automatically (via `--restart unless-stopped`)
- The systemd service itself doesn't need to restart for container restarts

To verify containers are running:
```bash
podman ps
```

## Notes

- The service is `Type=oneshot` because it just runs a script to start containers
- Containers handle their own restart logic via `--restart unless-stopped`
- The systemd service only needs to run once on boot to start everything
- Individual container restarts don't require the systemd service to restart
