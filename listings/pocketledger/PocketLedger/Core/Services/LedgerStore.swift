import Foundation

extension Notification.Name {
    /// 账本任何一处变动后广播。各界面收到就重新拉一遍数据 —— 这个 App 的数据量很小
    /// （几百条流水），整体重算比维护增量刷新简单得多，也不会出现界面与账本不一致。
    static let ledgerDidChange = Notification.Name("PocketLedger.ledgerDidChange")
}

/// 账本的唯一真相。全部数据存在 App 沙盒里的一个 JSON 文件中，**不出设备、不上传**。
///
/// 为什么用文件而不是 UserDefaults：UserDefaults 适合放几个设置项，流水会越攒越多，
/// 每次写入都要整体序列化进 plist 并同步，不合适。也没上 Core Data —— 这点数据量
/// 用不着，一个可读的 JSON 文件反而方便导出与排查。
final class LedgerStore {
    static let shared = LedgerStore()

    private(set) var accounts: [Account] = []
    private(set) var categories: [LedgerCategory] = []
    private(set) var transactions: [LedgerTransaction] = []

    private let fileURL: URL
    private let encoder = JSONEncoder()
    private let decoder = JSONDecoder()

    /// 落盘用的整体快照。
    private struct Snapshot: Codable {
        var accounts: [Account]
        var categories: [LedgerCategory]
        var transactions: [LedgerTransaction]
    }

    private init() {
        let base = FileManager.default
            .urls(for: .applicationSupportDirectory, in: .userDomainMask).first
            ?? FileManager.default.temporaryDirectory
        // iOS 上 Application Support 目录默认不存在，得自己建。
        try? FileManager.default.createDirectory(at: base, withIntermediateDirectories: true)
        fileURL = base.appendingPathComponent("pocketledger.json")

        encoder.dateEncodingStrategy = .iso8601
        decoder.dateDecodingStrategy = .iso8601
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]

