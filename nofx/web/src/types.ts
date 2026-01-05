export interface SystemStatus {
  trader_id: string;
  trader_name: string;
  ai_model: string;
  is_running: boolean;
  start_time: string;
  runtime_minutes: number;
  call_count: number;
  initial_balance: number;
  scan_interval: string;
  stop_until: string;
  last_reset_time: string;
  ai_provider: string;
}

export interface AccountInfo {
  total_equity: number;
  wallet_balance: number;
  unrealized_profit: number;
  available_balance: number;
  total_pnl: number;
  total_pnl_pct: number;
  total_unrealized_pnl: number;
  initial_balance: number;
  daily_pnl: number;
  position_count: number;
  margin_used: number;
  margin_used_pct: number;
  // 收益统计（可选，视后端提供情况）
  net_income?: number;
  income_window_days?: number;
  realized_pnl_sum?: number;
  funding_fee_sum?: number;
  commission_sum?: number;
  bonus_sum?: number;
  insurance_sum?: number;
  total_deposits?: number;
  total_withdrawals?: number;
}

export interface Position {
  symbol: string;
  side: string;
  entry_price: number;
  mark_price: number;
  quantity: number;
  leverage: number;
  unrealized_pnl: number;
  unrealized_pnl_pct: number;
  liquidation_price: number;
  margin_used: number;
}

export interface DecisionAction {
  action: string;
  symbol: string;
  quantity: number;
  leverage: number;
  price: number;
  order_id: number;
  timestamp: string;
  success: boolean;
  error?: string;
}

export interface AccountSnapshot {
  total_balance: number;
  available_balance: number;
  total_unrealized_profit: number;
  position_count: number;
  margin_used_pct: number;
}

export interface DecisionRecord {
  timestamp: string;
  cycle_number: number;
  input_prompt: string;
  cot_trace: string;
  decision_json: string;
  account_state: AccountSnapshot;
  positions: any[];
  candidate_coins: string[];
  decisions: DecisionAction[];
  execution_log: string[];
  success: boolean;
  error_message?: string;
  decision_type?: string; // "short_term" 或 "long_term"
}

export interface Statistics {
  total_cycles: number;
  successful_cycles: number;
  failed_cycles: number;
  total_open_positions: number;
  total_close_positions: number;
}

export interface OrderHistoryItem {
  order_id: number;
  client_order_id?: string;
  symbol: string;
  side: string;
  status: string;
  type: string;
  orig_type?: string;
  position_side?: string;
  time_in_force?: string;
  working_type?: string;
  reduce_only: boolean;
  close_position: boolean;
  avg_price: number;
  price: number;
  stop_price?: number | null;
  executed_qty: number;
  cum_quote: number;
  time: string;
  update_time: string;
  unix_time: number;
  unix_update: number;
}

export interface OrderHistoryStats {
  total_orders: number;
  buy_orders: number;
  sell_orders: number;
  total_volume: number;
  total_notional: number;
  symbols: string[];
  first_order_time?: string;
  last_order_time?: string;
}

export interface OrderHistoryResponse {
  trader_id: string;
  total: number;
  returned: number;
  limit: number;
  last_update?: string;
  orders: OrderHistoryItem[];
  stats: OrderHistoryStats;
}

export interface ManualOverride {
  override_id: string;
  trader_id: string;
  symbol: string;
  open_order_id: number;
  close_order_id: number;
  reason?: string;
  confidence?: 'low' | 'medium' | 'high' | string;
  status: 'draft' | 'active' | 'archived' | string;
  created_by?: string;
  updated_by?: string;
  created_at: string;
  updated_at?: string;
  audit_log?: string[];
}

export interface SnapshotAutoStats {
  matched: number;
  unmatched_open: number;
  unmatched_close: number;
}

export interface SnapshotMeta {
  version: number;
  generated_at: string;
  overrides_applied: number;
  rebuild_all_trades?: boolean; // 标记是否通过"一键修复"生成
  auto_stats: SnapshotAutoStats;
}

