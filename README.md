# 🚨 Meme AI Analyze

> **The flashlight in the dark forest of Meme tokens.**

Input any Meme token address → get a **six-dimension risk radar** + **AI analysis report** in 15-30 seconds. Know if it's safe before you ape.

---

## ✨ What It Does

- 🔍 **Paste any BSC Meme token address** into the search bar
- ⚡ **15-30 seconds** later, get a complete risk report
- 📊 **Six-dimension radar chart**: Contract Safety, LP Lock, Holder Concentration, Creator History, Liquidity, Social Signals
- 🤖 **AI-powered analysis**: Natural language explanation for non-technical users
- 🔗 **On-chain proof**: Every report is anchored to BSC via the RiskReportOracle contract
- 📋 **Creator tracking**: Full history of every token creator — serial rug pullers are exposed

## 🎯 Why It Matters

99% of Meme tokens are scams. Existing tools only look at one dimension:
- TokenSniffer → only contract code
- Honeypot.is → only honeypot check
- DexScreener → only price/liquidity

**Meme AI Analyze** combines all six dimensions with AI to tell the full story.

## 🏗️ Architecture

```
┌─────────────────────────────────────────────┐
│          Frontend (Next.js 14 + wagmi)        │
│    Search / Report / History / Share Card     │
└──────────────────┬──────────────────────────┘
                   │ HTTP + WebSocket
┌──────────────────┼──────────────────────────┐
│         Backend API (Go + go-zero)            │
│                                                │
│  ┌──────────┐ ┌────────────┐ ┌────────────┐  │
│  │ Query    │ │ Risk Engine│ │ Reporter   │  │
│  │ Service  │ │ • Scanner  │ │ • AI Report│  │
│  │          │ │ • Analyzer │ │ • Oracle   │  │
│  └──────────┘ │ • Historian│ └────────────┘  │
│               │ • Honeypot │                  │
│               └──────┬─────┘                  │
└──────────────────────┼────────────────────────┘
                       │
         ┌─────────────┼─────────────┐
         ▼             ▼             ▼
   ┌──────────┐ ┌──────────┐ ┌────────────┐
   │PostgreSQL│ │  Redis   │ │  BSC Chain │
   └──────────┘ └──────────┘ └────────────┘
```

### Tech Stack

| Layer | Technology |
|-------|-----------|
| **Frontend** | Next.js 14, React, Tailwind CSS, wagmi |
| **Backend** | Go, go-zero framework, goroutines |
| **Database** | PostgreSQL 16 (JSONB for dimensions) |
| **Cache** | Redis 7 |
| **Smart Contract** | Solidity (Foundry), deployed on BSC |
| **AI** | OpenAI GPT-4o / Google Gemini |
| **Infrastructure** | Docker Compose, Railway/Fly.io |

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Node.js 20+ (for local frontend dev)
- Go 1.22+ (for local backend dev)
- Foundry (for smart contract dev)

### One-Command Launch

```bash
# Clone and start everything
git clone https://github.com/your-org/meme-ai-analyze.git
cd meme-ai-analyze
cp .env.example .env   # Edit with your API keys
docker-compose up -d
```

The app will be available at:
- **Frontend**: http://localhost:3000
- **API**: http://localhost:8080
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379

### Local Development

```bash
# Backend
cd apps/api
go mod tidy
go run main.go -f etc/api.yaml

# Frontend
cd web
npm install
npm run dev

# Smart Contracts
cd contracts
forge test
forge script script/Deploy.s.sol --rpc-url bsc_testnet --broadcast
```

## 📁 Project Structure

