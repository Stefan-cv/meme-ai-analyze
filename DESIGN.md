# Meme AI Analyze — 完整技术设计文档

> Meme 代币 AI 风险分析平台
> 技术栈：Solidity + Go (go-zero) + PostgreSQL + Redis + AI (OpenAI/Gemini)
> 目标：输入任意 Meme 代币地址 → 15-30 秒生成六维风险雷达图 + AI 分析报告

---

## 一、产品定义

### 一句话

**输入任意 Meme 代币地址 → 15-30 秒后生成一份六维风险雷达图 + AI 分析报告，告诉用户这个币能不能碰。**

### 目标用户

- 每天面对数百个新 Meme 代币的散户投资者
- 不想被骗、但没有技术能力自己审计合约的人
- 想快速了解"这个币的创始人之前 rug 过多少次"的人

### 核心差异

| 现有工具 | 局限 | 我们怎么做 |
|---------|------|-----------|
| TokenSniffer | 只看合约代码，不追踪创建者行为 | 创始人地址全历史追踪 + 归零率分析 |
| Honeypot.is | 只看是否蜜罐 | 六维综合评分 + AI 自然语言解释 |
| DexScreener | 只看 K 线和流动性 | 合约安全 + 持仓集中度 + 创始人背景 |
| 各链浏览器 | 需要技术能力才能读懂 | AI 把链上数据翻译成散户能看懂的语言 |

### 一句话亮点

> "Meme 币赛道每天产生几千个代币，**99% 是骗局**。我们做的不是又一个发射台——我们做的是散户在这片黑暗森林里的**探照灯**。"

---

## 二、用户完整流程

```
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  Step 1: 用户在搜索框粘贴代币地址                           │
│  ┌──────────────────────────────────────────────────┐    │
│  │ 🔍 0x7a25...eF3b                        [分析]  │    │
│  └──────────────────────────────────────────────────┘    │
│                                                          │
│  Step 2: 等待 15-30s（进度条 + 展示正在分析的维度）          │
│  ┌──────────────────────────────────────────────────┐    │
│  │ 🔄 正在扫描...                                    │    │
│  │   ✅ 合约代码分析完成                              │    │
│  │   ✅ LP 池状态检查完成                             │    │
│  │   ⏳ 正在追踪创始人地址历史...                      │    │
│  │   ⏳ 正在分析持仓分布...                           │    │
│  └──────────────────────────────────────────────────┘    │
│                                                          │
│  Step 3: 展示报告                                        │
│  ┌──────────────────────────────────────────────────┐    │
│  │                                                  │    │
│  │  🚨 风险评分：8.2 / 10（高风险）                   │    │
│  │                                                  │    │
│  │  ┌─────────────┐  ┌──────────────────┐           │    │
│  │  │ 六维雷达图   │  │ 各维度得分：      │           │    │
│  │  │   (SVG)     │  │ 合约安全    8/10  │           │    │
│  │  │             │  │ LP锁定      2/10  │           │    │
│  │  │             │  │ 持仓集中   6/10   │           │    │
│  │  │             │  │ 创始人      8/10  │           │    │
│  │  │             │  │ 流动性      4/10  │           │    │
│  │  │             │  │ 社区信号    6/10  │           │    │
│  │  └─────────────┘  └──────────────────┘           │    │
│  │                                                  │    │
│  │  🤖 AI 分析师总结：                               │    │
│  │  ┌──────────────────────────────────────────┐    │    │
│  │  │ 该代币的创建者地址 0xABC... 在 30 天内      │    │    │
│  │  │ 部署了 23 个代币合约，其中 18 个已归零       │    │    │
│  │  │ （平均寿命 4.2 小时）。LP 完全未锁定，       │    │    │
│  │  │ 创始人可随时撤池。强烈建议不要参与。          │    │    │
│  │  └──────────────────────────────────────────┘    │    │
│  │                                                  │    │
│  │  📋 创始人历史：                                  │    │
│  │  ┌────────────────────┬──────┬──────┬──────┐    │    │
│  │  │ 代币                │ 状态 │ 寿命 │ 时间 │    │    │
│  │  ├────────────────────┼──────┼──────┼──────┤    │    │
│  │  │ 0x123... ($SCAM1)  │ 💀归零│ 2.3h │ 3天前│   │    │
│  │  │ 0x456... ($SCAM2)  │ 💀归零│ 6.1h │ 2天前│   │    │
│  │  │ 0x789... ($MOON1)  │ 💀归零│ 1.2h │ 昨天 │   │    │
│  │  │ ...共 23 个代币     │      │      │      │    │    │
│  │  └────────────────────┴──────┴──────┴──────┘    │    │
│  │                                                  │    │
│  │  [分享报告]  [查看链上证明]  [订阅该代币监控]       │    │
│  └──────────────────────────────────────────────────┘    │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

---

## 三、技术架构

```
┌─────────────────────────────────────────────────────────┐
│                  前端 (Next.js 14 + wagmi)                │
│          搜索页 / 报告展示页 / 历史查询 / 分享卡片          │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP + WebSocket（进度推送）
┌──────────────────────┼──────────────────────────────────┐
│                后端 API 服务 (go-zero)                     │
│                                                          │
│  ┌────────────┐  ┌─────────────────┐  ┌──────────────┐  │
│  │ 查询服务    │  │ 风险分析引擎     │  │ 报告服务      │  │
│  │ /api/scan  │  │ scanner.go      │  │ reporter.go  │  │
│  │            │  │ analyzer.go     │  │ oracle.go    │  │
│  │ • 接收地址  │  │ historian.go    │  │              │  │
│  │ • 查缓存   │  │ honeypot.go     │  │ • AI 生成报告 │  │
│  │ • 触发分析 │  │                 │  │ • 上链存哈希 │  │
│  └────────────┘  └────────┬────────┘  └──────┬───────┘  │
│                           │                   │          │
│                    ┌──────┴──────┐     ┌──────┴───────┐  │
│                    │ Indexer      │     │ AI Client    │  │
│                    │ • 新代币发现  │     │ GPT-4o/      │  │
│                    │ • 历史事件   │     │ Gemini       │  │
│                    │ • 创始人库   │     │              │  │
│                    └──────┬──────┘     └──────────────┘  │
└───────────────────────────┼──────────────────────────────┘
                            │
              ┌─────────────┼─────────────┐
              ▼             ▼             ▼
        ┌──────────┐ ┌──────────┐ ┌──────────────┐
        │PostgreSQL │ │  Redis   │ │   BSC 链     │
        │ • tokens  │ │ • 查询缓存│ │ • RiskOracle │
        │ • reports │ │ • 限流   │ │   合约        │
        │ • creators│ │ • 进度   │ │ • 事件查询   │
        └──────────┘ └──────────┘ └──────────────┘
