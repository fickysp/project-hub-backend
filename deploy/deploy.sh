#!/usr/bin/env bash
set -Eeuo pipefail
commit="${1:?Commit required}"
image="${2:?Image required}"
registry_user="${3:?Registry user required}"
[[ "$commit" =~ ^[0-9a-f]{40}$ ]] || exit 1
[[ "$image" =~ ^ghcr.io/[a-z0-9._/-]+$ ]] || exit 1
cd "$HOME/projecthub-backend"
test -s .env || { echo 'VM .env missing or empty'; exit 1; }
chmod 600 .env
auth_dir="$(mktemp -d)"
export DOCKER_CONFIG="$auth_dir"
trap 'rm -rf -- "$auth_dir"' EXIT
# Short-lived workflow token arrives on stdin, never as an argument or .env.
docker login ghcr.io -u "$registry_user" --password-stdin
docker pull "$image:$commit"
previous_image="$(docker inspect --format '{{.Config.Image}}' projecthub-api 2>/dev/null || true)"
previous_compose="incoming/$commit.previous.yaml"
if [ -f docker-compose.yaml ]; then cp docker-compose.yaml "$previous_compose"; fi
cp "incoming/$commit.compose.yaml" docker-compose.yaml
export REGISTRY_IMAGE="$image" IMAGE_TAG="$commit"
docker compose -p projecthub-backend -f docker-compose.yaml config --quiet
if docker compose -p projecthub-backend -f docker-compose.yaml up -d --wait --wait-timeout 150; then
 printf '%s\n' "$commit" > last-successful-commit
 echo 'Backend healthy on VM port 8080'
else
 echo 'Backend unhealthy; inspect logs on VM without sharing secrets.'
 if [ -n "$previous_image" ] && [ -f "$previous_compose" ]; then
  cp "$previous_compose" docker-compose.yaml
  export REGISTRY_IMAGE="${previous_image%:*}" IMAGE_TAG="${previous_image##*:}"
  docker compose -p projecthub-backend -f docker-compose.yaml up -d --wait --wait-timeout 150
  echo 'Application image restored; database schema is not rolled back.'
 fi
 exit 1
fi
