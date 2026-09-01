import UIKit

/// 流水列表。按天分组，新的在上；每天一个分组头，右侧是当天净额。
final class TransactionsViewController: LedgerObservingViewController {
    private let tableView = UITableView(frame: .zero, style: .grouped)
    private var sections: [DaySection] = []
    private var emptyState: EmptyStateView?

    /// 一天的流水。
    private struct DaySection {
        let day: Date
        let items: [LedgerTransaction]
    }

    override func viewDidLoad() {
        super.viewDidLoad()

        navigationItem.rightBarButtonItem = UIBarButtonItem(
            barButtonSystemItem: .add,
            target: self,
            action: #selector(addTransaction)
        )

        tableView.backgroundColor = AppTheme.background
        tableView.separatorStyle = .none
        tableView.dataSource = self
        tableView.delegate = self
        tableView.register(TransactionCell.self, forCellReuseIdentifier: TransactionCell.reuseID)
        tableView.rowHeight = 68
        tableView.sectionHeaderHeight = 34
        tableView.sectionFooterHeight = 0
        // grouped 表在 iOS 15 起默认给每段顶部加一段较大的留白，会把分组头顶得很散。
        // 本包最低就是 15.6，直接设即可，无需可用性判断。
        tableView.sectionHeaderTopPadding = 8
        view.addForAutoLayout(tableView)
        tableView.pinToEdges(of: view)

        reloadContent()
    }

    override func reloadContent() {
        sections = Self.groupByDay(LedgerStore.shared.transactionsNewestFirst())
        tableView.reloadData()
        updateEmptyState()
    }

    /// 把已按时间倒序排好的流水切成「天」。
    ///
    /// 依赖入参已经有序：这样一次线性扫描就够了，不必对每一天再排一次。
    private static func groupByDay(_ transactions: [LedgerTransaction],
                                   calendar: Calendar = .current) -> [DaySection] {
        var result: [DaySection] = []
        var currentDay: Date?
        var bucket: [LedgerTransaction] = []

        for transaction in transactions {
            let day = calendar.startOfDay(for: transaction.date)
            if let currentDay, currentDay != day {
                result.append(DaySection(day: currentDay, items: bucket))
                bucket = []
            }
            currentDay = day
            bucket.append(transaction)
        }
        if let currentDay, !bucket.isEmpty {
            result.append(DaySection(day: currentDay, items: bucket))
        }
        return result
    }

    private func updateEmptyState() {
        if sections.isEmpty {
            guard emptyState == nil else { return }
            let empty = EmptyStateView(
                symbolName: "list.bullet.rectangle",
                title: "No entries yet",
                message: "Record what you spend and receive. Everything stays on this device.",
                actionTitle: "Add entry"
            )
            empty.onAction { [weak self] in self?.addTransaction() }
            view.addForAutoLayout(empty)
            empty.pinToEdges(of: view)
            emptyState = empty
            tableView.isHidden = true
        } else {
            emptyState?.removeFromSuperview()
            emptyState = nil
            tableView.isHidden = false
        }
    }

    @objc private func addTransaction() {
        guard !LedgerStore.shared.accounts.isEmpty else {
            showBanner(message: "Add an account first, in the Accounts tab.", style: .failure)
            return
        }
        present(editor: TransactionEditorViewController(transaction: nil))
    }

    private func present(editor: TransactionEditorViewController) {
        let nav = UINavigationController(rootViewController: editor)
        AppTheme.applyNavigationAppearance(nav.navigationBar)
        present(nav, animated: true)
    }

    /// 一天的净额：收入减支出。转账不算（钱只是换了个口袋）。
    private func netAmount(of section: DaySection) -> Decimal {
        var net: Decimal = 0
        for item in section.items {
            switch item.kind {
            case .income: net += item.amount
            case .expense: net -= item.amount
            case .transfer: break
            }
        }
        return net
    }
}

extension TransactionsViewController: UITableViewDataSource, UITableViewDelegate {
    func numberOfSections(in tableView: UITableView) -> Int {
        sections.count
    }

    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        sections[section].items.count
    }

    func tableView(_ tableView: UITableView, viewForHeaderInSection section: Int) -> UIView? {
        let model = sections[section]
        let header = UIView()
        header.backgroundColor = .clear

        let title = UILabel()
        title.text = DateDisplay.sectionTitle(for: model.day)
        title.font = .systemFont(ofSize: 13, weight: .semibold)
        title.textColor = AppTheme.textSecondary

        let net = netAmount(of: model)
        let amount = UILabel()
        amount.font = .systemFont(ofSize: 13, weight: .semibold)
        amount.textColor = net < 0 ? AppTheme.expense : AppTheme.income
        amount.textAlignment = .right
        // 净额为 0（当天只有转账）就不显示，省得出现一个没有信息量的 "0"。
        amount.text = net == 0
            ? nil
            : MoneyFormatter.signed(
                abs(net),
                kind: net < 0 ? .expense : .income,
                currencyCode: currencyCode
            )

        header.addForAutoLayout(title)
        header.addForAutoLayout(amount)
        NSLayoutConstraint.activate([
            title.leadingAnchor.constraint(equalTo: header.leadingAnchor,
                                           constant: AppTheme.horizontalPadding + 4),
            title.bottomAnchor.constraint(equalTo: header.bottomAnchor, constant: -6),
            amount.trailingAnchor.constraint(equalTo: header.trailingAnchor,
                                             constant: -(AppTheme.horizontalPadding + 4)),
            amount.bottomAnchor.constraint(equalTo: title.bottomAnchor)
        ])
        return header
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: TransactionCell.reuseID, for: indexPath)
        if let cell = cell as? TransactionCell {
            cell.configure(
                transaction: sections[indexPath.section].items[indexPath.row],
                currencyCode: currencyCode
            )
        }
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let transaction = sections[indexPath.section].items[indexPath.row]
        present(editor: TransactionEditorViewController(transaction: transaction))
    }

    func tableView(
        _ tableView: UITableView,
        trailingSwipeActionsConfigurationForRowAt indexPath: IndexPath
    ) -> UISwipeActionsConfiguration? {
        let transaction = sections[indexPath.section].items[indexPath.row]
        let delete = UIContextualAction(style: .destructive, title: "Delete") { _, _, completion in
            // 先让 UIKit 收掉滑动态，再删。反过来的话：删除会同步走到
            // persist → 发通知 → reloadContent → reloadData()，而此时这一行的滑动
            // 还没收尾，UIKit 手里的 indexPath 已经失效 —— 轻则动画跳变，重则抛
            // NSInternalInconsistencyException。
            completion(true)
            DispatchQueue.main.async {
                LedgerStore.shared.deleteTransaction(id: transaction.id)
            }
        }
        return UISwipeActionsConfiguration(actions: [delete])
    }
}

