import UIKit

// MARK: - 单个方块

/// 棋盘上的一块。数字居中，归位后换成强调色。
final class TileView: UIView {
    let value: Int
    private let label = UILabel()
    private var isInPlace = false

    init(value: Int) {
        self.value = value
        super.init(frame: .zero)
        layer.cornerCurve = .continuous
        label.text = "\(value)"
        label.textAlignment = .center
        // 等宽数字：滑动时数字宽度不变，视觉上不会左右抖。
        label.font = .monospacedDigitSystemFont(ofSize: 24, weight: .semibold)
        addSubview(label)
        applyColors()
    }

    required init?(coder: NSCoder) { nil }

    override func layoutSubviews() {
        super.layoutSubviews()
        label.frame = bounds
        // 圆角与字号都跟着块大小走：3×3 的块比 5×5 的大得多，
        // 写死一个值会让某个尺寸下的比例失衡。
        layer.cornerRadius = bounds.width * 0.16
        label.font = .monospacedDigitSystemFont(ofSize: bounds.width * 0.42, weight: .semibold)
    }

    func setInPlace(_ inPlace: Bool) {
        guard inPlace != isInPlace else { return }
        isInPlace = inPlace
        applyColors()
    }

    private func applyColors() {
        backgroundColor = isInPlace ? AppTheme.tilePlacedFill : AppTheme.tileFill
        label.textColor = isInPlace ? AppTheme.tilePlacedText : AppTheme.tileText
    }
}

// MARK: - 棋盘

protocol BoardViewDelegate: AnyObject {
    /// 玩家走了一步（合法滑动）。
    func boardViewDidMove(_ board: BoardView)
    /// 棋盘复原了。
    func boardViewDidSolve(_ board: BoardView)
}

/// 棋盘。用 frame 布局而不是 Auto Layout —— 滑动就是「把块挪到另一个格子」，
/// 直接对 frame 做动画最自然；换成约束的话每走一步都要拆装约束，代码更长也更容易错。
final class BoardView: UIView {
    weak var delegate: BoardViewDelegate?

    private(set) var puzzle: SlidePuzzle
    private var tileViews: [TileView] = []
    private var slotViews: [UIView] = []

    /// 棋盘四周留白。
    private let inset: CGFloat = 8

