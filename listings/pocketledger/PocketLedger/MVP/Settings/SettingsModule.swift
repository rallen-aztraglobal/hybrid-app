import UIKit

/// 设置：币种、导出、隐私政策、清空数据。
final class SettingsViewController: LedgerObservingViewController {
    private let scrollView = UIScrollView()
    private let currencyRow = FormRowView(title: "Currency")
    private let exportRow = FormRowView(title: "Export as CSV")
    private let privacyRow = FormRowView(title: "Privacy policy")
    private let storageLabel = UILabel()

    override func viewDidLoad() {
        super.viewDidLoad()
        buildLayout()
        reloadContent()
    }

    private func buildLayout() {
        view.addForAutoLayout(scrollView)
        scrollView.pinToEdges(of: view)

        let content = UIStackView()
        content.axis = .vertical
        content.spacing = 18
        scrollView.addForAutoLayout(content)
        NSLayoutConstraint.activate([
            content.topAnchor.constraint(equalTo: scrollView.contentLayoutGuide.topAnchor, constant: 16),
            content.bottomAnchor.constraint(equalTo: scrollView.contentLayoutGuide.bottomAnchor, constant: -32),
            content.leadingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.leadingAnchor,
                                             constant: AppTheme.horizontalPadding),
            content.trailingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.trailingAnchor,
                                              constant: -AppTheme.horizontalPadding)
        ])

        currencyRow.addTarget(self, action: #selector(pickCurrency), for: .touchUpInside)
        exportRow.addTarget(self, action: #selector(exportCSV), for: .touchUpInside)
        privacyRow.addTarget(self, action: #selector(openPrivacyPolicy), for: .touchUpInside)

        let rowsCard = CardView()
        let rowsStack = UIStackView(arrangedSubviews: [
            currencyRow, HairlineView(), exportRow, HairlineView(), privacyRow
        ])
        rowsStack.axis = .vertical
        rowsCard.addForAutoLayout(rowsStack)
        rowsStack.pinToEdges(of: rowsCard)
        content.addArrangedSubview(rowsCard)

        // 数据在哪、会不会被上传 —— 直接写在设置页上，别让用户去猜。
        storageLabel.font = .systemFont(ofSize: 13)
        storageLabel.textColor = AppTheme.textSecondary
        storageLabel.numberOfLines = 0
        storageLabel.text = "Your accounts and entries are stored on this device only. "
            + "They are never uploaded and never leave the app unless you export them yourself."
        content.addArrangedSubview(storageLabel)

        let eraseButton = UIButton(type: .system)
        var eraseConfig = UIButton.Configuration.plain()
        eraseConfig.title = "Erase all data"
        eraseConfig.baseForegroundColor = AppTheme.expense
        eraseConfig.contentInsets = NSDirectionalEdgeInsets(top: 14, leading: 0, bottom: 14, trailing: 0)
        eraseButton.configuration = eraseConfig
        eraseButton.addTarget(self, action: #selector(eraseAll), for: .touchUpInside)

        let eraseCard = CardView()
        eraseCard.addForAutoLayout(eraseButton)
        eraseButton.pinToEdges(of: eraseCard)
        content.addArrangedSubview(eraseCard)

        let version = UILabel()
        version.text = "PocketLedger \(UserSettingsStore.appVersionDisplay)"
        version.font = .systemFont(ofSize: 12)
        version.textColor = AppTheme.textSecondary
        version.textAlignment = .center
        content.addArrangedSubview(version)
    }

    override func reloadContent() {
        currencyRow.setValue(UserSettingsStore.shared.currencyCode)
        exportRow.setValue(nil)
        privacyRow.setValue(nil)
    }

    // MARK: - 动作

    @objc private func pickCurrency() {
        let sheet = UIAlertController(title: "Currency", message: nil, preferredStyle: .actionSheet)
        for code in UserSettingsStore.supportedCurrencies {
            sheet.addAction(UIAlertAction(title: code, style: .default) { [weak self] _ in
                UserSettingsStore.shared.currencyCode = code
                // 币种一变，所有金额的显示都得重画 —— 直接借账本的变更通知走一遍全局刷新。
                NotificationCenter.default.post(name: .ledgerDidChange, object: nil)
                self?.reloadContent()
            })
        }
        sheet.addAction(UIAlertAction(title: "Cancel", style: .cancel))
        presentSheet(sheet)
    }

    /// 导出 CSV 并调起系统分享面板。
    ///
    /// 写到临时目录再分享，而不是把整个 CSV 当字符串丢给分享面板：
    /// 带文件名的附件在邮件/云盘里才是一个能直接打开的 .csv，字符串会变成正文。
    @objc private func exportCSV() {
        let csv = LedgerStore.shared.exportCSV()
        let url = FileManager.default.temporaryDirectory
            .appendingPathComponent("pocketledger-export.csv")
        do {
            try csv.write(to: url, atomically: true, encoding: .utf8)
        } catch {
            showBanner(message: "Could not prepare the export.", style: .failure)
            return
        }

        let share = UIActivityViewController(activityItems: [url], applicationActivities: nil)
        if let popover = share.popoverPresentationController {
            popover.sourceView = exportRow
            popover.sourceRect = exportRow.bounds
        }
        present(share, animated: true)
    }

    @objc private func openPrivacyPolicy() {
        guard let url = UserSettingsStore.privacyPolicyURL else {
            // 政策页还没托管（URL 仍是占位符）时给一句明确提示，而不是静默什么都不做。
            showBanner(message: "The privacy policy link is not set up yet.", style: .failure)
            return
        }
        UIApplication.shared.open(url)
    }

    @objc private func eraseAll() {
        confirmDestructive(
            title: "Erase all data?",
            message: "Every account and entry on this device will be deleted. This cannot be undone.",
            confirmTitle: "Erase"
        ) { [weak self] in
            LedgerStore.shared.eraseAll()
            self?.showBanner(message: "All data erased.", style: .success)
        }
    }

    private func presentSheet(_ sheet: UIAlertController) {
        if let popover = sheet.popoverPresentationController {
            popover.sourceView = currencyRow
            popover.sourceRect = currencyRow.bounds
        }
        present(sheet, animated: true)
    }
}
