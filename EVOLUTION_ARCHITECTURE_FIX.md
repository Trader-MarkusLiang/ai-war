# Evolution Lab 架构重构文档

## 📋 重构完成状态

**状态**: ✅ 已完成基础架构重构
**日期**: 2025-12-27
**版本**: v1.0 (基础版本，AI 功能使用 stub 实现)

## 问题分析

原始 Evolution Lab 的实现是**独立的**，没有复用现有的回测模块，这导致：

1. ❌ 重复实现回测逻辑
2. ❌ 无法利用现有的回测引擎
3. ❌ 回测结果和进化记录分离
4. ❌ 无法在回测页面查看进化任务的回测详情

## 正确的架构

Evolution Lab 应该是回测模块的**编排层**：

```
Evolution Lab (编排器)
    ↓
    调用 Backtest API
    ↓
    获取回测结果
    ↓
    AI 评估 + 优化 (当前为 stub)
    ↓
    生成新策略
    ↓
    循环迭代
```

## 实现方案

### 1. 数据库设计调整

#### evolutions 表（保持不变）
```sql
CREATE TABLE evolutions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    base_strategy_id TEXT NOT NULL,
    status TEXT NOT NULL,
    current_iteration INTEGER DEFAULT 0,
    max_iterations INTEGER DEFAULT 10,
    convergence_threshold INTEGER DEFAULT 3,
    best_version INTEGER DEFAULT 0,
    best_return REAL DEFAULT 0,
    config TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### evolution_iterations 表（关键修改）
```sql
CREATE TABLE evolution_iterations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    evolution_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    strategy_id TEXT NOT NULL,
    backtest_run_id TEXT NOT NULL,  -- ⭐ 关联到 backtest_runs 表
    status TEXT NOT NULL,

    -- 不再存储 metrics，直接从 backtest_runs 表查询
    -- metrics TEXT,  ❌ 删除

    evaluation_report TEXT,
    changes_summary TEXT,
    prompt_before TEXT,
    prompt_after TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (evolution_id) REFERENCES evolutions(id),
    FOREIGN KEY (backtest_run_id) REFERENCES backtest_runs(run_id)  -- ⭐ 外键关联
);
```

### 2. AutoEvolver 实现

```go
// autoevolver/evolver.go
type AutoEvolver struct {
    evolutionID   string
    config        *EvolutionConfig
    backtestMgr   *backtest.Manager  // ⭐ 注入回测管理器
    aiClient      *mcp.AIClient      // ⭐ 注入 AI 客户端
    store         *store.Store
    status        string
    stopChan      chan struct{}
}

func NewAutoEvolver(
    evolutionID string,
    config *EvolutionConfig,
    backtestMgr *backtest.Manager,  // ⭐ 依赖注入
    aiClient *mcp.AIClient,         // ⭐ 依赖注入
    store *store.Store,
) *AutoEvolver {
    return &AutoEvolver{
        evolutionID: evolutionID,
        config:      config,
        backtestMgr: backtestMgr,
        aiClient:    aiClient,
        store:       store,
        status:      StatusCreated,
        stopChan:    make(chan struct{}),
    }
}

func (e *AutoEvolver) Start(ctx context.Context) error {
    e.status = StatusRunning

    for version := 1; version <= e.config.MaxIterations; version++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-e.stopChan:
            return nil
        default:
        }

        // 执行单次迭代
        if err := e.runIteration(ctx, version); err != nil {
            return err
        }

        // 检查收敛
        if e.checkConvergence() {
            e.status = StatusCompleted
            return nil
        }
    }

    e.status = StatusCompleted
    return nil
}

