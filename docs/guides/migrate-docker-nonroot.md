# Migrating Docker Containers to Non-Root User

Starting with this release, the `evm`, `testapp`, and `local-da` Docker images run as a non-root user (`ev-node`, uid/gid 1000) instead of `root`. This aligns with the `grpc` image, which already ran as non-root.

If you are running any of these containers with **persistent volumes or bind mounts**, you need to fix file ownership before upgrading. Containers running without persistent storage (ephemeral) require no action.

## Who is affected

You are affected if **all** of the following are true:

- You run `evm`, `testapp`, or `local-da` via Docker (or docker-compose / Kubernetes)
- You use a volume or bind mount for the container's data directory
- The files in that volume were created by a previous (root-based) image

## Migration steps

### 1. Stop the running container

```bash
docker stop <container-name>
```

### 2. Fix file ownership on the volume

For **bind mounts** (host directory), run `chown` directly on the host:

```bash
# Replace /path/to/data with your actual data directory
sudo chown -R 1000:1000 /path/to/data
```

For **named Docker volumes**, use a temporary container:

```bash
# Replace <volume-name> with your Docker volume name
docker run --rm -v <volume-name>:/data alpine chown -R 1000:1000 /data
```

### 3. Pull the new image and restart

```bash
docker pull <image>
docker start <container-name>
```

### Kubernetes / docker-compose

If you manage containers through orchestration, you have two options:

**Option A: Init container (recommended for Kubernetes)**

Add an init container that fixes ownership before the main container starts:

```yaml
initContainers:
  - name: fix-permissions
    image: alpine:3.22
    command: ["chown", "-R", "1000:1000", "/home/ev-node"]
    volumeMounts:
      - name: data
        mountPath: /home/ev-node
securityContext:
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
```

**Option B: Set `fsGroup` in the pod security context**

If your volume driver supports it, setting `fsGroup: 1000` will automatically fix ownership on mount:

```yaml
securityContext:
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
```

**docker-compose**: update your `docker-compose.yml` to set the user:

```yaml
services:
  evm:
    image: evm:latest
    user: "1000:1000"
    volumes:
      - evm-data:/home/ev-node
```

## Verifying the migration

After restarting, confirm the container runs as the correct user:

```bash
docker exec <container-name> id
# Expected: uid=1000(ev-node) gid=1000(ev-node)
```

Check that the process can read and write its data directory:

```bash
docker exec <container-name> ls -la /home/ev-node
# All files should be owned by ev-node:ev-node
```

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `Permission denied` on startup | Volume files still owned by root | Re-run the `chown` step above |
| Container exits immediately | Data directory not writable | Check ownership and directory permissions (`drwxr-xr-x` or more permissive for uid 1000) |
| Application writes to wrong path | Old `WORKDIR` was `/root` or `/apps` | Update any custom volume mounts to target `/home/ev-node` instead |
