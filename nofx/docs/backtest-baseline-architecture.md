# 回测基线对比系统架构设计

## 概述

回测基线对比系统允许在运行 AI 策略回测的同时，并行运行一个基于传统技术指标的确定性策略作为基线（Baseline），用于对比和评估 AI 策略的表现。

## 系统架构

### 1. 核心组件

#### 1.1 BaselineEngine (基线引擎)
**文件**: `backtest/baseline_engine.go`

基于传统技术指标的确定性决策引擎，使用以下指标：
- **RSI** (Relative Strength Index) - 相对强弱指标
- **MACD** (Moving Average Convergence Divergence) - 指数平滑异同移动平均线
- **EMA** (Exponential Moving Average) - 指数移动平均线
- **StochRSI** (Stochastic RSI) - 随机相对强弱指标

**决策逻辑**:
```
开仓条件: 需要至少 2 个信号共振
- RSI 超卖/超买
- MACD 金叉/死叉
- 价格突破 EMA
- StochRSI 超卖/超买

平仓条件:
- 止盈: 达到 2x ATR
- 止损: 达到 1x ATR
- 反向信号出现
```

#### 1.2 Runner 集成
**文件**: `backtest/runner.go`

在回测运行器中集成了 Baseline 系统：

**新增字段**:
```go
type Runner struct {
    // ... 原有字段

    // Baseline 相关
    baselineEnabled bool
    baselineAccount *BacktestAccount
    baselineEngine  *BaselineEngine
    baselineState   *BacktestState
}
```

**执行流程**:
1. 每个回测周期，AI 和 Baseline 并行执行
2. 使用独立的账户管理（baselineAccount）
3. 独立记录权益曲线和交易事件
4. 互不干扰，完全隔离

#### 1.3 存储层
**文件**: `backtest/storage.go`, `backtest/storage_db_impl.go`

**数据库表**:
```sql
-- Baseline 权益曲线
CREATE TABLE backtest_baseline_equity (
    run_id TEXT,
    ts INTEGER,
    equity REAL,
    pnl_pct REAL
);

-- Baseline 交易记录
CREATE TABLE backtest_baseline_trades (
    run_id TEXT,
    ts INTEGER,
    symbol TEXT,
    action TEXT,
    side TEXT,
    quantity REAL,
    price REAL,
    realized_pnl REAL,
    cycle INTEGER
);
```

**存储函数**:
- `appendBaselineEquityPoint()` - 记录权益点
- `appendBaselineTradeEvent()` - 记录交易事件
- `LoadBaselineEquityPoints()` - 加载权益曲线
- `LoadBaselineTradeEvents()` - 加载交易记录

### 2. API 接口

#### 2.1 后端 API
**文件**: `api/backtest.go`

**新增端点**:
```
GET /api/backtest/baseline/equity?run_id={run_id}
GET /api/backtest/baseline/trades?run_id={run_id}
```

**响应格式**:
```json
// Equity 响应
[
  {
    "ts": 1704067200000,
    "equity": 1050.25,
    "pnl_pct": 5.025
  }
]

// Trades 响应
[
  {
    "timestamp": 1704067200000,
    "symbol": "BTCUSDT",
    "action": "open_long",
    "side": "long",
    "quantity": 0.1,
    "price": 42000.0,
    "realized_pnl": 0,
    "cycle": 10
  }
]
```

#### 2.2 前端 API
**文件**: `web/src/lib/api.ts`

**新增函数**:
```typescript
async getBaselineEquity(runId: string): Promise<BacktestEquityPoint[]>
async getBaselineTrades(runId: string): Promise<BacktestTradeEvent[]>
```

### 3. 前端展示

#### 3.1 启动配置
**文件**: `web/src/components/BacktestPage.tsx`

**位置**: 回测启动向导 - 第3步

**配置项**:
- ✅ 复用AI缓存
- ✅ 仅回放模式
- ✅ **启用基线对比** ← 新增

**说明**: 勾选后，回测将并行运行传统指标策略作为基线对比

#### 3.2 权益曲线图表
**文件**: `web/src/components/BacktestPage.tsx`

**图表组件**: `BacktestChart`