func (e *AutoEvolver) runIteration(ctx context.Context, version int) error {
    // 1. 获取当前策略
    strategy, err := e.store.Strategy().Get(e.config.UserID, e.config.BaseStrategyID)
    if err != nil {
        return err
    }

    // 2. 构建回测配置
    backtestRunID := fmt.Sprintf("%s_v%d_%d", e.evolutionID, version, time.Now().Unix())
    backtestConfig := backtest.BacktestConfig{
        RunID:          backtestRunID,
        UserID:         e.config.UserID,
        Symbols:        e.config.FixedParams.Symbols,
        Timeframes:     e.config.FixedParams.Timeframes,
        StartTS:        e.config.FixedParams.StartTS,
        EndTS:          e.config.FixedParams.EndTS,
        InitialBalance: e.config.FixedParams.InitialBalance,
        FeeBps:         e.config.FixedParams.FeeBps,
        SlippageBps:    e.config.FixedParams.SlippageBps,
        PromptVariant:  strategy.Prompt,  // 使用当前策略的 prompt
        AIModel:        e.config.EvaluationModel,
    }

    // 3. ⭐ 调用回测模块执行回测
    err = e.backtestMgr.Start(backtestConfig)
    if err != nil {
        return fmt.Errorf("backtest failed: %w", err)
    }

    // 4. 等待回测完成
    if err := e.waitForBacktestComplete(ctx, backtestRunID); err != nil {
        return err
    }

    // 5. ⭐ 从回测模块获取结果
    metrics, err := e.backtestMgr.GetMetrics(backtestRunID)
    if err != nil {
        return err
    }

    decisions, err := e.backtestMgr.GetDecisions(backtestRunID, 0, 100)
    if err != nil {
        return err
    }

    // 6. ⭐ 调用 AI 评估回测结果
    evaluation, err := e.evaluateBacktest(ctx, metrics, decisions)
    if err != nil {
        return err
    }

    // 7. ⭐ 调用 AI 生成优化建议
    optimization, err := e.optimizePrompt(ctx, strategy.Prompt, evaluation)
    if err != nil {
        return err
    }

    // 8. 保存迭代记录
    iteration := &evotypes.Iteration{
        EvolutionID:    e.evolutionID,
        Version:        version,
        StrategyID:     strategy.ID,
        BacktestRunID:  backtestRunID,  // ⭐ 关联回测 run_id
        Status:         IterStatusCompleted,
        EvalReport:     marshalJSON(evaluation),
        ChangesSummary: optimization.ExpectedEffect,
        PromptBefore:   strategy.Prompt,
        PromptAfter:    optimization.NewPrompt,
    }

    if err := e.store.Evolution().CreateIteration(iteration); err != nil {
        return err
    }

    // 9. 更新进化任务状态
    if metrics.TotalReturn > e.getBestReturn() {
        e.updateBestVersion(version, metrics.TotalReturn)
    }

    // 10. 创建新版本策略
    newStrategy := &store.Strategy{
        ID:          uuid.New().String(),
        UserID:      e.config.UserID,
        Name:        fmt.Sprintf("%s_v%d", strategy.Name, version),
        Prompt:      optimization.NewPrompt,
        ParentID:    strategy.ID,
        EvolutionID: e.evolutionID,
    }

    if err := e.store.Strategy().Create(newStrategy); err != nil {
        return err
    }

    // 11. 更新 base_strategy_id 为新策略
    e.config.BaseStrategyID = newStrategy.ID

    return nil
}

func (e *AutoEvolver) evaluateBacktest(
    ctx context.Context,
    metrics *backtest.Metrics,
    decisions []*decision.DecisionRecord,
) (*evotypes.EvaluationReport, error) {
    // 构建评估 prompt
    prompt := fmt.Sprintf(`
请评估以下回测结果：

性能指标：
- 总收益率: %.2f%%
- 最大回撤: %.2f%%
- 胜率: %.2f%%
- 夏普比率: %.2f
- 交易次数: %d

请分析：
1. 策略的优势（Strengths）
2. 策略的劣势（Weaknesses）
3. 改进建议（Suggestions）

请以 JSON 格式返回：
{
  "strengths": ["...", "..."],
  "weaknesses": ["...", "..."],
  "suggestions": ["...", "..."]
}
`, metrics.TotalReturn*100, metrics.MaxDrawdown*100,
   metrics.WinRate*100, metrics.SharpeRatio, metrics.TotalTrades)

    // ⭐ 调用 AI 客户端
    response, err := e.aiClient.SendMessage(ctx, prompt)
    if err != nil {
        return nil, err
    }

    var evaluation evotypes.EvaluationReport
    if err := json.Unmarshal([]byte(response), &evaluation); err != nil {
        return nil, err
    }

    return &evaluation, nil
}

