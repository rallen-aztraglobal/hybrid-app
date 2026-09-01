import Foundation

/// 金额的显示与解析。
///
/// `NumberFormatter` 的创建成本不低，而流水列表每个 cell 都要格式化一次金额，
/// 所以按币种缓存。**仅在主线程使用**（全部调用都来自 UI），故不加锁。
enum MoneyFormatter {
    private static var cache: [String: NumberFormatter] = [:]

    private static func formatter(for currencyCode: String) -> NumberFormatter {
        if let cached = cache[currencyCode] { return cached }
        let formatter = NumberFormatter()
        formatter.numberStyle = .currency
        formatter.currencyCode = currencyCode
        // **不要手动写死 2 位小数。** 支持的币种里 JPY / KRW / VND / IDR 都是零小数位，
        // 强设 2 位会显示成 "¥1,234.50" 这种当地人一眼看出不对的样子。
        // .currency 样式会按币种自己取正确位数。
        cache[currencyCode] = formatter
        return formatter
    }

    /// 带币种符号的金额，如 "₱1,234.50"。
    static func string(_ amount: Decimal, currencyCode: String) -> String {
        let number = NSDecimalNumber(decimal: amount)
        return formatter(for: currencyCode).string(from: number) ?? "\(amount)"
    }

    /// 带方向前缀的金额：支出 "−"、收入 "+"、转账不带号。
    ///
    /// 用的是数学减号 U+2212 而不是连字符，等宽且与 "+" 视觉重量一致，
    /// 金额右对齐时列不会看起来歪。
    static func signed(_ amount: Decimal, kind: TransactionKind, currencyCode: String) -> String {
        let body = string(amount, currencyCode: currencyCode)
        switch kind {
        case .expense: return "\u{2212}" + body
        case .income: return "+" + body
        case .transfer: return body
        }
    }

    /// 把已有金额回填到输入框时用的文本。
    ///
    /// **不要用 `"\(decimal)"`。** `Decimal.description` 恒以 `.` 作小数点，
    /// 而 `parseSigned` 的第一步是剥掉**本地**分组分隔符 —— 在德语、西班牙语、
    /// 印尼语、巴西葡语等以 `.` 作分组符的地区，回填出来的 `"12.50"` 会被剥成 `"1250"`，
    /// 用户只是打开编辑页再点保存，金额就变成一百倍。记账 App 出这种错是致命的。
    ///
    /// 这里用本地化的十进制格式并**关掉分组分隔符**：产出的文本正好是 parseSigned
    /// 能原样读回来的形状（小数点用本地符号，且不含任何分组符）。
    private static let editFormatter: NumberFormatter = {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        formatter.usesGroupingSeparator = false
        formatter.maximumFractionDigits = 2
        formatter.minimumFractionDigits = 0
        return formatter
    }()

    static func editableText(_ amount: Decimal) -> String {
        let number = NSDecimalNumber(decimal: amount)
        return editFormatter.string(from: number) ?? "\(amount)"
    }

    /// 把输入框里的文本解析成**正数**金额。解析不出、为负、或为零一律返回 nil ——
    /// 调用方据此提示「请输入有效金额」。流水金额恒为正（方向由 kind 决定），故用这个。
    static func parse(_ text: String, locale: Locale = .current) -> Decimal? {
        guard let value = parseSigned(text, locale: locale), value > 0 else { return nil }
        return value
    }

    /// 解析**可正可负、也可为零**的金额。期初余额用它：
    /// 信用卡建账时余额往往是负的（欠款），现金账户可能就是 0。
    ///
    /// 同时接受本地小数分隔符与英文句点：decimalPad 键盘给的是本地分隔符
    /// （很多地区是逗号），只认句点的话用户根本输不进小数。
    static func parseSigned(_ text: String, locale: Locale = .current) -> Decimal? {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return nil }

        // 负号可能是 ASCII 连字符，也可能是键盘/粘贴带来的数学减号。
        let isNegative = trimmed.hasPrefix("-") || trimmed.hasPrefix("\u{2212}")

        let separator = locale.decimalSeparator ?? "."
        var normalized = trimmed
        normalized = normalized.replacingOccurrences(of: locale.groupingSeparator ?? ",", with: "")
        normalized = normalized.replacingOccurrences(of: separator, with: ".")
        // 去掉除数字与小数点之外的一切（币种符号、空格、负号等）。
        normalized = normalized.filter { $0.isNumber || $0 == "." }

        guard !normalized.isEmpty, normalized != ".",
              let magnitude = Decimal(string: normalized, locale: nil) else {
            return nil
        }
        return isNegative ? -magnitude : magnitude
    }
}

/// 日期显示。集中在一处，免得各界面各写各的格式。
enum DateDisplay {
    private static let dayFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        formatter.timeStyle = .none
        return formatter
    }()

    /// 流水分组表头用：今天 / 昨天 显示成词，其余显示日期。
    static func sectionTitle(for date: Date, calendar: Calendar = .current) -> String {
        if calendar.isDateInToday(date) { return "Today" }
        if calendar.isDateInYesterday(date) { return "Yesterday" }
        return dayFormatter.string(from: date)
    }

    static func medium(_ date: Date) -> String {
        dayFormatter.string(from: date)
    }
}
