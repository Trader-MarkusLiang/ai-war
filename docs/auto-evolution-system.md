# 回测自动迭代优化系统 - 项目文档

> 创建时间: 2025-12-27
> 状态: 开发中

---

## 一、项目概述

### 1.1 目标

构建一个自动化链条：**回测完成 → AI评估 → Prompt优化 → 生成新策略 → 启动下一轮回测**，实现策略的自我迭代进化。

### 1.2 核心价值

- 自动化策略优化，减少人工干预
- 通过AI评估发现策略弱点
- 可控的单变量迭代（只优化Prompt）
- 完整的版本追溯和回滚能力

---

## 二、系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    AutoEvolver (自动进化器)                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐  │
│  │ 1.回测   │───▶│ 2.评估   │───▶│ 3.优化   │───▶│ 4.验证   │  │
│  │ Runner   │    │ Analyzer │    │ Optimizer│    │ Validator│  │
│  └──────────┘    └──────────┘    └──────────┘    └──────────┘  │
│       │                                               │         │
│       └───────────────── 循环 ◀───────────────────────┘         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 三、用户确认的配置

| 配置项 | 选择 |
|--------|------|
| AI模型 | Claude Opus (评估和优化) |
| 停止条件 | 收敛检测 (连续3次无明显改进) |
| 前端UI | 需要完整UI |
| 迭代上限 | 最多10次 |

---

## 四、核心流程

### Phase 1: 回测执行与监控

```
输入: 策略ID, 回测参数(固定)
输出: 回测结果(metrics, decisions, equity_curve)

流程:
1. 调用 POST /backtest/start 启动回测
2. 轮询 GET /backtest/status 监控进度
3. 回测完成后获取:
   - GET /backtest/metrics → 收益率、回撤、胜率等
   - GET /backtest/trades → 交易记录
   - 数据库查询 → 决策日志(cot_trace)
```

### Phase 2: AI评估分析

```
输入: 回测结果 + 当前策略Prompt
输出: 评估报告(strengths, weaknesses, suggestions)

评估维度:
1. 收益表现: 总收益率、夏普比率
2. 风险控制: 最大回撤、爆仓风险
3. 交易质量: 胜率、盈亏比、交易频率
4. 决策分析: 抽样分析AI的COT推理过程
5. 关键事件: 大亏损/大盈利的决策复盘
```

### Phase 3: Prompt优化生成

```
输入: 评估报告 + 当前Prompt + 历史迭代记录
输出: 优化后的新Prompt

优化原则:
1. 单次只修改1-2个关键点(可控变量)
2. 保留有效的规则,只调整问题部分
3. 修改需要可量化验证
```

### Phase 4: 策略验证与保存

```
输入: 新Prompt
输出: 新策略ID

验证步骤:
1. JSON格式校验
2. 必要字段完整性检查
3. 写入数据库生成新策略
4. 版本号递增(v1→v2→v3)
```

---

## 五、数据库设计

### 5.1 进化任务表 (evolutions)

```sql
CREATE TABLE evolutions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    base_strategy_id TEXT NOT NULL,
    status TEXT DEFAULT 'created',  -- created/running/paused/completed/stopped
    current_iteration INTEGER DEFAULT 0,
    max_iterations INTEGER DEFAULT 10,
    convergence_threshold INTEGER DEFAULT 3,
    best_version INTEGER DEFAULT 0,
    best_return REAL DEFAULT 0,
    config TEXT NOT NULL,  -- JSON格式的完整配置
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 5.2 迭代记录表 (evolution_iterations)

```sql
CREATE TABLE evolution_iterations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    evolution_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    strategy_id TEXT NOT NULL,
    backtest_run_id TEXT,
    status TEXT DEFAULT 'pending',  -- pending/backtesting/evaluating/optimizing/completed

    -- 回测指标
    total_return REAL,
    max_drawdown REAL,
    win_rate REAL,
    sharpe_ratio REAL,
    trades INTEGER,

    -- AI评估和优化
    evaluation_report TEXT,  -- JSON
    changes_summary TEXT,
    prompt_before TEXT,
    prompt_after TEXT,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (evolution_id) REFERENCES evolutions(id),
    UNIQUE(evolution_id, version)
);

