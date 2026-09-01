import UIKit

/// 总览：净资产、本月收支、钱花在哪、最近几笔。
///
/// 这一页只读、不做任何编辑 —— 打开 App 第一眼要回答的是「我还有多少钱、这个月花超了没」，
/// 记账动作在流水页完成。
final class OverviewViewController: LedgerObservingViewController {
    private let scrollView = UIScrollView()
    private let contentStack = UIStackView()

    private let balanceLabel = UILabel()
    private let monthLabel = UILabel()
    private let spentTile = StatTileView(title: "Spent", valueColor: AppTheme.expense)
    private let receivedTile = StatTileView(title: "Received", valueColor: AppTheme.income)

    /// 「钱花在哪」与「最近几笔」两块每次刷新都整体重建，故单独留出容器。
    private let categoriesCard = CardView()
    private let categoriesStack = UIStackView()
    private let recentCard = CardView()
    private let recentStack = UIStackView()

    override func viewDidLoad() {
        super.viewDidLoad()
        buildLayout()
        reloadContent()
    }

    private func buildLayout() {
        view.addForAutoLayout(scrollView)
        scrollView.pinToEdges(of: view)

        contentStack.axis = .vertical
        contentStack.spacing = 18
        scrollView.addForAutoLayout(contentStack)
        NSLayoutConstraint.activate([
            contentStack.topAnchor.constraint(equalTo: scrollView.contentLayoutGuide.topAnchor, constant: 12),
            contentStack.bottomAnchor.constraint(equalTo: scrollView.contentLayoutGuide.bottomAnchor, constant: -28),
            contentStack.leadingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.leadingAnchor,
                                                  constant: AppTheme.horizontalPadding),
            contentStack.trailingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.trailingAnchor,
                                                   constant: -AppTheme.horizontalPadding)
        ])

        contentStack.addArrangedSubview(makeBalanceCard())
        contentStack.addArrangedSubview(makeMonthCard())
        contentStack.addArrangedSubview(makeCategoriesCard())
        contentStack.addArrangedSubview(makeRecentCard())
    }

    // MARK: - 卡片

    private func makeBalanceCard() -> UIView {
        let caption = UILabel()
        caption.attributedText = NSAttributedString(
            string: "NET BALANCE", attributes: [.kern: 0.8]
        )
        caption.font = .systemFont(ofSize: 11, weight: .semibold)
        caption.textColor = AppTheme.textSecondary

        balanceLabel.font = .systemFont(ofSize: 34, weight: .bold)
        balanceLabel.textColor = AppTheme.textPrimary
        balanceLabel.adjustsFontSizeToFitWidth = true
        balanceLabel.minimumScaleFactor = 0.5

        let hint = UILabel()
        hint.text = "Across all your accounts"
        hint.font = .systemFont(ofSize: 13)
        hint.textColor = AppTheme.textSecondary

        let stack = UIStackView(arrangedSubviews: [caption, balanceLabel, hint])
        stack.axis = .vertical
        stack.spacing = 2

        let card = CardView()
        card.addForAutoLayout(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: card.topAnchor, constant: 18),
            stack.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 18),
            stack.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -18),
            stack.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -18)
        ])
        return card
    }

    private func makeMonthCard() -> UIView {
        monthLabel.font = .systemFont(ofSize: 15, weight: .semibold)
        monthLabel.textColor = AppTheme.textPrimary

        let tiles = UIStackView(arrangedSubviews: [spentTile, receivedTile])
        tiles.axis = .horizontal
        tiles.distribution = .fillEqually
        tiles.spacing = 12

        let stack = UIStackView(arrangedSubviews: [monthLabel, tiles])
        stack.axis = .vertical
        stack.spacing = 14

        let card = CardView()
        card.addForAutoLayout(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: card.topAnchor, constant: 18),
            stack.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 18),
            stack.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -18),
            stack.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -18)
        ])
        return card
    }

    private func makeCategoriesCard() -> UIView {
        categoriesStack.axis = .vertical
        categoriesStack.spacing = 12
        categoriesCard.addForAutoLayout(categoriesStack)
        NSLayoutConstraint.activate([
            categoriesStack.topAnchor.constraint(equalTo: categoriesCard.topAnchor, constant: 18),
            categoriesStack.leadingAnchor.constraint(equalTo: categoriesCard.leadingAnchor, constant: 18),
            categoriesStack.trailingAnchor.constraint(equalTo: categoriesCard.trailingAnchor, constant: -18),
            categoriesStack.bottomAnchor.constraint(equalTo: categoriesCard.bottomAnchor, constant: -18)
        ])
        return categoriesCard
    }

    private func makeRecentCard() -> UIView {
        recentStack.axis = .vertical
        recentStack.spacing = 12
        recentCard.addForAutoLayout(recentStack)
        NSLayoutConstraint.activate([
            recentStack.topAnchor.constraint(equalTo: recentCard.topAnchor, constant: 18),
            recentStack.leadingAnchor.constraint(equalTo: recentCard.leadingAnchor, constant: 18),
            recentStack.trailingAnchor.constraint(equalTo: recentCard.trailingAnchor, constant: -18),
            recentStack.bottomAnchor.constraint(equalTo: recentCard.bottomAnchor, constant: -18)
        ])
        return recentCard
    }

    // MARK: - 刷新

    override func reloadContent() {
        let store = LedgerStore.shared
        let month = LedgerMath.monthInterval(containing: Date())
        let code = currencyCode

        balanceLabel.text = MoneyFormatter.string(store.totalBalance, currencyCode: code)
        balanceLabel.textColor = store.totalBalance < 0 ? AppTheme.expense : AppTheme.textPrimary

        let formatter = DateFormatter()
        formatter.dateFormat = "LLLL yyyy"
        monthLabel.text = formatter.string(from: Date())

        let spent = LedgerMath.total(kind: .expense, transactions: store.transactions, in: month)
        let received = LedgerMath.total(kind: .income, transactions: store.transactions, in: month)
        spentTile.setValue(MoneyFormatter.string(spent, currencyCode: code))
        receivedTile.setValue(MoneyFormatter.string(received, currencyCode: code))

        rebuildCategories(store: store, month: month, code: code, monthlySpend: spent)
        rebuildRecent(store: store, code: code)
    }

    private func rebuildCategories(store: LedgerStore, month: DateInterval, code: String, monthlySpend: Decimal) {
        categoriesStack.arrangedSubviews.forEach { $0.removeFromSuperview() }

        let title = sectionTitle("Where it went")
        categoriesStack.addArrangedSubview(title)

        let totals = LedgerMath.categoryTotals(
            kind: .expense, transactions: store.transactions, in: month
        )
        guard let largest = totals.first?.total, largest > 0 else {
            categoriesStack.addArrangedSubview(
                placeholderLabel("No spending recorded this month yet.")
            )
            return
        }

        // 只列前五 —— 再往下的分类金额通常已经很小，列全反而看不出重点。
        for entry in totals.prefix(5) {
            let category = store.category(id: entry.categoryId)
            let row = CategoryBarRow()
            row.configure(
                name: category?.name ?? "Uncategorized",
                amount: MoneyFormatter.string(entry.total, currencyCode: code),
                // 条长按「占最大那一项的比例」画，不是占总额 ——
                // 占总额的话第一名往往也只有 30%，五根条都短得看不出差别。
                ratio: ratio(entry.total, of: largest),
                color: AppTheme.paletteColor(category?.colorIndex ?? 0)
            )
            categoriesStack.addArrangedSubview(row)
        }

        if monthlySpend > 0 && totals.count > 5 {
            categoriesStack.addArrangedSubview(
                placeholderLabel("and \(totals.count - 5) more")
            )
        }
    }

    /// Decimal 没有直接转 CGFloat 的构造器，经 NSDecimalNumber 的 doubleValue 过一手。
    /// 这里只是画条形图的长度，精度损失无关紧要（金额本身仍是 Decimal）。
    private func ratio(_ value: Decimal, of largest: Decimal) -> CGFloat {
        guard largest > 0 else { return 0 }
        let v = NSDecimalNumber(decimal: value).doubleValue
        let m = NSDecimalNumber(decimal: largest).doubleValue
        guard m > 0 else { return 0 }
        return CGFloat(max(0.0, min(1.0, v / m)))
    }

    private func rebuildRecent(store: LedgerStore, code: String) {
        recentStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        recentStack.addArrangedSubview(sectionTitle("Recent"))

        let recent = Array(store.transactionsNewestFirst().prefix(5))
        guard !recent.isEmpty else {
            recentStack.addArrangedSubview(placeholderLabel("Nothing recorded yet."))
            return
        }

        for transaction in recent {
            recentStack.addArrangedSubview(RecentEntryRow(transaction: transaction, currencyCode: code))
        }
    }

    private func sectionTitle(_ text: String) -> UILabel {
        let label = UILabel()
        label.text = text
        label.font = .systemFont(ofSize: 15, weight: .semibold)
        label.textColor = AppTheme.textPrimary
        return label
    }

    private func placeholderLabel(_ text: String) -> UILabel {
        let label = UILabel()
        label.text = text
        label.font = .systemFont(ofSize: 13)
        label.textColor = AppTheme.textSecondary
        label.numberOfLines = 0
        return label
    }
}