func (e *AutoEvolver) optimizePrompt(
    ctx context.Context,
    currentPrompt string,
    evaluation *evotypes.EvaluationReport,
) (*evotypes.OptimizationResult, error) {
    // 构建优化 prompt
    prompt := fmt.Sprintf(`
当前策略 prompt：
%s

评估结果：
优势：%v
劣势：%v
建议：%v

请根据评估结果优化策略 prompt，返回 JSON 格式：
{
  "changes": ["修改1", "修改2"],
  "new_prompt": "优化后的完整 prompt",
  "expected_effect": "预期效果说明"
}
`, currentPrompt, evaluation.Strengths, evaluation.Weaknesses, evaluation.Suggestions)

    // ⭐ 调用 AI 客户端
    response, err := e.aiClient.SendMessage(ctx, prompt)
    if err != nil {
        return nil, err
    }

    var optimization evotypes.OptimizationResult
    if err := json.Unmarshal([]byte(response), &optimization); err != nil {
        return nil, err
    }

    return &optimization, nil
}
```

### 3. API 层调整

```go
// api/evolution.go
func (s *Server) handleStartEvolution(c *gin.Context) {
    userID := c.GetString("user_id")
    evolutionID := c.Param("id")

    evolution, err := s.store.Evolution().Get(userID, evolutionID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Evolution not found"})
        return
    }

    // 解析配置
    var config autoevolver.EvolutionConfig
    json.Unmarshal([]byte(evolution.Config), &config)

    // ⭐ 创建 AutoEvolver，注入依赖
    evolver := autoevolver.NewAutoEvolver(
        evolutionID,
        &config,
        s.backtestMgr,  // ⭐ 注入回测管理器
        s.aiClient,     // ⭐ 注入 AI 客户端
        s.store,
    )

    // 启动进化任务（异步）
    go func() {
        ctx := context.Background()
        if err := evolver.Start(ctx); err != nil {
            logger.Errorf("Evolution failed: %v", err)
        }
    }()

    // 更新状态
    s.store.Evolution().UpdateStatus(evolutionID, autoevolver.StatusRunning)

    c.JSON(http.StatusOK, gin.H{
        "message": "Evolution started",
        "status":  autoevolver.StatusRunning,
    })
}
```

### 4. 前端调整

前端可以通过 `backtest_run_id` 直接跳转到回测详情页面：

```typescript
// IterationTable.tsx
const handleViewBacktest = (backtestRunId: string) => {
  // 跳转到回测详情页面
  navigate(`/backtest?run_id=${backtestRunId}`)
}

// 在表格中添加"查看回测"按钮
<button onClick={() => handleViewBacktest(iteration.backtest_run_id)}>
  View Backtest
