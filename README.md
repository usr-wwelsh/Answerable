# Answerable

[![Read about the commits](https://img.shields.io/badge/commits-code%20blog-1a1a1a?style=flat-square)](https://wwel.sh/digest.html?repo=answerable)

Agent-discoverable civic service endpoints. A provider (e.g. a shelter) drops a
CSV/md/txt/pdf or pastes a URL into a local admin page; Answerable republishes
those facts (hours, eligibility, availability) as a live public JSON-LD +
llms.txt + A2A agent-card endpoint, so any agent asking on someone's behalf
gets the provider's real current answer instead of stale scraped data.

Built for Venture 313 Buildathon 2026, category: Open, Accessible & Responsible
Government.

If you're adding Answerable to an existing web stack, you'll need to set up
Caddy (see `Dockerfile` / `deploy/`) and the admin UI password if you intend
for that to be public.

MIT licensed.
