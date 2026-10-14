<div align="center">

# ⚖️ LexScale (律衡)

**Open-Source Legal AI Agent Framework**

*Build enterprise-grade legal AI in minutes, not months.*

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react&logoColor=white)](https://react.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue?style=flat)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-24%20passing-brightgreen?style=flat)]()

[English](README.md) · [中文](README.zh-CN.md)

</div>

---

## What is LexScale?

LexScale is an **open-source Legal AI Agent framework** — think of it as **"LangChain for Legal"**. It provides purpose-built infrastructure for legal AI applications: evidence-tiered reasoning, approval gates, multi-agent orchestration, and audit trails.

### The Problem

Legal AI development today faces critical challenges:

| Challenge | Consequence |
|-----------|-------------|
| AI hallucination in legal advice | Malpractice, lawsuits |
| No evidence tracking | Can't verify AI conclusions |
| No approval workflow | AI sends wrong advice directly to clients |
| Everyone rebuilds the same infra | Wasted engineering effort |

LexScale solves all of these — out of the box.

## 30-Second Quickstart

```bash
git clone https://github.com/tongriyaotxt/lexscale.git
cd lexscale

# Build CLI
go build -o lexscale ./cmd/lexscale-cli/

# Review a contract
./lexscale -f examples/nda.txt
```

### Output

```
⚖️ LexScale — Contract Review Report
═══════════════════════════════════════
📄 File: examples/nda.txt
🔍 Findings: 4 risks identified

🔴 [RED] Excessive liquidated damages
   📎 Civil Code Art. 585
   💡 Damages exceeding actual loss by >30% may be
      adjusted downward by the court...

🟠 [ORANGE] Perpetual confidentiality obligation
   📎 Labor Contract Law Art. 23
   💡 "Perpetual" confidentiality may be deemed
      unconscionable and adjusted by the court...

═══════════════════════════════════════
Summary: 🔴1 🟠2 🟡1 🟢0
```

## Core Innovations

| Feature | What It Does | Why It Matters |
|---------|-------------|----------------|
| **Dual-Graph Model** | Legal Entity Graph + Argumentation Graph | AI sees relationships, not just clauses |
| **Three-Tier Evidence** | FACT (with citation) / INFERENCE (with reasoning) / UNKNOWN (explicitly uncertain) | Kills hallucination at the architecture level |
| **Approval Gates** | `deny` (non-overridable) / `ask` (human review) / `allow` (auto-pass) | AI never sends unsupervised conclusions to clients |
| **Multi-Agent Orchestration** | 6 legal roles: Counsel, Researcher, Analyst, Drafter, Verifier, Assessor | Single-agent systems can't handle legal complexity |
| **Audit Hash Chain** | Blockchain-style tamper-evident audit log | Compliance requirement for legal industry |
| **LLM Routing** | Route different agents to different models (Counsel→GPT-4, Workers→smaller) | Cost optimization without quality loss |

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    Developer Application                  │
│   Contract Review │ Legal Chatbot │ Compliance Check    │
├──────────────────────┬───────────────────────────────────┤
│                      ▼                                    │
│   ┌─────────────────────────────────────────────┐        │
│   │            LexScale Framework                │        │
│   │                                              │        │
│   │  ┌──────────┐  ┌─────────┐  ┌───────────┐  │        │
│   │  │ Dual-Graph│  │ Guard   │  │ Multi-Agent│  │        │
│   │  │ Model     │  │ Gates   │  │ Engine     │  │        │
│   │  └────┬─────┘  └────┬────┘  └─────┬─────┘  │        │
│   │       │              │              │         │        │
│   │  ┌────┴──────────────┴──────────────┴─────┐  │        │
│   │  │     Three-Tier Evidence System          │  │        │
│   │  │  FACT → must have citation              │  │        │
│   │  │  INFERENCE → must have reasoning chain  │  │        │
│   │  │  UNKNOWN → explicitly marked uncertain  │  │        │
│   │  └──────────────────────────────────────────┘  │        │
│   └─────────────────────────────────────────────┘        │
│          │              │              │                   │
├──────────┼──────────────┼──────────────┼───────────────────┤
│          ▼              ▼              ▼                   │
│   ┌──────────┐  ┌──────────┐  ┌─────────────────┐        │
│   │ OpenAI   │  │ Claude   │  │ Kimi/DeepSeek   │        │
│   │ GPT-4    │  │ Sonnet   │  │ (CN models)     │        │
│   └──────────┘  └──────────┘  └─────────────────┘        │
└──────────────────────────────────────────────────────────┘
```

## Multi-Agent Pipeline

```
Counsel Agent (Lead Attorney)
    │
    ├──→ Researcher  (statute retrieval with citations)
    ├──→ Analyst     (clause analysis, risk extraction)
    ├──→ Drafter     (document drafting, alternatives)
    ├──→ Verifier    (argumentation verification)
    └──→ Assessor    (risk assessment: severity × likelihood)
              │
              ▼
    ┌─── Guard Gates ───┐
    │  Every agent       │
    │  output must pass  │
    │  before delivery   │
    └────────────────────┘
```

## Installation

### CLI

```bash
go build -o lexscale ./cmd/lexscale-cli/

# Review with color output
./lexscale -f examples/nda.txt

# JSON output for pipelines
./lexscale -f examples/nda.txt -o json

# English output
./lexscale -f examples/nda.txt -l en
```

### Docker (Full Stack)

```bash
docker compose up -d

# API → http://localhost:8080
# Web  → http://localhost:3000
curl http://localhost:8080/healthz
```

### Go SDK

```go
import "github.com/tongriyaotxt/lexscale"

eng := lexscale.NewEngine(lexscale.Config{
    LLM: lexscale.NewOpenAIProvider(apiKey, "gpt-4o"),
})
```

## Web Dashboard

React frontend with:

- **Dual-graph visualization** (React Flow) — entities, claims, risks with color coding
- **Approval workbench** — pending queue, approve/reject
- **Matter management** — list/detail with tabs (Graph / Findings / Entities)
- **Risk dashboard** — Recharts pie/bar charts, stat cards
- **Real-time SSE** — live graph updates as agents work

## Example Contracts

| File | Type | Risks |
|------|------|-------|
| `examples/nda.txt` | NDA / Confidentiality | 4 |
| `examples/labor_contract.txt` | Labor Contract | 7 |
| `examples/procurement_contract.txt` | Procurement | 3 |

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.23+ |
| Frontend | Next.js 14, React 18, Tailwind CSS, Recharts |
| Database | PostgreSQL 15+ |
| Cache | Redis 7+ |
| Monitoring | Prometheus + Alert Rules |
| Testing | Go test + Playwright E2E |
| CI/CD | GitHub Actions |

## Documentation

| Document | Description |
|----------|-------------|
| [README.zh-CN.md](README.zh-CN.md) | 中文文档 |
| [PROPOSAL.md](PROPOSAL.md) | Full proposal & architecture deep-dive |
| [ROADMAP.md](ROADMAP.md) | Development roadmap |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contribution guide |
| [docs/API.md](docs/API.md) | REST API reference |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | Deployment guide |
| [docs/openapi.yaml](docs/openapi.yaml) | OpenAPI 3.1 spec |

## Project Stats

| Type | Lines |
|------|-------|
| Go | ~2,550 |
| TypeScript/React | ~850 |
| SQL | ~150 |
| Markdown/YAML | ~1,400 |
| **Total** | **~5,000** |

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...
```

## License

[Apache 2.0](LICENSE) — free for commercial use, with patent grant.

---

<div align="center">

**权衡之间，法律智能**

*Where Law Meets Intelligence*

⚖️

</div>
