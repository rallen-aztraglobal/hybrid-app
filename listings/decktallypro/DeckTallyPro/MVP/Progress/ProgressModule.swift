import UIKit

final class ProgressViewController: UIViewController, UITableViewDataSource {
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private let emptyStateLabel = UILabel()
    private var rows: [(String, String)] = []

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "Progress"
        view.backgroundColor = AppTheme.background
        tableView.backgroundColor = .clear
        tableView.dataSource = self
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "cell")

        emptyStateLabel.text = "No game history yet. Start a round in Arena to begin tracking progress."
        emptyStateLabel.textColor = AppTheme.textSecondary
        emptyStateLabel.font = .systemFont(ofSize: 15, weight: .medium)
        emptyStateLabel.textAlignment = .center
        emptyStateLabel.numberOfLines = 0
        tableView.backgroundView = emptyStateLabel

        view.addSubview(tableView)
        tableView.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            tableView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            tableView.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor)
        ])
        reloadData()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        reloadData()
    }

    private func reloadData() {
        let progress = ProgressStore.shared.progress
        let cardTotal = GameDataService.shared.cardCountLevels.count
        let hourglassTotal = GameDataService.shared.hourglassLevels.count
        let lastPlayed = progress.lastPlayedAt.map {
            DateFormatter.localizedString(from: $0, dateStyle: .medium, timeStyle: .short)
        } ?? "Not yet played"
        let hasHistory = progress.totalSessions > 0
            || progress.cardCountCompletedLevels > 0
            || progress.hourglassCompletedLevels > 0
            || progress.cardCountBestStreak > 0
            || progress.hourglassBestStreak > 0

        guard hasHistory else {
            rows = []
            emptyStateLabel.isHidden = false
            tableView.reloadData()
            return
        }

        rows = [
            ("Total Sessions", "\(progress.totalSessions)"),
            ("Card Spotter Progress", "\(min(progress.cardCountCompletedLevels, cardTotal)) / \(cardTotal)"),
            ("Sum Balance Progress", "\(min(progress.hourglassCompletedLevels, hourglassTotal)) / \(hourglassTotal)"),
            ("Card Spotter Current Streak", "\(progress.cardCountCurrentStreak)"),
            ("Card Spotter Best Streak", "\(progress.cardCountBestStreak)"),
            ("Sum Balance Current Streak", "\(progress.hourglassCurrentStreak)"),
            ("Sum Balance Best Streak", "\(progress.hourglassBestStreak)"),
            ("Last Session", lastPlayed)
        ]
        emptyStateLabel.isHidden = true
        tableView.reloadData()
    }

    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { rows.count }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "cell", for: indexPath)
        var config = UIListContentConfiguration.valueCell()
        config.text = rows[indexPath.row].0
        config.secondaryText = rows[indexPath.row].1
        config.textProperties.color = AppTheme.textPrimary
        config.secondaryTextProperties.color = AppTheme.accentSecondary
        cell.contentConfiguration = config
        cell.backgroundColor = AppTheme.surface
        cell.selectionStyle = .none
        return cell
    }
}
