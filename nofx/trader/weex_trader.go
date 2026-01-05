package trader

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"nofx/logger"
	"strconv"
	"strings"
	"sync"
	"time"
)

// WeexTrader WEEX USDT 永续合约交易器
type WeexTrader struct {
	apiKey           string
	secretKey        string
	accessPassphrase string
	baseURL          string

	// 余额缓存
	cachedBalance     map[string]interface{}
	balanceCacheTime  time.Time
	balanceCacheMutex sync.RWMutex

	// 持仓缓存
	cachedPositions     []map[string]interface{}
	positionsCacheTime  time.Time
	positionsCacheMutex sync.RWMutex

	// 交易对精度缓存 (symbol -> qtyStep)
	qtyStepCache      map[string]float64
	qtyStepCacheMutex sync.RWMutex

	// 保证金模式缓存 (symbol -> marginMode: 1=全仓, 3=逐仓)
	marginModeCache      map[string]int
	marginModeCacheMutex sync.RWMutex

	// 待设置的止盈止损价格 (symbol -> price)
	// WEEX需要在开仓时直接设置止盈止损，而不是开仓后单独设置
	pendingStopLoss      map[string]float64
	pendingTakeProfit    map[string]float64
	pendingPricesMutex   sync.RWMutex

	// 缓存时长（15秒）
	cacheDuration time.Duration

	// HTTP 客户端
	httpClient *http.Client
}