```

### 技术选型理由

| 层级 | 技术 | 选择理由 |
|------|------|---------|
| 前端 | Next.js 14 + wagmi | React 生态成熟，wagmi 原生支持钱包/合约交互，App Router 支持 SSR/OG Image |
| 后端 | Go (go-zero) | 高性能、并发原生支持（goroutine），适合链上数据并行抓取 |
| 数据库 | PostgreSQL | JSONB 支持灵活存储维度数据，ACID 保证报告完整性 |
| 缓存 | Redis | 查询缓存 + 限流 + 进度状态存储 |
| 合约 | Solidity (Foundry) | BSC 兼容 EVM，Foundry 测试框架高效 |
| AI | OpenAI GPT-4o / Gemini | 多模态能力强，结构化数据 → 自然语言效果好 |
| 部署 | Railway / Fly.io | 免费层即可启动，支持 Docker 一键部署 |

---

## 四、智能合约层

### 4.1 合约设计

合约极简——只做一件事：**存储风险报告的链上证据**。

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "@openzeppelin/contracts/access/AccessControl.sol";

/// @title RiskReportOracle
/// @notice 将 Meme 代币风险报告的哈希锚定到链上，提供不可篡改的审计轨迹
contract RiskReportOracle is AccessControl {

    bytes32 public constant ORACLE_ROLE = keccak256("ORACLE_ROLE");

    struct ReportRecord {
        bytes32 reportHash;   // 报告全文（存 Arweave/IPFS）的 keccak256
        uint8   riskScore;    // 1-10，越高越危险
        uint256 timestamp;    // 区块时间戳
        string  version;      // 分析引擎版本号（如 "v1.2.0"）
    }

    // tokenAddress => 最新报告
    mapping(address => ReportRecord) public latestReport;

    // tokenAddress => 历史报告数量
    mapping(address => uint256) public reportCount;

    // tokenAddress => 总查询次数
    mapping(address => uint256) public scanCount;

    // ========== Events ==========

    event ReportSubmitted(
        address indexed token,
        bytes32 indexed reportHash,
        uint8 riskScore,
        uint256 timestamp,
        uint256 totalReports
    );

    event TokenScanned(
        address indexed token,
        address indexed scannedBy,
        uint256 totalScans
    );

    // ========== Write ==========

    /// @notice 提交一份风险报告（仅 ORACLE_ROLE）
    function submitReport(
        address token,
        bytes32 reportHash,
        uint8 riskScore,
        string calldata version
    ) external onlyRole(ORACLE_ROLE) {
        require(token != address(0), "Invalid token");
        require(reportHash != bytes32(0), "Empty hash");
        require(riskScore >= 1 && riskScore <= 10, "Invalid score");

        latestReport[token] = ReportRecord({
            reportHash: reportHash,
            riskScore: riskScore,
            timestamp: block.timestamp,
            version: version
        });
        reportCount[token]++;

        emit ReportSubmitted(token, reportHash, riskScore, block.timestamp, reportCount[token]);
    }

    /// @notice 记录一次扫描（任何人可调用，前端查询时自动触发）
    function recordScan(address token) external {
        require(token != address(0), "Invalid token");
        scanCount[token]++;
        emit TokenScanned(token, msg.sender, scanCount[token]);
    }

    // ========== Read ==========

    function getLatestReport(address token) external view returns (ReportRecord memory) {
        return latestReport[token];
    }

    function getReportCount(address token) external view returns (uint256) {
        return reportCount[token];
    }

    function getScanStats(address token) external view
        returns (uint256 scans, uint256 reports, ReportRecord memory lastReport)
    {
        return (scanCount[token], reportCount[token], latestReport[token]);
    }
}
```

### 4.2 合约要点说明

| 设计决策 | 原因 |
|---------|------|
| 只存哈希不存全文 | Gas 成本。报告全文存后端/Arweave |
| `ORACLE_ROLE` 权限 | 只有后端可写，防止伪造报告 |
| `recordScan` 无权限 | 公开可调，记录热度/使用量 |
| `version` 字段 | 展示版本演进，便于迭代升级 |

---

## 五、后端设计

### 5.1 Go 项目结构

```
apps/api/internal/
├── config/
│   └── config.go                    # 配置管理（链 RPC、数据库、AI API Key）
├── svc/
│   └── servicecontext.go            # DI 容器（依赖注入）
├── handler/
│   ├── routes.go                    # 路由注册
│   └── risk/
│       └── handlers.go              # HTTP Handler（请求解析/响应封装/WebSocket）
├── logic/
│   └── risk/
│       └── scanlogic.go             # 扫描主流程编排（缓存→扫描→分析→存储→返回）
├── service/
│   └── risk/
│       ├── scanner.go               # 链上数据抓取（6 goroutine 并行）
│       ├── analyzer.go              # 六维评分引擎
│       ├── historian.go             # 创始人历史追踪
│       ├── honeypot.go              # 蜜罐检测
│       ├── reporter.go              # AI 报告生成
│       ├── oracle.go                # 链上报告锚定
│       └── cache.go                 # 查询缓存管理
├── model/
│   ├── token.go                     # 代币信息表模型
│   ├── report.go                    # 报告记录表模型
│   └── creator.go                   # 创始人档案表模型
└── types/
    └── types.go                     # 请求/响应类型定义
```

### 5.2 核心数据结构

