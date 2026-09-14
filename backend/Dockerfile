# syntax=docker/dockerfile:1
# Multi-stage, pinned versions (docs/conventions.md §Version pinning, D-015).

FROM golang:1.26.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 -h /home/app app
USER app
COPY --from=build /out/server /usr/local/bin/server
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1
# serve = migrate + seed + serve (D-014); override to "migrate" for migrations only
ENTRYPOINT ["server"]
CMD ["serve"]