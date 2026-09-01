import Foundation

/// 账目计算。**纯函数、无状态、不碰 UIKit** —— 余额与统计口径全在这里，
/// 界面只负责把结果画出来。
///
/// 金额一律用 `Decimal` 而不是 `Double`：0.1 + 0.2 在二进制浮点里不等于 0.3，
/// 记账 App 里这种误差会一分一分地累积到余额上，用户一眼就能看出对不上。
enum LedgerMath {

    /// 一条流水对某个账户余额的影响。
    ///
    /// - 支出：从 `accountId` 扣。
    /// - 收入：往 `accountId` 加。
    /// - 转账：从 `accountId` 扣、往 `toAccountId` 加 —— 同一条流水对两个账户各有一次影响，
    ///   故这里按传入的 account 分别计算，而不是给流水一个全局的正负号。
    static func delta(of transaction: LedgerTransaction, for accountId: UUID) -> Decimal {
        switch transaction.kind {
        case .expense:
            return transaction.accountId == accountId ? -transaction.amount : 0
        case .income:
            return transaction.accountId == accountId ? transaction.amount : 0
        case .transfer:
            if transaction.accountId == accountId { return -transaction.amount }
            if transaction.toAccountId == accountId { return transaction.amount }
            return 0
        }
    }

    /// 某账户的当前余额 = 期初余额 + 其上全部流水的净额。
    static func balance(of account: Account, transactions: [LedgerTransaction]) -> Decimal {
        var total = account.openingBalance
        for transaction in transactions {
            total += delta(of: transaction, for: account.id)
        }
        return total
    }

    /// 全部账户余额之和（净资产）。
    ///
    /// 转账天然不影响这个总和：转出方扣多少、转入方就加多少，两边抵消。
    static func totalBalance(accounts: [Account], transactions: [LedgerTransaction]) -> Decimal {
        var total: Decimal = 0
        for account in accounts {
            total += balance(of: account, transactions: transactions)
        }
        return total
    }

    /// 日期是否落在区间内。
    ///
    /// 刻意不用 `DateInterval.contains` —— 它把**结束时刻也算在内**，
    /// 而「本月」的结束时刻正是下月第一刻，用它会让下月第一秒的流水同时算进两个月。
    /// 这里统一取左闭右开。
    static func contains(_ date: Date, in interval: DateInterval) -> Bool {
        date >= interval.start && date < interval.end
    }

    /// 指定区间内、指定方向的合计。转账不计入收支统计（钱没离开你）。
    static func total(
        kind: TransactionKind,
        transactions: [LedgerTransaction],
        in interval: DateInterval
    ) -> Decimal {
        var total: Decimal = 0
        for transaction in transactions where transaction.kind == kind {
            if contains(transaction.date, in: interval) {
                total += transaction.amount
            }
        }
        return total
    }

    /// 区间内按分类汇总，按金额从大到小排。用于「这个月钱花在哪了」。
    /// 没有分类的流水（分类被删过）归到 nil 键下。
    static func categoryTotals(
        kind: TransactionKind,
        transactions: [LedgerTransaction],
        in interval: DateInterval
    ) -> [(categoryId: UUID?, total: Decimal)] {
        var buckets: [UUID?: Decimal] = [:]
        for transaction in transactions where transaction.kind == kind {
            guard contains(transaction.date, in: interval) else { continue }
            buckets[transaction.categoryId, default: 0] += transaction.amount
        }
        return buckets
            .map { (categoryId: $0.key, total: $0.value) }
            .sorted { $0.total > $1.total }
    }

    /// 某账户在区间内的流水条数。删除账户前用它提示用户会连带删掉多少条。
    static func transactionCount(forAccount accountId: UUID, transactions: [LedgerTransaction]) -> Int {
        transactions.filter {
            $0.accountId == accountId || $0.toAccountId == accountId
        }.count
    }

    /// 包含 `date` 的自然月区间（左闭右开）。
    ///
    /// `Calendar.dateInterval(of:for:)` 返回的正是「本月第一刻 → 下月第一刻」，
    /// 与上面 `contains` 的左闭右开语义刚好吻合。取不到时回退成 date 当天的空区间，
    /// 让调用方拿到 0 而不是崩。
    static func monthInterval(containing date: Date, calendar: Calendar = .current) -> DateInterval {
        calendar.dateInterval(of: .month, for: date) ?? DateInterval(start: date, duration: 0)
    }

    /// 包含 `date` 的自然日区间（左闭右开）。流水按天分组时用。
    static func dayInterval(containing date: Date, calendar: Calendar = .current) -> DateInterval {
        calendar.dateInterval(of: .day, for: date) ?? DateInterval(start: date, duration: 0)
    }
}