        load()
    }

    // MARK: - 读写

    private func load() {
        guard
            let data = try? Data(contentsOf: fileURL),
            let snapshot = try? decoder.decode(Snapshot.self, from: data)
        else {
            // notify: false —— 这里还在 init 里，**绝不能发通知**。原因见 persist(notify:)。
            seedFirstRun(notify: false)
            return
        }
        accounts = snapshot.accounts
        categories = snapshot.categories
        transactions = snapshot.transactions
    }

    /// 首次启动（或存档读不出来）时的初始账本。
    ///
    /// 给一套默认分类 + 一个「Cash」账户：不这么做的话，用户打开 App 得先建账户、
    /// 再建分类，才能记第一笔账 —— 三步之后才见到价值，多数人已经退出去了。
    /// 卡与电子钱包留给用户自己按需添加，那才是本 App 想让人用起来的部分。
    private func seedFirstRun(notify: Bool = true) {
        categories = DefaultCategories.make()
        accounts = [Account(name: "Cash", kind: .cash, openingBalance: 0, colorIndex: 3)]
        transactions = []
        persist(notify: notify)
    }

    /// 落盘。`notify: false` 专供 `init` 路径。
    ///
    /// **为什么需要这个开关**：`shared` 是 `static let`，它的惰性初始化由 `swift_once`
    /// （底层 `dispatch_once`）保护，而该锁**不可重入**。若在 `init` 里同步发出
    /// `.ledgerDidChange`，观察者（各页面的 `reloadContent()`）会立刻回头访问
    /// `LedgerStore.shared` —— 同一线程二次进入还没走完的 `swift_once`，
    /// libdispatch 直接 `DISPATCH_CLIENT_CRASH("trying to lock recursively")`。
    ///
    /// 这个崩溃只在**全新安装的第一次冷启动**出现（那时才会走 seedFirstRun → persist）；
    /// 崩过一次存档就落盘了，第二次启动读档成功、不再 persist，于是不复现 ——
    /// 极容易被误判成偶发问题，所以这里写清楚。
    ///
    /// 初始化期间也不需要通知：此刻还没有任何界面拿到过数据，没有「变更」可言。
    private func persist(notify: Bool = true) {
        let snapshot = Snapshot(accounts: accounts, categories: categories, transactions: transactions)
        do {
            let data = try encoder.encode(snapshot)
            // 原子写：中途崩溃也不会留下一个截断的、下次读不出来的账本。
            try data.write(to: fileURL, options: .atomic)
        } catch {
            assertionFailure("LedgerStore: 写入失败 — \(error)")
        }
        if notify {
            NotificationCenter.default.post(name: .ledgerDidChange, object: nil)
        }
    }

    // MARK: - 查询

    func account(id: UUID?) -> Account? {
        guard let id else { return nil }
        return accounts.first { $0.id == id }
    }

    func category(id: UUID?) -> LedgerCategory? {
        guard let id else { return nil }
        return categories.first { $0.id == id }
    }

    /// 某个方向下可选的分类。转账没有分类，传 .transfer 会得到空数组。
    func categories(for kind: TransactionKind) -> [LedgerCategory] {
        categories.filter { $0.kind == kind }
    }

    /// 全部流水，新的在前。同一时刻的按 id 稳定排序，避免每次刷新顺序跳动。
    func transactionsNewestFirst() -> [LedgerTransaction] {
        transactions.sorted {
            if $0.date != $1.date { return $0.date > $1.date }
            return $0.id.uuidString > $1.id.uuidString
        }
    }

    func balance(of account: Account) -> Decimal {
        LedgerMath.balance(of: account, transactions: transactions)
    }

    var totalBalance: Decimal {
        LedgerMath.totalBalance(accounts: accounts, transactions: transactions)
    }

    // MARK: - 账户

    func addAccount(_ account: Account) {
        accounts.append(account)
        persist()
    }

    func updateAccount(_ account: Account) {
        guard let index = accounts.firstIndex(where: { $0.id == account.id }) else { return }
        accounts[index] = account
        persist()
    }

    /// 删除账户，**连带删掉它上面的全部流水**（含以它为转入方的转账）。
    ///
    /// 另一种做法是把流水挪到别的账户上，但那会悄悄改掉用户没打算改的记录；
    /// 直接删掉、并在界面上先把条数告诉用户，语义更干净。
    func deleteAccount(id: UUID) {
        accounts.removeAll { $0.id == id }
        transactions.removeAll { $0.accountId == id || $0.toAccountId == id }
        persist()
    }

    // MARK: - 流水

    func addTransaction(_ transaction: LedgerTransaction) {
        transactions.append(transaction)
        persist()
    }

    func updateTransaction(_ transaction: LedgerTransaction) {
        guard let index = transactions.firstIndex(where: { $0.id == transaction.id }) else { return }
        transactions[index] = transaction
        persist()
    }

    func deleteTransaction(id: UUID) {
        transactions.removeAll { $0.id == id }
        persist()
    }

    // MARK: - 整体

    /// 清空全部数据并回到首次启动的初始账本。设置页的「Erase all data」用。
    func eraseAll() {
        seedFirstRun()
    }

    /// 导出成 CSV（设置页分享用）。表头固定，Excel / Google Sheets 都能直接打开。
    func exportCSV() -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withFullDate]

        var lines: [String] = ["Date,Type,Amount,Account,To account,Category,Note"]
        for transaction in transactionsNewestFirst() {
            let fields: [String] = [
                formatter.string(from: transaction.date),
                transaction.kind.rawValue,
                "\(transaction.amount)",
                account(id: transaction.accountId)?.name ?? "",
                account(id: transaction.toAccountId)?.name ?? "",
                category(id: transaction.categoryId)?.name ?? "",
                transaction.note
            ]
            lines.append(fields.map(Self.csvEscaped).joined(separator: ","))
        }
        return lines.joined(separator: "\n")
    }

    /// CSV 字段转义：含逗号、引号或换行的字段要整体加引号，内部的引号翻倍。
    /// 备注里出现逗号是很常见的事，不转义会把一列冲成两列。
    private static func csvEscaped(_ field: String) -> String {
        guard field.contains(",") || field.contains("\"") || field.contains("\n") else {
            return field
        }
        return "\"" + field.replacingOccurrences(of: "\"", with: "\"\"") + "\""
    }
}
