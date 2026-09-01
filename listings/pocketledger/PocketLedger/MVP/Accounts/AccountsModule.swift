import UIKit

/// 账户列表。每个账户显示类型图标、名字、类型与当前余额；顶部是全部账户的合计。
final class AccountsViewController: LedgerObservingViewController {
    private let tableView = UITableView(frame: .zero, style: .plain)
    private let headerView = AccountsHeaderView()
    private var accounts: [Account] = []
    private var emptyState: EmptyStateView?

    override func viewDidLoad() {
        super.viewDidLoad()

        navigationItem.rightBarButtonItem = UIBarButtonItem(
            barButtonSystemItem: .add,
            target: self,
            action: #selector(addAccount)
        )

        tableView.backgroundColor = AppTheme.background
        tableView.separatorStyle = .none
        tableView.dataSource = self
        tableView.delegate = self
        tableView.register(AccountCell.self, forCellReuseIdentifier: AccountCell.reuseID)
        tableView.rowHeight = 76
        tableView.contentInset = UIEdgeInsets(top: 8, left: 0, bottom: 24, right: 0)
        view.addForAutoLayout(tableView)
        tableView.pinToEdges(of: view)

        // tableHeaderView 不参与外层 Auto Layout，必须自己给 frame。
        // 先挂上占位高度，真实高度在 viewDidLayoutSubviews 里按内部约束算出来。
        headerView.frame = CGRect(x: 0, y: 0, width: view.bounds.width, height: 120)
        tableView.tableHeaderView = headerView

        reloadContent()
    }

    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        sizeHeaderToFit()
    }

    /// 把表头高度调成它内部约束真正需要的高度。
    ///
    /// **别写死高度**：表头里是「小标题 + 30pt 粗体金额 + 副标题」加上下内边距，
    /// 实际需要 110 多点；写死一个偏小的值会和内部约束打架，UIKit 只能打断其中一条，
    /// 于是控制台刷 `Unable to simultaneously satisfy constraints`、文字位置也会错。
    /// 而且这个高度随动态字体大小变，写死多少都不对。
    private func sizeHeaderToFit() {
        let width = tableView.bounds.width
        guard width > 0 else { return }
        headerView.frame.size.width = width
        let target = headerView.systemLayoutSizeFitting(
            CGSize(width: width, height: 0),
            withHorizontalFittingPriority: .required,
            verticalFittingPriority: .fittingSizeLevel
        ).height
        // 只在真的变了才重新赋值 —— 给 tableHeaderView 赋值会再触发一次布局，
        // 无条件赋值就是死循环。
        guard abs(headerView.frame.height - target) > 0.5 else { return }
        headerView.frame.size.height = target
        tableView.tableHeaderView = headerView
    }

    override func reloadContent() {
        let store = LedgerStore.shared
        accounts = store.accounts.sorted { $0.createdAt < $1.createdAt }
        headerView.setTotal(MoneyFormatter.string(store.totalBalance, currencyCode: currencyCode))
        headerView.setSubtitle(accounts.count == 1 ? "1 account" : "\(accounts.count) accounts")
        tableView.reloadData()
        updateEmptyState()
    }

    private func updateEmptyState() {
        if accounts.isEmpty {
            guard emptyState == nil else { return }
            let empty = EmptyStateView(
                symbolName: "creditcard",
                title: "No accounts yet",
                message: "Add a card, an e-wallet or a cash account to start tracking where your money sits.",
                actionTitle: "Add account"
            )
            empty.onAction { [weak self] in self?.addAccount() }
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

    @objc private func addAccount() {
        present(editor: AccountEditorViewController(account: nil))
    }

    private func present(editor: AccountEditorViewController) {
        let nav = UINavigationController(rootViewController: editor)
        AppTheme.applyNavigationAppearance(nav.navigationBar)
        present(nav, animated: true)
    }
}

extension AccountsViewController: UITableViewDataSource, UITableViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        accounts.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: AccountCell.reuseID, for: indexPath)
        if let cell = cell as? AccountCell {
            let account = accounts[indexPath.row]
            cell.configure(
                account: account,
                balance: LedgerStore.shared.balance(of: account),
                currencyCode: currencyCode
            )
        }
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        present(editor: AccountEditorViewController(account: accounts[indexPath.row]))
    }

    func tableView(
        _ tableView: UITableView,
        trailingSwipeActionsConfigurationForRowAt indexPath: IndexPath
    ) -> UISwipeActionsConfiguration? {
        let account = accounts[indexPath.row]
        let delete = UIContextualAction(style: .destructive, title: "Delete") { [weak self] _, _, completion in
            self?.confirmDelete(account: account)
            completion(true)
        }
        return UISwipeActionsConfiguration(actions: [delete])
    }

    /// 删账户前先把「会连带删掉几条流水」告诉用户 —— 这是不可撤销的，
    /// 不能让人删完才发现半年的记录没了。
    private func confirmDelete(account: Account) {
        let count = LedgerMath.transactionCount(
            forAccount: account.id,
            transactions: LedgerStore.shared.transactions
        )
        let message = count == 0
            ? "This account has no entries."
            : (count == 1
                ? "1 entry on this account will also be deleted."
                : "\(count) entries on this account will also be deleted.")
        confirmDestructive(
            title: "Delete \(account.name)?",
            message: message,
            confirmTitle: "Delete"
        ) {
            LedgerStore.shared.deleteAccount(id: account.id)
        }
    }
}