    init(size: Int) {
        puzzle = SlidePuzzle(size: size)
        super.init(frame: .zero)
        backgroundColor = AppTheme.surface
        layer.cornerRadius = AppTheme.cornerRadius
        layer.cornerCurve = .continuous
        isUserInteractionEnabled = true

        let tap = UITapGestureRecognizer(target: self, action: #selector(handleTap(_:)))
        addGestureRecognizer(tap)

        rebuild()
    }

    required init?(coder: NSCoder) { nil }

    // MARK: 布局

    /// 一格的边长（含缝）。
    private var cellSide: CGFloat {
        let available = min(bounds.width, bounds.height) - inset * 2
        return max(0, available) / CGFloat(puzzle.size)
    }

    /// 第 index 格的矩形（已扣掉缝）。
    private func frame(for index: Int) -> CGRect {
        let cell = cellSide
        let gap = cell * 0.08
        let row = CGFloat(puzzle.rowOf(index))
        let col = CGFloat(puzzle.colOf(index))
        return CGRect(
            x: inset + col * cell + gap / 2,
            y: inset + row * cell + gap / 2,
            width: cell - gap,
            height: cell - gap
        )
    }

    override func layoutSubviews() {
        super.layoutSubviews()
        for (index, slot) in slotViews.enumerated() {
            slot.frame = frame(for: index)
            slot.layer.cornerRadius = slot.bounds.width * 0.16
        }
        layoutTiles()
    }

    /// 把每一块摆到它当前所在的格子上。动画与否由调用方用 UIView.animate 包裹决定。
    private func layoutTiles() {
        // 先建一张「块的数字 → 当前下标」的表，避免对每一块都全盘查找一次。
        var indexOfValue: [Int: Int] = [:]
        for (index, value) in puzzle.tiles.enumerated() where value != 0 {
            indexOfValue[value] = index
        }
        for tile in tileViews {
            guard let index = indexOfValue[tile.value] else { continue }
            tile.frame = frame(for: index)
            tile.setInPlace(puzzle.isTileInPlace(index))
        }
    }

    /// 按当前尺寸重建全部子视图。换棋盘尺寸时调用。
    private func rebuild() {
        slotViews.forEach { $0.removeFromSuperview() }
        tileViews.forEach { $0.removeFromSuperview() }
        slotViews = []
        tileViews = []

        for _ in 0..<puzzle.count {
            let slot = UIView()
            slot.backgroundColor = AppTheme.slotFill
            slot.layer.cornerCurve = .continuous
            slot.isUserInteractionEnabled = false
            addSubview(slot)
            slotViews.append(slot)
        }
        // 块按数字建，位置由 layoutTiles 决定 —— 于是「滑动」只是改 frame，
        // 不需要销毁重建任何视图，动画天然连续。
        for value in 1..<puzzle.count {
            let tile = TileView(value: value)
            tile.isUserInteractionEnabled = false
            addSubview(tile)
            tileViews.append(tile)
        }
        setNeedsLayout()
    }

    // MARK: 交互

    /// 开新局。[size] 与当前不同则重建棋盘。
    func newGame(size: Int) {
        let sizeChanged = size != puzzle.size
        if sizeChanged {
            puzzle = SlidePuzzle(size: size)
            rebuild()
        }
        // **先 layoutIfNeeded 定住旧局面，再打乱。** 顺序反了的话，转场取的「前」快照
        // 就已经是新局面了，交叉淡入等于空转、什么也看不到。
        layoutIfNeeded()

        // 打乱步数随格数增长：3×3 走 100 多步就够乱，5×5 得多走些才看不出原样。
        let steps = puzzle.count * 12
        puzzle.shuffle(steps: steps)

        // 还没布局过（viewDidLoad 阶段就调到这里）时 bounds 是零，没有可淡入的起始帧，
        // 交给随后的 layoutSubviews 直接画出来即可。
        guard bounds.width > 0 else {
            setNeedsLayout()
            return
        }
        UIView.transition(with: self, duration: 0.25, options: .transitionCrossDissolve) {
            self.layoutTiles()
        }
    }

    @objc private func handleTap(_ recognizer: UITapGestureRecognizer) {
        let point = recognizer.location(in: self)
        guard let index = indexAt(point), puzzle.canMove(index) else { return }

        puzzle.move(index)
        UIView.animate(
            withDuration: 0.16,
            delay: 0,
            options: [.curveEaseOut, .beginFromCurrentState]
        ) {
            self.layoutTiles()
        }

        delegate?.boardViewDidMove(self)
        if puzzle.isSolved {
            delegate?.boardViewDidSolve(self)
        }
    }

    /// 把触点换算成格子下标；点在留白或界外返回 nil。
    private func indexAt(_ point: CGPoint) -> Int? {
        let cell = cellSide
        guard cell > 0 else { return nil }

        // **先减去留白再判负**，不要直接对 `Int((point.x - inset) / cell)` 判 `>= 0`：
        // 点落在左侧那 8pt 留白里时，商是个绝对值小于 1 的负数，`Int()` 向零取整正好得 0，
        // 于是被当成第 0 列放行。右侧留白却会算出 >= size 被正确挡掉 —— 左右不对称。
        let x = point.x - inset
        let y = point.y - inset
        guard x >= 0, y >= 0 else { return nil }

        let col = Int(x / cell)
        let row = Int(y / cell)
        guard row < puzzle.size, col < puzzle.size else { return nil }
        return row * puzzle.size + col
    }
}

// MARK: - 主界面

/// A 面本体：滑块拼图。
final class PlayViewController: UIViewController {
    private let sizeControl = UISegmentedControl(
        items: SettingsStore.boardSizes.map { "\($0)×\($0)" }
    )
    private let movesTile = StatTileView(title: "Moves")
    private let timeTile = StatTileView(title: "Time")
    private let bestTile = StatTileView(title: "Best")
    private lazy var boardView = BoardView(size: SettingsStore.shared.boardSize)
    private let winOverlay = WinOverlayView()