export interface ManualOverridesResponse {
  overrides: ManualOverride[];
  snapshot_meta?: SnapshotMeta;
}

export interface TradeOutcome {
  symbol: string;
  side: string;
  quantity: number;
  leverage: number;
  open_price: number;
  close_price: number;
  position_value: number;
  margin_used: number;
  pn_l: number;
  pn_l_pct: number;
  duration: string;
  open_time: string;
  close_time: string;
  was_stop_loss: boolean;
  open_order_id?: number;
  close_order_id?: number;
  open_cycle_number?: number;
  close_cycle_number?: number;
  open_reasoning?: string;
  close_reasoning?: string;
}

export interface SymbolPerformance {
  symbol: string;
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  win_rate: number;
  total_pn_l: number;
  avg_pn_l: number;
}

export interface PerformanceAnalysis {
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  win_rate: number;
  avg_win: number;
  avg_loss: number;
  profit_factor: number;
  sharpe_ratio?: number;
  recent_trades: TradeOutcome[];
  symbol_stats: Record<string, SymbolPerformance>;
  best_symbol?: string;
  worst_symbol?: string;
  recent_cot?: { timestamp: string; summary: string }[];
}

export interface PromptAuditRecord {
  timestamp: string;
  actor: string;
  action: string;
  comment?: string;
  version: number;
  changes?: string[];
}

export interface GistPromptSummary {
  id: string;
  title: string;
  version: number;
  tags: string[];
  updated_at: string;
  approved: boolean;
  approved_by?: string;
  approved_at?: string;
  last_editor?: string;
}

export interface GistPromptDetail extends GistPromptSummary {
  content: string;
  scenarios: string[];
  variables: string[];
  constraints: string[];
  guardrail?: string[];
  metadata?: Record<string, string>;
  changelog?: string;
  created_at: string;
  target_metric?: string;
  audit_trail?: PromptAuditRecord[];
}

export interface GistPromptProposal {
  id: string;
  prompt_id: string;
  title: string;
  summary: string;
  changes: string[];
  score: number;
  confidence: number;
  risk_notes?: string[];
  status: string;
  author: string;
  created_at: string;
  updated_at: string;
  suggested_prompt: string;
  proposed_strategy?: string;
  metadata?: Record<string, string>;
}

export interface GistPromptComment {
  id: string;
  prompt_id: string;
  author: string;
  message: string;
  mentions?: string[];
  created_at: string;
}

export interface GistStrategyDraft {
  strategy_id: string;
  name: string;
  version: string;
  source_prompt: string;
  type: string;
  guardrail?: string[];
  content_yaml: string;
  metadata?: Record<string, string>;
}

export interface StrategyFileResponse {
  path: string;
  relative?: string;
  content: string;
}

// 新增：竞赛相关类型
export interface TraderInfo {
  trader_id: string;
  trader_name: string;
  ai_model: string;
}

export interface CompetitionTraderData {
  trader_id: string;
  trader_name: string;
  ai_model: string;
  total_equity: number;
  total_pnl: number;
  total_pnl_pct: number;
  position_count: number;
  margin_used_pct: number;
  call_count: number;
  is_running: boolean;
}

export interface CompetitionData {
  traders: CompetitionTraderData[];
  count: number;
}

export interface ExternalSignalEvent {
  id: string;
  symbol: string;
  source: string;
  level?: string;
  score?: number;
  confidence?: number;
  title?: string;
  summary?: string;
  tags?: string[];
  triggered_at: string;
  received_at: string;
}

export interface ExternalSignalMetrics {
  enabled: boolean;
  source: string;
  total_received: number;
  total_filtered: number;
  consecutive_errors: number;
  last_success_time?: string;
  last_error_message?: string;
  active_subscribers: number;
  last_cursor?: string;
  last_persist_success?: string;
}