// NewWeexTrader 创建 WEEX 交易器
func NewWeexTrader(apiKey, secretKey, accessPassphrase string) *WeexTrader {
	trader := &WeexTrader{
		apiKey:            apiKey,
		secretKey:         secretKey,
		accessPassphrase:  accessPassphrase,
		baseURL:           "https://api-contract.weex.com",
		cacheDuration:     15 * time.Second,
		qtyStepCache:      make(map[string]float64),
		marginModeCache:   make(map[string]int),
		pendingStopLoss:   make(map[string]float64),
		pendingTakeProfit: make(map[string]float64),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	logger.Infof("🟢 [WEEX] 交易器初始化完成")

	return trader
}

// generateSignature 生成 WEEX API 签名
// 签名消息格式: timestamp + method + request_path + query_string + body
// 使用 HMAC-SHA256 + Base64 编码
func (t *WeexTrader) generateSignature(timestamp, method, requestPath, queryString, body string) string {
	message := timestamp + strings.ToUpper(method) + requestPath + queryString + body
	h := hmac.New(sha256.New, []byte(t.secretKey))
	h.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature
}

// sendRequest 发送 HTTP 请求到 WEEX API
func (t *WeexTrader) sendRequest(method, requestPath, queryString string, body interface{}) (map[string]interface{}, error) {
	respBody, err := t.sendRequestRaw(method, requestPath, queryString, body)
	if err != nil {
		return nil, err
	}

	// 解析响应为对象
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return result, nil
}

// sendRequestRaw 发送 HTTP 请求到 WEEX API 并返回原始响应体
func (t *WeexTrader) sendRequestRaw(method, requestPath, queryString string, body interface{}) ([]byte, error) {
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())

	// 构建请求体
	var bodyStr string
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyStr = string(bodyBytes)
	}

	// 生成签名（GET请求不包含body，POST请求包含body）
	var signature string
	if method == "GET" {
		// GET请求：签名消息 = timestamp + method + request_path + query_string
		message := timestamp + strings.ToUpper(method) + requestPath + queryString
		h := hmac.New(sha256.New, []byte(t.secretKey))
		h.Write([]byte(message))
		signature = base64.StdEncoding.EncodeToString(h.Sum(nil))
		logger.Debugf("🔍 [WEEX] GET签名消息: %s", message)
	} else {
		// POST请求：签名消息 = timestamp + method + request_path + query_string + body
		signature = t.generateSignature(timestamp, method, requestPath, queryString, bodyStr)
		logger.Debugf("🔍 [WEEX] POST签名消息: %s", timestamp+strings.ToUpper(method)+requestPath+queryString+bodyStr)
	}

	// 构建完整 URL
	url := t.baseURL + requestPath
	if queryString != "" {
		url += queryString
	}

	// 创建请求
	var req *http.Request
	var err error
	if method == "GET" {
		req, err = http.NewRequest("GET", url, nil)
	} else if method == "POST" {
		req, err = http.NewRequest("POST", url, strings.NewReader(bodyStr))
	} else {
		return nil, fmt.Errorf("不支持的请求方法: %s", method)
	}

	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("ACCESS-KEY", t.apiKey)
	req.Header.Set("ACCESS-SIGN", signature)
	req.Header.Set("ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("ACCESS-PASSPHRASE", t.accessPassphrase)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("locale", "zh-CN")

	// 发送请求
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetBalance 获取账户余额
func (t *WeexTrader) GetBalance() (map[string]interface{}, error) {
	// 检查缓存
	t.balanceCacheMutex.RLock()
	if t.cachedBalance != nil && time.Since(t.balanceCacheTime) < t.cacheDuration {
		balance := t.cachedBalance
		t.balanceCacheMutex.RUnlock()
		return balance, nil
	}
	t.balanceCacheMutex.RUnlock()

	// 调用 WEEX API 获取账户资产
	// GET /capi/v2/account/assets
	// 注意：WEEX API 直接返回数组，不是对象包装的数组
	respBody, err := t.sendRequestRaw("GET", "/capi/v2/account/assets", "", nil)
	if err != nil {
		return nil, fmt.Errorf("获取账户余额失败: %w", err)
	}

	// 解析响应数据（返回的是数组）
	// 响应格式: [{coinName, available, equity, frozen, unrealizePnl}]
	var assets []map[string]interface{}
	if err := json.Unmarshal(respBody, &assets); err != nil {
		return nil, fmt.Errorf("解析账户余额失败: %w", err)
	}

	// 查找 USDT 资产
	var totalEquity, availableBalance, unrealizedPnl float64
	for _, asset := range assets {
		coinName, _ := asset["coinName"].(string)
		if coinName == "USDT" {
			if equityStr, ok := asset["equity"].(string); ok {
				totalEquity, _ = strconv.ParseFloat(equityStr, 64)
			}
			if availStr, ok := asset["available"].(string); ok {
				availableBalance, _ = strconv.ParseFloat(availStr, 64)
			}
			if uplStr, ok := asset["unrealizePnl"].(string); ok {
				unrealizedPnl, _ = strconv.ParseFloat(uplStr, 64)
			}
			break
		}
	}

	// 构建统一格式的余额信息
	balance := map[string]interface{}{
		"totalEquity":           totalEquity,
		"totalWalletBalance":    totalEquity - unrealizedPnl, // 钱包余额 = 总权益 - 未实现盈亏
		"availableBalance":      availableBalance,
		"totalUnrealizedProfit": unrealizedPnl,
		"balance":               totalEquity, // 兼容其他交易所格式
	}

	// 更新缓存
	t.balanceCacheMutex.Lock()
	t.cachedBalance = balance
	t.balanceCacheTime = time.Now()
	t.balanceCacheMutex.Unlock()

	logger.Infof("✓ [WEEX] 获取账户余额成功: 总权益=%.2f, 可用=%.2f, 未实现盈亏=%.2f",
		totalEquity, availableBalance, unrealizedPnl)

	return balance, nil
}

// GetPositions 获取所有持仓
func (t *WeexTrader) GetPositions() ([]map[string]interface{}, error) {
	// 检查缓存
	t.positionsCacheMutex.RLock()
	if t.cachedPositions != nil && time.Since(t.positionsCacheTime) < t.cacheDuration {
		positions := t.cachedPositions
		t.positionsCacheMutex.RUnlock()
		return positions, nil
	}
	t.positionsCacheMutex.RUnlock()

	// 调用 WEEX API 获取所有持仓
	// GET /capi/v2/account/position/allPosition
	respBody, err := t.sendRequestRaw("GET", "/capi/v2/account/position/allPosition", "", nil)
	if err != nil {
		return nil, fmt.Errorf("获取持仓失败: %w", err)
	}

	// 解析响应数据（返回的是数组）
	var rawPositions []map[string]interface{}
	if err := json.Unmarshal(respBody, &rawPositions); err != nil {
		return nil, fmt.Errorf("解析持仓数据失败: %w", err)
	}

	// 转换为统一格式
	var positions []map[string]interface{}
	for _, rawPos := range rawPositions {
		// 解析持仓数量
		sizeStr, _ := rawPos["size"].(string)
		size, _ := strconv.ParseFloat(sizeStr, 64)

		// 跳过空持仓
		if size == 0 {
			continue
		}

		// 解析其他字段
		symbol, _ := rawPos["symbol"].(string)
		sideStr, _ := rawPos["side"].(string)
		leverageStr, _ := rawPos["leverage"].(string)
		leverage, _ := strconv.ParseFloat(leverageStr, 64)
		unrealizePnlStr, _ := rawPos["unrealizePnl"].(string)
		unrealizePnl, _ := strconv.ParseFloat(unrealizePnlStr, 64)
		liquidatePriceStr, _ := rawPos["liquidatePrice"].(string)
		liquidatePrice, _ := strconv.ParseFloat(liquidatePriceStr, 64)
		openValueStr, _ := rawPos["open_value"].(string)
		openValue, _ := strconv.ParseFloat(openValueStr, 64)

		// 计算入场价格（开仓价值 / 持仓数量）
		var entryPrice float64
		if size > 0 {
			entryPrice = openValue / size
		}

		// 获取当前市场价格作为标记价格
		markPrice, err := t.GetMarketPrice(symbol)
		if err != nil {
			logger.Infof("⚠️ [WEEX] 获取 %s 市场价格失败: %v", symbol, err)
			markPrice = entryPrice // 使用入场价格作为备用
		}

		// 转换持仓方向（LONG -> long, SHORT -> short）
		side := strings.ToLower(sideStr)

		// 计算持仓数量（多仓为正，空仓为负）
		positionAmt := size
		if side == "short" {
			positionAmt = -size
		}

		// 转换保证金模式（转换为小写，与前端期望一致）
		// WEEX返回: "SHARED"(全仓) 或 "ISOLATED"(逐仓)
		// 前端期望: "crossed"/"cross"(全仓) 或 "isolated"(逐仓)
		marginMode, _ := rawPos["margin_mode"].(string)
		if marginMode == "SHARED" {
			marginMode = "crossed" // 小写，与币安的"cross"格式一致
		} else if marginMode == "ISOLATED" {
			marginMode = "isolated" // 小写
		}

		// 确定持仓方向（用于查询止损止盈）
		positionSide := "LONG"
		if side == "short" {
			positionSide = "SHORT"
		}

		// 查询止损止盈订单
		stopLoss, takeProfit := t.getStopOrders(symbol, positionSide)

		// 将WEEX格式的symbol转换为标准格式（去掉cmt_前缀，转大写）
		// 例如: "cmt_btcusdt" -> "BTCUSDT"
		// 这样可以与AI决策的symbol格式匹配，同时下单时normalizeSymbol会自动转换回WEEX格式
		standardSymbol := strings.ToUpper(strings.TrimPrefix(symbol, "cmt_"))

		// 构建统一格式的持仓信息
		position := map[string]interface{}{
			"symbol":           standardSymbol, // 使用标准格式，方便与AI决策匹配
			"side":             side,
			"positionAmt":      positionAmt,
			"entryPrice":       entryPrice,
			"markPrice":        markPrice,
			"unRealizedProfit": unrealizePnl,
			"unrealizedPnL":    unrealizePnl,
			"liquidationPrice": liquidatePrice,
			"leverage":         leverage,
			"margin_type":      marginMode,  // CROSSED 或 ISOLATED
			"stop_loss":        stopLoss,    // 止损价格
			"take_profit":      takeProfit,  // 止盈价格
		}

		positions = append(positions, position)
	}

	// 更新缓存
	t.positionsCacheMutex.Lock()
	t.cachedPositions = positions
	t.positionsCacheTime = time.Now()
	t.positionsCacheMutex.Unlock()

	logger.Infof("✓ [WEEX] 获取持仓成功，共 %d 个持仓", len(positions))

	return positions, nil
}

// normalizeSymbol 将标准交易对格式转换为WEEX格式
// 例如: "BTCUSDT" -> "cmt_btcusdt"
func (t *WeexTrader) normalizeSymbol(symbol string) string {
	// 转换为小写
	symbol = strings.ToLower(symbol)
	// 如果已经有cmt_前缀，直接返回
	if strings.HasPrefix(symbol, "cmt_") {
		return symbol
	}
	// 添加cmt_前缀
	return "cmt_" + symbol
}

func weexStringValue(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case json.Number:
		return val.String()
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func weexFloatValue(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case nil:
		return 0, false
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	case string:
		s := strings.TrimSpace(val)
		if s == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		return f, err == nil
	default:
		s := strings.TrimSpace(weexStringValue(v))
		if s == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		return f, err == nil
	}
}

func weexMapString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			s := strings.TrimSpace(weexStringValue(v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func weexMapFloat(m map[string]interface{}, keys ...string) (float64, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if f, ok := weexFloatValue(v); ok {
				return f, true
			}
		}
	}
	return 0, false
}

func weexPlanOrderIsActive(status interface{}) bool {
	if f, ok := weexFloatValue(status); ok {
		// 文档: 0=等待触发/成交，-1=撤销，1/2=成交相关
		return f == 0
	}
	s := strings.ToUpper(strings.TrimSpace(weexStringValue(status)))
	switch s {
	case "UNTRIGGERED", "PENDING", "WAITING", "0":
		return true
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n == 0
	}
	return false
}

// weexPlanOrderCloseSide returns which position side this plan order is closing: "LONG"/"SHORT".
// WEEX currentPlan may return:
// - numeric: "3"(平多)/"4"(平空) ... as documented
// - string: "CLOSE_LONG"(平多)/"CLOSE_SHORT"(平空) (observed)
func weexPlanOrderCloseSide(orderType string) (string, bool) {
	typ := strings.ToUpper(strings.TrimSpace(orderType))
	if typ == "" {
		return "", false
	}
	if n, err := strconv.Atoi(typ); err == nil {
		switch n {
		case 3, 5, 7, 9:
			return "LONG", true
		case 4, 6, 8, 10:
			return "SHORT", true
		default:
			return "", false
		}
	}
	switch {
	case strings.HasPrefix(typ, "CLOSE_LONG"):
		return "LONG", true
	case strings.HasPrefix(typ, "CLOSE_SHORT"):
		return "SHORT", true
	default:
		return "", false
	}
}

// getStopOrders 查询止损止盈订单（简化版）
// 返回: stopLoss价格（0表示未设置），takeProfit价格（0表示未设置）
func (t *WeexTrader) getStopOrders(symbol string, positionSide string) (float64, float64) {
	var stopLoss, takeProfit float64
	positionSide = strings.ToUpper(strings.TrimSpace(positionSide))

	// 查询当前所有计划委托订单
	queryString := fmt.Sprintf("?symbol=%s", symbol)
	respBody, err := t.sendRequestRaw("GET", "/capi/v2/order/currentPlan", queryString, nil)
	if err != nil {
		return 0, 0
	}

	// 解析订单列表
	var orders []map[string]interface{}
	if err := json.Unmarshal(respBody, &orders); err != nil {
		return 0, 0
	}

	if len(orders) == 0 {
		return 0, 0
	}

	// 获取当前市场价格，用于判断是止损还是止盈
	marketPrice, err := t.GetMarketPrice(symbol)
	if err != nil {
		return 0, 0
	}

	// 遍历所有计划委托，筛选出止损止盈单
	for _, order := range orders {
		orderType := weexMapString(order, "type")
		triggerPrice, ok := weexMapFloat(order, "triggerPrice", "trigger_price")
		if !ok {
			continue
		}
		statusRaw, _ := order["status"]

		// 只处理未触发的订单
		if !weexPlanOrderIsActive(statusRaw) {
			continue
		}

		// 检查订单类型是否为平仓订单
		_, ok = weexPlanOrderCloseSide(orderType)
		if !ok {
			continue
		}

		// 根据持仓方向和触发价格判断止损/止盈
		if positionSide == "LONG" {
			if triggerPrice < marketPrice {
				// 止损单：取最高的止损价格
				if stopLoss == 0 || triggerPrice > stopLoss {
					stopLoss = triggerPrice
				}
			} else {
				// 止盈单：取最低的止盈价格
				if takeProfit == 0 || triggerPrice < takeProfit {
					takeProfit = triggerPrice
				}
			}
		} else if positionSide == "SHORT" {
			if triggerPrice > marketPrice {
				// 止损单：取最低的止损价格
				if stopLoss == 0 || triggerPrice < stopLoss {
					stopLoss = triggerPrice
				}
			} else {
				// 止盈单：取最高的止盈价格
				if takeProfit == 0 || triggerPrice > takeProfit {
					takeProfit = triggerPrice
				}
			}
		}
	}

	return stopLoss, takeProfit
}

// OpenLong 开多仓
func (t *WeexTrader) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)
	logger.Infof("[WEEX] 开多仓: %s 数量: %.6f 杠杆: %dx", symbol, quantity, leverage)

	// 1. 取消所有挂单（清理旧订单）
	if err := t.CancelAllOrders(symbol); err != nil {
		logger.Infof("  ⚠️ 取消旧挂单失败: %v", err)
	}

	// 2. 取消所有计划委托订单（包括止损止盈）
	if err := t.CancelPlanOrders(symbol); err != nil {
		logger.Infof("  ⚠️ 取消计划委托订单失败: %v", err)
	}

	// 3. 设置保证金模式为逐仓
	if err := t.SetMarginMode(symbol, false); err != nil {
		logger.Infof("  ⚠️ 设置保证金模式失败: %v", err)
	}

	// 4. 设置杠杆
	if err := t.SetLeverage(symbol, leverage); err != nil {
		logger.Infof("  ⚠️ 设置杠杆失败: %v", err)
	}

	// 格式化数量
	quantityStr, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return nil, err
	}

	// 生成唯一的订单ID
	clientOid := t.generateOrderID()

	// 5. 下单（使用逐仓模式）
	body := map[string]interface{}{
		"symbol":      symbol,
		"client_oid":  clientOid,
		"size":        quantityStr,
		"type":        "1",        // 1:开多
		"order_type":  "3",        // 3:立即成交并取消剩余（IOC）
		"match_price": "1",        // 1:使用市价
		"price":       "0",        // 市价单价格填0
		"marginMode":  3,          // 逐仓模式
	}

	// ✅ 添加预设的止盈止损价格（如果有的话）
	// 获取合约信息以确定价格精度（避免浮点精度问题）
	contractInfo, err := t.GetContractInfo(symbol)
	priceDecimals := 4 // 默认4位小数
	if err == nil {
		if tickSizeStr, ok := contractInfo["tick_size"].(string); ok {
			if val, err := strconv.ParseFloat(tickSizeStr, 64); err == nil {
				priceDecimals = int(val)
			}
		}
	}

	t.pendingPricesMutex.RLock()
	if stopLoss, ok := t.pendingStopLoss[symbol]; ok && stopLoss > 0 {
		// 格式化为字符串以避免浮点精度问题
		priceStr := fmt.Sprintf(fmt.Sprintf("%%.%df", priceDecimals), stopLoss)
		body["presetStopLossPrice"] = priceStr
		logger.Infof("  ✓ [WEEX] 开仓时设置止损价格: %s", priceStr)
	}
	if takeProfit, ok := t.pendingTakeProfit[symbol]; ok && takeProfit > 0 {
		// 格式化为字符串以避免浮点精度问题
		priceStr := fmt.Sprintf(fmt.Sprintf("%%.%df", priceDecimals), takeProfit)
		body["presetTakeProfitPrice"] = priceStr
		logger.Infof("  ✓ [WEEX] 开仓时设置止盈价格: %s", priceStr)
	}
	t.pendingPricesMutex.RUnlock()

	result, err := t.sendRequest("POST", "/capi/v2/order/placeOrder", "", body)
	if err != nil {
		return nil, fmt.Errorf("开多仓失败: %w", err)
	}

	// 解析返回结果
	orderID, _ := result["order_id"].(string)
	logger.Infof("✓ [WEEX] 开多仓成功: %s 数量: %s, 订单ID: %s", symbol, quantityStr, orderID)

	// 清除缓存
	t.clearCache()

	// 清除已使用的预设价格
	t.pendingPricesMutex.Lock()
	delete(t.pendingStopLoss, symbol)
	delete(t.pendingTakeProfit, symbol)
	t.pendingPricesMutex.Unlock()

	return map[string]interface{}{
		"orderId": orderID,
		"symbol":  symbol,
		"status":  "NEW",
	}, nil
}