```go
// types/types.go

// ScanRequest 查询请求
type ScanRequest struct {
    TokenAddress string `json:"token_address"` // 代币地址
    ChainID      int    `json:"chain_id"`      // 链 ID（默认 BSC=56）
}

// ScanResponse 查询响应（核心输出）
type ScanResponse struct {
    TokenAddress   string           `json:"token_address"`
    TokenName      string           `json:"token_name"`
    TokenSymbol    string           `json:"token_symbol"`
    RiskScore      float64          `json:"risk_score"`      // 1-10，综合分
    RiskLevel      string           `json:"risk_level"`      // "低风险"/"中风险"/"高风险"/"极度危险"
    Dimensions     [6]DimensionScore `json:"dimensions"`     // 六维评分
    AIReport       string           `json:"ai_report"`       // AI 自然语言分析
    CreatorHistory *CreatorProfile  `json:"creator_history"` // 创始人历史
    ReportHash     string           `json:"report_hash"`     // IPFS/Arweave 哈希
    TxHash         string           `json:"tx_hash"`         // 链上锚定 TX
    ScannedAt      int64            `json:"scanned_at"`      // Unix timestamp
}

type DimensionScore struct {
    Name    string  `json:"name"`     // "合约安全"
    Score   float64 `json:"score"`    // 1-10
    Weight  float64 `json:"weight"`   // 权重
    Details string  `json:"details"`  // 一句话说明
    Status  string  `json:"status"`   // "good"/"warning"/"danger"
}

type CreatorProfile struct {
    Address        string       `json:"address"`
    TotalTokens    int          `json:"total_tokens"`      // 创建过多少代币
    ActiveTokens   int          `json:"active_tokens"`     // 存活的
    RuggedTokens   int          `json:"rugged_tokens"`     // 归零的
    RugRate        float64      `json:"rug_rate"`          // 归零率 0-1
    AvgLifetimeHrs float64      `json:"avg_lifetime_hrs"`  // 平均存活时间
    FirstSeenDays  int          `json:"first_seen_days"`   // 首次出现距今
    RiskLabel      string       `json:"risk_label"`        // 风险标签
    RecentTokens   []TokenBrief `json:"recent_tokens"`     // 最近创建的代币
}

type TokenBrief struct {
    Address   string  `json:"address"`
    Name      string  `json:"name"`
    Symbol    string  `json:"symbol"`
    Status    string  `json:"status"`   // "active"/"rugged"/"unknown"
    Lifetime  float64 `json:"lifetime_hrs"`
    CreatedAt int64   `json:"created_at"`
}
```

### 5.3 扫描器（scanner.go）——并行数据抓取

```go
// scanner.go
// 核心职责：从链上并行抓取代币的所有相关数据

package risk

import (
    "context"
    "sync"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/ethclient"
)

type Scanner struct {
    client    *ethclient.Client  // 链 RPC 客户端
    redis     *redis.Client      // 缓存
    historian *Historian         // 创始人追踪
}

// TokenRiskProfile 代币完整风险画像（所有维度原始数据）
type TokenRiskProfile struct {
    Address            string
    ContractSafety     *ContractSafetyData
    LPStatus           *LPStatusData
    HolderConcentration *HolderData
    CreatorProfile     *CreatorProfile
    Liquidity          *LiquidityData
    SocialSignals      *SocialSignalData
}

// Scan 执行一次完整扫描（6 个 goroutine 并行）
func (s *Scanner) Scan(ctx context.Context, tokenAddr string) (*TokenRiskProfile, error) {
    addr := common.HexToAddress(tokenAddr)

    profile := &TokenRiskProfile{Address: tokenAddr}

    var wg sync.WaitGroup
    errCh := make(chan error, 6)

    // 维度 1：合约安全
    wg.Add(1)
    go func() {
        defer wg.Done()
        if data, err := s.analyzeContractSafety(ctx, addr); err != nil {
            errCh <- fmt.Errorf("contract_safety: %w", err)
        } else {
            profile.ContractSafety = data
        }
    }()

    // 维度 2：LP 锁定状态
    wg.Add(1)
    go func() {
        defer wg.Done()
        if data, err := s.analyzeLPStatus(ctx, addr); err != nil {
            errCh <- fmt.Errorf("lp_status: %w", err)
        } else {
            profile.LPStatus = data
        }
    }()

    // 维度 3：持仓集中度
    wg.Add(1)
    go func() {
        defer wg.Done()
        if data, err := s.analyzeHolderConcentration(ctx, addr); err != nil {
            errCh <- fmt.Errorf("holder_concentration: %w", err)
        } else {
            profile.HolderConcentration = data
        }
    }()

    // 维度 4：创始人历史（最核心）
    wg.Add(1)
    go func() {
        defer wg.Done()
        if data, err := s.historian.TraceCreator(ctx, addr); err != nil {
            errCh <- fmt.Errorf("creator_history: %w", err)
        } else {
            profile.CreatorProfile = data
        }
    }()

    // 维度 5：流动性分析
    wg.Add(1)
    go func() {
        defer wg.Done()
        if data, err := s.analyzeLiquidity(ctx, addr); err != nil {
            errCh <- fmt.Errorf("liquidity: %w", err)
        } else {
            profile.Liquidity = data
        }
    }()

    // 维度 6：社区信号
    wg.Add(1)
    go func() {
        defer wg.Done()
        if data, err := s.analyzeSocialSignals(ctx, addr); err != nil {
            errCh <- fmt.Errorf("social_signals: %w", err)
        } else {
            profile.SocialSignals = data
        }
    }()

    wg.Wait()
    close(errCh)

    // 收集错误（非致命错误不阻断流程，但记录日志）
    for err := range errCh {
        log.Printf("[Scanner] non-fatal error: %v", err)
    }

    return profile, nil
}
```

### 5.4 创始人追踪器（historian.go）——核心差异化模块

