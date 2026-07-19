import UIKit

final class SettingsViewController: UIViewController, UITableViewDataSource, UITableViewDelegate {
    private enum Row {
        case timer, guide, privacy, share, version
    }

    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private let rows: [Row] = [.timer, .guide, .privacy, .share, .version]

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "Settings"
        view.backgroundColor = AppTheme.background
        tableView.backgroundColor = .clear
        tableView.dataSource = self
        tableView.delegate = self
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "cell")
        view.addSubview(tableView)
        tableView.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            tableView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            tableView.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor)
        ])
    }

    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { rows.count }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "cell", for: indexPath)
        var config = UIListContentConfiguration.valueCell()
        switch rows[indexPath.row] {
        case .timer:
            config.text = "Round Timer"
            config.secondaryText = UserSettingsStore.shared.timerLabels[UserSettingsStore.shared.timerPresetIndex]
        case .guide:
            config.text = "How To Play"
        case .privacy:
            config.text = "Privacy Policy"
        case .share:
            config.text = "Share App"
        case .version:
            config.text = "Version"
            config.secondaryText = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String
        }
        config.textProperties.color = AppTheme.textPrimary
        config.secondaryTextProperties.color = AppTheme.textSecondary
        cell.contentConfiguration = config
        cell.backgroundColor = AppTheme.surface
        cell.accessoryType = rows[indexPath.row] == .version ? .none : .disclosureIndicator
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let anchor = tableView.cellForRow(at: indexPath)
        switch rows[indexPath.row] {
        case .timer:
            presentTimerPicker(anchor: anchor)
        case .guide:
            let guide = GuideViewController()
            let nav = UINavigationController(rootViewController: guide)
            AppTheme.applyNavigationAppearance(nav.navigationBar)
            present(nav, animated: true)
        case .privacy:
            UIApplication.shared.open(UserSettingsStore.privacyPolicyURL)
        case .share:
            presentShareSheet(anchor: anchor)
        case .version:
            break
        }
    }

    private func presentTimerPicker(anchor: UIView?) {
        let alert = UIAlertController(title: "Round Timer", message: "Choose the countdown duration for each round.", preferredStyle: .actionSheet)
        for (index, label) in UserSettingsStore.shared.timerLabels.enumerated() {
            alert.addAction(UIAlertAction(title: label, style: .default) { [weak self] _ in
                guard let self else { return }
                UserSettingsStore.shared.timerPresetIndex = index
                self.tableView.reloadData()
                self.showBanner(message: "Timer set to \(label).", style: .success)
            })
        }
        alert.addAction(UIAlertAction(title: "Cancel", style: .cancel))
        configurePopover(alert.popoverPresentationController, anchor: anchor)
        present(alert, animated: true)
    }

    private func presentShareSheet(anchor: UIView?) {
        let items: [Any] = ["Deck Tally Pro", UserSettingsStore.appStoreURL]
        let activity = UIActivityViewController(activityItems: items, applicationActivities: nil)
        configurePopover(activity.popoverPresentationController, anchor: anchor)
        present(activity, animated: true)
    }

    private func configurePopover(_ popover: UIPopoverPresentationController?, anchor: UIView?) {
        guard let popover else { return }
        if let anchor {
            popover.sourceView = anchor
            popover.sourceRect = anchor.bounds
            popover.permittedArrowDirections = [.up, .down]
        } else {
            popover.sourceView = view
            popover.sourceRect = CGRect(x: view.bounds.midX, y: view.bounds.midY, width: 0, height: 0)
            popover.permittedArrowDirections = []
        }
    }
}
