FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/answerable ./cmd/answerable

FROM caddy:2-alpine
COPY --from=build /out/answerable /usr/local/bin/answerable
COPY deploy/Caddyfile /etc/caddy/Caddyfile
COPY deploy/entrypoint.sh /entrypoint.sh
COPY site/ /srv/site/
RUN chmod +x /entrypoint.sh && mkdir -p /data

ENV ANSWERABLE_PORT=8081
ENV ANSWERABLE_ADMIN_PORT=8090

# The public port is dynamic (Railway sets $PORT, Caddy binds to it).
# 8090 (admin) is a separate, fixed port: give it its own public domain in
# Railway's networking settings so it doesn't need path-rewriting through
# Caddy. Requires ANSWERABLE_ADMIN_PASSWORD to be set, see entrypoint.sh.
EXPOSE 8090

ENTRYPOINT ["/entrypoint.sh"]