```go
// historian.go
// 这是本项目的"灵魂模块"——全网唯一追踪 Meme 币创始人行为历史的引擎

type Historian struct {
    db      *pgxpool.Pool
    indexer *Indexer // 链上事件索引器
}

// CreatorRecord 数据库中的创始人档案
type CreatorRecord struct {
    Address          string    `db:"address"`
    TotalTokens      int       `db:"total_tokens"`
    RuggedTokens     int       `db:"rugged_tokens"`
    AvgLifetimeHours float64   `db:"avg_lifetime_hours"`
    FirstSeenAt      time.Time `db:"first_seen_at"`
    LastSeenAt       time.Time `db:"last_seen_at"`
    RiskLabel        string    `db:"risk_label"` // "clean"/"suspicious"/"serial_rugger"
}

// TraceCreator 追踪创始人
func (h *Historian) TraceCreator(ctx context.Context, tokenAddr common.Address) (*CreatorProfile, error) {
    // 1. 从链上获取该代币的 creator（取 deploy 交易 from 字段）
    creator, err := h.getDeployer(ctx, tokenAddr)
    if err != nil {
        return nil, fmt.Errorf("get deployer: %w", err)
    }

    // 2. 查数据库中的创始人档案（由 Indexer 持续维护）
    record, err := h.getCreatorRecord(ctx, creator)
    if err != nil {
        // 新地址，无历史记录
        return &CreatorProfile{
            Address:     creator.Hex(),
            TotalTokens: 0,
            RiskLabel:   "✅ 新地址，无不良记录",
        }, nil
    }

    // 3. 生成风险标签
    rugRate := float64(record.RuggedTokens) / float64(record.TotalTokens)
    riskLabel := h.classifyCreator(record.TotalTokens, rugRate)

    return &CreatorProfile{
        Address:        creator.Hex(),
        TotalTokens:    record.TotalTokens,
        RuggedTokens:   record.RuggedTokens,
        RugRate:        rugRate,
        AvgLifetimeHrs: record.AvgLifetimeHours,
        FirstSeenDays:  int(time.Since(record.FirstSeenAt).Hours() / 24),
        RiskLabel:      riskLabel,
        RecentTokens:   h.recentTokens(ctx, creator),
    }, nil
}

// classifyCreator 创始人风险分类
func (h *Historian) classifyCreator(totalTokens int, rugRate float64) string {
    switch {
    case totalTokens >= 10 && rugRate >= 0.7:
        return fmt.Sprintf("⚠️ 惯犯：该地址创建了大量已归零代币（归零率 %.0f%%）", rugRate*100)
    case rugRate >= 0.5:
        return "⚠️ 可疑：超过半数创建的代币已归零"
    case totalTokens >= 5:
        return "ℹ️ 该地址较活跃，但暂未发现明显风险模式"
    default:
        return "✅ 新地址，无不良记录"
    }
}

// getDeployer 获取代币合约的部署者地址
func (h *Historian) getDeployer(ctx context.Context, tokenAddr common.Address) (common.Address, error) {
    // 方案 A：从合约创建交易的 receipt 中获取 from 地址
    // 需要先找到创建该合约的交易哈希（通过区块浏览器 API 或归档节点查询）
    
    // 方案 B：如果合约实现了 owner() 方法，调用 owner()
    // 注意：owner 不一定是 deployer（可能已转移所有权）
    
    // 这里以方案 A 为例
    txHash, err := h.indexer.FindContractCreationTx(ctx, tokenAddr)
    if err != nil {
        return common.Address{}, err
    }
    
    tx, _, err := h.indexer.client.TransactionByHash(ctx, txHash)
    if err != nil {
        return common.Address{}, err
    }
    
    from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
    if err != nil {
        return common.Address{}, err
    }
    
    return from, nil
}
```

### 5.5 Indexer 层——创始人数据库的建设

```go
// 在现有 Indexer 架构上增加 Creator Tracker

// watchNewTokens 协程：
// 监听链上新部署的代币合约事件
// 每发现一个新代币 → 记录 deployer 地址 → 更新创始人数据库

func (idx *Indexer) trackNewTokens(ctx context.Context, log types.Log) error {
    // 解析合约创建事件
    deployer := getDeployerFromReceipt(ctx, idx.client, log.TxHash)
    tokenAddr := log.Address

    // 获取代币基本信息
    tokenName, tokenSymbol := idx.getTokenInfo(ctx, tokenAddr)

    // UPSERT 创始人记录
    _, err := idx.db.Exec(ctx, `
        INSERT INTO creator_records (address, total_tokens, first_seen_at, last_seen_at)
        VALUES ($1, 1, NOW(), NOW())
        ON CONFLICT (address) DO UPDATE SET
            total_tokens = creator_records.total_tokens + 1,
            last_seen_at = NOW()
    `, deployer.Hex())
    if err != nil {
        return fmt.Errorf("upsert creator: %w", err)
    }

    // 插入代币记录
    _, err = idx.db.Exec(ctx, `
        INSERT INTO creator_tokens (creator_address, token_address, token_name, token_symbol, created_at)
        VALUES ($1, $2, $3, $4, NOW())
    `, deployer.Hex(), tokenAddr.Hex(), tokenName, tokenSymbol)

    return err
}

// 定期更新归零状态（cron job，每 5 分钟）
func (idx *Indexer) updateRugStatus(ctx context.Context) error {
    // 检查所有 active 代币的当前流动性
    // 如果流动性归零 → 标记为 rugged → 更新 creator_records.rugged_tokens
    _, err := idx.db.Exec(ctx, `
        UPDATE creator_records cr
        SET rugged_tokens = (
            SELECT COUNT(*) FROM creator_tokens ct
            WHERE ct.creator_address = cr.address AND ct.status = 'rugged'
        ),
        avg_lifetime_hours = (
            SELECT COALESCE(AVG(
                EXTRACT(EPOCH FROM (COALESCE(rugged_at, NOW()) - created_at)) / 3600
            ), 0)
            FROM creator_tokens ct
            WHERE ct.creator_address = cr.address AND ct.status = 'rugged'
        )
    `)
    return err
}
```

### 5.6 AI 分析层（reporter.go）

```go
// reporter.go
// 把结构化数据翻译成散户能看懂的 AI 分析

type Reporter struct {
    aiClient *AIClient // OpenAI / Gemini
}

func (r *Reporter) Generate(ctx context.Context, profile *TokenRiskProfile) (string, error) {
    systemPrompt := `你是一位专业的链上安全分析师，擅长把复杂的链上数据翻译成散户能听懂的话。

分析规则：
1. 先看创始人历史——如果 rug 率 > 50%，这是最危险的信号，必须警告
2. 再看 LP 锁定——未锁定的池子创始人随时可以跑路
3. 再看合约安全——有 mint/blacklist/pause 函数的代币风险更高
4. 最后看持仓——前 3 地址持有 > 50% 说明高度集中

输出要求：
- 150-200 字
- 第一句给结论（"该代币风险较高，建议谨慎" 或 "暂未发现明显风险信号"）
- 用 2-3 条具体数据支撑结论
- 如果创始人历史可疑，重点强调
- 语言直白。使用"建议不要参与"，而不是"存在较高的投资风险"
- 如果代币一切正常，最后加上"Meme 代币天然高风险，请自行判断"