// OpenShort 开空仓
func (t *WeexTrader) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)
	logger.Infof("[WEEX] 开空仓: %s 数量: %.6f 杠杆: %dx", symbol, quantity, leverage)

	// 1. 取消所有挂单（清理旧订单）
	if err := t.CancelAllOrders(symbol); err != nil {
		logger.Infof("  ⚠️ 取消旧挂单失败: %v", err)
	}

	// 2. 取消所有计划委托订单（包括止损止盈）
	if err := t.CancelPlanOrders(symbol); err != nil {
		logger.Infof("  ⚠️ 取消计划委托订单失败: %v", err)
	}

	// 3. 设置保证金模式为逐仓
	if err := t.SetMarginMode(symbol, false); err != nil {
		logger.Infof("  ⚠️ 设置保证金模式失败: %v", err)
	}

	// 4. 设置杠杆
	if err := t.SetLeverage(symbol, leverage); err != nil {
		logger.Infof("  ⚠️ 设置杠杆失败: %v", err)
	}

	// 格式化数量
	quantityStr, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return nil, err
	}

	// 生成唯一的订单ID
	clientOid := t.generateOrderID()

	// 5. 下单（使用逐仓模式）
	body := map[string]interface{}{
		"symbol":      symbol,
		"client_oid":  clientOid,
		"size":        quantityStr,
		"type":        "2",        // 2:开空
		"order_type":  "3",        // 3:立即成交并取消剩余（IOC）
		"match_price": "1",        // 1:使用市价
		"price":       "0",        // 市价单价格填0
		"marginMode":  3,          // 逐仓模式
	}

	// ✅ 添加预设的止盈止损价格（如果有的话）
	// 获取合约信息以确定价格精度（避免浮点精度问题）
	contractInfo, err := t.GetContractInfo(symbol)
	priceDecimals := 4 // 默认4位小数
	if err == nil {
		if tickSizeStr, ok := contractInfo["tick_size"].(string); ok {
			if val, err := strconv.ParseFloat(tickSizeStr, 64); err == nil {
				priceDecimals = int(val)
			}
		}
	}

	t.pendingPricesMutex.RLock()
	if stopLoss, ok := t.pendingStopLoss[symbol]; ok && stopLoss > 0 {
		// 格式化为字符串以避免浮点精度问题
		priceStr := fmt.Sprintf(fmt.Sprintf("%%.%df", priceDecimals), stopLoss)
		body["presetStopLossPrice"] = priceStr
		logger.Infof("  ✓ [WEEX] 开仓时设置止损价格: %s", priceStr)
	}
	if takeProfit, ok := t.pendingTakeProfit[symbol]; ok && takeProfit > 0 {
		// 格式化为字符串以避免浮点精度问题
		priceStr := fmt.Sprintf(fmt.Sprintf("%%.%df", priceDecimals), takeProfit)
		body["presetTakeProfitPrice"] = priceStr
		logger.Infof("  ✓ [WEEX] 开仓时设置止盈价格: %s", priceStr)
	}
	t.pendingPricesMutex.RUnlock()

	result, err := t.sendRequest("POST", "/capi/v2/order/placeOrder", "", body)
	if err != nil {
		return nil, fmt.Errorf("开空仓失败: %w", err)
	}

	// 解析返回结果
	orderID, _ := result["order_id"].(string)
	logger.Infof("✓ [WEEX] 开空仓成功: %s 数量: %s, 订单ID: %s", symbol, quantityStr, orderID)

	// 清除缓存
	t.clearCache()

	// 清除已使用的预设价格
	t.pendingPricesMutex.Lock()
	delete(t.pendingStopLoss, symbol)
	delete(t.pendingTakeProfit, symbol)
	t.pendingPricesMutex.Unlock()

	return map[string]interface{}{
		"orderId": orderID,
		"symbol":  symbol,
		"status":  "NEW",
	}, nil
}

// CloseLong 平多仓
func (t *WeexTrader) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	// 保存原始symbol用于查找持仓（GetPositions返回的是标准格式）
	originalSymbol := strings.ToUpper(symbol)

	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// 如果 quantity = 0，获取当前持仓数量
	if quantity == 0 {
		positions, err := t.GetPositions()
		if err != nil {
			return nil, err
		}
		for _, pos := range positions {
			side, _ := pos["side"].(string)
			posSymbol, _ := pos["symbol"].(string)
			// 使用标准格式比较（GetPositions返回的symbol是标准格式）
			if posSymbol == originalSymbol && strings.ToLower(side) == "long" {
				quantity = pos["positionAmt"].(float64)
				break
			}
		}
	}

	if quantity <= 0 {
		return nil, fmt.Errorf("没有多仓可平")
	}

	// 格式化数量
	quantityStr, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return nil, err
	}

	// 生成唯一的订单ID
	clientOid := t.generateOrderID()

	// 获取保证金模式（必须与开仓时一致）
	marginMode := t.getMarginMode(symbol)

	// 调用 WEEX API 平仓
	body := map[string]interface{}{
		"symbol":      symbol,
		"client_oid":  clientOid,
		"size":        quantityStr,
		"type":        "3",        // 3:平多
		"order_type":  "3",        // 3:立即成交并取消剩余（IOC）
		"match_price": "1",        // 1:市价
		"price":       "0",        // 市价单价格填0
		"marginMode":  marginMode, // 必须与开仓时一致
	}

	result, err := t.sendRequest("POST", "/capi/v2/order/placeOrder", "", body)
	if err != nil {
		return nil, fmt.Errorf("平多仓失败: %w", err)
	}

	// 解析返回结果
	orderID, _ := result["order_id"].(string)
	logger.Infof("✓ [WEEX] 平多仓成功: %s 数量: %s, 订单ID: %s", symbol, quantityStr, orderID)

	// 清除缓存
	t.clearCache()

	return map[string]interface{}{
		"orderId": orderID,
		"symbol":  symbol,
		"status":  "NEW",
	}, nil
}

// CloseShort 平空仓
func (t *WeexTrader) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	// 保存原始symbol用于查找持仓（GetPositions返回的是标准格式）
	originalSymbol := strings.ToUpper(symbol)

	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// 如果 quantity = 0，获取当前持仓数量
	if quantity == 0 {
		positions, err := t.GetPositions()
		if err != nil {
			return nil, err
		}
		for _, pos := range positions {
			side, _ := pos["side"].(string)
			posSymbol, _ := pos["symbol"].(string)
			// 使用标准格式比较（GetPositions返回的symbol是标准格式）
			if posSymbol == originalSymbol && strings.ToLower(side) == "short" {
				quantity = -pos["positionAmt"].(float64) // 空仓是负数
				break
			}
		}
	}

	if quantity <= 0 {
		return nil, fmt.Errorf("没有空仓可平")
	}

	// 格式化数量
	quantityStr, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return nil, err
	}

	// 生成唯一的订单ID
	clientOid := t.generateOrderID()

	// 获取保证金模式（必须与开仓时一致）
	marginMode := t.getMarginMode(symbol)

	// 调用 WEEX API 平仓
	body := map[string]interface{}{
		"symbol":      symbol,
		"client_oid":  clientOid,
		"size":        quantityStr,
		"type":        "4",        // 4:平空
		"order_type":  "3",        // 3:立即成交并取消剩余（IOC）
		"match_price": "1",        // 1:市价
		"price":       "0",        // 市价单价格填0
		"marginMode":  marginMode, // 必须与开仓时一致
	}

	result, err := t.sendRequest("POST", "/capi/v2/order/placeOrder", "", body)
	if err != nil {
		return nil, fmt.Errorf("平空仓失败: %w", err)
	}

	// 解析返回结果
	orderID, _ := result["order_id"].(string)
	logger.Infof("✓ [WEEX] 平空仓成功: %s 数量: %s, 订单ID: %s", symbol, quantityStr, orderID)

	// 清除缓存
	t.clearCache()

	return map[string]interface{}{
		"orderId": orderID,
		"symbol":  symbol,
		"status":  "NEW",
	}, nil
}