export interface ExternalSignalRuntimeConfig {
  source: string;
  poll_interval: number;
  timeout: number;
  symbol_cooldown: number;
  event_cooldown: number;
  min_score: number;
  min_confidence: number;
  max_latency: number;
  storage_directory: string;
}

export interface ExternalSignalResponse {
  enabled: boolean;
  source?: string;
  items: ExternalSignalEvent[];
  metrics?: ExternalSignalMetrics;
  config?: ExternalSignalRuntimeConfig;
}

export interface ExternalSignalSnapshot {
  symbol: string;
  latest?: ExternalSignalEvent;
  history?: ExternalSignalEvent[];
  last_updated?: string;
}

export interface ExternalSignalHistoryResponse {
  enabled: boolean;
  symbol: string;
  snapshot: ExternalSignalSnapshot;
}

export interface CandidateEntry {
  symbol: string;
  source: string;
  level?: string;
  score?: number;
  confidence?: number;
  summary?: string;
  triggered_at?: string;
  expires_at?: string;
  event_id?: string;
}

export interface CandidateAction {
  symbol: string;
  action: string;
  level?: string;
  score?: number;
  reason?: string;
  at: string;
}

export interface CandidateState {
  enabled: boolean;
  source?: string;
  last_updated?: string;
  active: CandidateEntry[];
  history: CandidateAction[];
}

export interface CandidateStateResponse {
  trader_id: string;
  state: CandidateState;
}

// ==================== Grid Trading Types ====================

export interface GridTraderInfo {
  id: string;
  name: string;
  enabled: boolean;  // 是否启用
  exchange: string;  // 交易所
  is_running: boolean;
  is_paused: boolean;
  start_time: string;
  uptime_seconds: number;
  update_count: number;
  last_update_time: string;
  error_count: number;
  last_error?: string;
  current_symbols: string[];
  
  // 统计信息
  position_count?: number;       // 持仓数量
  order_count?: number;          // 订单总数
  pending_order_count?: number; // 待处理订单数
  total_unrealized_pnl?: number; // 总未实现盈亏
  
  // AI调度器信息
  ai_scheduler?: {
    enabled: boolean;
    next_schedule_time?: string;
    schedule_time: string;
    schedule_days: number;
    candidate_symbols_count: number;
    select_count: number;
  };
  
  // 资金分配信息
  fund_allocation?: {
    grid_allocation_pct: number;
    ai_allocation_pct: number;
    grid_balance?: number;
    grid_available?: number;
    grid_unrealized_pnl?: number;
  };
  
  // 网格配置摘要
  grid_config_summary?: {
    grid_spacing_pct: number;
    order_size_pct: number;
    leverage: number;
    max_positions: number;
  };
  
  config?: GridTraderConfig;
}

export interface GridTraderListResponse {
  workers: GridTraderInfo[];
  count: number;
}

export interface GridTraderDetailResponse {
  worker: GridTraderInfo;
}

export interface GridTraderConfig {
  id: string;
  name: string;
  enabled: boolean;
  exchange: string;
  ai_scheduler: GridAISchedulerConfig;
  grid_config: GridConfig;
  fund_allocation: GridFundAllocationConfig;
  execution_config: GridExecutionConfig;
}

export interface GridAISchedulerConfig {
  enabled: boolean;
  schedule_time: string;
  schedule_days: number;
  ai_model: string;
  deepseek_key?: string;
  qwen_key?: string;
  custom_api_url?: string;
  custom_api_key?: string;
  candidate_symbols: string[];
  select_count: number;
}

export interface GridConfig {
  grid_spacing_pct: number;
  order_size_pct: number;
  upper_bound_pct: number;
  lower_bound_pct: number;
  leverage: number;
  max_positions: number;
  unilateral_protection: number;
  max_floating_loss_pct: number;
  max_slippage_pct: number;
}