    private var moves = 0
    private let clock = GameClock()
    private var displayTimer: Timer?

    /// 本页当前是否在屏上。
    ///
    /// **少了这一维会出两个方向相反的计时错误**：本页是 tab 的根控制器、永不销毁，
    /// 所以人在 Records 页时它照样收得到「App 回到前台」的通知 —— 不判可见性就会
    /// 给一个看不见的页面继续计时；而反过来，切走时暂停了却在切回来时忘了恢复，
    /// 这一局的用时会就此冻结，冻结值还会被当成「最短用时」提交进纪录。
    private var isOnScreen = false

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.background
        // 本页关掉大标题：它恒占安全区顶部约 96pt，而这一页要在竖直方向上塞下
        // 分段控件 + 统计条 + 正方形棋盘 + 按钮，小屏上那 52pt 的差别就是「按钮点得到」
        // 和「按钮在 tab bar 底下」的分界。Records / Settings 两页仍用大标题。
        navigationItem.largeTitleDisplayMode = .never
        buildLayout()

        boardView.delegate = self
        sizeControl.selectedSegmentIndex =
            SettingsStore.boardSizes.firstIndex(of: SettingsStore.shared.boardSize) ?? 1
        sizeControl.addTarget(self, action: #selector(sizeChanged), for: .valueChanged)

        NotificationCenter.default.addObserver(
            self, selector: #selector(appDidEnterBackground),
            name: UIApplication.didEnterBackgroundNotification, object: nil
        )
        NotificationCenter.default.addObserver(
            self, selector: #selector(appWillEnterForeground),
            name: UIApplication.willEnterForegroundNotification, object: nil
        )

        startNewGame()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        isOnScreen = true

        // 设置页可能改过默认棋盘尺寸。不同步的话会出现「分段控件显示 4×4、棋盘是 4×4、
        // Best 却显示 5×5 的纪录」，玩完还会用旧尺寸记账。
        let stored = SettingsStore.shared.boardSize
        if stored != boardView.puzzle.size {
            sizeControl.selectedSegmentIndex =
                SettingsStore.boardSizes.firstIndex(of: stored) ?? 1
            startNewGame()   // 内部已含 refreshBest / startDisplayTimer
            return
        }

        refreshBest()
        resumeClockIfInProgress()
        startDisplayTimer()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        isOnScreen = false
        // 离开页面就把计时停掉：切到 Records 页看半天成绩，回来时间不该白涨。
        clock.pause()
        stopDisplayTimer()
        refreshStats()
    }

    /// 牌局是否正在进行中（已经走过步、且还没解开）。
    private var isGameInProgress: Bool {
        moves > 0 && !boardView.puzzle.isSolved
    }

    /// 只有在进行中的牌局才恢复计时 —— 没开始的局和已经赢了的局都不该走表。
    private func resumeClockIfInProgress() {
        guard isGameInProgress else { return }
        clock.resume()
    }

    deinit {
        displayTimer?.invalidate()
    }

    // MARK: 布局

