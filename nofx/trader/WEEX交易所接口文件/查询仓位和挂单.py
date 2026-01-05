import time
import hmac
import hashlib
import base64
import requests
import json

api_key = "你的"
secret_key = "你的"
access_passphrase = "你的"


def generate_signature(secret_key, timestamp, method, request_path, query_string, body):
    message = timestamp + method.upper() + request_path + query_string + str(body)
    signature = hmac.new(secret_key.encode(), message.encode(), hashlib.sha256).digest()
    return base64.b64encode(signature).decode()


def generate_signature_get(secret_key, timestamp, method, request_path, query_string):
    message = timestamp + method.upper() + request_path + query_string
    signature = hmac.new(secret_key.encode(), message.encode(), hashlib.sha256).digest()
    return base64.b64encode(signature).decode()


def send_request_post(api_key, secret_key, access_passphrase, method, request_path, query_string, body):
    timestamp = str(int(time.time() * 1000))
    body = json.dumps(body)
    signature = generate_signature(secret_key, timestamp, method, request_path, query_string, body)

    headers = {
        "ACCESS-KEY": api_key,
        "ACCESS-SIGN": signature,
        "ACCESS-TIMESTAMP": timestamp,
        "ACCESS-PASSPHRASE": access_passphrase,
        "Content-Type": "application/json",
        "locale": "zh-CN"
    }

    # 配置v2rayN代理
    proxies = {
        'http': 'http://127.0.0.1:10808',
        'https': 'http://127.0.0.1:10808'
    }

    url = "https://api-contract.weex.com/"
    if method == "GET":
        response = requests.get(url + request_path, headers=headers, proxies=proxies)
    elif method == "POST":
        response = requests.post(url + request_path, headers=headers, data=body, proxies=proxies)
    return response


def send_request_get(api_key, secret_key, access_passphrase, method, request_path, query_string):
    timestamp = str(int(time.time() * 1000))
    signature = generate_signature_get(secret_key, timestamp, method, request_path, query_string)

    headers = {
        "ACCESS-KEY": api_key,
        "ACCESS-SIGN": signature,
        "ACCESS-TIMESTAMP": timestamp,
        "ACCESS-PASSPHRASE": access_passphrase,
        "Content-Type": "application/json",
        "locale": "zh-CN"
    }

    # 配置v2rayN代理
    proxies = {
        'http': 'http://127.0.0.1:10808',
        'https': 'http://127.0.0.1:10808'
    }

    url = "https://api-contract.weex.com/"
    if method == "GET":
        response = requests.get(url + request_path + query_string, headers=headers, proxies=proxies)
    return response


def get_account_balance():
    """获取账户信息列表"""
    request_path = "/capi/v2/account/getAccounts"
    query_string = ""
    response = send_request_get(api_key, secret_key, access_passphrase, "GET", request_path, query_string)
    print("账户余额信息:")
    print("状态码:", response.status_code)
    print("响应内容:")
    print(json.dumps(response.json(), indent=2, ensure_ascii=False))
    return response


def get_position():
    """获取单个持仓信息"""
    request_path = "/capi/v2/account/position/singlePosition"
    query_string = '?symbol=cmt_btcusdt'
    response = send_request_get(api_key, secret_key, access_passphrase, "GET", request_path, query_string)
    print("\n持仓信息:")
    print("状态码:", response.status_code)
    print("响应内容:", response.text)
    return response