/// 「分类 + 金额 + 一根比例条」。
private final class CategoryBarRow: UIView {
    private let nameLabel = UILabel()
    private let amountLabel = UILabel()
    private let track = UIView()
    private let fill = UIView()
    private var fillWidthConstraint: NSLayoutConstraint!

    init() {
        super.init(frame: .zero)

        nameLabel.font = .systemFont(ofSize: 14)
        nameLabel.textColor = AppTheme.textPrimary
        nameLabel.lineBreakMode = .byTruncatingTail

        amountLabel.font = .systemFont(ofSize: 14, weight: .semibold)
        amountLabel.textColor = AppTheme.textPrimary
        amountLabel.textAlignment = .right
        amountLabel.setContentCompressionResistancePriority(.required, for: .horizontal)
        nameLabel.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)

        track.backgroundColor = AppTheme.separator
        track.layer.cornerRadius = 3
        fill.layer.cornerRadius = 3

        let labels = UIStackView(arrangedSubviews: [nameLabel, amountLabel])
        labels.axis = .horizontal
        labels.spacing = 8

        addForAutoLayout(labels)
        addForAutoLayout(track)
        track.addForAutoLayout(fill)

        fillWidthConstraint = fill.widthAnchor.constraint(equalTo: track.widthAnchor, multiplier: 0)
        NSLayoutConstraint.activate([
            labels.topAnchor.constraint(equalTo: topAnchor),
            labels.leadingAnchor.constraint(equalTo: leadingAnchor),
            labels.trailingAnchor.constraint(equalTo: trailingAnchor),

            track.topAnchor.constraint(equalTo: labels.bottomAnchor, constant: 6),
            track.leadingAnchor.constraint(equalTo: leadingAnchor),
            track.trailingAnchor.constraint(equalTo: trailingAnchor),
            track.bottomAnchor.constraint(equalTo: bottomAnchor),
            track.heightAnchor.constraint(equalToConstant: 6),

            fill.leadingAnchor.constraint(equalTo: track.leadingAnchor),
            fill.topAnchor.constraint(equalTo: track.topAnchor),
            fill.bottomAnchor.constraint(equalTo: track.bottomAnchor),
            fillWidthConstraint
        ])
    }

    required init?(coder: NSCoder) { nil }

    func configure(name: String, amount: String, ratio: CGFloat, color: UIColor) {
        nameLabel.text = name
        amountLabel.text = amount
        fill.backgroundColor = color
        // multiplier 不能改，只能换一条约束。ratio 为 0 时给一个极小值，
        // 免得 multiplier=0 让约束退化。
        fillWidthConstraint.isActive = false
        fillWidthConstraint = fill.widthAnchor.constraint(
            equalTo: track.widthAnchor,
            multiplier: max(0.02, ratio)
        )
        fillWidthConstraint.isActive = true
    }
}