```
meme_ai_analyze/
├── contracts/                          # Smart Contracts
│   ├── foundry.toml
│   ├── src/RiskReportOracle.sol        # On-chain report anchoring
│   ├── test/RiskReportOracle.t.sol     # 20+ unit tests
│   └── script/Deploy.s.sol             # Deployment script
├── apps/api/                           # Go Backend
│   ├── main.go                         # Entry point
│   ├── etc/api.yaml                    # go-zero config
│   └── internal/
│       ├── config/                     # App configuration
│       ├── svc/                        # DI container
│       ├── handler/                    # HTTP/WS handlers
│       ├── logic/                      # Business logic
│       ├── service/risk/               # Core services
│       │   ├── scanner.go              # Parallel data fetching
│       │   ├── analyzer.go             # 6-dim weighted scoring
│       │   ├── historian.go            # Creator history tracking
│       │   ├── honeypot.go             # Honeypot detection
│       │   ├── reporter.go             # AI report generation
│       │   ├── oracle.go               # On-chain anchoring
│       │   └── cache.go                # Redis caching
│       ├── model/                      # Database models
│       └── types/                      # Request/response types
├── web/                                # Frontend
│   ├── src/app/                        # Pages (Next.js App Router)
│   │   ├── page.tsx                    # Home (search + risk wall)
│   │   ├── scan/[tokenAddress]/        # Report page
│   │   ├── history/                    # Scan history
│   │   └── api/share/                  # OG Image generation
│   ├── src/components/                 # React components
│   │   ├── SearchBar.tsx
│   │   ├── RadarChart.tsx              # SVG 6-dim radar
│   │   ├── ScoreHeader.tsx
│   │   ├── DimensionList.tsx
│   │   ├── AIReportBlock.tsx
│   │   ├── CreatorTimeline.tsx
│   │   ├── RiskWall.tsx
│   │   ├── StatsBar.tsx
│   │   └── ScanProgress.tsx
│   └── src/hooks/                      # Custom hooks
├── db/migrations/                      # PostgreSQL migrations
├── docker-compose.yml                  # One-click deployment
└── DESIGN.md                           # Full technical design doc
```

## 🔬 Six-Dimension Scoring Algorithm

| Dimension | Weight | What It Checks |
|-----------|--------|---------------|
| **Creator History** | 30% | Total tokens deployed, rug rate, avg lifetime, serial rugger detection |
| **LP Lock Status** | 25% | Is liquidity locked? How long? Can the founder withdraw? |
| **Holder Concentration** | 15% | Top 3 / Top 10 holder percentage, whale risk |
| **Contract Safety** | 10% | Honeypot check, mint/blacklist/pause functions, high tax |
| **Liquidity Depth** | 10% | Total liquidity in USD, ability to exit position |
| **Social Signals** | 10% | Website, Twitter age, Telegram members, community authenticity |

**Total Score** = Σ(dimension_score × weight), 1.0 (safe) to 10.0 (danger)

## 🧪 Smart Contract

```solidity
// RiskReportOracle: Anchors report hashes on-chain
// BSC Testnet: [address after deployment]
// BscScan: [link after verification]

function submitReport(address token, bytes32 reportHash, uint8 riskScore, string version)
function recordScan(address token)                          // permissionless
function getScanStats(address token)                        // view scans + reports
```

### Run Tests

```bash
cd contracts
forge test -vvv

# Expected output: 18+ tests passing
# ✓ test_SubmitReport_Success
# ✓ test_SubmitReport_RevertIf_ScoreOutOfRange
# ✓ test_RecordScan_Success
# ✓ test_GetScanStats_ReturnsCorrectData
# ... etc
```

## 📡 API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/scan` | Submit a token for analysis |
| `GET` | `/api/v1/report/:address` | Get cached report |
| `GET` | `/api/v1/creator/:address` | Get creator profile |
| `GET` | `/api/v1/stats` | Platform statistics |
| `GET` | `/api/v1/risk-wall` | Recent high-risk tokens |
| `WS` | `/ws/scan?token=0x...` | Real-time scan progress |

## 📝 Environment Variables

See `.env.example` for all required variables. Key ones:

```bash
# Required
BSC_RPC_URL=           # BSC RPC endpoint
OPENAI_API_KEY=        # OpenAI API key
ORACLE_PRIVATE_KEY=    # Private key for report signing
ORACLE_CONTRACT_ADDRESS= # Deployed oracle contract

# Optional (has defaults)
POSTGRES_HOST=localhost
REDIS_HOST=localhost
```

## 🗺️ Roadmap

- [x] Six-dimension risk scoring engine
- [x] Creator history tracking
- [x] AI-powered natural language reports
- [x] On-chain report anchoring via RiskReportOracle
- [x] Real-time WebSocket progress
- [x] Docker Compose one-click launch
- [ ] Multi-chain support (Ethereum, Arbitrum, Base)
- [ ] Real-time token monitoring with push alerts
- [ ] Community scam evidence submission + verification
- [ ] Paid API tier for DApp integration
- [ ] Fine-tuned AI model based on rug case data

## ⚠️ Disclaimer

This tool is for informational purposes only. Meme tokens are inherently high-risk. Always do your own research (DYOR). Nothing in this analysis constitutes financial advice.

## 📄 License

MIT License

---

> **"Show, don't tell."** — Build something that speaks for itself.
