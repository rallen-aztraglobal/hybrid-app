import Foundation

/// 账户类型。**新建账户时由用户自己选** —— 这是本 App 的核心设定：
/// 同一笔钱放在信用卡、电子钱包还是现金里，花销的口径完全不同。
///
/// 之所以把「电子钱包」单列一类而不是塞进「银行账户」：目标市场里 GCash / Maya
/// 这类钱包是和银行卡并列的主力支付方式，混在一起用户就分不清余额了。
enum AccountKind: String, Codable, CaseIterable {
    case card
    case eWallet
    case cash
    case bank

    var displayName: String {
        switch self {
        case .card: return "Card"
        case .eWallet: return "E-wallet"
        case .cash: return "Cash"
        case .bank: return "Bank account"
        }
    }

    /// 分段控件里用的短标签。"Bank account" 在四选一的分段控件里会被挤到省略号，
    /// 那里只放 "Bank"，完整名字由下方的说明文字补齐。
    var shortName: String {
        switch self {
        case .card: return "Card"
        case .eWallet: return "E-wallet"
        case .cash: return "Cash"
        case .bank: return "Bank"
        }
    }

    /// 建账时给用户的一句说明，帮他分清该选哪个。
    var hint: String {
        switch self {
        case .card: return "Credit or debit card. Balance may go negative when you owe."
        case .eWallet: return "Digital wallet such as a mobile payment app."
        case .cash: return "Physical cash in your pocket or at home."
        case .bank: return "A savings or checking account at a bank."
        }
    }

    /// SF Symbol 名。全部选用 iOS 14 及更早就有的符号 —— 本包最低支持 iOS 15.6，
    /// 用了 iOS 16+ 才有的符号（如 `banknote.fill`）在老系统上会画不出来。
    var symbolName: String {
        switch self {
        case .card: return "creditcard.fill"
        case .eWallet: return "wallet.pass.fill"
        case .cash: return "dollarsign.circle.fill"
        case .bank: return "building.columns.fill"
        }
    }
}

/// 一个账户：一张卡、一个电子钱包、一笔现金或一个银行账户。
struct Account: Codable, Identifiable, Equatable {
    var id: UUID
    var name: String
    var kind: AccountKind
    /// 建账时的期初余额。账户当前余额 = 期初余额 ± 其上的全部流水（见 LedgerMath）。
    var openingBalance: Decimal
    var colorIndex: Int
    var createdAt: Date

    init(
        id: UUID = UUID(),
        name: String,
        kind: AccountKind,
        openingBalance: Decimal = 0,
        colorIndex: Int = 0,
        createdAt: Date = Date()
    ) {
        self.id = id
        self.name = name
        self.kind = kind
        self.openingBalance = openingBalance
        self.colorIndex = colorIndex
        self.createdAt = createdAt
    }
}

/// 流水的方向。
enum TransactionKind: String, Codable, CaseIterable {
    case expense
    case income
    /// 账户之间转账（例如从银行卡充值到电子钱包）。不计入收支统计，只挪动余额。
    case transfer

    var displayName: String {
        switch self {
        case .expense: return "Expense"
        case .income: return "Income"
        case .transfer: return "Transfer"
        }
    }
}

/// 分类。转账没有分类，故只区分支出与收入两类。
struct LedgerCategory: Codable, Identifiable, Equatable {
    var id: UUID
    var name: String
    var symbolName: String
    var colorIndex: Int
    /// 该分类属于支出还是收入。转账不取分类，故这里只会是 .expense / .income。
    var kind: TransactionKind

    init(
        id: UUID = UUID(),
        name: String,
        symbolName: String,
        colorIndex: Int,
        kind: TransactionKind
    ) {
        self.id = id
        self.name = name
        self.symbolName = symbolName
        self.colorIndex = colorIndex
        self.kind = kind
    }
}

/// 一条流水。
///
/// `amount` **恒为正数**，方向完全由 `kind` 决定 —— 让金额带符号会让「编辑时把支出
/// 改成收入」这类操作产生两处真相（符号与 kind），迟早对不上。
struct LedgerTransaction: Codable, Identifiable, Equatable {
    var id: UUID
    var kind: TransactionKind
    var amount: Decimal
    var date: Date
    /// 支出/收入所属账户；转账时是**转出方**。
    var accountId: UUID
    /// 仅转账使用：**转入方**。其余类型恒为 nil。
    var toAccountId: UUID?
    /// 转账恒为 nil；支出/收入指向 LedgerCategory。分类被删时也可能为 nil。
    var categoryId: UUID?
    var note: String

    init(
        id: UUID = UUID(),
        kind: TransactionKind,
        amount: Decimal,
        date: Date = Date(),
        accountId: UUID,
        toAccountId: UUID? = nil,
        categoryId: UUID? = nil,
        note: String = ""
    ) {
        self.id = id
        self.kind = kind
        self.amount = amount
        self.date = date
        self.accountId = accountId
        self.toAccountId = toAccountId
        self.categoryId = categoryId
        self.note = note
    }
}

/// 首次启动时写入的默认分类。用户不必先建分类才能记第一笔账。
enum DefaultCategories {
    static func make() -> [LedgerCategory] {
        let expenses: [(String, String, Int)] = [
            ("Food & drink", "fork.knife", 5),
            ("Groceries", "cart.fill", 3),
            ("Transport", "car.fill", 0),
            ("Shopping", "bag.fill", 7),
            ("Bills", "bolt.fill", 4),
            ("Home", "house.fill", 2),
            ("Health", "cross.case.fill", 6),
            ("Entertainment", "gamecontroller.fill", 1),
            ("Education", "book.fill", 2),
            ("Other", "ellipsis.circle.fill", 1)
        ]
        let incomes: [(String, String, Int)] = [
            ("Salary", "briefcase.fill", 3),
            ("Bonus", "star.fill", 4),
            ("Refund", "arrow.uturn.backward.circle.fill", 0),
            ("Other income", "plus.circle.fill", 2)
        ]

        var result: [LedgerCategory] = []
        for item in expenses {
            result.append(
                LedgerCategory(name: item.0, symbolName: item.1, colorIndex: item.2, kind: .expense)
            )
        }
        for item in incomes {
            result.append(
                LedgerCategory(name: item.0, symbolName: item.1, colorIndex: item.2, kind: .income)
            )
        }
        return result
    }
}