/// 总览页「最近」里的一行，比流水页的 cell 更轻。
private final class RecentEntryRow: UIView {
    init(transaction: LedgerTransaction, currencyCode: String) {
        super.init(frame: .zero)

        let store = LedgerStore.shared
        let title = UILabel()
        title.font = .systemFont(ofSize: 14)
        title.textColor = AppTheme.textPrimary
        title.lineBreakMode = .byTruncatingTail

        let subtitle = UILabel()
        subtitle.font = .systemFont(ofSize: 12)
        subtitle.textColor = AppTheme.textSecondary

        let amount = UILabel()
        amount.font = .systemFont(ofSize: 14, weight: .semibold)
        amount.textAlignment = .right
        amount.setContentCompressionResistancePriority(.required, for: .horizontal)
        title.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)

        switch transaction.kind {
        case .transfer:
            title.text = "Transfer"
            let from = store.account(id: transaction.accountId)?.name ?? "—"
            let to = store.account(id: transaction.toAccountId)?.name ?? "—"
            subtitle.text = "\(from) \u{2192} \(to)"
            amount.textColor = AppTheme.textSecondary
        case .expense, .income:
            title.text = store.category(id: transaction.categoryId)?.name ?? "Uncategorized"
            subtitle.text = store.account(id: transaction.accountId)?.name ?? "—"
            amount.textColor = transaction.kind == .expense ? AppTheme.expense : AppTheme.income
        }
        amount.text = MoneyFormatter.signed(
            transaction.amount, kind: transaction.kind, currencyCode: currencyCode
        )

        let texts = UIStackView(arrangedSubviews: [title, subtitle])
        texts.axis = .vertical
        texts.spacing = 1

        let row = UIStackView(arrangedSubviews: [texts, amount])
        row.axis = .horizontal
        row.alignment = .center
        row.spacing = 8

        addForAutoLayout(row)
        row.pinToEdges(of: self)
    }

    required init?(coder: NSCoder) { nil }
}
