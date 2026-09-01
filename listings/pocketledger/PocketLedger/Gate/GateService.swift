import Foundation

/// B 面打开方式：internal=内开（WKWebView）/ external=外开（系统浏览器）。
/// 仅 mode=B 时有意义，服务端只在判 B 时下发；缺省/非法一律内开。
enum GateOpenMode {
    case internalWebView
    case externalBrowser

    /// 只认 "external"，其余（含缺省 nil）一律内开。
    static func from(_ raw: String?) -> GateOpenMode {
        return raw == "external" ? .externalBrowser : .internalWebView
    }
}

/// 网关判定结果。默认即 A 面——任何不确定都落到这个安全默认值。
enum GateDecision {
    case aSide
    case bSide(url: URL, openMode: GateOpenMode)
}

/// 调用服务端 AB 面网关。判定全在服务端（按请求真实 IP 查国家 + 时区/IP 规则）；
/// 客户端只上报 platform/bundleId/timezone，从不上报 IP，也从不内置 B 面地址。
///
/// 安全不变量：**任何异常（超时、网络错误、非 200、解析失败）一律 A 面**。
enum GateService {
    /// 逐个尝试候选基址，任一成功即返回；全部失败 → A 面。
    static func evaluate(timezone: String) async -> GateDecision {
        let payload: [String: String] = [
            "platform": GateConfig.platform,
            "bundleId": GateConfig.bundleId,
            "timezone": timezone
        ]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            return .aSide
        }

        for base in GateConfig.apiBases {
            if let decision = await post(base: base, body: body) {
                return decision
            }
        }
        return .aSide
    }

    /// 向单个基址发一次判定请求。返回 nil = 该基址连不上（外层继续试下一个）；
    /// 返回 .aSide = 拿到了明确的「非 B」响应（或任何应判 A 的情形）。
    private static func post(base: String, body: Data) async -> GateDecision? {
        guard let url = URL(string: base + GateConfig.gatePath) else { return nil }

        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.httpBody = body
        req.timeoutInterval = GateConfig.requestTimeout

        do {
            let (data, response) = try await URLSession.shared.data(for: req)
            guard let http = response as? HTTPURLResponse else { return .aSide }
            guard http.statusCode == 200 else { return .aSide }
            return parse(data)
        } catch {
            // 连接类错误：让外层尝试下一个候选基址。其余无法区分时也回退 A 面。
            let code = (error as? URLError)?.code
            if code == .cannotConnectToHost || code == .cannotFindHost
                || code == .notConnectedToInternet || code == .timedOut {
                return nil
            }
            return .aSide
        }
    }

    /// 解析响应；形状不符一律 A 面。
    private static func parse(_ data: Data) -> GateDecision {
        guard
            let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
            (obj["mode"] as? String) == "B",
            let urlString = obj["url"] as? String,
            let url = usableURL(urlString)
        else {
            return .aSide
        }
        return .bSide(url: url, openMode: GateOpenMode.from(obj["openMode"] as? String))
    }

    /// B 面 url 是否可用：能解析、是 http/https、且有主机名。
    ///
    /// 只判 `URL(string:)` 非 nil 是不够的 —— `URL(string: "somestring")` 会成功，
    /// 得到一个只有 path、没有 scheme 与 host 的相对 URL，WKWebView 加载它只会白屏。
    /// 白屏比回退 A 面糟得多（用户看到的是个坏掉的 App），所以这里前置挡掉：
    /// 挡不住的畸形值宁可判 A。与 colorstack / calcpad / tilefit 的 Dart 侧同口径。
    private static func usableURL(_ raw: String) -> URL? {
        guard !raw.isEmpty, let url = URL(string: raw) else { return nil }
        guard let scheme = url.scheme?.lowercased(), scheme == "http" || scheme == "https" else {
            return nil
        }
        guard let host = url.host, !host.isEmpty else { return nil }
        return url
    }
}