def get_all_positions():
    """获取所有持仓信息"""
    request_path = "/capi/v2/account/position/allPosition"
    query_string = ""
    response = send_request_get(api_key, secret_key, access_passphrase, "GET", request_path, query_string)

    data = response.json()

    # API返回的是列表，不是字典
    if isinstance(data, list):
        positions = data
    else:
        # 如果是字典格式，尝试获取data字段
        positions = data.get('data', []) if isinstance(data, dict) else []

    print("\n" + "="*60)
    print("【持仓信息】")
    print("="*60)

    if positions:
        print(f"当前持仓数量: {len(positions)}\n")
        for pos in positions:
            # 提取币种名称（去掉cmt_前缀和usdt后缀）
            symbol = pos.get('symbol', '')
            coin = symbol.replace('cmt_', '').replace('usdt', '').upper()

            # 方向
            side = pos.get('side', '')
            side_cn = '多单' if side == 'LONG' else '空单'

            # 持仓量
            size = pos.get('size', '0')

            # 未实现盈亏
            unrealize_pnl = float(pos.get('unrealizePnl', 0))
            pnl_sign = '+' if unrealize_pnl >= 0 else ''

            # 开仓价值
            open_value = pos.get('open_value', '0')

            # 杠杆
            leverage = pos.get('leverage', '0')

            # 开仓时间
            created_time = pos.get('created_time', '')
            if created_time:
                timestamp = int(created_time) / 1000
                time_str = time.strftime('%m-%d %H:%M', time.localtime(timestamp))
            else:
                time_str = '未知'

            # 持仓ID
            position_id = pos.get('position_id', pos.get('id', ''))

            # 一行显示
            print(f"{coin:6} | {side_cn:4} | 数量:{size:8} | 价值:{open_value:10} | 杠杆:{leverage:3}x | 盈亏:{pnl_sign}{unrealize_pnl:8} | {time_str} | ID:{position_id}")
    else:
        print("当前无持仓\n")

    return response


def get_open_orders():
    """获取当前挂单列表"""
    request_path = "/capi/v2/order/openOrders"
    query_string = ""
    response = send_request_get(api_key, secret_key, access_passphrase, "GET", request_path, query_string)

    data = response.json()

    # 处理不同的返回格式
    if isinstance(data, list):
        orders = data
    elif isinstance(data, dict) and data.get('code') == '0':
        orders = data.get('data', [])
    else:
        orders = []

    print("\n" + "="*60)
    print("【挂单信息】")
    print("="*60)

    if orders:
        print(f"当前挂单数量: {len(orders)}\n")
        for order in orders:
            # 提取币种名称
            symbol = order.get('symbol', '')
            coin = symbol.replace('cmt_', '').replace('usdt', '').upper()

            # 方向
            order_type = order.get('type', '')
            side_cn = '买入(做多)' if order_type == '1' else '卖出(做空)'

            # 数量
            size = order.get('size', '0')

            # 价格
            price = order.get('price', '0')

            # 转换时间戳为可读格式
            create_time = order.get('create_time', '')
            if create_time:
                timestamp = int(create_time) / 1000
                time_str = time.strftime('%Y-%m-%d %H:%M:%S', time.localtime(timestamp))
            else:
                time_str = '未知'

            # 订单ID
            order_id = order.get('order_id', '')

            # 一行显示
            print(f"{coin:6} | {side_cn:10} | 数量:{size:8} | 挂单价:{price:10} | {time_str} | ID:{order_id}")
    else:
        print("当前无挂单\n")

    return response