不要输出 markdown 格式。不要用 bullet point。用自然段落。`

    dataJSON := r.formatProfileForAI(profile)
    response, err := r.aiClient.Chat(ctx, systemPrompt, dataJSON)
    if err != nil {
        return "", fmt.Errorf("AI analysis failed: %w", err)
    }

    return response, nil
}

// formatProfileForAI 将风险画像转为 AI 友好的 JSON
func (r *Reporter) formatProfileForAI(profile *TokenRiskProfile) string {
    data := map[string]interface{}{
        "token_name":     profile.Address,
        "contract_safety": map[string]interface{}{
            "is_honeypot":       profile.ContractSafety.IsHoneypot,
            "has_mint_function": profile.ContractSafety.HasMint,
            "has_blacklist":     profile.ContractSafety.HasBlacklist,
            "has_pause":         profile.ContractSafety.HasPause,
            "owner_privileges":  profile.ContractSafety.OwnerPrivileges,
        },
        "lp_status": map[string]interface{}{
            "is_locked":       profile.LPStatus.Locked,
            "lock_remaining":  profile.LPStatus.LockRemaining,
            "lp_token_amount": profile.LPStatus.TokenAmount,
        },
        "holder_concentration": map[string]interface{}{
            "top3_percentage":   profile.HolderConcentration.Top3Percent,
            "top10_percentage":  profile.HolderConcentration.Top10Percent,
            "total_holders":     profile.HolderConcentration.TotalHolders,
        },
        "creator": map[string]interface{}{
            "total_tokens":     profile.CreatorProfile.TotalTokens,
            "rugged_tokens":    profile.CreatorProfile.RuggedTokens,
            "rug_rate":         profile.CreatorProfile.RugRate,
            "avg_lifetime_hrs": profile.CreatorProfile.AvgLifetimeHrs,
            "first_seen_days":  profile.CreatorProfile.FirstSeenDays,
        },
    }
    b, _ := json.Marshal(data)
    return string(b)
}
```

### 5.7 缓存策略（cache.go）

```go
// cache.go
// 智能缓存策略：
//   - 低风险（score 1-3）代币：缓存 12 小时（变化不大）
//   - 中风险（score 4-6）代币：缓存 1 小时
//   - 高风险（score 7-10）代币：缓存 10 分钟（可能随时 rug）
//   - Redis key: "risk:report:{chain_id}:{token_address}"
//   - 同时存 ETag，支持 304 Not Modified

func (c *Cache) GetOrScan(ctx context.Context, tokenAddr string) (*ScanResponse, bool) {
    key := fmt.Sprintf("risk:report:56:%s", tokenAddr) // 56 = BSC chain ID

    cached, err := c.redis.Get(ctx, key).Result()
    if err != nil {
        return nil, false // 缓存未命中
    }

    var resp ScanResponse
    if err := json.Unmarshal([]byte(cached), &resp); err != nil {
        return nil, false
    }

    // 根据风险等级判断是否过期
    ttl := c.calcExpiry(resp.RiskScore)
    if time.Now().Unix() > resp.ScannedAt+ttl {
        return nil, false // 过期
    }

    return &resp, true
}

func (c *Cache) calcExpiry(score float64) int64 {
    switch {
    case score <= 3:
        return 12 * 3600  // 12 hours
    case score <= 6:
        return 3600       // 1 hour
    default:
        return 600        // 10 minutes（高风险代币随时可能变化）
    }
}

// Set 缓存扫描结果
func (c *Cache) Set(ctx context.Context, resp *ScanResponse) error {
    key := fmt.Sprintf("risk:report:56:%s", resp.TokenAddress)
    ttl := time.Duration(c.calcExpiry(resp.RiskScore)) * time.Second
    
    data, err := json.Marshal(resp)
    if err != nil {
        return err
    }
    
    return c.redis.Set(ctx, key, data, ttl).Err()
}
```

### 5.8 PostgreSQL 数据库 Schema

```sql
-- ============================================================
-- Meme AI Analyze - 数据库 Schema
-- ============================================================

-- 代币风险报告表
CREATE TABLE risk_reports (
    id              BIGSERIAL PRIMARY KEY,
    chain_id        INT NOT NULL DEFAULT 56,          -- 链 ID（56=BSC, 1=Ethereum）
    token_address   VARCHAR(42) NOT NULL,
    token_name      VARCHAR(128),
    token_symbol    VARCHAR(32),
    risk_score      DECIMAL(3,1) NOT NULL,            -- 1.0-10.0
    risk_level      VARCHAR(16) NOT NULL,             -- low/medium/high/critical
    dimension_json  JSONB NOT NULL,                   -- 六维评分详情
    ai_report       TEXT,                             -- AI 生成的分析报告
    report_hash     VARCHAR(128),                     -- IPFS/Arweave 哈希
    oracle_tx_hash  VARCHAR(66),                      -- 链上锚定交易哈希
    creator_address VARCHAR(42),
    scanned_at      BIGINT NOT NULL,
    created_at      BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())*1000)::BIGINT,
    UNIQUE(chain_id, token_address)
);

CREATE INDEX idx_reports_score ON risk_reports(chain_id, risk_score);
CREATE INDEX idx_reports_creator ON risk_reports(creator_address);
CREATE INDEX idx_reports_scanned ON risk_reports(scanned_at DESC);

-- 创始人档案表
CREATE TABLE creator_records (
    address             VARCHAR(42) PRIMARY KEY,
    total_tokens        INT NOT NULL DEFAULT 0,
    rugged_tokens       INT NOT NULL DEFAULT 0,
    avg_lifetime_hours  DECIMAL(8,2) DEFAULT 0,
    first_seen_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    risk_label          VARCHAR(32) DEFAULT 'new'    -- clean/suspicious/serial_rugger/new
);

CREATE INDEX idx_creator_label ON creator_records(risk_label);
CREATE INDEX idx_creator_tokens ON creator_records(total_tokens DESC);

-- 创始人的代币历史
CREATE TABLE creator_tokens (
    id               BIGSERIAL PRIMARY KEY,
    creator_address  VARCHAR(42) NOT NULL REFERENCES creator_records(address),
    token_address    VARCHAR(42) NOT NULL,
    token_name       VARCHAR(128),
    token_symbol     VARCHAR(32),
    status           VARCHAR(16) DEFAULT 'active',   -- active/rugged/unknown
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rugged_at        TIMESTAMPTZ,                     -- 归零时间
    UNIQUE(token_address)
);

CREATE INDEX idx_ct_creator ON creator_tokens(creator_address, status);
CREATE INDEX idx_ct_created ON creator_tokens(creator_address, created_at DESC);

-- 扫描日志（用于统计热度）
CREATE TABLE scan_logs (
    id             BIGSERIAL PRIMARY KEY,
    token_address  VARCHAR(42) NOT NULL,
    scanned_by     VARCHAR(42),                      -- 查询者钱包地址（可为空）
    risk_score     DECIMAL(3,1),
    scanned_at     BIGINT NOT NULL
);

CREATE INDEX idx_scan_token ON scan_logs(token_address, scanned_at DESC);
CREATE INDEX idx_scan_time ON scan_logs(scanned_at DESC);

-- 链上事件监听状态（Indexer 断点续传）
CREATE TABLE indexer_state (
    chain_id        INT PRIMARY KEY,
    last_block      BIGINT NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 六、前端设计

### 6.1 页面结构

```
/                         # 首页（搜索框 + 热门查询 + 高风险警示墙）
/scan/[tokenAddress]      # 报告详情页（六维雷达图 + AI 分析 + 创始人历史）
/history                  # 最近扫描记录
/api/share/[tokenAddress] # OG Image 生成（分享到社交媒体的卡片图）
```

### 6.2 搜索页组件树

```
HomePage
├── HeroSection
│   ├── SearchBar（主搜索框 + [分析] 按钮 + 链选择器）
│   └── 热门代币标签云（可点击快速查询）
├── RiskWall（高风险警示墙——实时展示最近 10 个 7+ 分的代币）
│   └── RiskCard[]（代币名 / 分数 / 一句话警告）
└── StatsBar（已分析 X 个代币 / 发现 Y 个高风险 / Z 个 rug 创始人）