// SetLeverage 设置杠杆
func (t *WeexTrader) SetLeverage(symbol string, leverage int) error {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	leverageStr := fmt.Sprintf("%d", leverage)

	// 获取保证金模式（从持仓中智能推断，包括其他交易对）
	marginMode := t.getMarginMode(symbol)

	// 调用 WEEX API 设置杠杆
	body := map[string]interface{}{
		"symbol":         symbol,
		"marginMode":     marginMode,
		"longLeverage":   leverageStr,
		"shortLeverage":  leverageStr,
	}

	result, err := t.sendRequest("POST", "/capi/v2/account/leverage", "", body)
	if err != nil {
		// 如果杠杆已经是目标值，忽略错误
		if strings.Contains(err.Error(), "No need to change") || strings.Contains(err.Error(), "leverage not modified") {
			logger.Infof("  ✓ [WEEX] %s 杠杆已经是 %dx", symbol, leverage)
			return nil
		}
		return fmt.Errorf("设置杠杆失败: %w", err)
	}

	// 检查返回码
	if code, ok := result["code"].(string); ok && code == "200" {
		logger.Infof("  ✓ [WEEX] %s 杠杆设置为 %dx", symbol, leverage)
		return nil
	}

	// 如果返回码不是 200，返回错误信息
	if msg, ok := result["msg"].(string); ok {
		return fmt.Errorf("设置杠杆失败: %s", msg)
	}

	return nil
}

// SetMarginMode 设置保证金模式
func (t *WeexTrader) SetMarginMode(symbol string, isCrossMargin bool) error {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// 确定目标保证金模式
	targetMode := 3 // 逐仓
	targetModeStr := "逐仓"
	if isCrossMargin {
		targetMode = 1 // 全仓
		targetModeStr = "全仓"
	}

	// 先查询账户实际使用的保证金模式
	actualMode := t.queryActualMarginMode()
	actualModeStr := "逐仓"
	if actualMode == 1 {
		actualModeStr = "全仓"
	}
	logger.Infof("  ℹ️ [WEEX] 账户当前保证金模式: %s (mode=%d)", actualModeStr, actualMode)

	// 如果账户当前模式与目标模式一致，直接缓存并返回
	if actualMode == targetMode {
		t.marginModeCacheMutex.Lock()
		t.marginModeCache[symbol] = actualMode
		t.marginModeCacheMutex.Unlock()
		logger.Infof("  ✓ [WEEX] %s 保证金模式已是 %s，无需切换", symbol, actualModeStr)
		return nil
	}

	// 尝试切换保证金模式
	positions, err := t.GetPositions()
	if err != nil {
		logger.Infof("  ⚠️ [WEEX] 获取持仓信息失败: %v，使用账户当前模式", err)
		// 使用账户当前的模式
		t.marginModeCacheMutex.Lock()
		t.marginModeCache[symbol] = actualMode
		t.marginModeCacheMutex.Unlock()
		return nil
	}

	// 查找该交易对的持仓，获取当前杠杆
	currentLeverage := 10 // 默认杠杆
	for _, pos := range positions {
		posSymbol, _ := pos["symbol"].(string)
		standardSymbol := strings.ToUpper(strings.TrimPrefix(symbol, "cmt_"))
		if posSymbol == standardSymbol {
			if lev, ok := pos["leverage"].(float64); ok {
				currentLeverage = int(lev)
			}
			break
		}
	}

	// 调用杠杆接口尝试切换保证金模式
	err = t.setMarginModeWithLeverage(symbol, targetMode, currentLeverage)
	if err != nil {
		logger.Infof("  ⚠️ [WEEX] 切换保证金模式失败: %v，使用账户当前模式 %s", err, actualModeStr)
		// 切换失败，使用账户当前的模式
		t.marginModeCacheMutex.Lock()
		t.marginModeCache[symbol] = actualMode
		t.marginModeCacheMutex.Unlock()
		return nil
	}

	// 切换成功，缓存目标模式
	t.marginModeCacheMutex.Lock()
	t.marginModeCache[symbol] = targetMode
	t.marginModeCacheMutex.Unlock()
	logger.Infof("  ✓ [WEEX] %s 保证金模式切换成功: %s", symbol, targetModeStr)
	return nil
}

// queryActualMarginMode 直接查询账户的实际保证金模式（不使用缓存）
// 从持仓信息中获取，如果没有持仓则默认返回3（逐仓）
func (t *WeexTrader) queryActualMarginMode() int {
	// 直接从持仓查询，不使用缓存
	positions, err := t.GetPositions()
	if err == nil && len(positions) > 0 {
		// 从第一个持仓获取保证金模式（假设账户所有币对使用相同模式）
		for _, pos := range positions {
			if marginType, ok := pos["margin_type"].(string); ok {
				if marginType == "crossed" {
					return 1 // 全仓
				} else if marginType == "isolated" {
					return 3 // 逐仓
				}
			}
		}
	}
	// 如果没有持仓，默认返回3（逐仓）
	return 3
}

// getMarginModeSimple 简化版保证金模式获取
// 优先级：1.缓存（SetMarginMode设置的） 2.持仓信息 3.默认值（全仓）
func (t *WeexTrader) getMarginModeSimple(symbol string) int {
	// 1. 优先从缓存获取（SetMarginMode会缓存目标模式）
	t.marginModeCacheMutex.RLock()
	if mode, ok := t.marginModeCache[symbol]; ok {
		t.marginModeCacheMutex.RUnlock()
		return mode
	}
	t.marginModeCacheMutex.RUnlock()

	// 2. 从持仓中检测
	positions, err := t.GetPositions()
	if err == nil && len(positions) > 0 {
		// 将symbol转换为标准格式用于比较（GetPositions返回的是标准格式）
		standardSymbol := strings.ToUpper(strings.TrimPrefix(symbol, "cmt_"))

		// 查找该交易对的持仓
		for _, pos := range positions {
			posSymbol, _ := pos["symbol"].(string)
			if posSymbol == standardSymbol {
				if marginType, ok := pos["margin_type"].(string); ok {
					mode := 1 // 全仓
					if marginType == "isolated" {
						mode = 3 // 逐仓
					}
					// 缓存检测到的模式
					t.marginModeCacheMutex.Lock()
					t.marginModeCache[symbol] = mode
					t.marginModeCacheMutex.Unlock()
					return mode
				}
			}
		}
	}

	// 3. 使用默认值（全仓）
	return 1
}

// getMarginMode 智能获取保证金模式（保留用于兼容性）
// 优先级: 1.缓存 2.该交易对持仓 3.其他交易对持仓 4.默认值(全仓)
func (t *WeexTrader) getMarginMode(symbol string) int {
	// 1. 优先从缓存中获取
	t.marginModeCacheMutex.RLock()
	if mode, ok := t.marginModeCache[symbol]; ok {
		t.marginModeCacheMutex.RUnlock()
		logger.Infof("  [WEEX] 从缓存获取保证金模式: %s (mode=%d)", symbol, mode)
		return mode
	}
	t.marginModeCacheMutex.RUnlock()

	// 2. 从持仓中检测
	positions, err := t.GetPositions()
	if err == nil && len(positions) > 0 {
		// 将symbol转换为标准格式用于比较（GetPositions返回的是标准格式）
		standardSymbol := strings.ToUpper(strings.TrimPrefix(symbol, "cmt_"))

		// 2.1 优先查找该交易对的持仓
		for _, pos := range positions {
			posSymbol, _ := pos["symbol"].(string)
			if posSymbol == standardSymbol {
				if marginType, ok := pos["margin_type"].(string); ok {
					mode := 1 // 全仓
					if marginType == "isolated" { // 小写，与GetPositions返回的格式一致
						mode = 3 // 逐仓
					}
					logger.Infof("  [WEEX] 从持仓获取保证金模式: %s (mode=%d, marginType=%s)", symbol, mode, marginType)

					// 缓存检测到的保证金模式
					t.marginModeCacheMutex.Lock()
					t.marginModeCache[symbol] = mode
					t.marginModeCacheMutex.Unlock()

					return mode
				}
			}
		}

		// 2.2 如果该交易对没有持仓，从其他交易对的持仓中推断
		// 假设所有交易对使用相同的保证金模式
		for _, pos := range positions {
			if marginType, ok := pos["margin_type"].(string); ok {
				mode := 1 // 全仓
				if marginType == "isolated" { // 修复：使用小写判断
					mode = 3 // 逐仓
				}
				otherSymbol, _ := pos["symbol"].(string)
				logger.Infof("  [WEEX] 从其他交易对(%s)推断保证金模式: %s (mode=%d, marginType=%s)", otherSymbol, symbol, mode, marginType)

				// 缓存推断的保证金模式
				t.marginModeCacheMutex.Lock()
				t.marginModeCache[symbol] = mode
				t.marginModeCacheMutex.Unlock()

				return mode
			}
		}
	}

	// 3. 使用默认值（全仓）
	logger.Infof("  [WEEX] 使用默认保证金模式: %s (mode=1, 全仓)", symbol)
	return 1
}

