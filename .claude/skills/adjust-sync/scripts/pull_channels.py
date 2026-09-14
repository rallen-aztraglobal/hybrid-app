#!/usr/bin/env python3
"""从渠道中台 Console 拉 AP+BP 渠道清单，输出全量 + 未绑定 Adjust 的增量。

用法:
    ADJUST_CONSOLE_URL=https://... ADJUST_CONSOLE_USER=admin ADJUST_CONSOLE_PASS='***' \
    python3 pull_channels.py <out.json>

输出 <out.json>: {"channels":[{brand,id,flavor,applicationId,palCode,appName,adjustAppToken,
                               adjustBpRawEvents,adjustEvents}],
                  "delta":[[flavor,applicationId,bpRaw], ...]}
delta = 需要在 Adjust 建/补的渠道：adjustAppToken 为空者，或「BP 原始事件」开关已开但事件表缺
14 个中任一者（渠道从 legacy 切到 bpRaw 时要补建 8 个事件，见 ADR-0018）。
并打印增量清单，供第 2 步内联进 provision.js 的 BATCH（第 3 位 bpRaw 决定建哪套事件）。

用 curl 走 HTTP（macOS 上 urllib 常因缺 CA bundle 校验 HTTPS 失败；curl 用系统证书更稳）。
兼容响应体 data 为「列表」或「{items,total} 分页对象」两种形态。
"""
import os, sys, json, subprocess

# 两套事件集（与 provision.js EVENT_SETS、08-adjust.md §11 对齐）：渠道当前模式下缺任一都视为需同步，
# 这样「开过 BP 原始事件又关回去」的渠道（事件表被 14 个覆盖、缺老 6 个）也会被拉回来补建。
LEGACY_EVENTS = ["AddToCart", "CompleteRegistration", "Login", "OldRegPurchase", "Purchase", "TPFirstDeposit"]
BP_RAW_EVENTS = ["Action_BUFD", "Action_BURD", "Action_Deposit", "Action_FDRD", "Action_Registration",
                 "ad_app_opened", "ad_deeplink_opened", "ad_deposit", "ad_game_open", "ad_registration",
                 "ad_web_deposit", "ad_web_login", "ad_web_pageview", "ad_web_reg"]


def curl(url, token=None, method="GET", body=None):
    cmd = ["curl", "-s", "--max-time", "25", "-X", method, url]
    if token:
        cmd += ["-H", f"Authorization: Bearer {token}"]
    if body is not None:
        cmd += ["-H", "Content-Type: application/json", "-d", json.dumps(body)]
    out = subprocess.run(cmd, capture_output=True, text=True).stdout
    return json.loads(out)


def data_rows(resp):
    """取 resp['data']，兼容 list 与 {items:[...]} 两种。"""
    d = resp.get("data")
    if isinstance(d, dict):
        return d.get("items") or d.get("list") or []
    return d or []


def main():
    out = sys.argv[1] if len(sys.argv) > 1 else "adjust_channels.json"
    base = os.environ["ADJUST_CONSOLE_URL"].rstrip("/")
    u, p = os.environ["ADJUST_CONSOLE_USER"], os.environ["ADJUST_CONSOLE_PASS"]
    tok = curl(f"{base}/api/auth/login", method="POST",
               body={"username": u, "password": p})["data"]["accessToken"]
    rows = []
    for brand in ("ap", "bp"):  # 只有 AP/BP 接 Adjust；GP 不接
        for c in data_rows(curl(f"{base}/api/channels?brand={brand}", token=tok)):
            rows.append({"brand": brand, "id": c["id"], "flavor": c["flavorName"],
                         "applicationId": c["applicationId"], "palCode": c.get("palCode"),
                         "appName": c.get("appName"), "adjustAppToken": c.get("adjustAppToken"),
                         "adjustBpRawEvents": bool(c.get("adjustBpRawEvents")),
                         "adjustEvents": c.get("adjustEvents") or {}})

    def needs_sync(r):
        if not r.get("adjustAppToken"):
            return True  # 未绑定
        want = BP_RAW_EVENTS if r["adjustBpRawEvents"] else LEGACY_EVENTS
        return any(e not in r["adjustEvents"] for e in want)  # 当前模式事件不齐（含模式切换后）

    delta = [[r["flavor"], r["applicationId"], r["adjustBpRawEvents"]] for r in rows if needs_sync(r)]
    json.dump({"channels": rows, "delta": delta}, open(out, "w"), ensure_ascii=False, indent=2)
    ap = sum(1 for r in rows if r["brand"] == "ap")
    unbound = sum(1 for r in rows if not r.get("adjustAppToken"))
    print(f"总 {len(rows)}（AP {ap} / BP {len(rows) - ap}）；需同步（增量）{len(delta)}"
          f"（未绑定 {unbound}，当前模式事件不齐 {len(delta) - unbound}）:")
    for fl, ai, raw in delta:
        print(f"  {fl}  {ai}{'  [BP 原始事件]' if raw else ''}")
    print("→", out)
    print("\nBATCH 数组（复制进 provision.js，按 ≤14 分批）:")
    print(json.dumps(delta, ensure_ascii=False))


if __name__ == "__main__":
    main()