ReportPage
├── ScoreHeader（风险评分大数字 + 等级标签颜色）
├── RadarChart（SVG/CSS 六维雷达图，彩色区域）
├── DimensionList（六个维度分别展开详情）
├── AIReportBlock（AI 分析文本 + "此分析由 AI 生成" 免责声明）
├── CreatorTimeline（创始人历史时间轴，代币卡片列表）
├── ActionButtons（分享到 Twitter / 查看链上 TX / 重新分析 / 订阅监控）
└── Disclaimer（风险提示和法律免责）
```

### 6.3 WebSocket 进度推送

```go
// 后端 WebSocket 推送扫描进度（handler 层）
func (h *Handler) ScanWS(w http.ResponseWriter, r *http.Request) {
    // 升级为 WebSocket
    conn, _ := upgrader.Upgrade(w, r, nil)
    defer conn.Close()

    // 从 URL 参数中获取代币地址
    tokenAddr := r.URL.Query().Get("token")

    steps := []string{
        "正在读取合约代码...",
        "正在检查 LP 池锁定状态...",
        "正在分析持仓分布...",
        "正在追踪创始人地址历史...",
        "正在评估流动性深度...",
        "正在抓取社区信号...",
        "AI 正在分析生成报告...",
        "正在将报告锚定到链上...",
    }

    for i, step := range steps {
        progress := float64(i+1) / float64(len(steps)) * 100
        conn.WriteJSON(map[string]interface{}{
            "step":     step,
            "progress": progress,
            "status":   "scanning",
        })
        time.Sleep(200 * time.Millisecond) // 给前端渲染时间
    }

    // 执行实际扫描
    result, err := h.logic.Scan(r.Context(), tokenAddr)
    if err != nil {
        conn.WriteJSON(map[string]interface{}{
            "status": "error",
            "error":  err.Error(),
        })
        return
    }

    // 推送完整结果
    conn.WriteJSON(map[string]interface{}{
        "status": "completed",
        "data":   result,
    })
}
```

### 6.4 技术选型

| 功能 | 技术方案 |
|------|---------|
| 框架 | Next.js 14 (App Router) |
| Web3 交互 | wagmi + viem |
| 雷达图 | 纯 SVG（轻量、可控、适合分享卡片渲染） |
| 进度推送 | WebSocket（实时展示分析步骤） |
| OG Image | `@vercel/og`（服务端渲染分享卡片） |
| 状态管理 | React Query（服务端状态）+ Zustand（客户端状态） |
| 样式 | Tailwind CSS |
| 部署 | Vercel（免费层） |

---

## 七、六维评分算法（analyzer.go）

```go
// analyzer.go
// 加权综合评分算法

const (
    WeightContract  = 0.10  // 合约安全
    WeightLP        = 0.25  // LP 锁定（最重要——没锁就是跑路风险）
    WeightHolder    = 0.15  // 持仓集中度
    WeightCreator   = 0.30  // 创始人历史（最最重要——rug 惯犯直接拉黑）
    WeightLiquidity = 0.10  // 流动性
    WeightSocial    = 0.10  // 社区信号
)

// CalculateScore 总评分 = Σ(各维度得分 × 权重)
func (a *Analyzer) CalculateScore(profile *TokenRiskProfile) float64 {
    scores := []struct {
        score  float64
        weight float64
    }{
        {a.scoreContract(profile.ContractSafety), WeightContract},
        {a.scoreLP(profile.LPStatus), WeightLP},
        {a.scoreHolder(profile.HolderConcentration), WeightHolder},
        {a.scoreCreator(profile.CreatorProfile), WeightCreator},
        {a.scoreLiquidity(profile.Liquidity), WeightLiquidity},
        {a.scoreSocial(profile.SocialSignals), WeightSocial},
    }

    var total float64
    for _, s := range scores {
        total += s.score * s.weight
    }
    return math.Round(total*10) / 10 // 保留 1 位小数
}

// 创始人评分（核心算法）
func (a *Analyzer) scoreCreator(profile *CreatorProfile) float64 {
    // 新地址：不算危险，给低分
    if profile.TotalTokens == 0 {
        return 2.0
    }

    rugRate := profile.RugRate

    switch {
    case profile.TotalTokens >= 10 && rugRate >= 0.7:
        return 10.0 // 惯犯
    case rugRate >= 0.5:
        return 8.0  // 高危
    case profile.TotalTokens >= 5:
        return 5.0  // 存疑
    case profile.TotalTokens >= 2:
        return 3.0  // 轻度关注
    default:
        return 1.5  // 基本安全
    }
}