// setMarginModeWithLeverage 通过杠杆接口设置保证金模式
func (t *WeexTrader) setMarginModeWithLeverage(symbol string, marginMode int, leverage int) error {
	leverageStr := fmt.Sprintf("%d", leverage)

	body := map[string]interface{}{
		"symbol":         symbol,
		"marginMode":     marginMode, // 1=全仓, 3=逐仓
		"longLeverage":   leverageStr,
		"shortLeverage":  leverageStr,
	}

	result, err := t.sendRequest("POST", "/capi/v2/account/leverage", "", body)
	if err != nil {
		// 如果保证金模式已经是目标值，可能会返回错误，忽略这种情况
		if strings.Contains(err.Error(), "No need to change") || strings.Contains(err.Error(), "margin mode not modified") {
			return nil
		}
		return fmt.Errorf("设置保证金模式失败: %w", err)
	}

	// 检查返回码
	if code, ok := result["code"].(string); ok && code == "200" {
		return nil
	}

	// 如果返回码不是 200，返回错误信息
	if msg, ok := result["msg"].(string); ok {
		return fmt.Errorf("设置保证金模式失败: %s", msg)
	}

	return nil
}

// GetMarketPrice 获取市场价格
func (t *WeexTrader) GetMarketPrice(symbol string) (float64, error) {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// 调用 WEEX API 获取市场行情
	// GET /capi/v2/market/ticker?symbol=cmt_btcusdt
	queryString := fmt.Sprintf("?symbol=%s", symbol)
	result, err := t.sendRequest("GET", "/capi/v2/market/ticker", queryString, nil)
	if err != nil {
		return 0, fmt.Errorf("获取市场价格失败: %w", err)
	}

	// 解析最新价格
	// 响应格式: {last, markPrice, indexPrice, ...}
	var price float64
	if lastStr, ok := result["last"].(string); ok {
		price, err = strconv.ParseFloat(lastStr, 64)
		if err != nil {
			return 0, fmt.Errorf("解析价格失败: %w", err)
		}
	} else if lastFloat, ok := result["last"].(float64); ok {
		price = lastFloat
	} else {
		return 0, fmt.Errorf("未找到价格数据")
	}

	return price, nil
}

// SetStopLoss 设置止损单
// ✅ WEEX特殊处理：
// - 如果有持仓：创建计划委托订单
// - 如果无持仓：存储到pending map，开仓时通过presetStopLossPrice参数设置
func (t *WeexTrader) SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// ✅ 修复：从合约信息中获取 tick_size 和 priceEndStep 来计算 stepSize
	contractInfo, err := t.GetContractInfo(symbol)
	var stepSize float64 = 0.1 // 默认stepSize
	var priceDecimals int = 4   // 默认4位小数
	if err == nil {
		var tickSize float64 = 1.0
		var priceEndStep float64 = 1.0

		// 解析 tick_size（可能是字符串）
		if tickSizeStr, ok := contractInfo["tick_size"].(string); ok {
			if val, err := strconv.ParseFloat(tickSizeStr, 64); err == nil && val > 0 {
				tickSize = val
				priceDecimals = int(val)
			}
		}

		// 解析 priceEndStep（可能是数字或字符串）
		if priceEndStepFloat, ok := contractInfo["priceEndStep"].(float64); ok && priceEndStepFloat > 0 {
			priceEndStep = priceEndStepFloat
		} else if priceEndStepInt, ok := contractInfo["priceEndStep"].(int); ok && priceEndStepInt > 0 {
			priceEndStep = float64(priceEndStepInt)
		} else if priceEndStepStr, ok := contractInfo["priceEndStep"].(string); ok {
			if val, err := strconv.ParseFloat(priceEndStepStr, 64); err == nil && val > 0 {
				priceEndStep = val
			}
		}

		// 计算 stepSize = priceEndStep / (10 ^ tick_size)
		stepSize = priceEndStep / math.Pow(10, tickSize)
	}

	// 对于空仓止损，向上取整（更高的止损价格，更安全）
	// 对于多仓止损，向下取整（更低的止损价格，更安全）
	var alignedPrice float64
	if positionSide == "SHORT" {
		alignedPrice = math.Ceil(stopPrice/stepSize) * stepSize
	} else {
		alignedPrice = math.Floor(stopPrice/stepSize) * stepSize
	}

	// 检查是否有持仓
	positions, err := t.GetPositions()
	hasPosition := false
	if err == nil {
		standardSymbol := strings.ToUpper(strings.TrimPrefix(symbol, "cmt_"))
		for _, pos := range positions {
			posSymbol, _ := pos["symbol"].(string)
			if posSymbol == standardSymbol {
				hasPosition = true
				break
			}
		}
	}

	// 如果有持仓，创建计划委托订单
	if hasPosition {
		return t.createStopLossPlanOrder(symbol, positionSide, quantity, alignedPrice, priceDecimals)
	}

	// 如果没有持仓，存储止损价格，在开仓时使用
	t.pendingPricesMutex.Lock()
	t.pendingStopLoss[symbol] = alignedPrice
	t.pendingPricesMutex.Unlock()

	logger.Infof("  ✓ [WEEX] 止损价格已存储: %s @ %.4f (将在开仓时设置)", symbol, alignedPrice)
	return nil
}

// createStopLossPlanOrder 创建计划委托止损单（用于已有持仓）
func (t *WeexTrader) createStopLossPlanOrder(symbol string, positionSide string, quantity, triggerPrice float64, priceDecimals int) error {
	// 格式化数量
	quantityStr, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return fmt.Errorf("格式化数量失败: %w", err)
	}

	// 生成唯一的订单ID
	clientOid := t.generateOrderID()

	// 确定订单类型（3:平多 4:平空）
	orderType := "3" // 平多
	if positionSide == "SHORT" {
		orderType = "4" // 平空
	}

	// 获取保证金模式
	marginMode := t.getMarginMode(symbol)

	// 格式化触发价格为字符串（避免浮点精度问题）
	triggerPriceStr := fmt.Sprintf(fmt.Sprintf("%%.%df", priceDecimals), triggerPrice)

	// 调用计划委托API
	body := map[string]interface{}{
		"symbol":        symbol,
		"client_oid":    clientOid,
		"size":          quantityStr,
		"type":          orderType,
		"match_type":    "1",              // 1:市价
		"execute_price": triggerPriceStr,  // 执行价格=触发价格
		"trigger_price": triggerPriceStr,  // 触发价格
		"marginMode":    marginMode,
	}

	result, err := t.sendRequest("POST", "/capi/v2/order/plan_order", "", body)
	if err != nil {
		return fmt.Errorf("创建计划委托止损单失败: %w", err)
	}

	// 解析返回结果
	orderID, _ := result["order_id"].(string)
	logger.Infof("  ✓ [WEEX] 计划委托止损单创建成功: %s @ %s, 订单ID: %s", symbol, triggerPriceStr, orderID)

	return nil
}

// SetTakeProfit 设置止盈单
// ✅ WEEX特殊处理：
// - 如果有持仓：创建计划委托订单
// - 如果无持仓：存储到pending map，开仓时通过presetTakeProfitPrice参数设置
func (t *WeexTrader) SetTakeProfit(symbol string, positionSide string, quantity, takeProfitPrice float64) error {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// ✅ 修复：从合约信息中获取 tick_size 和 priceEndStep 来计算 stepSize
	contractInfo, err := t.GetContractInfo(symbol)
	var stepSize float64 = 0.1 // 默认stepSize
	var priceDecimals int = 4   // 默认4位小数
	if err == nil {
		var tickSize float64 = 1.0
		var priceEndStep float64 = 1.0

		// 解析 tick_size（可能是字符串）
		if tickSizeStr, ok := contractInfo["tick_size"].(string); ok {
			if val, err := strconv.ParseFloat(tickSizeStr, 64); err == nil && val > 0 {
				tickSize = val
				priceDecimals = int(val)
			}
		}

		// 解析 priceEndStep（可能是数字或字符串）
		if priceEndStepFloat, ok := contractInfo["priceEndStep"].(float64); ok && priceEndStepFloat > 0 {
			priceEndStep = priceEndStepFloat
		} else if priceEndStepInt, ok := contractInfo["priceEndStep"].(int); ok && priceEndStepInt > 0 {
			priceEndStep = float64(priceEndStepInt)
		} else if priceEndStepStr, ok := contractInfo["priceEndStep"].(string); ok {
			if val, err := strconv.ParseFloat(priceEndStepStr, 64); err == nil && val > 0 {
				priceEndStep = val
			}
		}

		// 计算 stepSize = priceEndStep / (10 ^ tick_size)
		stepSize = priceEndStep / math.Pow(10, tickSize)
	}

	// 对于空仓止盈，向下取整（更低的止盈价格，更保守）
	// 对于多仓止盈，向上取整（更高的止盈价格，更保守）
	var alignedPrice float64
	if positionSide == "SHORT" {
		alignedPrice = math.Floor(takeProfitPrice/stepSize) * stepSize
	} else {
		alignedPrice = math.Ceil(takeProfitPrice/stepSize) * stepSize
	}

	// 检查是否有持仓
	positions, err := t.GetPositions()
	hasPosition := false
	if err == nil {
		standardSymbol := strings.ToUpper(strings.TrimPrefix(symbol, "cmt_"))
		for _, pos := range positions {
			posSymbol, _ := pos["symbol"].(string)
			if posSymbol == standardSymbol {
				hasPosition = true
				break
			}
		}
	}

	// 如果有持仓，创建计划委托订单
	if hasPosition {
		return t.createTakeProfitPlanOrder(symbol, positionSide, quantity, alignedPrice, priceDecimals)
	}

	// 如果没有持仓，存储止盈价格，在开仓时使用
	t.pendingPricesMutex.Lock()
	t.pendingTakeProfit[symbol] = alignedPrice
	t.pendingPricesMutex.Unlock()

	logger.Infof("  ✓ [WEEX] 止盈价格已存储: %s @ %.4f (将在开仓时设置)", symbol, alignedPrice)
	return nil
}

