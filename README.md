# Answerable

[![Read about the commits](https://img.shields.io/badge/commits-code%20blog-1a1a1a?style=flat-square)](https://wwel.sh/digest.html?repo=answerable)

**Everyone uses AI these days — and a lot of people use it for things it was
never built to be responsible for.** They spill their secrets. They ask
deeply personal questions. Sometimes those problems have real, local
solutions — a shelter bed, a bus route, a food pantry, a walk-in clinic. AI
will often try to suggest one. The problem: the information it finds is
uncontrolled, often outdated, or simply wrong. This isn't a hunch — a 2026
MIT Media Lab study found leading chatbots give measurably less accurate,
less truthful answers to users with less formal education or lower English
proficiency, and a 2025 review by the UK's Open Data Institute found AI
models answering government-information questions show "extremely high
variance and unreliability." The people most likely to ask an AI where to
find a shelter bed tonight are exactly the users these models serve worst.

**Answerable fixes that.** It's a lightweight dependency that bolts onto any
stack — built in Go, so it's multiplatform and scales naturally. The binary
is ~16MB stripped.

Answerable makes any website ready for the agent-native web:

- Serves the latest facts from the documents your business already uses — to everyone's AI agents
- Serves MCP so AI can securely and traceably interact with your website (e.g. book an appointment)
- Speaks webhooks and SMTP — so your Slack/Teams/Discord and company email see agent requests coming in real time
- Ships a web UI and CLI, so it fits your IaC scripts
- Does all of this automatically

Because it's a single static binary that bolts onto infrastructure a
provider already runs — the same box or platform already serving their
site — there's no new server to provision or pay for. A shelter, food
pantry, or city department adopting Answerable doesn't take on new hosting
cost or vendor lock-in, so "free" isn't a subsidized loss-leader — it's the
actual marginal cost.

Imagine you don't have somewhere to sleep tonight, or you can't get to work
and don't know how the bus runs. With Answerable:

- Anyone's AI can find and book a bed
- Anyone's AI can find real-time bus stops and times
- Anyone's AI can tell you exactly what's on the shelf at your local pantry
- Anyone's AI can tell you a health clinic's walk-in availability

This isn't hypothetical for Detroit. [CAM Detroit's own Q2 2026
report](https://camdetroit.org/wp-content/uploads/2026/09/2026-Quarter-2-Report.pdf)
shows 75–90% of intake calls (depending on household type) ended with the
caller added to the shelter placement waitlist rather than a same-day bed,
with average waitlist time ranging from about 8 weeks to over 6 months
depending on household type and month. Detroit already publishes some of
this as real-time open data (DDOT's GTFS/realtime feed via
[DDOT.info](https://ddot.info), the city's open data portal) — the gap isn't
that the data doesn't exist, it's that no general-purpose AI agent has a
standard way to find and query it live. Answerable is that layer.

---

Agent-discoverable civic service endpoints. A provider (e.g. a shelter) drops a
CSV/md/txt/pdf or pastes a URL into a local admin page; Answerable republishes
those facts (hours, eligibility, availability) as a live public JSON-LD +
llms.txt + A2A agent-card endpoint, so any agent asking on someone's behalf
gets the provider's real current answer instead of stale scraped data.

**What this doesn't solve (yet): discovery.** Answerable makes a provider's
facts agent-queryable once an agent knows to look — it doesn't solve how a
stranger's AI finds the *right* shelter's endpoint out of many. That's a
directory/discoverability layer, deliberately out of scope for this build;
llms.txt/JSON-LD crawl-and-rank discovery is a real path but not one a few
days can prove out.

If you're adding Answerable to an existing web stack, you'll need to set up
Caddy (see `Dockerfile` / `deploy/`) and the admin UI password if you intend
for that to be public.

Built for Venture 313 Buildathon 2026, targeting the **Open, Accessible &
Responsible Government** pillar — agent-discoverable public info is a
government-transparency problem first. (The shelter/bus/pantry/clinic
examples above touch several of Detroit's other Rise Higher pillars too, but
Government is the one this is built and pitched against.)

Made possible with support from Venture 313's partners: The Gilbert Family
Foundation, TechTown Detroit, Invest Detroit Ventures, and the Detroit
Development Fund.

MIT licensed.