// LP 锁定评分
func (a *Analyzer) scoreLP(lp *LPStatusData) float64 {
    if !lp.Found {
        return 10.0 // 连 LP 池都没有 —— 极度危险
    }
    if !lp.Locked {
        return 9.0  // 有 LP 但未锁定 —— 随时跑路
    }
    if lp.LockRemaining < 24*3600 {
        return 7.0  // 即将解锁（< 1 天）
    }
    if lp.LockRemaining < 7*24*3600 {
        return 4.0  // 一周内解锁
    }
    return 1.0 // 长期锁定，安全
}

// 合约安全评分
func (a *Analyzer) scoreContract(safety *ContractSafetyData) float64 {
    score := 0.0
    if safety.IsHoneypot {
        score += 10.0 // 蜜罐直接满分危险
    }
    if safety.HasMint {
        score += 3.0  // 可增发
    }
    if safety.HasBlacklist {
        score += 2.0  // 黑名单功能
    }
    if safety.HasPause {
        score += 2.0  // 可暂停交易
    }
    if safety.HasHighTax {
        score += 2.0  // 高交易税
    }
    return math.Min(score, 10.0)
}

// 持仓集中度评分
func (a *Analyzer) scoreHolder(h *HolderData) float64 {
    if h.TotalHolders == 0 {
        return 10.0
    }
    if h.Top3Percent >= 80 {
        return 10.0
    }
    if h.Top3Percent >= 50 {
        return 7.0
    }
    if h.Top10Percent >= 80 {
        return 5.0
    }
    return 2.0
}

// 流动性评分
func (a *Analyzer) scoreLiquidity(l *LiquidityData) float64 {
    if l.TotalLiquidityUSD < 1000 {
        return 9.0  // 流动性极低
    }
    if l.TotalLiquidityUSD < 10000 {
        return 6.0
    }
    if l.TotalLiquidityUSD < 100000 {
        return 3.0
    }
    return 1.0  // 流动性充足
}

// 社区信号评分
func (a *Analyzer) scoreSocial(s *SocialSignalData) float64 {
    score := 5.0 // 默认中等
    if s.HasWebsite {
        score -= 1.0
    }
    if s.TwitterAgeDays > 30 {
        score -= 1.5
    }
    if s.TelegramMembers > 1000 {
        score -= 1.0
    }
    return math.Max(score, 1.0)
}

// getRiskLevel 根据分数映射风险等级
func (a *Analyzer) getRiskLevel(score float64) string {
    switch {
    case score <= 3.0:
        return "低风险"
    case score <= 5.0:
        return "中风险"
    case score <= 7.0:
        return "高风险"
    default:
        return "极度危险"
    }
}
```

### 权重设计理念

```
创始人历史 ████████████████████████████████ 30%  ← 最核心差异
LP 锁定   █████████████████████████ 25%        ← 跑路直接指标
持仓集中  ███████████████ 15%                  ← 操纵风险
合约安全  ██████████ 10%                       ← 技术风险
流动性    ██████████ 10%                       ← 退出难度
社区信号  ██████████ 10%                       ← 辅助参考
```

---

## 八、API 接口设计

### 8.1 RESTful API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/scan` | 提交代币地址进行扫描 |
| `GET` | `/api/v1/report/{tokenAddress}` | 查询历史报告 |
| `GET` | `/api/v1/creator/{address}` | 查询创始人档案 |
| `GET` | `/api/v1/stats` | 全局统计数据 |
| `GET` | `/api/v1/hot` | 热门查询代币列表 |
| `GET` | `/api/v1/risk-wall` | 高风险警示墙（实时） |
| `WS` | `/ws/scan` | WebSocket 扫描进度推送 |

### 8.2 API 请求/响应示例

```json
// POST /api/v1/scan
// Request
{
    "token_address": "0x7a25...eF3b",
    "chain_id": 56
}

// Response
{
    "code": 0,
    "message": "success",
    "data": {
        "token_address": "0x7a25...eF3b",
        "token_name": "ScamMoon",
        "token_symbol": "SCAM",
        "risk_score": 8.2,
        "risk_level": "极度危险",
        "dimensions": [
            {
                "name": "合约安全",
                "score": 8.0,
                "weight": 0.10,
                "details": "合约存在 mint 函数和黑名单功能",
                "status": "danger"
            },
            {
                "name": "LP锁定",
                "score": 2.0,
                "weight": 0.25,
                "details": "LP 完全未锁定，创始人可随时撤池",
                "status": "danger"
            }
            // ... 其余四维
        ],
        "ai_report": "该代币风险极高，强烈建议不要参与。创建者地址在30天内部署了23个代币合约，其中18个已归零...",
        "creator_history": {
            "address": "0xABC...123",
            "total_tokens": 23,
            "rugged_tokens": 18,
            "rug_rate": 0.78,
            "avg_lifetime_hrs": 4.2,
            "risk_label": "⚠️ 惯犯：该地址创建了大量已归零代币（归零率 78%）"
        },
        "report_hash": "QmXx...YyZ",
        "tx_hash": "0xTx...Hash",
        "scanned_at": 1720458000
    }
}
```

---

## 九、开发时间线

### Day 1：合约 + 数据库 + 项目骨架（~3h）

| 时间 | 任务 | AI 协助 |
|------|------|---------|
| 19:30-20:00 | 写 RiskReportOracle.sol | Claude 生成初版 → review → 修改 |
| 20:00-20:30 | 写 Foundry 测试（提交报告/权限验证） | Claude 生成测试框架 |
| 20:30-21:00 | 休息 | — |
| 21:00-22:00 | 创建 go-zero 项目骨架（main.go + config + svc） | Claude 生成脚手架 |
| 22:00-22:30 | 写 PostgreSQL schema + goose/atlas 迁移 | Claude 生成 DDL |

**产出**：合约编译通过 + 测试通过 + 后端骨架可运行 + 数据库表建好

---

### Day 2：链上数据抓取（~4h）