CREATE INDEX idx_iterations_evolution ON evolution_iterations(evolution_id);
```

---

## 六、后端文件结构

```
nofx/
├── autoevolver/
│   ├── evolver.go       # 主控制器 (~300行)
│   ├── analyzer.go      # 回测分析器 (~200行)
│   ├── optimizer.go     # Prompt优化器 (~250行)
│   ├── validator.go     # 策略验证器 (~100行)
│   ├── types.go         # 数据结构 (~150行)
│   └── prompts.go       # AI提示词模板 (~100行)
├── store/
│   └── evolution.go     # 进化记录存储 (~200行)
├── api/
│   └── evolution.go     # 进化API处理器 (~300行)
```

---

## 七、API接口设计

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /evolution/create | 创建进化任务 |
| POST | /evolution/{id}/start | 启动进化 |
| POST | /evolution/{id}/pause | 暂停进化 |
| POST | /evolution/{id}/stop | 停止进化 |
| GET | /evolution/{id}/status | 获取进化状态 |
| GET | /evolution/{id}/iterations | 获取迭代历史 |
| GET | /evolution/{id}/iterations/{version} | 获取单次迭代详情 |
| GET | /evolution/list | 获取所有进化任务列表 |
| POST | /evolution/{id}/rollback/{version} | 回滚到指定版本 |

---

## 八、前端文件结构

```
nofx/web/src/
├── pages/
│   └── EvolutionPage.tsx          # 进化实验室主页面 (新增)
├── components/
│   ├── evolution/
│   │   ├── EvolutionCard.tsx      # 进化任务卡片
│   │   ├── IterationTable.tsx     # 迭代历史表格
│   │   ├── EvolutionChart.tsx     # 多版本收益对比图
│   │   ├── EvaluationReport.tsx   # AI评估报告展示
│   │   ├── PromptDiff.tsx         # Prompt差异对比
│   │   └── CreateEvolutionModal.tsx # 创建进化任务弹窗
├── lib/
│   └── api.ts                     # 添加进化相关API (修改)
└── types.ts                       # 添加进化相关类型 (修改)
```

---

## 九、安全机制

1. **迭代上限**: 最多10次迭代，防止无限循环
2. **收敛检测**: 连续3次无改进则停止
3. **回滚机制**: 保留所有版本，可随时回滚
4. **人工审核**: 可选的人工确认环节

---

## 十、任务清单 (TODO)

### 第一阶段: 后端核心

| # | 任务 | 状态 | 备注 |
|---|------|------|------|
| 1 | 创建 autoevolver 包基础结构和类型定义 | ✅ 已完成 | types.go, prompts.go |
| 2 | 实现 store/evolution.go 数据存储层 | ✅ 已完成 | 包含所有CRUD操作 |
| 3 | 实现 evolver.go 主控制流程 | ✅ 已完成 | 完整集成analyzer/optimizer/validator |
| 4 | 实现 analyzer.go 回测分析器 | ✅ 已完成 | AI评估报告生成 |
| 5 | 实现 optimizer.go Prompt优化器 | ✅ 已完成 | 调用Claude Opus |
| 6 | 实现 validator.go 策略验证器 | ✅ 已完成 | JSON验证和策略保存 |
| 7 | 添加 api/evolution.go REST接口 | ✅ 已完成 | 9个API端点 + 路由注册 |

### 第二阶段: 前端UI

| # | 任务 | 状态 | 备注 |
|---|------|------|------|
| 8 | 创建 EvolutionPage.tsx 主页面 | ⏳ 待开始 | |
| 9 | 实现迭代历史表格组件 | ⏳ 待开始 | IterationTable.tsx |
| 10 | 实现多版本收益曲线对比图 | ⏳ 待开始 | 基于Recharts |
| 11 | 实现AI评估报告展示 | ⏳ 待开始 | EvaluationReport.tsx |
| 12 | 实现Prompt差异对比组件 | ⏳ 待开始 | PromptDiff.tsx |
| 13 | 添加创建进化任务弹窗 | ⏳ 待开始 | CreateEvolutionModal.tsx |

### 第三阶段: 集成测试

| # | 任务 | 状态 | 备注 |
|---|------|------|------|
| 14 | 端到端测试完整流程 | ⏳ 待开始 | |
| 15 | 收敛检测逻辑验证 | ⏳ 待开始 | |
| 16 | 异常处理和恢复机制测试 | ⏳ 待开始 | |

---

## 十一、核心代码示例

### 11.1 主控制流程 (evolver.go)

```go
func (e *AutoEvolver) Run(ctx context.Context) error {
    for iteration := 1; iteration <= e.config.MaxIterations; iteration++ {
        // 1. 启动回测
        runID, err := e.startBacktest(ctx, e.currentStrategyID)
        if err != nil { return err }

        // 2. 等待回测完成
        metrics, err := e.waitBacktestComplete(ctx, runID)
        if err != nil { return err }

        // 3. 记录迭代结果
        e.recordIteration(iteration, runID, metrics)

        // 4. 收敛检测
        if e.isConverged() {
            log.Info("收敛检测触发，停止进化")
            break
        }

        // 5. AI评估
        evaluation, err := e.evaluate(ctx, metrics)
        if err != nil { return err }

        // 6. AI优化Prompt
        newPrompt, err := e.optimize(ctx, evaluation)
        if err != nil { return err }

        // 7. 验证并保存新策略
        newStrategyID, err := e.saveNewStrategy(newPrompt, iteration+1)
        if err != nil { return err }

        e.currentStrategyID = newStrategyID
    }
    return nil
}
```

### 11.2 收敛检测 (evolver.go)

```go
func (e *AutoEvolver) isConverged() bool {
    history := e.getRecentHistory(3)
    if len(history) < 3 { return false }

    // 检查最近3次迭代的收益率变化
    for i := 1; i < len(history); i++ {
        improvement := history[i].TotalReturn - history[i-1].TotalReturn
        if improvement > 1.0 { // 超过1%的改进
            return false
        }
    }
    return true // 连续3次无明显改进
}
```

---

## 十二、前端UI设计

```
┌─────────────────────────────────────────────────────────────┐
│  策略自动进化实验室                              [新建进化]  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 进化任务: evolution_001                              │   │
│  │ 基础策略: 4h-Stoch-RSI-v2                           │   │
│  │ 状态: 运行中 (第3轮/最多10轮)                        │   │
│  │ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │   │
│  │ [暂停] [停止]                                        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─ 迭代历史 ──────────────────────────────────────────┐   │
│  │                                                      │   │
│  │  版本    收益率   回撤    胜率   状态    操作        │   │
│  │  ─────────────────────────────────────────────────  │   │
│  │  v1      +2.1%   -24%   45%    完成    [查看]       │   │
│  │  v2      +19.3%  -20%   52%    完成    [查看]       │   │
│  │  v3      +15.8%  -18%   48%    运行中  [查看]       │   │
│  │                                                      │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─ 收益曲线对比 ──────────────────────────────────────┐   │
│  │     ^                                                │   │
│  │  +20%│      ╭──v2                                   │   │
│  │     │    ╭─╯                                        │   │
│  │  +10%│  ╭╯    ╭──v3                                 │   │
│  │     │ ╭╯   ╭─╯                                      │   │
│  │   0%├─╯──╭╯────v1                                   │   │
│  │     │   ╰╯                                          │   │
│  │ -10%│                                               │   │
│  │     └────────────────────────────────────────▶      │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─ 当前迭代详情 (v3) ─────────────────────────────────┐   │
│  │  [评估报告] [Prompt差异] [决策样本]                  │   │
│  │                                                      │   │
│  │  ## AI评估报告                                       │   │
│  │  优势: 止损放宽后减少了假突破止损...                 │   │
│  │  劣势: 强制止盈8%可能过早平仓...                     │   │
│  │  建议: 将强制止盈调整为10%...                        │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## 十三、更新日志

| 日期 | 更新内容 |
|------|----------|
| 2025-12-27 | 创建项目文档，完成方案设计 |
| 2025-12-27 | 用户确认配置：Claude Opus + 收敛检测 + 完整UI |
| 2025-12-27 | 开始后端开发 |
| 2025-12-27 | ✅ 完成后端核心开发（第一阶段全部完成） |
| 2025-12-27 | - 创建 autoevolver 包：types.go, prompts.go |
| 2025-12-27 | - 实现数据存储层：store/evolution.go (~322行) |
| 2025-12-27 | - 实现主控制器：evolver.go (~346行，完整集成) |
| 2025-12-27 | - 实现AI分析器：analyzer.go (~113行) |
| 2025-12-27 | - 实现Prompt优化器：optimizer.go (~85行) |
| 2025-12-27 | - 实现策略验证器：validator.go (~50行) |
| 2025-12-27 | - 实现REST API：api/evolution.go (~476行，9个端点) |
| 2025-12-27 | - 更新Store接口：添加Evolution()方法 |

---

## 十四、相关文件

- 计划文件: `/home/mc/.claude/plans/groovy-puzzling-eich.md`
- 策略v2: `stoch-rsi-4h-v2-512E5057`
- 回测分析: `bt_20251227_004500`