    private func buildLayout() {
        // 分段控件在深色底上要自己上色，否则用的是系统浅色外观，和整页格格不入。
        sizeControl.backgroundColor = AppTheme.surface
        sizeControl.selectedSegmentTintColor = AppTheme.accent
        sizeControl.setTitleTextAttributes(
            [.foregroundColor: AppTheme.textSecondary,
             .font: UIFont.systemFont(ofSize: 14, weight: .medium)],
            for: .normal
        )
        sizeControl.setTitleTextAttributes(
            [.foregroundColor: AppTheme.tilePlacedText,
             .font: UIFont.systemFont(ofSize: 14, weight: .semibold)],
            for: .selected
        )

        let statsCard = CardView()
        let stats = UIStackView(arrangedSubviews: [movesTile, timeTile, bestTile])
        stats.axis = .horizontal
        stats.distribution = .fillEqually
        statsCard.addForAutoLayout(stats)
        NSLayoutConstraint.activate([
            stats.topAnchor.constraint(equalTo: statsCard.topAnchor, constant: 14),
            stats.leadingAnchor.constraint(equalTo: statsCard.leadingAnchor, constant: 12),
            stats.trailingAnchor.constraint(equalTo: statsCard.trailingAnchor, constant: -12),
            stats.bottomAnchor.constraint(equalTo: statsCard.bottomAnchor, constant: -14)
        ])

        let newGameButton = UIButton(type: .system)
        var config = UIButton.Configuration.filled()
        config.baseBackgroundColor = AppTheme.accent
        config.baseForegroundColor = AppTheme.tilePlacedText
        config.cornerStyle = .capsule
        config.contentInsets = NSDirectionalEdgeInsets(top: 13, leading: 30, bottom: 13, trailing: 30)
        var titleAttributes = AttributeContainer()
        // 写全 `UIFont.` 而不是靠前导点做隐式成员查找：这里要先经 AttributeContainer 的
        // dynamicMemberLookup 推出 Value == UIFont，再在其上查找 —— 双重推导，
        // 而本工程里这是唯一一处没有「同款写法已验证能编译」背书的 API，不值得省这几个字符。
        titleAttributes.font = UIFont.systemFont(ofSize: 16, weight: .semibold)
        config.attributedTitle = AttributedString("New game", attributes: titleAttributes)
        newGameButton.configuration = config
        newGameButton.addTarget(self, action: #selector(newGameTapped), for: .touchUpInside)

        view.addForAutoLayout(sizeControl)
        view.addForAutoLayout(statsCard)
        view.addForAutoLayout(boardView)
        view.addForAutoLayout(newGameButton)

        let guide = view.safeAreaLayoutGuide
        let pad = AppTheme.horizontalPadding

        // 棋盘宽度是**可以让步**的那一项：空间够就顶满宽（.defaultHigh），
        // 不够就被下面那条「按钮不许越过安全区底边」压小。
        //
        // 一开始这里写的是「全必需约束、不给下边界」，看起来最确定，实际是错的：
        // `MainTabBarController` 给每个 tab 开了 prefersLargeTitles，而本页没有
        // scroll view，所以大标题栏恒为展开态、直接吃掉安全区顶部约 96pt。
        // 算上它之后，iPhone SE 这一档根本放不下，而没有退让余地的约束链只会把
        // 「New game」按钮顶到 tab bar 底下去 —— 按钮还在，只是点不到。
        // 所以本页把大标题关掉（省 52pt），并让棋盘可被压缩。
        let boardWidth = boardView.widthAnchor.constraint(
            equalTo: guide.widthAnchor, constant: -pad * 2
        )
        boardWidth.priority = .defaultHigh

        NSLayoutConstraint.activate([
            sizeControl.topAnchor.constraint(equalTo: guide.topAnchor, constant: 12),
            sizeControl.leadingAnchor.constraint(equalTo: guide.leadingAnchor, constant: pad),
            sizeControl.trailingAnchor.constraint(equalTo: guide.trailingAnchor, constant: -pad),

            statsCard.topAnchor.constraint(equalTo: sizeControl.bottomAnchor, constant: 14),
            statsCard.leadingAnchor.constraint(equalTo: guide.leadingAnchor, constant: pad),
            statsCard.trailingAnchor.constraint(equalTo: guide.trailingAnchor, constant: -pad),

            boardView.topAnchor.constraint(equalTo: statsCard.bottomAnchor, constant: 20),
            boardView.centerXAnchor.constraint(equalTo: guide.centerXAnchor),
            boardView.widthAnchor.constraint(lessThanOrEqualTo: guide.widthAnchor, constant: -pad * 2),
            boardWidth,
            // 棋盘是正方形。
            boardView.heightAnchor.constraint(equalTo: boardView.widthAnchor),

            newGameButton.topAnchor.constraint(equalTo: boardView.bottomAnchor, constant: 20),
            newGameButton.centerXAnchor.constraint(equalTo: guide.centerXAnchor),
            // 这条是关键：小屏上它会把棋盘压小，而不是把按钮挤出屏幕。
            newGameButton.bottomAnchor.constraint(
                lessThanOrEqualTo: guide.bottomAnchor, constant: -12
            )
        ])

        winOverlay.isHidden = true
        winOverlay.onNewGame = { [weak self] in self?.startNewGame() }
        view.addForAutoLayout(winOverlay)
        winOverlay.pinToEdges(of: view)
    }

    // MARK: 一局的生命周期

    private func startNewGame() {
        moves = 0
        clock.reset()
        winOverlay.isHidden = true
        boardView.newGame(size: SettingsStore.shared.boardSize)
        refreshStats()
        refreshBest()
        startDisplayTimer()
    }

    @objc private func newGameTapped() {
        startNewGame()
    }

    @objc private func sizeChanged() {
        let index = sizeControl.selectedSegmentIndex
        guard index >= 0 && index < SettingsStore.boardSizes.count else { return }
        SettingsStore.shared.boardSize = SettingsStore.boardSizes[index]
        startNewGame()
    }

    private func refreshStats() {
        movesTile.setValue("\(moves)")
        timeTile.setValue(TimeDisplay.clock(clock.elapsedSeconds))
    }

    private func refreshBest() {
        let record = RecordStore.shared.record(forSize: SettingsStore.shared.boardSize)
        if let bestMoves = record.bestMoves {
            bestTile.setValue("\(bestMoves)")
        } else {
            bestTile.setValue("—")
        }
    }

    // MARK: 计时

    /// 每秒刷一次用时显示。真正的计时在 GameClock 里按时间戳算，
    /// 这个 timer 只负责把数字画出来 —— 于是丢几次回调也不会让时间走偏。
    private func startDisplayTimer() {
        stopDisplayTimer()
        displayTimer = Timer.scheduledTimer(withTimeInterval: 0.5, repeats: true) { [weak self] _ in
            self?.refreshStats()
        }
    }

    private func stopDisplayTimer() {
        displayTimer?.invalidate()
        displayTimer = nil
    }

    @objc private func appDidEnterBackground() {
        clock.pause()
        stopDisplayTimer()
    }

    @objc private func appWillEnterForeground() {
        // 不在屏上就什么都不做。本页是 tab 根控制器、永不销毁，人在 Records 页时
        // 它照样收得到这个通知 —— 少了这道判断就会给一个看不见的页面继续计时。
        guard isOnScreen else { return }
        resumeClockIfInProgress()
        startDisplayTimer()
    }
}

extension PlayViewController: BoardViewDelegate {
    func boardViewDidMove(_ board: BoardView) {
        // 第一步落下才开始计时 —— 打开 App 放着不动不该被算进成绩。
        if moves == 0 { clock.start() }
        moves += 1
        refreshStats()

        if SettingsStore.shared.hapticsEnabled {
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
        }
    }

