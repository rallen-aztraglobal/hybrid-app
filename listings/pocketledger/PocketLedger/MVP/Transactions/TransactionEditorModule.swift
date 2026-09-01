import UIKit

/// 新建 / 编辑一笔流水。
///
/// 三种方向共用一张表单：支出、收入、账户间转账。转账时「分类」行换成「转入账户」行 ——
/// 转账不该有分类（钱没花掉，只是换了个口袋），给它分类会把统计口径搞乱。
final class TransactionEditorViewController: UIViewController {
    private let existing: LedgerTransaction?

    private let scrollView = UIScrollView()
    private let kindControl = UISegmentedControl(
        items: TransactionKind.allCases.map { $0.displayName }
    )
    private let amountField = UITextField()
    private let currencyLabel = UILabel()
    private let accountRow = FormRowView(title: "Account")
    private let toAccountRow = FormRowView(title: "To account")
    private let categoryRow = FormRowView(title: "Category")
    private let datePicker = UIDatePicker()
    private let noteField = UITextField()

    private var selectedAccountId: UUID?
    private var selectedToAccountId: UUID?
    private var selectedCategoryId: UUID?

    init(transaction: LedgerTransaction?) {
        self.existing = transaction
        super.init(nibName: nil, bundle: nil)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) { fatalError("init(coder:) not supported") }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.background
        title = existing == nil ? "New entry" : "Edit entry"

