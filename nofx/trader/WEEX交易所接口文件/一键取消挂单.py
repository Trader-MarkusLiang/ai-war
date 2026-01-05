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
    body_str = json.dumps(body)
    signature = generate_signature(secret_key, timestamp, method, request_path, query_string, body_str)

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
    response = requests.post(url + request_path, headers=headers, data=body_str, proxies=proxies)
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
    response = requests.get(url + request_path + query_string, headers=headers, proxies=proxies)
    return response


def get_open_orders():
    """获取所有普通挂单"""
    request_path = "/capi/v2/order/openOrders"
    query_string = ""
    response = send_request_get(api_key, secret_key, access_passphrase, "GET", request_path, query_string)

    data = response.json()
    if isinstance(data, list):
        return data
    elif isinstance(data, dict) and data.get('code') == '0':
        return data.get('data', [])
    return []


def get_plan_orders():
    """获取所有计划委托"""
    request_path = "/capi/v2/order/currentPlan"
    query_string = ""
    response = send_request_get(api_key, secret_key, access_passphrase, "GET", request_path, query_string)

    data = response.json()
    if isinstance(data, list):
        return data
    elif isinstance(data, dict) and data.get('code') == '0':
        return data.get('data', [])
    return []


def cancel_order(order_id):
    """取消普通挂单"""
    request_path = "/capi/v2/order/cancel_order"
    body = {"orderId": order_id}
    query_string = ""
    response = send_request_post(api_key, secret_key, access_passphrase, "POST", request_path, query_string, body)
    return response.json()


def cancel_plan_order(order_id):
    """取消计划委托"""
    request_path = "/capi/v2/order/cancel_plan"
    body = {"orderId": order_id}
    query_string = ""
    response = send_request_post(api_key, secret_key, access_passphrase, "POST", request_path, query_string, body)
    return response.json()


def cancel_all_orders():
    """一键取消所有订单"""
    print("="*60)
    print("开始取消所有订单...")
    print("="*60)

    # 获取所有普通挂单
    print("\n正在获取普通挂单...")
    open_orders = get_open_orders()
    print(f"找到 {len(open_orders)} 个普通挂单")

    # 取消所有普通挂单
    if open_orders:
        print("\n开始取消普通挂单:")
        for order in open_orders:
            order_id = order.get('order_id', '')
            symbol = order.get('symbol', '').replace('cmt_', '').replace('usdt', '').upper()

            try:
                result = cancel_order(order_id)
                if result.get('result'):
                    print(f"  ✓ 成功取消 {symbol} 挂单 (ID: {order_id})")
                else:
                    err_msg = result.get('err_msg', '未知错误')
                    print(f"  ✗ 取消失败 {symbol} 挂单 (ID: {order_id}): {err_msg}")
                time.sleep(0.1)  # 避免请求过快
            except Exception as e:
                print(f"  ✗ 取消失败 {symbol} 挂单 (ID: {order_id}): {str(e)}")

    # 获取所有计划委托
    print("\n正在获取计划委托...")
    plan_orders = get_plan_orders()
    print(f"找到 {len(plan_orders)} 个计划委托")

    # 取消所有计划委托
    if plan_orders:
        print("\n开始取消计划委托:")
        for order in plan_orders:
            order_id = order.get('order_id', '')
            symbol = order.get('symbol', '').replace('cmt_', '').replace('usdt', '').upper()
            trigger_price = order.get('triggerPrice', '0')

            try:
                result = cancel_plan_order(order_id)
                if result.get('result'):
                    print(f"  ✓ 成功取消 {symbol} 计划委托 (触发价: {trigger_price}, ID: {order_id})")
                else:
                    err_msg = result.get('err_msg', '未知错误')
                    print(f"  ✗ 取消失败 {symbol} 计划委托 (ID: {order_id}): {err_msg}")
                time.sleep(0.1)  # 避免请求过快
            except Exception as e:
                print(f"  ✗ 取消失败 {symbol} 计划委托 (ID: {order_id}): {str(e)}")

    print("\n" + "="*60)
    print("所有订单取消完成！")
    print("="*60)


if __name__ == '__main__':
    # 确认操作
    print("\n⚠️  警告：此操作将取消所有普通挂单和计划委托！")
    confirm = input("确认要继续吗？(输入 yes 继续): ")

    if confirm.lower() == 'yes':
        cancel_all_orders()
    else:
        print("操作已取消")