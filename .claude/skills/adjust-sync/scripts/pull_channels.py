#!/usr/bin/env python3
"""从渠道中台 Console 拉 AP+BP 渠道清单，输出全量 + 未绑定 Adjust 的增量。

用法:
    ADJUST_CONSOLE_URL=https://... ADJUST_CONSOLE_USER=admin ADJUST_CONSOLE_PASS='***' \
    python3 pull_channels.py <out.json>

输出 <out.json>: {"channels":[{brand,id,flavor,applicationId,palCode,appName,adjustAppToken}],
                  "delta":[[flavor,applicationId], ...]}   # delta = adjustAppToken 为空者
并打印增量清单，供第 2 步内联进 provision.js 的 BATCH。

用 curl 走 HTTP（macOS 上 urllib 常因缺 CA bundle 校验 HTTPS 失败；curl 用系统证书更稳）。
兼容响应体 data 为「列表」或「{items,total} 分页对象」两种形态。
"""
import os, sys, json, subprocess


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
                         "appName": c.get("appName"), "adjustAppToken": c.get("adjustAppToken")})
    delta = [[r["flavor"], r["applicationId"]] for r in rows if not r.get("adjustAppToken")]
    json.dump({"channels": rows, "delta": delta}, open(out, "w"), ensure_ascii=False, indent=2)
    ap = sum(1 for r in rows if r["brand"] == "ap")
    print(f"总 {len(rows)}（AP {ap} / BP {len(rows) - ap}）；未绑定 Adjust（增量）{len(delta)}:")
    for fl, ai in delta:
        print(f"  {fl}  {ai}")
    print("→", out)
    print("\nBATCH 数组（复制进 provision.js，按 ≤14 分批）:")
    print(json.dumps(delta, ensure_ascii=False))


if __name__ == "__main__":
    main()
