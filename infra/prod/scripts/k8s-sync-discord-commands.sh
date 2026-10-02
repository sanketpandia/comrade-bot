#!/usr/bin/env bash
# k8s-sync-discord-commands.sh — Run Discord slash-command deploy via a one-off Kubernetes Job.
#
# Usage:
#   IMAGE=ghcr.io/owner/comrade-bot:main SCOPE=global ./k8s-sync-discord-commands.sh
#
# Env:
#   IMAGE     (required) Container image with dist/deploy-commands.js
#   SCOPE     global (default) or local (requires GUILD_ID in comrade-bot-env)
#   NAMESPACE default ie-apps
#   GITHUB_RUN_ID optional; used for unique Job names

set -euo pipefail

IMAGE="${IMAGE:?IMAGE is required}"
SCOPE="${SCOPE:-global}"
NAMESPACE="${NAMESPACE:-ie-apps}"
RUN_ID="${GITHUB_RUN_ID:-manual}"
JOB_NAME="comrade-bot-sync-cmds-${RUN_ID}-$(date +%s)"

if [[ "$SCOPE" != "global" && "$SCOPE" != "local" ]]; then
  echo "SCOPE must be global or local, got: $SCOPE" >&2
  exit 1
fi

echo "Creating Job ${JOB_NAME} in ${NAMESPACE} (image=${IMAGE}, scope=${SCOPE})"

kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: ${JOB_NAME}
  namespace: ${NAMESPACE}
spec:
  ttlSecondsAfterFinished: 600
  backoffLimit: 0
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: deploy-commands
          image: ${IMAGE}
          imagePullPolicy: Always
          command: ["node", "dist/deploy-commands.js", "${SCOPE}"]
          envFrom:
            - secretRef:
                name: comrade-bot-env
EOF

cleanup() {
  kubectl -n "$NAMESPACE" delete job "$JOB_NAME" --ignore-not-found --wait=false 2>/dev/null || true
}

trap cleanup EXIT

if ! kubectl -n "$NAMESPACE" wait --for=condition=complete "job/${JOB_NAME}" --timeout=5m; then
  echo "Job did not complete successfully; fetching logs:" >&2
  kubectl -n "$NAMESPACE" logs "job/${JOB_NAME}" --tail=200 2>/dev/null || true
  kubectl -n "$NAMESPACE" describe "job/${JOB_NAME}" >&2 || true
  exit 1
fi

kubectl -n "$NAMESPACE" logs "job/${JOB_NAME}"
echo "Discord command sync completed (Job ${JOB_NAME})"