</button>
```

## 优势

✅ **复用现有代码**：不需要重新实现回测逻辑
✅ **数据一致性**：回测结果统一存储在 `backtest_runs` 表
✅ **功能完整性**：可以查看进化任务的完整回测详情（equity curve, trades, decisions）
✅ **易于维护**：回测逻辑的改进会自动应用到进化任务
✅ **清晰的职责分离**：Evolution Lab 只负责编排，Backtest 负责执行

## ✅ 已完成的实施步骤

### 1. 数据库 Schema ✅
- `evolution_iterations` 表已包含 `backtest_run_id` 字段
- 可通过此字段关联到 `backtest_runs` 表查看完整回测详情

### 2. AutoEvolver 核心实现 ✅

**文件**: `nofx/autoevolver/evolver.go`
- 实现了完整的生命周期管理（Start/Pause/Resume/Stop）
- 支持迭代循环和收敛检测
- 正确注入 `backtestMgr` 和 `store` 依赖

**文件**: `nofx/autoevolver/iteration.go`
- 实现了单次迭代流程：
  1. 获取当前策略
  2. 构建回测配置
  3. **调用 `backtestMgr.Start()` 执行回测**
  4. 等待回测完成
  5. 获取回测结果
  6. AI 评估（当前为 stub）
  7. AI 优化（当前为 stub）
  8. 保存迭代记录
  9. 更新最佳版本
  10. 创建新策略版本

**文件**: `nofx/autoevolver/helpers.go`
- 收敛检测逻辑
- 最佳版本管理

### 3. API 层实现 ✅

**文件**: `nofx/api/evolution.go`
- `handleStartEvolution`: 创建 AutoEvolver 实例并启动
- `handlePauseEvolution`: 暂停进化任务
- `handleResumeEvolution`: 恢复进化任务
- `handleStopEvolution`: 停止进化任务
- `handleGetEvolutionIterations`: 获取迭代历史
- `handleGetEvolutionIteration`: 获取单次迭代详情

### 4. 类型定义 ✅

**文件**: `nofx/evotypes/types.go`
- 添加 `UserID` 字段到 `EvolutionConfig`
- 定义完整的类型结构

**文件**: `nofx/store/evolution.go`
- 添加 `UpdateCurrentIteration()` 方法
- 添加 `UpdateBestVersion()` 方法

### 5. 前端组件 ✅

已创建完整的前端 UI：
- `EvolutionPage.tsx` - 主页面
- `EvolutionCard.tsx` - 任务卡片
- `CreateEvolutionModal.tsx` - 创建对话框
- `IterationTable.tsx` - 迭代历史表格
- `EvolutionDetailModal.tsx` - 任务详情
- `IterationDetailModal.tsx` - 迭代详情

### 6. 部署状态 ✅

- ✅ 后端已编译并部署到服务器（46MB）
- ⏳ 前端需要手动部署：`./local_build_deploy.sh frontend`

## 🎯 核心优势

1. **复用回测引擎**：不需要重新实现回测逻辑
2. **数据一致性**：回测结果统一存储在 `backtest_runs` 表
3. **功能完整性**：可以查看进化任务的完整回测详情（equity curve, trades, decisions）
4. **易于维护**：回测逻辑的改进会自动应用到进化任务
5. **清晰的职责分离**：Evolution Lab 只负责编排，Backtest 负责执行

## 📝 当前限制

### AI 功能使用 Stub 实现

当前版本的 AI 评估和优化功能使用简化的 stub 实现：

```go
// 6. Create stub evaluation (AI evaluation not yet implemented)
evaluation := &evotypes.EvaluationReport{
    Strengths:   []string{fmt.Sprintf("Return: %.2f%%", metrics.TotalReturnPct)},
    Weaknesses:  []string{"AI evaluation not yet implemented"},
    Suggestions: []string{"Manual review recommended"},
}

// 7. Create stub optimization (AI optimization not yet implemented)
optimization := &evotypes.OptimizationResult{
    Changes:        []string{"No changes - AI optimization not yet implemented"},
    NewPrompt:      promptVariant,
    ExpectedEffect: "No changes applied",
}
```

### 后续改进方向

1. **实现真正的 AI 评估**：
   - 分析回测指标
   - 分析交易决策样本
   - 生成结构化的评估报告

2. **实现真正的 AI 优化**：
   - 根据评估结果优化策略 prompt
   - 生成具体的修改建议
   - 预测优化效果

3. **增强前端功能**：
   - 添加"查看回测"按钮，跳转到回测详情页面
   - 可视化迭代进度和收敛曲线
   - 对比不同版本的策略表现

## 总结

Evolution Lab 现在是回测模块的**智能编排器**，通过调用现有的回测引擎来执行策略评估，并为未来的 AI 驱动优化预留了接口。
