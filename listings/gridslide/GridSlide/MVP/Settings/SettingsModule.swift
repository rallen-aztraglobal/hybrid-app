import UIKit

/// 设置：默认棋盘尺寸、震动反馈、清空成绩、隐私政策。
final class SettingsViewController: UIViewController {
    private let scrollView = UIScrollView()
    private let sizeRow = FormRowView(title: "Board size")
    private let hapticsRow = FormRowView(title: "Haptics", showsChevron: false)
    private let hapticsSwitch = UISwitch()
    private let privacyRow = FormRowView(title: "Privacy policy")

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.background
        buildLayout()
        refresh()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        refresh()
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
            content.leadingAnchor.constraint(
                equalTo: scrollView.frameLayoutGuide.leadingAnchor,
                constant: AppTheme.horizontalPadding
            ),
            content.trailingAnchor.constraint(
                equalTo: scrollView.frameLayoutGuide.trailingAnchor,
                constant: -AppTheme.horizontalPadding
            )
        ])

        sizeRow.addTarget(self, action: #selector(pickSize), for: .touchUpInside)
        privacyRow.addTarget(self, action: #selector(openPrivacyPolicy), for: .touchUpInside)

        hapticsSwitch.onTintColor = AppTheme.accent
        hapticsSwitch.addTarget(self, action: #selector(hapticsChanged), for: .valueChanged)
        hapticsRow.attachSwitch(hapticsSwitch)

        let rowsCard = CardView()
        let rowsStack = UIStackView(arrangedSubviews: [
            sizeRow, HairlineView(), hapticsRow, HairlineView(), privacyRow
        ])
        rowsStack.axis = .vertical
        rowsCard.addForAutoLayout(rowsStack)
        rowsStack.pinToEdges(of: rowsCard)
        content.addArrangedSubview(rowsCard)

        // 数据在哪、会不会被上传——直接写在设置页上，别让用户去猜。
        let note = UILabel()
        note.font = .systemFont(ofSize: 13)
        note.textColor = AppTheme.textSecondary
        note.numberOfLines = 0
        note.text = "Your best scores are stored on this device only. "
            + "There is no account and nothing to sign in to."
        content.addArrangedSubview(note)

        let resetButton = UIButton(type: .system)
        var resetConfig = UIButton.Configuration.plain()
        resetConfig.title = "Reset all records"
        resetConfig.baseForegroundColor = AppTheme.danger
        resetConfig.contentInsets = NSDirectionalEdgeInsets(top: 14, leading: 0, bottom: 14, trailing: 0)
        resetButton.configuration = resetConfig
        resetButton.addTarget(self, action: #selector(resetRecords), for: .touchUpInside)

        let resetCard = CardView()
        resetCard.addForAutoLayout(resetButton)
        resetButton.pinToEdges(of: resetCard)
        content.addArrangedSubview(resetCard)

        let version = UILabel()
        version.text = "GridSlide \(SettingsStore.appVersionDisplay)"
        version.font = .systemFont(ofSize: 12)
        version.textColor = AppTheme.textSecondary
        version.textAlignment = .center
        content.addArrangedSubview(version)
    }

    private func refresh() {
        let size = SettingsStore.shared.boardSize
        sizeRow.setValue("\(size)×\(size)")
        hapticsSwitch.isOn = SettingsStore.shared.hapticsEnabled
    }

    // MARK: - 动作

    @objc private func pickSize() {
        let sheet = UIAlertController(title: "Board size", message: nil, preferredStyle: .actionSheet)
        for size in SettingsStore.boardSizes {
            sheet.addAction(UIAlertAction(title: "\(size)×\(size)", style: .default) { [weak self] _ in
                SettingsStore.shared.boardSize = size
                self?.refresh()
            })
        }
        sheet.addAction(UIAlertAction(title: "Cancel", style: .cancel))
        // 本包只发 iPhone，但 iPad 兼容模式下 action sheet 缺锚点会直接崩，加上很便宜。
        if let popover = sheet.popoverPresentationController {
            popover.sourceView = sizeRow
            popover.sourceRect = sizeRow.bounds
        }
        present(sheet, animated: true)
    }

    @objc private func hapticsChanged() {
        SettingsStore.shared.hapticsEnabled = hapticsSwitch.isOn
        // 开启的那一下先震一次，让用户当场知道这个开关管的是什么。
        if hapticsSwitch.isOn {
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
        }
    }

    @objc private func openPrivacyPolicy() {
        guard let url = SettingsStore.privacyPolicyURL else {
            // 政策页还没托管（URL 仍是占位符）时给一句明确提示，而不是静默什么都不做。
            showBanner(message: "The privacy policy link is not set up yet.", style: .failure)
            return
        }
        UIApplication.shared.open(url)
    }

    @objc private func resetRecords() {
        confirmDestructive(
            title: "Reset all records?",
            message: "Best moves and best times for every board size will be cleared. "
                + "This cannot be undone.",
            confirmTitle: "Reset"
        ) { [weak self] in
            RecordStore.shared.reset()
            self?.showBanner(message: "Records cleared.", style: .success)
        }
    }
}