def get_plan_orders():
    """获取计划委托列表（止损单、止盈单等）"""
    request_path = "/capi/v2/order/currentPlan"
    query_string = ""
    response = send_request_get(api_key, secret_key, access_passphrase, "GET", request_path, query_string)

    data = response.json()

    # API返回的是列表
    if isinstance(data, list):
        orders = data
    elif isinstance(data, dict) and data.get('code') == '0':
        orders = data.get('data', [])
    else:
        orders = []

    print("\n" + "="*60)
    print("【计划委托信息（止损/止盈）】")
    print("="*60)

    if orders:
        print(f"当前计划委托数量: {len(orders)}\n")
        for order in orders:
            # 提取币种名称
            symbol = order.get('symbol', '')
            coin = symbol.replace('cmt_', '').replace('usdt', '').upper()

            # 订单类型
            order_type = order.get('type', '')
            type_map = {
                '1': '开多', '2': '开空', '3': '平多', '4': '平空',
                '5': '减仓平多', '6': '减仓平空', '7': '协议平多', '8': '协议平空',
                '9': '爆仓平多', '10': '爆仓平空',
                'OPEN_LONG': '开多', 'OPEN_SHORT': '开空',
                'CLOSE_LONG': '平多', 'CLOSE_SHORT': '平空'
            }
            side_cn = type_map.get(order_type, order_type)

            # 数量
            size = order.get('size', '0')

            # 触发价格
            trigger_price = order.get('triggerPrice', '0')

            # 预设的止盈止损价格
            take_profit = order.get('presetTakeProfitPrice')
            stop_loss = order.get('presetStopLossPrice')

            # 判断订单类型
            if stop_loss and stop_loss != 'null' and stop_loss != '0' and stop_loss is not None:
                plan_type_cn = '止损'
            elif take_profit and take_profit != 'null' and take_profit != '0' and take_profit is not None:
                plan_type_cn = '止盈'
            else:
                plan_type_cn = '计划'

            # 转换时间戳为可读格式
            create_time = order.get('createTime', '')
            if create_time:
                timestamp = int(create_time) / 1000
                time_str = time.strftime('%m-%d %H:%M', time.localtime(timestamp))
            else:
                time_str = '未知'

            # 订单状态
            status = order.get('status', '')
            status_map = {
                '-1': '已撤销', '0': '等待中', '1': '部分成交', '2': '已成交',
                'UNTRIGGERED': '未触发', 'TRIGGERED': '已触发', 'CANCELLED': '已撤销'
            }
            status_cn = status_map.get(status, status)

            # 订单ID
            order_id = order.get('order_id', order.get('id', ''))

            # 一行显示
            print(f"{coin:6} | {plan_type_cn:4} | {side_cn:6} | 数量:{size:8} | 触发价:{trigger_price:10} | {status_cn:6} | {time_str} | ID:{order_id}")
    else:
        print("当前无计划委托\n")

    return response


def get_order_history(page_size=5):
    """获取历史订单记录"""
    request_path = "/capi/v2/order/history"
    # 查询最近90天的记录
    start_time = int((time.time() - 90 * 24 * 3600) * 1000)
    query_string = f"?pageSize={page_size}&createDate={start_time}"
    response = send_request_get(api_key, secret_key, access_passphrase, "GET", request_path, query_string)

    data = response.json()

    # 处理返回数据
    if isinstance(data, list):
        orders = data
    elif isinstance(data, dict) and data.get('code') == '0':
        orders = data.get('data', [])
    else:
        orders = []

    print("\n" + "="*60)
    print("【历史订单记录】")
    print("="*60)

    if orders:
        print(f"历史订单数量: {len(orders)}\n")
        for order in orders:
            # 提取币种名称
            symbol = order.get('symbol', '')
            coin = symbol.replace('cmt_', '').replace('usdt', '').upper()

            # 订单类型
            order_type = order.get('type', '')
            type_map = {
                'open_long': '开多', 'open_short': '开空',
                'close_long': '平多', 'close_short': '平空',
                'offset_liquidate_long': '减仓平多', 'offset_liquidate_short': '减仓平空',
                'agreement_close_long': '协议平多', 'agreement_close_short': '协议平空',
                'burst_liquidate_long': '爆仓平多', 'burst_liquidate_short': '爆仓平空'
            }
            side_cn = type_map.get(order_type, order_type)

            # 数量和成交信息
            size = order.get('size', '0')
            filled_qty = order.get('filled_qty', '0')
            price_avg = order.get('price_avg', '0')
            fee = order.get('fee', '0')
            total_profits = order.get('totalProfits', '0')

            # 订单状态
            status = order.get('status', '')
            status_map = {
                'pending': '处理中', 'open': '已挂单', 'filled': '已成交',
                'canceling': '取消中', 'canceled': '已撤销', 'untriggered': '未触发'
            }
            status_cn = status_map.get(status, status)

            # 转换时间戳
            create_time = order.get('createTime', '')
            if create_time:
                timestamp = int(create_time) / 1000
                time_str = time.strftime('%m-%d %H:%M', time.localtime(timestamp))
            else:
                time_str = '未知'

            # 订单ID
            order_id = order.get('order_id', order.get('id', ''))

            # 一行显示
            print(f"{coin:6} | {side_cn:8} | 数量:{size:8} | 均价:{price_avg:10} | 盈亏:{total_profits:8} | 手续费:{fee:6} | {status_cn:6} | {time_str} | ID:{order_id}")
    else:
        print("暂无历史订单\n")

    return response


