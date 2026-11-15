#!/usr/bin/env bash
set -Eeuo pipefail
commit="${1:?Commit SHA required}"
[[ "$commit" =~ ^[0-9a-f]{40}$ ]] || { echo 'Invalid commit SHA'; exit 1; }
deploy_dir="$HOME/projecthub-backend"
cd "$deploy_dir"
test -f .env || { echo 'Create the VM .env first'; exit 1; }
chmod 600 .env
docker info >/dev/null
docker compose version >/dev/null
archive="incoming/$commit.tar.gz"
candidate="incoming/$commit.compose.yaml"
test -f "$archive" && test -f "$candidate"
docker load -i "$archive" >/dev/null
previous_image="$(docker inspect --format '{{.Config.Image}}' projecthub-api 2>/dev/null || true)"
previous_compose="incoming/$commit.previous.yaml"
if [ -f docker-compose.yaml ]; then cp docker-compose.yaml "$previous_compose"; fi
cp "$candidate" docker-compose.yaml
export IMAGE_TAG="$commit"
docker compose -p projecthub-backend -f docker-compose.yaml config --quiet
if docker compose -p projecthub-backend -f docker-compose.yaml up -d --wait --wait-timeout 150; then
  printf '%s\n' "$commit" > last-successful-commit
  rm -f "$archive"
  echo "Deployment healthy: $commit"
else
  echo 'Deployment failed its health check.'
  if [[ "$previous_image" == projecthub-api:* ]] && [ -f "$previous_compose" ]; then
    cp "$previous_compose" docker-compose.yaml
    export IMAGE_TAG="${previous_image#projecthub-api:}"
    docker compose -p projecthub-backend -f docker-compose.yaml up -d --wait --wait-timeout 150
    echo 'Previous application image restored; database schema is not rolled back.'
  fi
  exit 1
fi