    func boardViewDidSolve(_ board: BoardView) {
        clock.pause()
        stopDisplayTimer()
        refreshStats()

        let size = board.puzzle.size
        let seconds = clock.elapsedSeconds
        let outcome = RecordStore.shared.submit(size: size, moves: moves, seconds: seconds)
        refreshBest()

        if SettingsStore.shared.hapticsEnabled {
            UINotificationFeedbackGenerator().notificationOccurred(.success)
        }

        winOverlay.configure(moves: moves, seconds: seconds, outcome: outcome)
        winOverlay.isHidden = false
        winOverlay.alpha = 0
        UIView.animate(withDuration: 0.25) { self.winOverlay.alpha = 1 }
    }
}

// MARK: - 计时器

/// 一局的计时。
///
/// 按**时间戳**算而不是「每秒加一」：后者在 App 退到后台、timer 被系统停掉时会少算，
/// 回到前台又接着加，最终显示的时间既不是真实耗时也不是游玩时长，两头不靠。
/// 这里累计的是「确实在玩的那几段」的时长之和。
final class GameClock {
    private var accumulated: TimeInterval = 0
    private var runningSince: Date?

    var elapsedSeconds: Int {
        var total = accumulated
        if let runningSince {
            total += Date().timeIntervalSince(runningSince)
        }
        return Int(total)
    }