/// 账户页表头：全部账户余额合计。
private final class AccountsHeaderView: UIView {
    private let card = CardView()
    private let titleLabel = UILabel()
    private let totalLabel = UILabel()
    private let subtitleLabel = UILabel()

    init() {
        super.init(frame: .zero)

        titleLabel.attributedText = NSAttributedString(
            string: "TOTAL BALANCE",
            attributes: [.kern: 0.8]
        )
        titleLabel.font = .systemFont(ofSize: 11, weight: .semibold)
        titleLabel.textColor = AppTheme.textSecondary

        totalLabel.font = .systemFont(ofSize: 30, weight: .bold)
        totalLabel.textColor = AppTheme.textPrimary
        totalLabel.adjustsFontSizeToFitWidth = true
        totalLabel.minimumScaleFactor = 0.5

        subtitleLabel.font = .systemFont(ofSize: 13)
        subtitleLabel.textColor = AppTheme.textSecondary

        let stack = UIStackView(arrangedSubviews: [titleLabel, totalLabel, subtitleLabel])
        stack.axis = .vertical
        stack.spacing = 2

        addForAutoLayout(card)
        card.addForAutoLayout(stack)
        NSLayoutConstraint.activate([
            card.topAnchor.constraint(equalTo: topAnchor, constant: 4),
            card.leadingAnchor.constraint(equalTo: leadingAnchor, constant: AppTheme.horizontalPadding),
            card.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -AppTheme.horizontalPadding),
            card.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -12),
            stack.topAnchor.constraint(equalTo: card.topAnchor, constant: 16),
            stack.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 18),
            stack.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -18),
            stack.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -16)
        ])
    }

    required init?(coder: NSCoder) { nil }

    func setTotal(_ text: String) { totalLabel.text = text }
    func setSubtitle(_ text: String) { subtitleLabel.text = text }
}

/// 账户行。
private final class AccountCell: UITableViewCell {
    static let reuseID = "AccountCell"

    private let card = CardView()
    private let badge = IconBadgeView(diameter: 42)
    private let nameLabel = UILabel()
    private let kindLabel = UILabel()
    private let balanceLabel = UILabel()

    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        backgroundColor = .clear
        contentView.backgroundColor = .clear
        selectionStyle = .none

        nameLabel.font = .systemFont(ofSize: 16, weight: .semibold)
        nameLabel.textColor = AppTheme.textPrimary

        kindLabel.font = .systemFont(ofSize: 13)
        kindLabel.textColor = AppTheme.textSecondary

        balanceLabel.font = .systemFont(ofSize: 17, weight: .semibold)
        balanceLabel.textColor = AppTheme.textPrimary
        balanceLabel.textAlignment = .right
        // 名字很长时先压缩名字、保住金额 —— 余额是这一行的主信息。
        balanceLabel.setContentCompressionResistancePriority(.required, for: .horizontal)
        nameLabel.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        nameLabel.lineBreakMode = .byTruncatingTail

        let textStack = UIStackView(arrangedSubviews: [nameLabel, kindLabel])
        textStack.axis = .vertical
        textStack.spacing = 2

        contentView.addForAutoLayout(card)
        card.addForAutoLayout(badge)
        card.addForAutoLayout(textStack)
        card.addForAutoLayout(balanceLabel)

        NSLayoutConstraint.activate([
            card.topAnchor.constraint(equalTo: contentView.topAnchor, constant: 4),
            card.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -4),
            card.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: AppTheme.horizontalPadding),
            card.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -AppTheme.horizontalPadding),

            badge.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 14),
            badge.centerYAnchor.constraint(equalTo: card.centerYAnchor),

            textStack.leadingAnchor.constraint(equalTo: badge.trailingAnchor, constant: 12),
            textStack.centerYAnchor.constraint(equalTo: card.centerYAnchor),

            balanceLabel.leadingAnchor.constraint(equalTo: textStack.trailingAnchor, constant: 10),
            balanceLabel.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -14),
            balanceLabel.centerYAnchor.constraint(equalTo: card.centerYAnchor)
        ])
    }

    required init?(coder: NSCoder) { nil }

    func configure(account: Account, balance: Decimal, currencyCode: String) {
        badge.configure(
            symbolName: account.kind.symbolName,
            color: AppTheme.paletteColor(account.colorIndex)
        )
        nameLabel.text = account.name
        kindLabel.text = account.kind.displayName
        balanceLabel.text = MoneyFormatter.string(balance, currencyCode: currencyCode)
        // 余额为负（比如信用卡欠款）标红。**按数值判断而不是看字符串开头** ——
        // 货币格式里的负数在不同地区长得不一样（"-₱100"、"(₱100)"、"₱-100"），
        // 判前缀会在部分地区失灵。
        balanceLabel.textColor = balance < 0 ? AppTheme.expense : AppTheme.textPrimary
    }
}