def get_trade_fills(limit=5):
    """获取成交明细"""
    request_path = "/capi/v2/order/fills"
    query_string = f"?limit={limit}"
    response = send_request_get(api_key, secret_key, access_passphrase, "GET", request_path, query_string)

    data = response.json()

    # 处理返回数据
    if isinstance(data, dict):
        trades = data.get('list', [])
    elif isinstance(data, list):
        trades = data
    else:
        trades = []

    print("\n" + "="*60)
    print("【成交明细】")
    print("="*60)

    if trades:
        print(f"成交记录数量: {len(trades)}\n")
        for trade in trades:
            # 提取币种名称
            symbol = trade.get('symbol', '')
            coin = symbol.replace('cmt_', '').replace('usdt', '').upper()

            # 成交方向
            direction = trade.get('direction', '')
            direction_map = {
                'OPEN_LONG': '开多', 'OPEN_SHORT': '开空',
                'CLOSE_LONG': '平多', 'CLOSE_SHORT': '平空'
            }
            direction_cn = direction_map.get(direction, direction)

            # 成交信息
            fill_size = trade.get('fillSize', '0')
            fill_value = trade.get('fillValue', '0')
            fill_fee = trade.get('fillFee', '0')
            realize_pnl = trade.get('realizePnl', '0')

            # 转换时间戳
            created_time = trade.get('createdTime', '')
            if created_time:
                timestamp = int(created_time) / 1000
                time_str = time.strftime('%m-%d %H:%M', time.localtime(timestamp))
            else:
                time_str = '未知'

            # 成交ID
            trade_id = trade.get('trade_id', trade.get('id', trade.get('tradeId', '')))

            # 一行显示
            print(f"{coin:6} | {direction_cn:6} | 数量:{fill_size:8} | 价值:{fill_value:10} | 盈亏:{realize_pnl:8} | 手续费:{fill_fee:6} | {time_str} | ID:{trade_id}")
    else:
        print("暂无成交记录\n")

    return response


def place_order():
    """下单示例"""
    request_path = "/capi/v2/order/placeOrder"
    body = {
        "symbol": "cmt_ethusdt",
        "client_oid": "71557515757447",
        "size": "0.01",
        "type": "1",
        "order_type": "0",
        "match_price": "1",
        "price": "80000"
    }
    query_string = ""
    response = send_request_post(api_key, secret_key, access_passphrase, "POST", request_path, query_string, body)
    print("\n下单结果:")
    print("状态码:", response.status_code)
    print("响应内容:", response.text)
    return response


if __name__ == '__main__':
    # 查看账户余额
    get_account_balance()

    # 查看所有持仓信息
    get_all_positions()

    # 查看当前挂单信息
    get_open_orders()

    # 查看条件单信息（止损单、止盈单）
    get_plan_orders()

    # 查看历史订单记录(最近5条)
    get_order_history(page_size=5)

    # 查看成交明细(最近5条)
    get_trade_fills(limit=5)

    # 如果需要查看单个持仓信息，取消下面的注释
    # get_position()

    # 如果需要下单，取消下面的注释
    # place_order()