        navigationItem.leftBarButtonItem = UIBarButtonItem(
            barButtonSystemItem: .cancel, target: self, action: #selector(cancel)
        )
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            barButtonSystemItem: .save, target: self, action: #selector(save)
        )

        buildLayout()
        applyInitialValues()
        refreshRows()
    }

    override func viewDidAppear(_ animated: Bool) {
        super.viewDidAppear(animated)
        // 新建时直接把光标放到金额上：记账最高频的动作就是敲一个数字。
        if existing == nil {
            amountField.becomeFirstResponder()
        }
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

        // —— 方向 ——
        kindControl.selectedSegmentIndex = 0
        kindControl.addTarget(self, action: #selector(kindChanged), for: .valueChanged)
        content.addArrangedSubview(kindControl)

        // —— 金额 ——
        currencyLabel.text = UserSettingsStore.shared.currencyCode
        currencyLabel.font = .systemFont(ofSize: 15, weight: .semibold)
        currencyLabel.textColor = AppTheme.textSecondary
        currencyLabel.setContentHuggingPriority(.required, for: .horizontal)

        amountField.placeholder = "0.00"
        amountField.font = .systemFont(ofSize: 34, weight: .bold)
        amountField.textColor = AppTheme.textPrimary
        amountField.keyboardType = .decimalPad
        amountField.adjustsFontSizeToFitWidth = true
        amountField.minimumFontSize = 20

        let amountStack = UIStackView(arrangedSubviews: [currencyLabel, amountField])
        amountStack.axis = .horizontal
        amountStack.alignment = .firstBaseline
        amountStack.spacing = 8

        let amountCard = CardView()
        amountCard.addForAutoLayout(amountStack)
        NSLayoutConstraint.activate([
            amountStack.topAnchor.constraint(equalTo: amountCard.topAnchor, constant: 18),
            amountStack.leadingAnchor.constraint(equalTo: amountCard.leadingAnchor, constant: 18),
            amountStack.trailingAnchor.constraint(equalTo: amountCard.trailingAnchor, constant: -18),
            amountStack.bottomAnchor.constraint(equalTo: amountCard.bottomAnchor, constant: -18)
        ])
        content.addArrangedSubview(amountCard)

        // —— 账户 / 转入账户 / 分类 ——
        accountRow.addTarget(self, action: #selector(pickAccount), for: .touchUpInside)
        toAccountRow.addTarget(self, action: #selector(pickToAccount), for: .touchUpInside)
        categoryRow.addTarget(self, action: #selector(pickCategory), for: .touchUpInside)

        let rowsCard = CardView()
        let rowsStack = UIStackView(arrangedSubviews: [
            accountRow, HairlineView(), toAccountRow, categoryRow
        ])
        rowsStack.axis = .vertical
        rowsCard.addForAutoLayout(rowsStack)
        rowsStack.pinToEdges(of: rowsCard)
        content.addArrangedSubview(rowsCard)

        // —— 日期 ——
        datePicker.datePickerMode = .date
        datePicker.preferredDatePickerStyle = .compact
        // 不允许记未来的账：记账是对已发生的事的记录，未来日期几乎总是误操作。
        datePicker.maximumDate = Date()

        let dateLabel = UILabel()
        dateLabel.text = "Date"
        dateLabel.font = .systemFont(ofSize: 16)
        dateLabel.textColor = AppTheme.textPrimary

        let dateStack = UIStackView(arrangedSubviews: [dateLabel, UIView(), datePicker])
        dateStack.axis = .horizontal
        dateStack.alignment = .center
        dateStack.spacing = 10

        let dateCard = CardView()
        dateCard.addForAutoLayout(dateStack)
        NSLayoutConstraint.activate([
            dateStack.topAnchor.constraint(equalTo: dateCard.topAnchor, constant: 10),
            dateStack.leadingAnchor.constraint(equalTo: dateCard.leadingAnchor, constant: 16),
            dateStack.trailingAnchor.constraint(equalTo: dateCard.trailingAnchor, constant: -12),
            dateStack.bottomAnchor.constraint(equalTo: dateCard.bottomAnchor, constant: -10)
        ])
        content.addArrangedSubview(dateCard)

        // —— 备注 ——
        noteField.placeholder = "Note (optional)"
        noteField.font = .systemFont(ofSize: 16)
        noteField.textColor = AppTheme.textPrimary
        noteField.clearButtonMode = .whileEditing
        noteField.returnKeyType = .done
        noteField.delegate = self

        let noteCard = CardView()
        noteCard.addForAutoLayout(noteField)
        NSLayoutConstraint.activate([
            noteField.topAnchor.constraint(equalTo: noteCard.topAnchor, constant: 14),
            noteField.leadingAnchor.constraint(equalTo: noteCard.leadingAnchor, constant: 16),
            noteField.trailingAnchor.constraint(equalTo: noteCard.trailingAnchor, constant: -16),
            noteField.bottomAnchor.constraint(equalTo: noteCard.bottomAnchor, constant: -14),
            noteField.heightAnchor.constraint(greaterThanOrEqualToConstant: 24)
        ])
        content.addArrangedSubview(noteCard)

        // —— 删除（仅编辑态）——
        if existing != nil {
            let deleteButton = UIButton(type: .system)
            var config = UIButton.Configuration.plain()
            config.title = "Delete entry"
            config.baseForegroundColor = AppTheme.expense
            config.contentInsets = NSDirectionalEdgeInsets(top: 14, leading: 0, bottom: 14, trailing: 0)
            deleteButton.configuration = config
            deleteButton.addTarget(self, action: #selector(deleteEntry), for: .touchUpInside)

            let card = CardView()
            card.addForAutoLayout(deleteButton)
            deleteButton.pinToEdges(of: card)
            content.addArrangedSubview(card)
        }
    }

    // MARK: - 初值

    private func applyInitialValues() {
        let store = LedgerStore.shared

        guard let transaction = existing else {
            selectedAccountId = store.accounts.first?.id
            selectedCategoryId = store.categories(for: .expense).first?.id
            datePicker.date = Date()
            return
        }

        if let index = TransactionKind.allCases.firstIndex(of: transaction.kind) {
            kindControl.selectedSegmentIndex = index
        }
        // 用 editableText 而不是 "\(amount)"：后者恒以 "." 作小数点，
        // 在以 "." 作分组符的地区会被 parseSigned 剥成整数（见 MoneyFormatter 注释）。
        amountField.text = MoneyFormatter.editableText(transaction.amount)
        selectedAccountId = transaction.accountId
        selectedToAccountId = transaction.toAccountId
        selectedCategoryId = transaction.categoryId
        datePicker.date = transaction.date
        noteField.text = transaction.note
    }

    private var selectedKind: TransactionKind {
        let all = TransactionKind.allCases
        let index = kindControl.selectedSegmentIndex
        guard index >= 0 && index < all.count else { return .expense }
        return all[index]
    }

    @objc private func kindChanged() {
        // 换了方向，原来的分类多半不再适用（支出分类 ≠ 收入分类），重挑一个默认值。
        if selectedKind == .transfer {
            selectedCategoryId = nil
            if selectedToAccountId == nil {
                selectedToAccountId = LedgerStore.shared.accounts
                    .first { $0.id != selectedAccountId }?.id
            }
        } else {
            let available = LedgerStore.shared.categories(for: selectedKind)
            if let current = selectedCategoryId,
               available.contains(where: { $0.id == current }) {
                // 保留原选择。
            } else {
                selectedCategoryId = available.first?.id
            }
        }
        refreshRows()
    }

    /// 按当前方向显示/隐藏对应的行，并刷新每行右侧的值。
    private func refreshRows() {
        let store = LedgerStore.shared
        let isTransfer = selectedKind == .transfer

        toAccountRow.isHidden = !isTransfer
        categoryRow.isHidden = isTransfer

        accountRow.setValue(store.account(id: selectedAccountId)?.name ?? "Select")
        toAccountRow.setValue(store.account(id: selectedToAccountId)?.name ?? "Select")
        categoryRow.setValue(store.category(id: selectedCategoryId)?.name ?? "Select")
    }

    // MARK: - 选择器

    @objc private func pickAccount() {
        chooseAccount(title: "Account", exclude: nil) { [weak self] id in
            guard let self else { return }
            self.selectedAccountId = id
            // 转账两端不能是同一个账户；撞了就把转入端清掉，逼用户重选。
            if self.selectedKind == .transfer, self.selectedToAccountId == id {
                self.selectedToAccountId = nil
            }
            self.refreshRows()
        }
    }

    @objc private func pickToAccount() {
        chooseAccount(title: "To account", exclude: selectedAccountId) { [weak self] id in
            self?.selectedToAccountId = id
            self?.refreshRows()
        }
    }

    private func chooseAccount(title: String, exclude: UUID?, onPick: @escaping (UUID) -> Void) {
        let accounts = LedgerStore.shared.accounts.filter { $0.id != exclude }
        guard !accounts.isEmpty else {
            showBanner(message: "You need another account for a transfer.", style: .failure)
            return
        }
        let sheet = UIAlertController(title: title, message: nil, preferredStyle: .actionSheet)
        for account in accounts {
            sheet.addAction(UIAlertAction(title: "\(account.name) \u{00B7} \(account.kind.displayName)",
                                          style: .default) { _ in
                onPick(account.id)
            })
        }
        sheet.addAction(UIAlertAction(title: "Cancel", style: .cancel))
        presentSheet(sheet)
    }

    @objc private func pickCategory() {
        let categories = LedgerStore.shared.categories(for: selectedKind)
        guard !categories.isEmpty else { return }
        let sheet = UIAlertController(title: "Category", message: nil, preferredStyle: .actionSheet)
        for category in categories {
            sheet.addAction(UIAlertAction(title: category.name, style: .default) { [weak self] _ in
                self?.selectedCategoryId = category.id
                self?.refreshRows()
            })
        }
        sheet.addAction(UIAlertAction(title: "Cancel", style: .cancel))
        presentSheet(sheet)
    }

    /// 弹 action sheet。本包只发 iPhone，但 iPad 上以兼容模式跑时 sheet 需要锚点，
    /// 不给就会直接崩 —— 这一步很便宜，加上。
    private func presentSheet(_ sheet: UIAlertController) {
        if let popover = sheet.popoverPresentationController {
            popover.sourceView = view
            popover.sourceRect = CGRect(x: view.bounds.midX, y: view.bounds.midY, width: 0, height: 0)
            popover.permittedArrowDirections = []
        }
        view.endEditing(true)
        present(sheet, animated: true)
    }

    // MARK: - 保存 / 删除

    @objc private func cancel() {
        dismiss(animated: true)
    }

    @objc private func save() {
        guard let amount = MoneyFormatter.parse(amountField.text ?? "") else {
            showBanner(message: "Enter an amount greater than zero.", style: .failure)
            return
        }
        guard let accountId = selectedAccountId else {
            showBanner(message: "Pick an account.", style: .failure)
            return
        }

        let kind = selectedKind
        var toAccountId: UUID?
        var categoryId: UUID?

        if kind == .transfer {
            guard let destination = selectedToAccountId else {
                showBanner(message: "Pick the account to transfer into.", style: .failure)
                return
            }
            guard destination != accountId else {
                showBanner(message: "Pick two different accounts.", style: .failure)
                return
            }
            toAccountId = destination
        } else {
            guard let category = selectedCategoryId else {
                showBanner(message: "Pick a category.", style: .failure)
                return
            }
            categoryId = category
        }

        let note = (noteField.text ?? "").trimmingCharacters(in: .whitespacesAndNewlines)

        if var transaction = existing {
            transaction.kind = kind
            transaction.amount = amount
            transaction.date = datePicker.date
            transaction.accountId = accountId
            transaction.toAccountId = toAccountId
            transaction.categoryId = categoryId
            transaction.note = note
            LedgerStore.shared.updateTransaction(transaction)
        } else {
            LedgerStore.shared.addTransaction(
                LedgerTransaction(
                    kind: kind,
                    amount: amount,
                    date: datePicker.date,
                    accountId: accountId,
                    toAccountId: toAccountId,
                    categoryId: categoryId,
                    note: note
                )
            )
        }
        dismiss(animated: true)
    }

    @objc private func deleteEntry() {
        guard let transaction = existing else { return }
        confirmDestructive(
            title: "Delete this entry?",
            message: "This cannot be undone.",
            confirmTitle: "Delete"
        ) { [weak self] in
            LedgerStore.shared.deleteTransaction(id: transaction.id)
            // 下一个 runloop 再关：alert 的关闭动画此刻未必走完，
            // 直接 dismiss 有可能被消耗在 alert 上，留下一个「删了但没关」的编辑页。
            DispatchQueue.main.async { self?.dismiss(animated: true) }
        }
    }
}

extension TransactionEditorViewController: UITextFieldDelegate {
    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        textField.resignFirstResponder()
        return true
    }
}