export interface GridFundAllocationConfig {
  grid_allocation_pct: number;
  ai_allocation_pct: number;
}

export interface GridExecutionConfig {
  update_interval_seconds: number;
  order_timeout_seconds: number;
  max_retry_count: number;
}

export interface GridPosition {
  symbol: string;
  side: string;
  entry_price: number;
  mark_price: number;
  quantity: number;
  leverage: number;
  unrealized_pnl: number;
  unrealized_pnl_pct: number;
  grid_level: number;
}

export interface GridOrder {
  order_id: string;
  symbol: string;
  side: string;
  type: string;
  price: number;
  quantity: number;
  status: string;
  grid_level: number;
  timestamp: string;
  filled_time?: string;
}

export interface GridScheduleRecord {
  timestamp: string;
  selected_symbols: string[];
  reasoning: string;
  confidence: number;
  allocation: Record<string, number>;
  success: boolean;
  error?: string;
}

export interface GridStatistics {
  total_trades: number;
  win_trades: number;
  loss_trades: number;
  win_rate: number;
  total_pnl: number;
  realized_pnl: number;
  unrealized_pnl: number;
  max_drawdown: number;
  sharpe_ratio: number;
  grid_efficiency: number;
}

export interface GridEquityPoint {
  timestamp: string;
  equity: number;
  total_pnl: number;
  floating_pnl: number;
  realized_pnl: number;
}

export interface GridTrade {
  trade_id: string;
  symbol: string;
  action: string;
  price: number;
  quantity: number;
  leverage: number;
  grid_level: number;
  pnl?: number;
  timestamp: string;
}

export interface GridFundAllocation {
  total_equity: number;
  grid_allocation_pct: number;
  ai_allocation_pct: number;
  grid: {
    initial_allocation: number;
    current_balance: number;
    realized_pnl: number;
    unrealized_pnl: number;
    margin_used: number;
    total_pnl: number;
  };
  ai: {
    initial_allocation: number;
    current_balance: number;
    realized_pnl: number;
    unrealized_pnl: number;
    margin_used: number;
    total_pnl: number;
  };
}

// Baseline Strategy Pool Types - Base types first
export interface BaselineSignalThresholds {
  rsi_oversold: number
  rsi_overbought: number
  stoch_oversold: number
  stoch_overbought: number
  min_signal_count: number
}

export interface BaselineRiskManagement {
  equity_multiplier: number
  leverage: number
  hard_stop_loss_pct: number
  trailing_tp1_pct: number
  trailing_tp1_lock: number
  trailing_tp2_pct: number
  trailing_tp2_lock: number
  trailing_tp3_pct: number
  trailing_tp3_lock: number
  trailing_sl1_pct: number
  trailing_sl1_lock: number
  trailing_sl2_pct: number
  trailing_sl2_lock: number
}

export interface BaselineConfig {
  rsi_period: number
  macd_fast: number
  macd_slow: number
  macd_signal: number
  ema_period: number
  stoch_rsi_period: number
  atr_period: number
  signal_thresholds: BaselineSignalThresholds
  risk_management: BaselineRiskManagement
}

export interface BaselineStrategyStats {
  total_runs: number
  avg_return_pct: number
  avg_drawdown_pct: number
  avg_sharpe_ratio: number
  avg_win_rate: number
  best_return_pct: number
  worst_return_pct: number
}

export interface BaselineStrategy {
  id: string
  user_id: string
  name: string
  description: string
  config: BaselineConfig
  is_system_default: boolean
  stats?: BaselineStrategyStats
  created_at: string
  updated_at: string
}

export interface BaselineStrategyPerformance {
  id: number
  baseline_strategy_id: string
  run_id: string
  symbols: string[]
  timeframe: string
  start_ts: number
  end_ts: number
  initial_balance: number
  final_equity: number
  total_return_pct: number
  max_drawdown_pct: number
  sharpe_ratio: number
  win_rate: number
  total_trades: number
  created_at: string
}