// createTakeProfitPlanOrder 创建计划委托止盈单（用于已有持仓）
func (t *WeexTrader) createTakeProfitPlanOrder(symbol string, positionSide string, quantity, triggerPrice float64, priceDecimals int) error {
	// 格式化数量
	quantityStr, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return fmt.Errorf("格式化数量失败: %w", err)
	}

	// 生成唯一的订单ID
	clientOid := t.generateOrderID()

	// 确定订单类型（3:平多 4:平空）
	orderType := "3" // 平多
	if positionSide == "SHORT" {
		orderType = "4" // 平空
	}

	// 获取保证金模式
	marginMode := t.getMarginMode(symbol)

	// 格式化触发价格为字符串（避免浮点精度问题）
	triggerPriceStr := fmt.Sprintf(fmt.Sprintf("%%.%df", priceDecimals), triggerPrice)

	// 调用计划委托API
	body := map[string]interface{}{
		"symbol":        symbol,
		"client_oid":    clientOid,
		"size":          quantityStr,
		"type":          orderType,
		"match_type":    "1",              // 1:市价
		"execute_price": triggerPriceStr,  // 执行价格=触发价格
		"trigger_price": triggerPriceStr,  // 触发价格
		"marginMode":    marginMode,
	}

	result, err := t.sendRequest("POST", "/capi/v2/order/plan_order", "", body)
	if err != nil {
		return fmt.Errorf("创建计划委托止盈单失败: %w", err)
	}

	// 解析返回结果
	orderID, _ := result["order_id"].(string)
	logger.Infof("  ✓ [WEEX] 计划委托止盈单创建成功: %s @ %s, 订单ID: %s", symbol, triggerPriceStr, orderID)

	return nil
}

// CancelStopLossOrders 取消止损单
func (t *WeexTrader) CancelStopLossOrders(symbol string) error {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// 查询当前所有计划委托订单
	// GET /capi/v2/order/currentPlan?symbol=xxx
	queryString := fmt.Sprintf("?symbol=%s", symbol)
	respBody, err := t.sendRequestRaw("GET", "/capi/v2/order/currentPlan", queryString, nil)
	if err != nil {
		return fmt.Errorf("获取计划委托失败: %w", err)
	}

	// 解析订单列表（返回的是数组）
	var orders []map[string]interface{}
	if err := json.Unmarshal(respBody, &orders); err != nil {
		return fmt.Errorf("解析计划委托列表失败: %w", err)
	}

	// 如果没有计划委托，直接返回
	if len(orders) == 0 {
		logger.Infof("  ℹ [WEEX] %s 没有计划委托需要取消", symbol)
		return nil
	}

	// 获取当前市场价格，用于判断是止损还是止盈
	marketPrice, err := t.GetMarketPrice(symbol)
	if err != nil {
		logger.Infof("⚠️ [WEEX] 获取市场价格失败: %v", err)
		return err
	}

	// 遍历所有计划委托，筛选出止损单并取消
	canceledCount := 0
	for _, order := range orders {
		orderID := weexMapString(order, "order_id", "orderId")
		orderType := weexMapString(order, "type")
		triggerPrice, ok := weexMapFloat(order, "triggerPrice", "trigger_price")
		if orderID == "" || !ok {
			continue
		}
		statusRaw, _ := order["status"]
		if !weexPlanOrderIsActive(statusRaw) {
			continue
		}

		// 判断是否为止损单
		isStopLoss := false
		closeSide, ok := weexPlanOrderCloseSide(orderType)
		if !ok {
			continue
		}
		if closeSide == "LONG" {
			// 触发价格 < 市场价格 = 止损单
			isStopLoss = triggerPrice < marketPrice
		} else if closeSide == "SHORT" {
			// 触发价格 > 市场价格 = 止损单
			isStopLoss = triggerPrice > marketPrice
		}

		// 如果不是止损单，跳过
		if !isStopLoss {
			continue
		}

		// 调用撤单接口
		// POST /capi/v2/order/cancel_plan
		body := map[string]interface{}{
			"orderId": orderID,
		}

		result, err := t.sendRequest("POST", "/capi/v2/order/cancel_plan", "", body)
		if err != nil {
			logger.Infof("  ⚠️ [WEEX] 取消止损单 %s 失败: %v", orderID, err)
			continue
		}

		// 检查取消结果
		if resultBool, ok := result["result"].(bool); ok && resultBool {
			canceledCount++
			logger.Infof("  ✓ [WEEX] 取消止损单成功: %s @ %.2f", orderID, triggerPrice)
		} else {
			errMsg, _ := result["err_msg"].(string)
			logger.Infof("  ⚠️ [WEEX] 取消止损单 %s 失败: %s", orderID, errMsg)
		}
	}

	if canceledCount > 0 {
		logger.Infof("  ✓ [WEEX] 取消了 %d 个止损单", canceledCount)
	}
	return nil
}

// CancelTakeProfitOrders 取消止盈单
func (t *WeexTrader) CancelTakeProfitOrders(symbol string) error {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// 查询当前所有计划委托订单
	// GET /capi/v2/order/currentPlan?symbol=xxx
	queryString := fmt.Sprintf("?symbol=%s", symbol)
	respBody, err := t.sendRequestRaw("GET", "/capi/v2/order/currentPlan", queryString, nil)
	if err != nil {
		return fmt.Errorf("获取计划委托失败: %w", err)
	}

	// 解析订单列表（返回的是数组）
	var orders []map[string]interface{}
	if err := json.Unmarshal(respBody, &orders); err != nil {
		return fmt.Errorf("解析计划委托列表失败: %w", err)
	}

	// 如果没有计划委托，直接返回
	if len(orders) == 0 {
		logger.Infof("  ℹ [WEEX] %s 没有计划委托需要取消", symbol)
		return nil
	}

	// 获取当前市场价格，用于判断是止损还是止盈
	marketPrice, err := t.GetMarketPrice(symbol)
	if err != nil {
		logger.Infof("⚠️ [WEEX] 获取市场价格失败: %v", err)
		return err
	}

	// 遍历所有计划委托，筛选出止盈单并取消
	canceledCount := 0
	for _, order := range orders {
		orderID := weexMapString(order, "order_id", "orderId")
		orderType := weexMapString(order, "type")
		triggerPrice, ok := weexMapFloat(order, "triggerPrice", "trigger_price")
		if orderID == "" || !ok {
			continue
		}
		statusRaw, _ := order["status"]
		if !weexPlanOrderIsActive(statusRaw) {
			continue
		}

		// 判断是否为止盈单
		isTakeProfit := false
		closeSide, ok := weexPlanOrderCloseSide(orderType)
		if !ok {
			continue
		}
		if closeSide == "LONG" {
			// 触发价格 > 市场价格 = 止盈单
			isTakeProfit = triggerPrice > marketPrice
		} else if closeSide == "SHORT" {
			// 触发价格 < 市场价格 = 止盈单
			isTakeProfit = triggerPrice < marketPrice
		}

		// 如果不是止盈单，跳过
		if !isTakeProfit {
			continue
		}

		// 调用撤单接口
		// POST /capi/v2/order/cancel_plan
		body := map[string]interface{}{
			"orderId": orderID,
		}

		result, err := t.sendRequest("POST", "/capi/v2/order/cancel_plan", "", body)
		if err != nil {
			logger.Infof("  ⚠️ [WEEX] 取消止盈单 %s 失败: %v", orderID, err)
			continue
		}

		// 检查取消结果
		if resultBool, ok := result["result"].(bool); ok && resultBool {
			canceledCount++
			logger.Infof("  ✓ [WEEX] 取消止盈单成功: %s @ %.2f", orderID, triggerPrice)
		} else {
			errMsg, _ := result["err_msg"].(string)
			logger.Infof("  ⚠️ [WEEX] 取消止盈单 %s 失败: %s", orderID, errMsg)
		}
	}

	if canceledCount > 0 {
		logger.Infof("  ✓ [WEEX] 取消了 %d 个止盈单", canceledCount)
	}
	return nil
}

