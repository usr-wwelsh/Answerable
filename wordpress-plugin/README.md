# Answerable Connect (WordPress plugin)

Makes a WordPress site agent-discoverable without server access, DNS
changes, or an IT team. Install like any other plugin, paste your running
Answerable instance's URL into Settings → Answerable Connect, done.

Answerable still does all the actual work — facts, agent card, MCP,
booking. This plugin only:

- Proxies `/llms.txt`, `/.well-known/agent.json`, `/.well-known/mcp.json`
  from your WordPress domain to your Answerable instance (these are
  supposed to resolve same-origin, so a plain script tag can't serve them —
  they need this).
- Adds the same discovery `<link>` tags to `<head>` that Answerable's own
  reverse-proxy mode injects.

## Install

1. WordPress admin → Plugins → Add New → Upload Plugin → pick this folder
   zipped up → Activate.
2. Settings → Answerable Connect → paste your Answerable instance's base
   URL (e.g. `https://myorg-answerable.up.railway.app`).
3. Requires "pretty permalinks" (Settings → Permalinks → anything but
   "Plain") — the plugin warns you in wp-admin if they're off.

## Limitations

- You still need an Answerable instance running somewhere (your own host,
  Railway, etc.) — this plugin is a connector, not a reimplementation.
- `/mcp` and booking endpoints aren't proxied; they don't need to be
  same-origin, so the agent card and manifest just point agents straight
  at your Answerable instance for those.
- Some hosts intercept `.well-known/` at the web server level before
  WordPress ever sees the request. Rare on shared WP hosting, but worth
  checking if the proxy paths 404.
