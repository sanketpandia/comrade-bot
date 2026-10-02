#!/usr/bin/env bash
# Path-filtered checks mirroring .github/workflows/ci.yml.
# Install: make hooks-install   Skip one push: git push --no-verify   Or: SKIP_PREPUSH=1
set -euo pipefail

if [[ "${SKIP_PREPUSH:-}" == "1" ]]; then
	exit 0
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
POLITBURO_DIR="${REPO_ROOT}/services/politburo"
BOT_DIR="${REPO_ROOT}/services/comrade-bot-discord"
CICD_DIR="${REPO_ROOT}/cicd"

collect_changed_files() {
	local range files="" line
	if [[ $# -gt 0 ]]; then
		git -C "$REPO_ROOT" diff --name-only "$1"
		return
	fi

	if [[ -t 0 ]]; then
		if git -C "$REPO_ROOT" rev-parse --verify '@{u}' >/dev/null 2>&1; then
			git -C "$REPO_ROOT" diff --name-only '@{u}..'
		elif git -C "$REPO_ROOT" rev-parse --verify origin/main >/dev/null 2>&1; then
			git -C "$REPO_ROOT" diff --name-only origin/main..HEAD
		elif git -C "$REPO_ROOT" rev-parse --verify main >/dev/null 2>&1; then
			git -C "$REPO_ROOT" diff --name-only main..HEAD
		else
			git -C "$REPO_ROOT" diff --name-only HEAD~1..HEAD 2>/dev/null || true
		fi
		return
	fi

	while read -r _local_ref local_sha _remote_ref remote_sha; do
		if [[ "$local_sha" == "0000000000000000000000000000000000000000" ]]; then
			continue
		fi
		if [[ "$remote_sha" == "0000000000000000000000000000000000000000" ]]; then
			range="$local_sha"
			for base in "origin/main" "main"; do
				if git -C "$REPO_ROOT" rev-parse --verify "$base" >/dev/null 2>&1; then
					range="$(git -C "$REPO_ROOT" merge-base "$base" "$local_sha")..$local_sha"
					break
				fi
			done
		else
			range="${remote_sha}..${local_sha}"
		fi
		while IFS= read -r line; do
			[[ -n "$line" ]] && files+="${line}"$'\n'
		done < <(git -C "$REPO_ROOT" diff --name-only "$range" 2>/dev/null || true)
	done

	printf '%s' "$files" | sed '/^$/d' | sort -u
}

matches_any() {
	local file="$1"
	local pattern
	for pattern in "$@"; do
		case "$file" in
		$pattern) return 0 ;;
		esac
	done
	return 1
}

file_triggers_politburo_ci() {
	matches_any "$1" \
		"services/politburo/*" \
		"openapi/*" \
		"cicd/*" \
		"Makefile" \
		".github/workflows/ci.yml"
}

file_triggers_bot_ci() {
	matches_any "$1" \
		"services/comrade-bot-discord/*" \
		"openapi/*" \
		"cicd/*" \
		"Makefile" \
		".github/workflows/ci.yml" \
		"infra/prod/scripts/k8s-sync-discord-commands.sh"
}

file_triggers_openapi_ci() {
	matches_any "$1" "openapi/*" "cicd/*" "Makefile"
}

changed="$(collect_changed_files "$@")"
if [[ -z "$changed" ]]; then
	exit 0
fi

run_politburo=false
run_bot=false
run_openapi=false

while IFS= read -r path; do
	[[ -z "$path" ]] && continue
	if file_triggers_politburo_ci "$path"; then run_politburo=true; fi
	if file_triggers_bot_ci "$path"; then run_bot=true; fi
	if file_triggers_openapi_ci "$path"; then run_openapi=true; fi
done <<< "$changed"

if [[ "$run_politburo" == false && "$run_bot" == false && "$run_openapi" == false ]]; then
	exit 0
fi

echo "pre-push: running CI-aligned checks for changed paths..."
echo "$changed" | sed 's/^/  /'

if [[ "$run_openapi" == true ]]; then
	echo "→ OpenAPI: bundle freshness"
	make -C "$REPO_ROOT" openapi-bundle-check
fi

if [[ "$run_politburo" == true || "$run_openapi" == true ]]; then
	echo "→ Politburo: generate, test, build"
	make -C "$CICD_DIR" generate-politburo generate-infinite-flight
	(
		cd "$POLITBURO_DIR"
		go test ./...
		go build -buildvcs=false ./cmd/politburo
	)
fi

if [[ "$run_bot" == true || "$run_openapi" == true ]]; then
	echo "→ Discord bot: api:generate, test, build, commands:validate"
	if [[ ! -d "$BOT_DIR/node_modules" ]]; then
		echo "pre-push: run 'npm ci' in services/comrade-bot-discord first" >&2
		exit 1
	fi
	(
		cd "$BOT_DIR"
		npm run api:generate
		npm test
		npm run build
		npm run commands:validate
	)
fi

echo "pre-push: OK"
