#!/bin/sh
set -e

ANSWERABLE_PORT="${ANSWERABLE_PORT:-8081}"
ANSWERABLE_ADMIN_PORT="${ANSWERABLE_ADMIN_PORT:-8090}"

# The admin webui is bound to 0.0.0.0 so it can get its own public port on
# Railway (see README). That makes it reachable off this machine, so the
# app itself fails closed without a password: see RequireAuthForBind in
# internal/admin/auth.go. Fail here too, with a clearer message, instead of
# letting the backgrounded process die silently while Caddy keeps running.
if [ -z "$ANSWERABLE_ADMIN_PASSWORD" ]; then
	echo "entrypoint: ANSWERABLE_ADMIN_PASSWORD must be set (admin webui binds to 0.0.0.0 in this image)" >&2
	exit 1
fi

/usr/local/bin/answerable \
	-port "$ANSWERABLE_PORT" \
	-admin-bind 0.0.0.0 \
	-admin-port "$ANSWERABLE_ADMIN_PORT" \
	-db /data/answerable.db \
	-no-browser &

exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
