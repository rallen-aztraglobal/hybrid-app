import UIKit

protocol ArenaViewProtocol: AnyObject {
    func render(modes: [ArenaModeItem])
}

struct ArenaModeItem {
    let mode: GameMode
    let progressText: String
}

final class ArenaPresenter {
    weak var view: ArenaViewProtocol?

    func viewDidLoad() {
        reload()
    }

    func reload() {
        let progress = ProgressStore.shared.progress
        let data = GameDataService.shared
        view?.render(modes: [
            ArenaModeItem(
                mode: .cardCount,
                progressText: "Unlocked \(min(progress.cardCountCompletedLevels + 1, data.cardCountLevels.count)) of \(data.cardCountLevels.count)"
            ),
            ArenaModeItem(
                mode: .hourglass,
                progressText: "Unlocked \(min(progress.hourglassCompletedLevels + 1, data.hourglassLevels.count)) of \(data.hourglassLevels.count)"
            )
        ])
    }

    func start(mode: GameMode, from viewController: UIViewController) {
        // recordSession() is intentionally NOT called here — it is called by
        // each game presenter on the first answer submission, so sessions that
        // the user entered but immediately left back are not counted.
        switch mode {
        case .cardCount:
            let level = min(ProgressStore.shared.progress.cardCountCompletedLevels, GameDataService.shared.cardCountLevels.count - 1)
            let vc = CardCountViewController(levelIndex: max(0, level))
            viewController.navigationController?.pushViewController(vc, animated: true)
        case .hourglass:
            let level = min(ProgressStore.shared.progress.hourglassCompletedLevels, GameDataService.shared.hourglassLevels.count - 1)
            let vc = HourglassViewController(levelIndex: max(0, level))
            viewController.navigationController?.pushViewController(vc, animated: true)
        }
    }
}

final class ArenaViewController: UIViewController, ArenaViewProtocol, UITableViewDataSource, UITableViewDelegate {
    private let presenter = ArenaPresenter()
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private var items: [ArenaModeItem] = []

    override func viewDidLoad() {
        super.viewDidLoad()
        presenter.view = self
        view.backgroundColor = AppTheme.background
        tableView.backgroundColor = .clear
        tableView.dataSource = self
        tableView.delegate = self
        tableView.register(ArenaCell.self, forCellReuseIdentifier: ArenaCell.reuseID)
        tableView.separatorStyle = .none
        view.addSubview(tableView)
        tableView.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            tableView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            tableView.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor)
        ])
        presenter.viewDidLoad()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        presenter.reload()
    }

    func render(modes: [ArenaModeItem]) {
        items = modes
        tableView.reloadData()
    }

    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { items.count }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: ArenaCell.reuseID, for: indexPath) as! ArenaCell
        cell.configure(item: items[indexPath.row])
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        presenter.start(mode: items[indexPath.row].mode, from: self)
    }
}

final class ArenaCell: UITableViewCell {
    static let reuseID = "ArenaCell"
    private let card = GameModeCardView()

    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        backgroundColor = .clear
        selectionStyle = .none
        contentView.addSubview(card)
        card.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            card.topAnchor.constraint(equalTo: contentView.topAnchor, constant: 8),
            card.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 16),
            card.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -16),
            card.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -8)
        ])
    }

    required init?(coder: NSCoder) { nil }

    func configure(item: ArenaModeItem) {
        card.configure(
            title: item.mode.title,
            subtitle: item.mode.subtitle,
            progress: item.progressText,
            symbolName: item.mode.symbolName
        )
    }
}