/// 流水行：分类图标 + 标题（分类名 / 转账两端） + 副标题（账户与备注） + 金额。
final class TransactionCell: UITableViewCell {
    static let reuseID = "TransactionCell"

    private let card = CardView()
    private let badge = IconBadgeView(diameter: 38)
    private let titleLabel = UILabel()
    private let subtitleLabel = UILabel()
    private let amountLabel = UILabel()

    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        backgroundColor = .clear
        contentView.backgroundColor = .clear
        selectionStyle = .none

        titleLabel.font = .systemFont(ofSize: 15, weight: .semibold)
        titleLabel.textColor = AppTheme.textPrimary
        titleLabel.lineBreakMode = .byTruncatingTail

        subtitleLabel.font = .systemFont(ofSize: 12)
        subtitleLabel.textColor = AppTheme.textSecondary
        subtitleLabel.lineBreakMode = .byTruncatingTail

        amountLabel.font = .systemFont(ofSize: 16, weight: .semibold)
        amountLabel.textAlignment = .right
        // 金额永远完整显示，需要压缩时压左边的文字。
        amountLabel.setContentCompressionResistancePriority(.required, for: .horizontal)
        titleLabel.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        subtitleLabel.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)

        let textStack = UIStackView(arrangedSubviews: [titleLabel, subtitleLabel])
        textStack.axis = .vertical
        textStack.spacing = 2

        contentView.addForAutoLayout(card)
        card.addForAutoLayout(badge)
        card.addForAutoLayout(textStack)
        card.addForAutoLayout(amountLabel)

        NSLayoutConstraint.activate([
            card.topAnchor.constraint(equalTo: contentView.topAnchor, constant: 3),
            card.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -3),
            card.leadingAnchor.constraint(equalTo: contentView.leadingAnchor,
                                          constant: AppTheme.horizontalPadding),
            card.trailingAnchor.constraint(equalTo: contentView.trailingAnchor,
                                           constant: -AppTheme.horizontalPadding),

            badge.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 12),
            badge.centerYAnchor.constraint(equalTo: card.centerYAnchor),

            textStack.leadingAnchor.constraint(equalTo: badge.trailingAnchor, constant: 12),
            textStack.centerYAnchor.constraint(equalTo: card.centerYAnchor),

            amountLabel.leadingAnchor.constraint(equalTo: textStack.trailingAnchor, constant: 8),
            amountLabel.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -12),
            amountLabel.centerYAnchor.constraint(equalTo: card.centerYAnchor)
        ])
    }

    required init?(coder: NSCoder) { nil }

    func configure(transaction: LedgerTransaction, currencyCode: String) {
        let store = LedgerStore.shared
        let account = store.account(id: transaction.accountId)

        switch transaction.kind {
        case .transfer:
            // 转账没有分类，用一个箭头图标 + 「A → B」把两端说清楚。
            badge.configure(symbolName: "arrow.left.arrow.right", color: AppTheme.textSecondary)
            titleLabel.text = "Transfer"
            let to = store.account(id: transaction.toAccountId)?.name ?? "—"
            subtitleLabel.text = "\(account?.name ?? "—") \u{2192} \(to)"
            amountLabel.textColor = AppTheme.textSecondary
        case .expense, .income:
            let category = store.category(id: transaction.categoryId)
            badge.configure(
                symbolName: category?.symbolName ?? "questionmark.circle.fill",
                color: AppTheme.paletteColor(category?.colorIndex ?? 0)
            )
            titleLabel.text = category?.name ?? "Uncategorized"
            subtitleLabel.text = account?.name ?? "—"
            amountLabel.textColor = transaction.kind == .expense ? AppTheme.expense : AppTheme.income
        }

        // 有备注就把它接在副标题后面 —— 备注往往比账户名更能提醒这笔是什么。
        let note = transaction.note.trimmingCharacters(in: .whitespacesAndNewlines)
        if !note.isEmpty {
            let base = subtitleLabel.text ?? ""
            subtitleLabel.text = base.isEmpty ? note : "\(base) \u{00B7} \(note)"
        }

        amountLabel.text = MoneyFormatter.signed(
            transaction.amount,
            kind: transaction.kind,
            currencyCode: currencyCode
        )
    }
}