// CancelAllOrders 取消所有挂单
func (t *WeexTrader) CancelAllOrders(symbol string) error {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// 先获取当前所有挂单
	// GET /capi/v2/order/current?symbol=xxx
	queryString := fmt.Sprintf("?symbol=%s", symbol)
	respBody, err := t.sendRequestRaw("GET", "/capi/v2/order/current", queryString, nil)
	if err != nil {
		return fmt.Errorf("获取当前挂单失败: %w", err)
	}

	// 解析订单列表（返回的是数组）
	var orders []map[string]interface{}
	if err := json.Unmarshal(respBody, &orders); err != nil {
		return fmt.Errorf("解析订单列表失败: %w", err)
	}

	// 如果没有挂单，直接返回
	if len(orders) == 0 {
		logger.Infof("  ℹ [WEEX] %s 没有挂单需要取消", symbol)
		return nil
	}

	// 遍历所有订单，逐个取消
	canceledCount := 0
	for _, order := range orders {
		orderID, _ := order["order_id"].(string)
		if orderID == "" {
			continue
		}

		// 调用撤单接口
		// POST /capi/v2/order/cancel_order
		body := map[string]interface{}{
			"orderId": orderID,
		}

		result, err := t.sendRequest("POST", "/capi/v2/order/cancel_order", "", body)
		if err != nil {
			logger.Infof("  ⚠️ [WEEX] 取消订单 %s 失败: %v", orderID, err)
			continue
		}

		// 检查取消结果
		if resultBool, ok := result["result"].(bool); ok && resultBool {
			canceledCount++
			logger.Infof("  ✓ [WEEX] 取消订单成功: %s", orderID)
		} else {
			errMsg, _ := result["err_msg"].(string)
			logger.Infof("  ⚠️ [WEEX] 取消订单 %s 失败: %s", orderID, errMsg)
		}
	}

	logger.Infof("  ✓ [WEEX] 取消了 %d/%d 个挂单", canceledCount, len(orders))
	return nil
}

// CancelStopOrders 取消止损止盈单
func (t *WeexTrader) CancelStopOrders(symbol string) error {
	if err := t.CancelStopLossOrders(symbol); err != nil {
		logger.Infof("⚠️ [WEEX] 取消止损单失败: %v", err)
	}
	if err := t.CancelTakeProfitOrders(symbol); err != nil {
		logger.Infof("⚠️ [WEEX] 取消止盈单失败: %v", err)
	}
	return nil
}

// CancelPlanOrders 取消所有计划委托订单（包括止损止盈）
func (t *WeexTrader) CancelPlanOrders(symbol string) error {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// 查询当前计划委托订单
	// GET /capi/v2/order/currentPlan?symbol=xxx
	queryString := fmt.Sprintf("?symbol=%s", symbol)
	respBody, err := t.sendRequestRaw("GET", "/capi/v2/order/currentPlan", queryString, nil)
	if err != nil {
		return fmt.Errorf("获取计划委托订单失败: %w", err)
	}

	// 解析订单列表（返回的是数组）
	var orders []map[string]interface{}
	if err := json.Unmarshal(respBody, &orders); err != nil {
		return fmt.Errorf("解析计划委托订单列表失败: %w", err)
	}

	// 如果没有计划委托订单，直接返回
	if len(orders) == 0 {
		logger.Infof("  ℹ [WEEX] %s 没有计划委托订单需要取消", symbol)
		return nil
	}

	// 遍历所有订单，逐个取消
	canceledCount := 0
	for _, order := range orders {
		orderID, _ := order["order_id"].(string)
		if orderID == "" {
			continue
		}

		// 调用撤单接口
		// POST /capi/v2/order/cancel_plan
		body := map[string]interface{}{
			"orderId": orderID,
		}

		result, err := t.sendRequest("POST", "/capi/v2/order/cancel_plan", "", body)
		if err != nil {
			logger.Infof("  ⚠️ [WEEX] 取消计划委托订单 %s 失败: %v", orderID, err)
			continue
		}

		// 检查取消结果
		if resultBool, ok := result["result"].(bool); ok && resultBool {
			canceledCount++
			logger.Infof("  ✓ [WEEX] 取消计划委托订单成功: %s", orderID)
		} else {
			errMsg, _ := result["err_msg"].(string)
			logger.Infof("  ⚠️ [WEEX] 取消计划委托订单 %s 失败: %s", orderID, errMsg)
		}
	}

	logger.Infof("  ✓ [WEEX] 取消了 %d/%d 个计划委托订单", canceledCount, len(orders))
	return nil
}

// FormatQuantity 格式化数量到正确精度
func (t *WeexTrader) FormatQuantity(symbol string, quantity float64) (string, error) {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// 获取该交易对的 qtyStep
	qtyStep := t.getQtyStep(symbol)

	// 根据 qtyStep 对齐数量（向下取整到最近的步长）
	alignedQty := math.Floor(quantity/qtyStep) * qtyStep

	// 计算需要的小数位数
	decimals := 0
	if qtyStep < 1 {
		stepStr := strconv.FormatFloat(qtyStep, 'f', -1, 64)
		if idx := strings.Index(stepStr, "."); idx >= 0 {
			decimals = len(stepStr) - idx - 1
		}
	}

	// 格式化
	format := fmt.Sprintf("%%.%df", decimals)
	formatted := fmt.Sprintf(format, alignedQty)

	return formatted, nil
}

// GetOrderStatus 获取订单状态
func (t *WeexTrader) GetOrderStatus(symbol string, orderID string) (map[string]interface{}, error) {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// 调用 WEEX API 获取订单状态
	// GET /capi/v2/order/current?orderId=xxx
	queryString := fmt.Sprintf("?orderId=%s", orderID)
	respBody, err := t.sendRequestRaw("GET", "/capi/v2/order/current", queryString, nil)
	if err != nil {
		return nil, fmt.Errorf("获取订单状态失败: %w", err)
	}

	// 解析订单列表（返回的是数组）
	var orders []map[string]interface{}
	if err := json.Unmarshal(respBody, &orders); err != nil {
		return nil, fmt.Errorf("解析订单数据失败: %w", err)
	}

	// 如果没有找到订单，返回错误
	if len(orders) == 0 {
		return nil, fmt.Errorf("订单 %s 不存在", orderID)
	}

	// 获取第一个订单（应该只有一个）
	order := orders[0]

	// 解析订单状态
	status, _ := order["status"].(string)
	priceAvgStr, _ := order["price_avg"].(string)
	priceAvg, _ := strconv.ParseFloat(priceAvgStr, 64)
	filledQtyStr, _ := order["filled_qty"].(string)
	filledQty, _ := strconv.ParseFloat(filledQtyStr, 64)
	feeStr, _ := order["fee"].(string)
	fee, _ := strconv.ParseFloat(feeStr, 64)

	// 转换状态为统一格式
	// WEEX: pending, open, filled, canceling, canceled, untriggered
	// 统一: NEW, FILLED, CANCELED, PARTIALLY_FILLED
	unifiedStatus := "NEW"
	switch status {
	case "filled":
		unifiedStatus = "FILLED"
	case "canceled":
		unifiedStatus = "CANCELED"
	case "open":
		if filledQty > 0 {
			unifiedStatus = "PARTIALLY_FILLED"
		} else {
			unifiedStatus = "NEW"
		}
	case "pending":
		unifiedStatus = "NEW"
	}

	result := map[string]interface{}{
		"orderId":     orderID,
		"symbol":      order["symbol"],
		"status":      unifiedStatus,
		"avgPrice":    priceAvg,
		"executedQty": filledQty,
		"commission":  fee,
	}

	return result, nil
}

// GetClosedPnL 获取已平仓盈亏记录
func (t *WeexTrader) GetClosedPnL(startTime time.Time, limit int) ([]ClosedPnLRecord, error) {
	// 调用 WEEX API 获取成交明细
	// GET /capi/v2/order/fills?startTime=xxx&limit=xxx
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}

	queryString := fmt.Sprintf("?startTime=%d&limit=%d", startTime.UnixMilli(), limit)
	respBody, err := t.sendRequestRaw("GET", "/capi/v2/order/fills", queryString, nil)
	if err != nil {
		return nil, fmt.Errorf("获取成交明细失败: %w", err)
	}

	// 解析响应数据
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析成交明细失败: %w", err)
	}

	// 获取成交明细列表
	list, ok := result["list"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("成交明细格式错误")
	}

	// 转换为 ClosedPnLRecord 格式
	var records []ClosedPnLRecord
	for _, item := range list {
		fill, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// 只处理有已实现盈亏的记录（平仓记录）
		realizePnlStr, _ := fill["realizePnl"].(string)
		realizePnl, _ := strconv.ParseFloat(realizePnlStr, 64)
		if realizePnl == 0 {
			continue // 跳过开仓记录
		}

		// 解析字段
		symbol, _ := fill["symbol"].(string)
		direction, _ := fill["direction"].(string)
		fillSizeStr, _ := fill["fillSize"].(string)
		fillSize, _ := strconv.ParseFloat(fillSizeStr, 64)
		fillValueStr, _ := fill["fillValue"].(string)
		fillValue, _ := strconv.ParseFloat(fillValueStr, 64)
		fillFeeStr, _ := fill["fillFee"].(string)
		fillFee, _ := strconv.ParseFloat(fillFeeStr, 64)
		createdTime, _ := fill["createdTime"].(float64)
		tradeID := fmt.Sprintf("%v", fill["tradeId"])

		// 计算价格
		var price float64
		if fillSize > 0 {
			price = fillValue / fillSize
		}

		// 确定持仓方向
		side := "long"
		if strings.Contains(strings.ToLower(direction), "short") || strings.Contains(strings.ToLower(direction), "空") {
			side = "short"
		}

		// 创建记录
		record := ClosedPnLRecord{
			Symbol:      symbol,
			Side:        side,
			EntryPrice:  0,           // WEEX 不提供入场价格
			ExitPrice:   price,       // 使用成交价格
			Quantity:    fillSize,
			RealizedPnL: realizePnl,
			Fee:         fillFee,
			ExitTime:    time.UnixMilli(int64(createdTime)),
			EntryTime:   time.UnixMilli(int64(createdTime)), // 使用相同时间
			OrderID:     tradeID,
			CloseType:   "unknown",
			ExchangeID:  tradeID,
		}

		records = append(records, record)
	}

	return records, nil
}

