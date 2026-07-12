#!/usr/bin/env python3
"""把 provision 结果（app token + 事件 token）回填到 Console 的渠道。

用法:
    ADJUST_CONSOLE_URL=https://... ADJUST_CONSOLE_USER=admin ADJUST_CONSOLE_PASS='***' \
    python3 fill_console.py <channels.json> <final.json>

    channels.json  来自 pull_channels.py（提供每渠道 id/palCode/appName）
    final.json     = [{channelId 或 flavor, adjustAppToken, adjustEvents}, ...]

对每渠道 PUT /api/channels/:id（带 palCode/appName/adjustAppToken/adjustEvents），再 GET 回读校验。
前提：线上已发布带 Adjust 字段的新代码（否则回读为 None，脚本会明确报出 → 先跑 release skill）。

用 curl 走 HTTP（macOS urllib 常因缺 CA bundle 失败）。兼容 data 为 list 或 {items} 分页对象。
"""
import os, sys, json, subprocess


def curl(url, token=None, method="GET", body=None):
    cmd = ["curl", "-s", "--max-time", "25", "-X", method, url]
    if token:
        cmd += ["-H", f"Authorization: Bearer {token}"]
    if body is not None:
        cmd += ["-H", "Content-Type: application/json", "-d", json.dumps(body)]
    out = subprocess.run(cmd, capture_output=True, text=True).stdout
    try:
        return json.loads(out)
    except Exception:
        return {"_raw": out[:200]}


def main():
    chf, finf = sys.argv[1], sys.argv[2]
    base = os.environ["ADJUST_CONSOLE_URL"].rstrip("/")
    u, p = os.environ["ADJUST_CONSOLE_USER"], os.environ["ADJUST_CONSOLE_PASS"]
    tok = curl(f"{base}/api/auth/login", method="POST",
               body={"username": u, "password": p})["data"]["accessToken"]

    chans = json.load(open(chf))["channels"]
    by_flavor = {c["flavor"]: c for c in chans}
    by_id = {c["id"]: c for c in chans}
    final = json.load(open(finf))

    ok, fail = 0, []
    for r in final:
        ch = by_id.get(r.get("channelId")) or by_flavor.get(r.get("flavor"))
        if not ch:
            fail.append((r.get("flavor") or r.get("channelId"), "渠道未匹配"))
            continue
        body = {"palCode": ch.get("palCode"), "appName": ch.get("appName"),
                "adjustAppToken": r["adjustAppToken"], "adjustEvents": r["adjustEvents"]}
        curl(f"{base}/api/channels/{ch['id']}", token=tok, method="PUT", body=body)
        g = curl(f"{base}/api/channels/{ch['id']}", token=tok).get("data", {})
        got, nev = g.get("adjustAppToken"), len(g.get("adjustEvents") or {})
        if got == r["adjustAppToken"] and nev == len(r["adjustEvents"]):
            ok += 1
        else:
            fail.append((ch["flavor"], f"回读 token={got!r} events={nev}（未持久化？线上是否已发布新代码）"))

    print(f"回填+回读校验通过 {ok}/{len(final)}")
    if fail:
        print("失败/异常:")
        for fl, why in fail:
            print(f"  {fl}: {why}")


if __name__ == "__main__":
    main()