    func start() {
        guard runningSince == nil else { return }
        runningSince = Date()
    }

    func resume() {
        start()
    }

    func pause() {
        guard let runningSince else { return }
        accumulated += Date().timeIntervalSince(runningSince)
        self.runningSince = nil
    }

    func reset() {
        accumulated = 0
        runningSince = nil
    }
}

// MARK: - 通关浮层

/// 复原之后盖上来的结算层。
final class WinOverlayView: UIView {
    var onNewGame: (() -> Void)?

    private let titleLabel = UILabel()
    private let detailLabel = UILabel()
    private let badgeLabel = UILabel()

    init() {
        super.init(frame: .zero)
        backgroundColor = AppTheme.background.withAlphaComponent(0.92)

        titleLabel.text = "Solved"
        titleLabel.font = .systemFont(ofSize: 30, weight: .bold)
        titleLabel.textColor = AppTheme.textPrimary
        titleLabel.textAlignment = .center

        detailLabel.font = .monospacedDigitSystemFont(ofSize: 16, weight: .regular)
        detailLabel.textColor = AppTheme.textSecondary
        detailLabel.textAlignment = .center

        badgeLabel.font = .systemFont(ofSize: 14, weight: .semibold)
        badgeLabel.textColor = AppTheme.accent
        badgeLabel.textAlignment = .center
        badgeLabel.numberOfLines = 0

        let button = UIButton(type: .system)
        var config = UIButton.Configuration.filled()
        config.baseBackgroundColor = AppTheme.accent
        config.baseForegroundColor = AppTheme.tilePlacedText
        config.cornerStyle = .capsule
        config.contentInsets = NSDirectionalEdgeInsets(top: 13, leading: 32, bottom: 13, trailing: 32)
        var titleAttributes = AttributeContainer()
        // 写全 `UIFont.` 而不是靠前导点做隐式成员查找：这里要先经 AttributeContainer 的
        // dynamicMemberLookup 推出 Value == UIFont，再在其上查找 —— 双重推导，
        // 而本工程里这是唯一一处没有「同款写法已验证能编译」背书的 API，不值得省这几个字符。
        titleAttributes.font = UIFont.systemFont(ofSize: 16, weight: .semibold)
        config.attributedTitle = AttributedString("Play again", attributes: titleAttributes)
        button.configuration = config
        button.addTarget(self, action: #selector(newGameTapped), for: .touchUpInside)

        let stack = UIStackView(arrangedSubviews: [titleLabel, detailLabel, badgeLabel, button])
        stack.axis = .vertical
        stack.alignment = .center
        stack.spacing = 8
        stack.setCustomSpacing(24, after: badgeLabel)

        addForAutoLayout(stack)
        NSLayoutConstraint.activate([
            stack.centerXAnchor.constraint(equalTo: centerXAnchor),
            stack.centerYAnchor.constraint(equalTo: centerYAnchor),
            stack.leadingAnchor.constraint(greaterThanOrEqualTo: leadingAnchor, constant: 32),
            stack.trailingAnchor.constraint(lessThanOrEqualTo: trailingAnchor, constant: -32)
        ])
    }

    required init?(coder: NSCoder) { nil }

    func configure(moves: Int, seconds: Int, outcome: RecordOutcome) {
        detailLabel.text = "\(moves) moves \u{00B7} \(TimeDisplay.clock(seconds))"

        if outcome.isNewMovesBest && outcome.isNewTimeBest {
            badgeLabel.text = "New best moves and time"
        } else if outcome.isNewMovesBest {
            badgeLabel.text = "New best moves"
        } else if outcome.isNewTimeBest {
            badgeLabel.text = "New best time"
        } else {
            badgeLabel.text = nil
        }
    }

    @objc private func newGameTapped() {
        onNewGame?()
    }
}