**显示内容**:
- **黄色实线** + 渐变填充 - AI 策略权益曲线
- **橙色虚线** (#FF6B35) - Baseline 指标策略曲线
- **图例** - 区分 "AI Strategy" 和 "Baseline (Indicators)"
- **Tooltip** - 鼠标悬停显示两条曲线的具体数值

**数据获取**:
```typescript
// 仅当 enable_baseline 为 true 时获取
const baselineEquity = useSWR(
  selectedRunId && baselineEnabled ? ['bt-baseline-equity', selectedRunId] : null,
  () => api.getBaselineEquity(selectedRunId!)
)
```

## 4. 使用说明

### 4.1 启动带 Baseline 的回测

1. 进入回测页面，点击"启动回测"
2. 完成第1步（选择交易对和时间范围）
3. 完成第2步（选择策略和杠杆）
4. 在第3步中，勾选 **"启用基线对比"** 复选框
5. 点击"启动回测"

### 4.2 查看对比结果

回测运行后，在权益曲线图表中可以看到：
- **AI 策略曲线**（黄色实线）- 基于 AI 决策的表现
- **Baseline 曲线**（橙色虚线）- 基于传统指标的表现

通过对比两条曲线，可以评估：
- AI 策略是否优于传统指标策略
- AI 策略的优势在哪些时间段
- 是否存在 AI 决策的随机性问题

## 5. 技术细节

### 5.1 指标参数配置

Baseline 引擎使用策略配置中的 `IndicatorConfig` 参数：

```go
type IndicatorConfig struct {
    RSIPeriod        int     // RSI 周期，默认 14
    MACDFast         int     // MACD 快线，默认 12
    MACDSlow         int     // MACD 慢线，默认 26
    MACDSignal       int     // MACD 信号线，默认 9
    EMAPeriod        int     // EMA 周期，默认 20
    StochRSIPeriod   int     // StochRSI 周期，默认 14
    ATRPeriod        int     // ATR 周期，默认 14
}
```

### 5.2 信号阈值

```go
const (
    RSI_OVERSOLD  = 30.0  // RSI 超卖
    RSI_OVERBOUGHT = 70.0 // RSI 超买
    STOCH_OVERSOLD = 20.0 // StochRSI 超卖
    STOCH_OVERBOUGHT = 80.0 // StochRSI 超买
)
```

### 5.3 风险管理

**仓位管理**:
- 使用与 AI 策略相同的杠杆配置
- 每次开仓使用可用资金的固定比例
- 独立的账户管理，不影响 AI 策略

**止盈止损**:
- 止盈: 2x ATR
- 止损: 1x ATR
- 基于 ATR 动态调整，适应市场波动

### 5.4 数据流程

```
┌─────────────────────────────────────────────────────┐
│                  Backtest Runner                     │
│                                                      │
│  ┌──────────────┐              ┌──────────────┐    │
│  │  AI Engine   │              │   Baseline   │    │
│  │              │              │    Engine    │    │
│  │  - LLM Call  │              │  - RSI       │    │
│  │  - Decision  │              │  - MACD      │    │
│  │              │              │  - EMA       │    │
│  └──────┬───────┘              └──────┬───────┘    │
│         │                             │             │
│         ▼                             ▼             │
│  ┌──────────────┐              ┌──────────────┐    │
│  │ AI Account   │              │   Baseline   │    │
│  │              │              │   Account    │    │
│  └──────┬───────┘              └──────┬───────┘    │
│         │                             │             │
│         ▼                             ▼             │
│  ┌──────────────┐              ┌──────────────┐    │
│  │  AI Equity   │              │   Baseline   │    │
│  │   & Trades   │              │ Equity/Trade │    │
│  └──────────────┘              └──────────────┘    │
└─────────────────────────────────────────────────────┘
```


## 6. 注意事项

### 6.1 性能影响
- Baseline 引擎计算开销很小（纯数学计算）
- 不会显著影响回测速度
- 数据库写入量增加约 2倍（equity + trades）

### 6.2 数据一致性
- Baseline 和 AI 使用相同的市场数据
- 使用相同的时间戳和周期
- 确保对比的公平性

### 6.3 策略配置
- Baseline 使用策略的 `IndicatorConfig` 参数
- 如果策略未配置指标参数，使用默认值
- 可以通过修改策略配置来调整 Baseline 行为


## 7. 文件清单

### 7.1 后端文件
```
backtest/
├── baseline_engine.go      # Baseline 决策引擎
├── config.go               # 配置结构（新增 EnableBaseline）
├── runner.go               # 回测运行器（集成 Baseline）
├── storage.go              # 存储接口（新增 Baseline 函数）
├── storage_db_impl.go      # 数据库实现（新增 Baseline 表）
└── account.go              # 账户管理（复用）

api/
└── backtest.go             # API 端点（新增 Baseline 路由）

store/
└── backtest.go             # 数据库表定义（新增 Baseline 表）
```

### 7.2 前端文件
```
web/src/
├── lib/
│   └── api.ts              # API 函数（新增 Baseline 接口）
├── types.ts                # 类型定义（新增 enable_baseline）
└── components/
    └── BacktestPage.tsx    # 回测页面（新增复选框和图表）
```


## 8. 总结

### 8.1 核心优势
1. **确定性基准** - 提供可重复的对比基线
2. **并行执行** - 不影响 AI 策略运行
3. **独立账户** - 完全隔离，互不干扰
4. **可视化对比** - 直观展示 AI vs 传统指标
5. **低开销** - 计算成本极低

### 8.2 应用场景
- 评估 AI 策略是否优于传统方法
- 识别 AI 决策的随机性问题
- 验证策略配置的有效性
- 为策略优化提供参考基准

### 8.3 未来扩展
- [ ] 支持更多技术指标（Bollinger Bands, Fibonacci 等）
- [ ] 可配置的信号权重
- [ ] Baseline 策略的独立配置界面
- [ ] 多个 Baseline 策略对比
- [ ] Baseline 性能指标统计

---

**文档版本**: v1.0  
**最后更新**: 2026-01-02  
**作者**: NOFX Team
