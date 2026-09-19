# Answerable

[![Read about the commits](https://img.shields.io/badge/commits-code%20blog-1a1a1a?style=flat-square)](https://wwel.sh/digest.html?repo=answerable)

**Everyone uses AI these days — and a lot of people use it for things it was
never built to be responsible for.** They spill their secrets. They ask
deeply personal questions. Sometimes those problems have real, local
solutions — a shelter bed, a bus route, a food pantry, a walk-in clinic. AI
will often try to suggest one. The problem: the information it finds is
uncontrolled, often outdated, or simply wrong.

**Answerable fixes that.** It's a lightweight dependency that bolts onto any
stack — built in Go, so it's multiplatform and scales naturally. The binary
is only 15MB.

Answerable makes any website ready for the agent-native web:

- Serves the latest facts from the documents your business already uses — to everyone's AI agents
- Serves MCP so AI can securely and traceably interact with your website (e.g. book an appointment)
- Speaks webhooks and SMTP
- Ships a web UI and CLI, so it fits your IaC scripts
- Does all of this automatically

Imagine you don't have somewhere to sleep tonight, or you can't get to work
and don't know how the bus runs. With Answerable:

- Anyone's AI can find and book a bed
- Anyone's AI can find real-time bus stops and times
- Anyone's AI can tell you exactly what's on the shelf at your local pantry
- Anyone's AI can tell you a health clinic's walk-in availability

Answerable directly answers all five of Mary Sheffield's pillars — a free,
drag-and-drop solution any team can deploy using documents they already
have.

---

Agent-discoverable civic service endpoints. A provider (e.g. a shelter) drops a
CSV/md/txt/pdf or pastes a URL into a local admin page; Answerable republishes
those facts (hours, eligibility, availability) as a live public JSON-LD +
llms.txt + A2A agent-card endpoint, so any agent asking on someone's behalf
gets the provider's real current answer instead of stale scraped data.

If you're adding Answerable to an existing web stack, you'll need to set up
Caddy (see `Dockerfile` / `deploy/`) and the admin UI password if you intend
for that to be public.

Built for Venture 313 Buildathon 2026, category: Open, Accessible & Responsible
Government.

MIT licensed.
