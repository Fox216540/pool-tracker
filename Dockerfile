# syntax=docker/dockerfile:1.7
FROM golang:1.26-bookworm AS backend-build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 go build -o /pool-tracker ./cmd/api

FROM node:20-bookworm-slim AS frontend-build
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM debian:bookworm-slim AS api
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wget && rm -rf /var/lib/apt/lists/* && useradd --system --uid 10001 app && mkdir /data && chown app:app /data
WORKDIR /app
COPY --from=backend-build /pool-tracker /app/pool-tracker
COPY migrations /app/migrations
USER app
EXPOSE 8080
ENTRYPOINT ["/app/pool-tracker"]

FROM nginx:1.27-alpine AS web
COPY --from=frontend-build /src/frontend/dist /usr/share/nginx/html
COPY deploy/nginx/default.conf /etc/nginx/conf.d/default.conf