// 辅助方法

// generateOrderID 生成唯一的订单ID
// 格式: WEEX{时间戳}{随机数}
// 限制: 不超过40个字符
func (t *WeexTrader) generateOrderID() string {
	timestamp := time.Now().UnixNano() / 1000000 // 毫秒时间戳
	return fmt.Sprintf("WEEX%d", timestamp)
}

// clearCache 清除缓存
func (t *WeexTrader) clearCache() {
	t.balanceCacheMutex.Lock()
	t.cachedBalance = nil
	t.balanceCacheMutex.Unlock()

	t.positionsCacheMutex.Lock()
	t.cachedPositions = nil
	t.positionsCacheMutex.Unlock()
}

// getQtyStep 获取交易对的数量步长
func (t *WeexTrader) getQtyStep(symbol string) float64 {
	// 检查缓存
	t.qtyStepCacheMutex.RLock()
	if step, ok := t.qtyStepCache[symbol]; ok {
		t.qtyStepCacheMutex.RUnlock()
		return step
	}
	t.qtyStepCacheMutex.RUnlock()

	// 调用 WEEX API 获取交易对精度信息
	contractInfo, err := t.GetContractInfo(symbol)
	if err != nil {
		logger.Infof("⚠️ [WEEX] 获取合约信息失败: %v，使用默认精度", err)
		return 0.001 // 默认精度
	}

	// 从合约信息中获取 minOrderSize 作为步长
	minOrderSizeStr, _ := contractInfo["minOrderSize"].(string)
	minOrderSize, err := strconv.ParseFloat(minOrderSizeStr, 64)
	if err != nil || minOrderSize <= 0 {
		minOrderSize = 0.001
	}

	// 缓存结果
	t.qtyStepCacheMutex.Lock()
	t.qtyStepCache[symbol] = minOrderSize
	t.qtyStepCacheMutex.Unlock()

	return minOrderSize
}

// CalculatePositionSize 计算持仓大小
// 根据账户余额、风险百分比、价格和杠杆计算应该开仓的数量
func (t *WeexTrader) CalculatePositionSize(balance, riskPercent, price float64, leverage int) float64 {
	riskAmount := balance * (riskPercent / 100.0)
	positionValue := riskAmount * float64(leverage)
	quantity := positionValue / price
	return quantity
}

// GetContractInfo 获取合约信息
func (t *WeexTrader) GetContractInfo(symbol string) (map[string]interface{}, error) {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	// 调用 WEEX API 获取合约信息
	// GET /capi/v2/market/contracts?symbol=xxx
	queryString := fmt.Sprintf("?symbol=%s", symbol)
	respBody, err := t.sendRequestRaw("GET", "/capi/v2/market/contracts", queryString, nil)
	if err != nil {
		return nil, fmt.Errorf("获取合约信息失败: %w", err)
	}

	// 解析响应数据（返回的是数组）
	var contracts []map[string]interface{}
	if err := json.Unmarshal(respBody, &contracts); err != nil {
		return nil, fmt.Errorf("解析合约信息失败: %w", err)
	}

	// 如果没有找到合约信息，返回错误
	if len(contracts) == 0 {
		return nil, fmt.Errorf("未找到 %s 的合约信息", symbol)
	}

	// 返回第一个合约信息
	contract := contracts[0]

	// 🔍 调试日志：打印合约信息中的关键字段
	logger.Infof("  🔍 [WEEX] %s 合约信息: tick_size=%v, priceEndStep=%v", symbol, contract["tick_size"], contract["priceEndStep"])

	return contract, nil
}

// GetSymbolPrecision 获取交易对的数量精度（小数位数）
func (t *WeexTrader) GetSymbolPrecision(symbol string) (int, error) {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	contractInfo, err := t.GetContractInfo(symbol)
	if err != nil {
		return 3, err // 默认精度3
	}

	// 从 minOrderSize 计算精度
	minOrderSizeStr, _ := contractInfo["minOrderSize"].(string)
	minOrderSize, err := strconv.ParseFloat(minOrderSizeStr, 64)
	if err != nil || minOrderSize <= 0 {
		return 3, nil // 默认精度3
	}

	// 计算小数位数
	precision := calculatePrecisionFromValue(minOrderSize)
	logger.Infof("  [WEEX] %s 数量精度: %d (minOrderSize: %s)", symbol, precision, minOrderSizeStr)

	return precision, nil
}

// GetPricePrecision 获取交易对的价格精度（小数位数）
func (t *WeexTrader) GetPricePrecision(symbol string) (int, error) {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	contractInfo, err := t.GetContractInfo(symbol)
	if err != nil {
		return 2, err // 默认精度2
	}

	// 从 tick_size 计算精度
	tickSizeStr, _ := contractInfo["tick_size"].(string)
	tickSize, err := strconv.ParseFloat(tickSizeStr, 64)
	if err != nil || tickSize <= 0 {
		return 2, nil // 默认精度2
	}

	// 计算小数位数
	precision := calculatePrecisionFromValue(tickSize)
	logger.Infof("  [WEEX] %s 价格精度: %d (tick_size: %s)", symbol, precision, tickSizeStr)

	return precision, nil
}

// calculatePrecisionFromValue 从数值计算小数位数
func calculatePrecisionFromValue(value float64) int {
	if value >= 1 {
		return 0
	}

	// 转换为字符串并计算小数位数
	str := strconv.FormatFloat(value, 'f', -1, 64)
	if idx := strings.Index(str, "."); idx >= 0 {
		return len(str) - idx - 1
	}

	return 0
}

// GetMinNotional 获取最小名义价值（最小订单金额）
func (t *WeexTrader) GetMinNotional(symbol string) float64 {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	contractInfo, err := t.GetContractInfo(symbol)
	if err != nil {
		logger.Infof("⚠️ [WEEX] 获取合约信息失败: %v，使用默认最小名义价值", err)
		return 10.0 // 默认10 USDT
	}

	// 从 minOrderSize 计算最小名义价值
	minOrderSizeStr, _ := contractInfo["minOrderSize"].(string)
	minOrderSize, err := strconv.ParseFloat(minOrderSizeStr, 64)
	if err != nil || minOrderSize <= 0 {
		return 10.0
	}

	// 获取当前市场价格
	price, err := t.GetMarketPrice(symbol)
	if err != nil {
		logger.Infof("⚠️ [WEEX] 获取市场价格失败: %v，使用默认最小名义价值", err)
		return 10.0
	}

	// 计算最小名义价值 = 最小数量 * 价格
	minNotional := minOrderSize * price
	if minNotional < 10.0 {
		minNotional = 10.0 // 至少10 USDT
	}

	return minNotional
}

// CheckMinNotional 检查订单是否满足最小名义价值要求
func (t *WeexTrader) CheckMinNotional(symbol string, quantity float64) error {
	// 转换交易对格式为WEEX格式
	symbol = t.normalizeSymbol(symbol)

	price, err := t.GetMarketPrice(symbol)
	if err != nil {
		return fmt.Errorf("获取市场价格失败: %w", err)
	}

	notionalValue := quantity * price
	minNotional := t.GetMinNotional(symbol)

	if notionalValue < minNotional {
		return fmt.Errorf(
			"订单金额 %.2f USDT 低于最小要求 %.2f USDT (数量: %.4f, 价格: %.4f)",
			notionalValue, minNotional, quantity, price,
		)
	}

	return nil
}

// UploadAILog 上传AI交易日志（用于WEEX AI Wars比赛）
// orderId: 订单ID（可选）
// stage: AI参与阶段，如 "Strategy Generation", "Decision Making", "Risk Assessment"
// model: AI模型名称/版本，如 "GPT-4-turbo", "Claude-3"
// input: 输入数据（提示词/查询）
// output: 输出数据（模型结果）
// explanation: 逻辑说明（最多1000字符）
func (t *WeexTrader) UploadAILog(orderId int64, stage, model string, input, output map[string]interface{}, explanation string) error {
	// 构建请求体
	body := map[string]interface{}{
		"stage":       stage,
		"model":       model,
		"input":       input,
		"output":      output,
		"explanation": explanation,
	}

	// 如果有订单ID，添加到请求体
	if orderId > 0 {
		body["orderId"] = orderId
	}

	// 发送请求
	result, err := t.sendRequest("POST", "/capi/v2/order/uploadAiLog", "", body)
	if err != nil {
		return fmt.Errorf("上传AI日志失败: %w", err)
	}

	// 检查响应
	code, _ := result["code"].(string)
	if code != "00000" {
		msg, _ := result["msg"].(string)
		return fmt.Errorf("上传AI日志失败: code=%s, msg=%s", code, msg)
	}

	logger.Infof("🤖 [WEEX] AI日志上传成功: stage=%s, model=%s", stage, model)
	return nil
}
