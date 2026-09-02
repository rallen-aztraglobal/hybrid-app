import UIKit

/// 成绩页：每个棋盘尺寸一张卡，显示最少步数、最短用时与通关次数。
final class RecordsViewController: UIViewController {
    private let scrollView = UIScrollView()
    private let contentStack = UIStackView()
    private var cards: [Int: RecordCardView] = [:]

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.background
        buildLayout()

        NotificationCenter.default.addObserver(
            self, selector: #selector(refresh),
            name: .recordsDidChange, object: nil
        )
        refresh()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        refresh()
    }

    private func buildLayout() {
        view.addForAutoLayout(scrollView)
        scrollView.pinToEdges(of: view)

        contentStack.axis = .vertical
        contentStack.spacing = 14
        scrollView.addForAutoLayout(contentStack)
        NSLayoutConstraint.activate([
            contentStack.topAnchor.constraint(
                equalTo: scrollView.contentLayoutGuide.topAnchor, constant: 12
            ),
            contentStack.bottomAnchor.constraint(
                equalTo: scrollView.contentLayoutGuide.bottomAnchor, constant: -28
            ),
            contentStack.leadingAnchor.constraint(
                equalTo: scrollView.frameLayoutGuide.leadingAnchor,
                constant: AppTheme.horizontalPadding
            ),
            contentStack.trailingAnchor.constraint(
                equalTo: scrollView.frameLayoutGuide.trailingAnchor,
                constant: -AppTheme.horizontalPadding
            )
        ])

        for size in SettingsStore.boardSizes {
            let card = RecordCardView(size: size)
            cards[size] = card
            contentStack.addArrangedSubview(card)
        }

        let note = UILabel()
        note.text = "Best moves and best time are tracked separately, "
            + "so a careful run and a fast run both count."
        note.font = .systemFont(ofSize: 13)
        note.textColor = AppTheme.textSecondary
        note.numberOfLines = 0
        contentStack.addArrangedSubview(note)
    }

    @objc private func refresh() {
        for size in SettingsStore.boardSizes {
            cards[size]?.configure(with: RecordStore.shared.record(forSize: size))
        }
    }
}

/// 一个尺寸的成绩卡。
private final class RecordCardView: CardView {
    private let movesTile = StatTileView(title: "Best moves")
    private let timeTile = StatTileView(title: "Best time")
    private let playsTile = StatTileView(title: "Solved")

    init(size: Int) {
        super.init(cornerRadius: AppTheme.cornerRadius)

        let title = UILabel()
        title.text = "\(size)×\(size)"
        title.font = .systemFont(ofSize: 20, weight: .bold)
        title.textColor = AppTheme.textPrimary

        let tiles = UIStackView(arrangedSubviews: [movesTile, timeTile, playsTile])
        tiles.axis = .horizontal
        tiles.distribution = .fillEqually
        tiles.spacing = 10

        let stack = UIStackView(arrangedSubviews: [title, tiles])
        stack.axis = .vertical
        stack.spacing = 14

        addForAutoLayout(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: topAnchor, constant: 16),
            stack.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -16)
        ])
    }

    required init?(coder: NSCoder) { nil }

    func configure(with record: BoardRecord) {
        // 还没通关过就画一个破折号，而不是 0 —— 0 步 0 秒会被误读成「有过一次神纪录」。
        movesTile.setValue(record.bestMoves.map { "\($0)" } ?? "—")
        timeTile.setValue(record.bestSeconds.map { TimeDisplay.clock($0) } ?? "—")
        playsTile.setValue("\(record.completions)")

        let hasRecord = record.completions > 0
        let color = hasRecord ? AppTheme.textPrimary : AppTheme.textSecondary
        movesTile.setValueColor(color)
        timeTile.setValueColor(color)
        playsTile.setValueColor(color)
    }
}
