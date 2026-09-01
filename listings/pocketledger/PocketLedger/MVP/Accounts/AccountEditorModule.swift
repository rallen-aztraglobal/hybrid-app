import UIKit

/// 新建 / 编辑账户。
///
/// 这一页是本 App 的核心设定所在：**账户类型由用户自己选** —— 卡、电子钱包、现金、
/// 银行账户。类型不只是个图标，它决定了用户怎么理解这笔余额（卡可以是负的＝欠款，
/// 电子钱包是预充值的），所以选择器放在名字下面第一位，并配一句说明。
final class AccountEditorViewController: UIViewController {
    /// nil = 新建；非 nil = 编辑既有账户。
    private let existing: Account?

    private let scrollView = UIScrollView()
    private let nameField = UITextField()
    private let kindControl = UISegmentedControl(
        items: AccountKind.allCases.map { $0.shortName }
    )
    private let kindHintLabel = UILabel()
    private let balanceField = UITextField()
    private let colorPicker = ColorPickerView()

    init(account: Account?) {
        self.existing = account
        super.init(nibName: nil, bundle: nil)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) not supported") }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.background
        title = existing == nil ? "New account" : "Edit account"

        navigationItem.leftBarButtonItem = UIBarButtonItem(
            barButtonSystemItem: .cancel, target: self, action: #selector(cancel)
        )
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            barButtonSystemItem: .save, target: self, action: #selector(save)
        )

        buildLayout()
        applyExistingValues()
        updateKindHint()
    }

    // MARK: - 布局

    private func buildLayout() {
        scrollView.keyboardDismissMode = .interactive
        view.addForAutoLayout(scrollView)
        scrollView.pinToEdges(of: view)

        let content = UIStackView()
        content.axis = .vertical
        content.spacing = 18
        scrollView.addForAutoLayout(content)
        NSLayoutConstraint.activate([
            content.topAnchor.constraint(equalTo: scrollView.contentLayoutGuide.topAnchor, constant: 20),
            content.bottomAnchor.constraint(equalTo: scrollView.contentLayoutGuide.bottomAnchor, constant: -32),
            content.leadingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.leadingAnchor,
                                             constant: AppTheme.horizontalPadding),
            content.trailingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.trailingAnchor,
                                              constant: -AppTheme.horizontalPadding)
        ])

        // —— 名字 ——
        nameField.placeholder = "e.g. Main card, GCash, Wallet"
        nameField.font = .systemFont(ofSize: 16)
        nameField.textColor = AppTheme.textPrimary
        nameField.autocapitalizationType = .words
        nameField.clearButtonMode = .whileEditing
        nameField.returnKeyType = .done
        nameField.delegate = self
        content.addArrangedSubview(section(title: "Name", content: fieldRow(nameField)))

        // —— 类型（本页的主角）——
        kindControl.selectedSegmentIndex = 0
        kindControl.addTarget(self, action: #selector(kindChanged), for: .valueChanged)
        kindHintLabel.font = .systemFont(ofSize: 13)
        kindHintLabel.textColor = AppTheme.textSecondary
        kindHintLabel.numberOfLines = 0

        let kindStack = UIStackView(arrangedSubviews: [kindControl, kindHintLabel])
        kindStack.axis = .vertical
        kindStack.spacing = 10
        content.addArrangedSubview(section(title: "Type", content: padded(kindStack)))

        // —— 期初余额 ——
        balanceField.placeholder = "0"
        balanceField.font = .systemFont(ofSize: 16)
        balanceField.textColor = AppTheme.textPrimary
        // decimalPad 没有负号键，所以下面额外给一个「切换正负」按钮。
        balanceField.keyboardType = .decimalPad
        balanceField.delegate = self

        let negateButton = UIButton(type: .system)
        negateButton.setTitle("+/\u{2212}", for: .normal)
        negateButton.titleLabel?.font = .systemFont(ofSize: 16, weight: .semibold)
        negateButton.tintColor = AppTheme.accent
        negateButton.addTarget(self, action: #selector(toggleBalanceSign), for: .touchUpInside)
        negateButton.setContentHuggingPriority(.required, for: .horizontal)

        let balanceStack = UIStackView(arrangedSubviews: [balanceField, negateButton])
        balanceStack.axis = .horizontal
        balanceStack.spacing = 12
        content.addArrangedSubview(
            section(
                title: "Opening balance",
                content: fieldRow(balanceStack),
                footnote: "What is in this account right now. Tap +\u{2212} for a card you still owe on."
            )
        )

        // —— 颜色 ——
        content.addArrangedSubview(section(title: "Color", content: padded(colorPicker)))

        // —— 删除（仅编辑态）——
        if existing != nil {
            let deleteButton = UIButton(type: .system)
            var config = UIButton.Configuration.plain()
            config.title = "Delete account"
            config.baseForegroundColor = AppTheme.expense
            config.contentInsets = NSDirectionalEdgeInsets(top: 14, leading: 0, bottom: 14, trailing: 0)
            deleteButton.configuration = config
            deleteButton.addTarget(self, action: #selector(deleteAccount), for: .touchUpInside)

            let card = CardView()
            card.addForAutoLayout(deleteButton)
            deleteButton.pinToEdges(of: card)
            content.addArrangedSubview(card)
        }
    }

    /// 一个「小标题 + 白卡片内容 + 可选脚注」的表单段落。
    private func section(title: String, content: UIView, footnote: String? = nil) -> UIView {
        let titleLabel = UILabel()
        titleLabel.attributedText = NSAttributedString(
            string: title.uppercased(), attributes: [.kern: 0.8]
        )
        titleLabel.font = .systemFont(ofSize: 11, weight: .semibold)
        titleLabel.textColor = AppTheme.textSecondary

        let stack = UIStackView(arrangedSubviews: [titleLabel, content])
        stack.axis = .vertical
        stack.spacing = 8

        if let footnote {
            let footLabel = UILabel()
            footLabel.text = footnote
            footLabel.font = .systemFont(ofSize: 12)
            footLabel.textColor = AppTheme.textSecondary
            footLabel.numberOfLines = 0
            stack.addArrangedSubview(footLabel)
            stack.setCustomSpacing(8, after: content)
        }
        return stack
    }

    /// 把一个控件放进白卡片里，四周留出内边距。
    private func padded(_ inner: UIView) -> UIView {
        let card = CardView()
        card.addForAutoLayout(inner)
        NSLayoutConstraint.activate([
            inner.topAnchor.constraint(equalTo: card.topAnchor, constant: 14),
            inner.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 14),
            inner.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -14),
            inner.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -14)
        ])
        return card
    }

    private func fieldRow(_ inner: UIView) -> UIView {
        let card = CardView()
        card.addForAutoLayout(inner)
        NSLayoutConstraint.activate([
            inner.topAnchor.constraint(equalTo: card.topAnchor, constant: 14),
            inner.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 16),
            inner.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -16),
            inner.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -14),
            inner.heightAnchor.constraint(greaterThanOrEqualToConstant: 24)
        ])
        return card
    }

    // MARK: - 取值 / 存值

    private func applyExistingValues() {
        guard let account = existing else {
            colorPicker.select(index: 0)
            return
        }
        nameField.text = account.name
        if let index = AccountKind.allCases.firstIndex(of: account.kind) {
            kindControl.selectedSegmentIndex = index
        }
        // 期初余额为 0 时留空，让占位符「0」出面 —— 显示一个孤零零的 "0" 更像是没填。
        // 用 editableText 而不是 "\(amount)"：后者恒以 "." 作小数点，
        // 在以 "." 作分组符的地区会被 parseSigned 剥成整数（见 MoneyFormatter 注释）。
        if account.openingBalance != 0 {
            balanceField.text = MoneyFormatter.editableText(account.openingBalance)
        }
        colorPicker.select(index: account.colorIndex)
    }

    private var selectedKind: AccountKind {
        let index = kindControl.selectedSegmentIndex
        let all = AccountKind.allCases
        guard index >= 0 && index < all.count else { return .cash }
        return all[index]
    }

    @objc private func kindChanged() {
        updateKindHint()
    }

    private func updateKindHint() {
        kindHintLabel.text = selectedKind.hint
    }

    @objc private func toggleBalanceSign() {
        let text = balanceField.text ?? ""
        guard !text.isEmpty else { return }
        if text.hasPrefix("-") {
            balanceField.text = String(text.dropFirst())
        } else if text.hasPrefix("\u{2212}") {
            balanceField.text = String(text.dropFirst())
        } else {
            balanceField.text = "-" + text
        }
    }

    @objc private func cancel() {
        dismiss(animated: true)
    }

    @objc private func save() {
        let name = (nameField.text ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        guard !name.isEmpty else {
            showBanner(message: "Give this account a name.", style: .failure)
            return
        }

        // 留空 = 0；填了但解析不出来才算错。
        let rawBalance = (balanceField.text ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let openingBalance: Decimal
        if rawBalance.isEmpty {
            openingBalance = 0
        } else if let parsed = MoneyFormatter.parseSigned(rawBalance) {
            openingBalance = parsed
        } else {
            showBanner(message: "That opening balance is not a valid number.", style: .failure)
            return
        }

        if var account = existing {
            account.name = name
            account.kind = selectedKind
            account.openingBalance = openingBalance
            account.colorIndex = colorPicker.selectedIndex
            LedgerStore.shared.updateAccount(account)
        } else {
            LedgerStore.shared.addAccount(
                Account(
                    name: name,
                    kind: selectedKind,
                    openingBalance: openingBalance,
                    colorIndex: colorPicker.selectedIndex
                )
            )
        }
        dismiss(animated: true)
    }

    @objc private func deleteAccount() {
        guard let account = existing else { return }
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
        ) { [weak self] in
            LedgerStore.shared.deleteAccount(id: account.id)
            // 下一个 runloop 再关：alert 的关闭动画此刻未必走完，
            // 直接 dismiss 有可能被消耗在 alert 上，留下一个「删了但没关」的编辑页。
            DispatchQueue.main.async { self?.dismiss(animated: true) }
        }
    }
}

