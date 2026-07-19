import Foundation

/// 网关判定结果。默认即 A 面——任何不确定都落到这个安全默认值。
enum GateDecision {
    case aSide
    case bSide(url: URL)
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
            !urlString.isEmpty,
            let url = URL(string: urlString)
        else {
            return .aSide
        }
        return .bSide(url: url)
    }
}