| 时间 | 任务 | AI 协助 |
|------|------|---------|
| 19:30-20:30 | 写 scanner.go（6 goroutine 并行抓取框架） | Claude 生成 go-ethereum 调用代码 |
| 20:30-21:00 | 写 honeypot.go（模拟交易检测蜜罐） | Claude 生成检测逻辑 |
| 21:00-21:15 | 休息 | — |
| 21:15-22:30 | 写 historian.go + getDeployer 实现 | Claude 生成 receipt 解析代码 |
| 22:30-23:00 | 写 Indexer 的 token tracker 协程 | Claude 生成 Indexer 适配代码 |

**产出**：scanner 可对真实地址跑通并返回 profile

---

### Day 3：评分引擎 + AI 报告（~4h）

| 时间 | 任务 | AI 协助 |
|------|------|---------|
| 19:30-20:30 | 写 analyzer.go（六维评分算法） | Claude 生成评分函数 |
| 20:30-21:00 | 写单元测试（输入已知 profile → 验证分数） | Claude 生成测试数据 |
| 21:00-21:15 | 休息 | — |
| 21:15-22:30 | 写 reporter.go（调用 OpenAI/Gemini） | Claude 生成 prompt + API 调用 |
| 22:30-23:00 | 写 oracle.go（ECDSA 签名 + 调合约 submitReport） | Claude 生成签名和合约调用 |

**产出**：完整分析链路跑通（输入代币地址 → 返回完整 ScanResponse）

---

### Day 4：API + 缓存 + WebSocket（~3h）

| 时间 | 任务 | AI 协助 |
|------|------|---------|
| 19:30-20:30 | 写 handler + logic + types + 路由注册 | Claude 生成 go-zero handler/logic |
| 20:30-21:00 | 写 cache.go + Redis 集成 | Claude 生成缓存逻辑 |
| 21:00-21:15 | 休息 | — |
| 21:15-22:30 | 写 WebSocket 进度推送 | Claude 生成 WebSocket handler |

**产出**：所有 API 可 curl 调试，缓存生效

---

### Day 5：前端（~4h）

| 时间 | 任务 | AI 协助 |
|------|------|---------|
| 19:30-20:30 | Next.js 项目创建 + SearchBar + 调 API | Cursor 生成首页组件 |
| 20:30-21:00 | ReportPage（六维雷达图 SVG） | Cursor 生成雷达图 + 颜色映射 |
| 21:00-21:15 | 休息 | — |
| 21:15-22:30 | AI 报告卡片 + 创始人时间轴 | Cursor 生成卡片组件 |
| 22:30-23:00 | 加载动画 + 进度条 WebSocket 连接 | Cursor 生成动画 |

**产出**：前端可交互，输入地址可看到完整报告

---

### Day 6：集成 + 部署 + 提交（~4h）

| 时间 | 任务 | AI 协助 |
|------|------|---------|
| 19:30-20:30 | 端到端联调 + 修 bug | Claude debug |
| 20:30-21:00 | 部署合约到 BSC Testnet → BscScan 验证 | Claude 生成部署脚本 |
| 21:00-21:15 | 休息 | — |
| 21:15-22:00 | 部署后端（Railway/Fly.io 免费层） | Claude 生成 Dockerfile |
| 22:00-22:30 | 录 demo 视频（2分钟） | Claude 生成视频脚本 |
| 22:30-23:00 | 写 README + 提交 | Claude 生成 README |

**产出**：线上可用的完整产品 + 提交

---

## 十、Demo 视频脚本（2 分钟）

```
[0:00-0:15] 痛点引入
"每天都有数千个 Meme 代币上线，但 99% 都是骗局。
我今天展示的工具帮助用户在 20 秒内判断一个币能不能碰。"

[0:15-0:45] 演示扫描一个真实 scam 代币
（打开 DApp → 粘贴一个已知 scam 代币地址 → 点击分析）
"我输入一个地址，等待 20 秒。
看——进度条展示了我们正在检查的 6 个维度。"

[0:45-1:15] 展示报告
"得分 8.2/10，高风险。
点击创始人历史——看！这个地址 30 天内部署了 23 个代币，
其中 18 个已经归零，平均寿命 4 小时。这是典型的 serial rugger。"

[1:15-1:45] 展示链上锚定
"每一次分析结果都通过 RiskReportOracle 合约锚定到链上，
哈希不可篡改，任何人都可以独立验证。展示一下链上 TX。"

[1:45-2:00] 总结
"Meme AI Analyze——给散户一把在黑暗森林中探路的灯。
技术栈：Solidity + Go + AI + PostgreSQL。代码全部开源在 GitHub。"
```

---

## 十一、提交检查清单

- [ ] 合约部署到 BSC Testnet 并 BscScan 验证
- [ ] 后端 API 可公网访问（Railway/Fly.io）
- [ ] 前端可访问且有 demo
- [ ] GitHub 仓库 README 写清楚（英文）：
  - What it does（一句话）
  - How it works（架构图 + 技术栈）
  - Why it matters（解决什么问题）
  - How to run（docker-compose up 一键启动）
  - Demo video link
  - Contract address (BscScan link)
- [ ] Demo 视频 < 3 分钟
- [ ] 合约开源且至少有 2 笔链上交易
- [ ] 发给 2-3 个朋友测试，录 feedback

---

## 十二、附录

### A. 技术风险与应对

| 风险 | 影响 | 应对策略 |
|------|------|---------|
| RPC 节点限流 | 扫描中断 | 多节点轮询 + 请求队列 + 指数退避重试 |
| AI API 超时 | 报告无 AI 分析 | 超时降级：返回纯数据评分，标记"AI 分析暂时不可用" |
| 创始地址获取失败 | 缺失核心维度 | 标记为"unknown"，降低创始人维度权重，不影响其他维度 |
| 缓存击穿（热点代币） | 数据库压力 | 互斥锁 + 单飞模式（singleflight）防止并发重复扫描 |

### B. 后续迭代方向

1. **多链支持**：从 BSC 扩展到 Ethereum、Arbitrum、Base 等 EVM 兼容链
2. **实时监控**：对已扫描代币建立 watch 机制，关键指标变化时推送告警
3. **社区贡献**：允许用户提交 scam 证据，社区验证后上链存证
4. **API 开放**：提供付费 API，供其他 DApp 集成风险评分
5. **AI 模型微调**：基于积累的 rug 案例数据微调专用风险分析模型

---

> **核心理念**：我们做的不是又一个发射台或交易工具——我们做的是 Meme 赛道的基础设施，是散户在黑暗森林里的探照灯。**Show, don't tell.**