extension AccountEditorViewController: UITextFieldDelegate {
    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        textField.resignFirstResponder()
        return true
    }
}

/// 一排可选的圆形色块。选中的那个套一圈描边。
final class ColorPickerView: UIView {
    private(set) var selectedIndex = 0
    private var dots: [UIButton] = []

    init() {
        super.init(frame: .zero)
        let stack = UIStackView()
        stack.axis = .horizontal
        stack.distribution = .fillEqually
        stack.spacing = 10

        for index in 0..<AppTheme.palette.count {
            let dot = UIButton(type: .custom)
            dot.backgroundColor = AppTheme.paletteColor(index)
            dot.layer.cornerRadius = 15
            dot.layer.borderColor = AppTheme.textPrimary.cgColor
            dot.tag = index
            dot.addTarget(self, action: #selector(pick(_:)), for: .touchUpInside)
            dot.heightAnchor.constraint(equalToConstant: 30).isActive = true
            dots.append(dot)
            stack.addArrangedSubview(dot)
        }

        addForAutoLayout(stack)
        stack.pinToEdges(of: self)
        select(index: 0)
    }

    required init?(coder: NSCoder) { nil }

    func select(index: Int) {
        let clamped = (index >= 0 && index < dots.count) ? index : 0
        selectedIndex = clamped
        for (i, dot) in dots.enumerated() {
            dot.layer.borderWidth = (i == clamped) ? 3 : 0
        }
    }

    @objc private func pick(_ sender: UIButton) {
        select(index: sender.tag)
    }
}
