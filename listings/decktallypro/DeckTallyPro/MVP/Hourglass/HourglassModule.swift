import UIKit

protocol HourglassViewProtocol: AnyObject {
    func render(level: HourglassLevel, levelIndex: Int, totalLevels: Int)
    func updateTimer(text: String)
    func showFeedback(correct: Bool)
    func showTimerExpired()
    func reloadAnswers()
}

final class HourglassPresenter {
    weak var view: HourglassViewProtocol?
    private let levelIndex: Int
    private var timer: Timer?
    private var remaining: TimeInterval = UserSettingsStore.shared.timerDuration
    private var selectedIndex: Int?
    private var isExpired = false
    private var hasRecordedResult = false

    init(levelIndex: Int) {
        self.levelIndex = levelIndex
    }

    func viewDidLoad() {
        let levels = GameDataService.shared.hourglassLevels
        guard levelIndex < levels.count else { return }
        view?.render(level: levels[levelIndex], levelIndex: levelIndex, totalLevels: levels.count)
        remaining = UserSettingsStore.shared.timerDuration
        view?.updateTimer(text: TimeFormatter.display(for: remaining))
    }

    func viewWillAppear() {
        guard selectedIndex == nil, !hasRecordedResult, !isExpired else {
            stopTimer()
            return
        }
        startTimer()
    }

    deinit {
        stopTimer()
    }

    func viewWillDisappear() {
        stopTimer()
    }

    func viewDidDisappear() {
        stopTimer()
    }

    func selectAnswer(at index: Int) {
        guard !isExpired else {
            view?.showTimerExpired()
            return
        }
        guard selectedIndex == nil, !hasRecordedResult else { return }

        let levels = GameDataService.shared.hourglassLevels
        guard levelIndex < levels.count else { return }
        let choices = levels[levelIndex].answerChoices
        guard choices.indices.contains(index) else { return }

        selectedIndex = index
        let answer = choices[index]

        recordResult(correct: answer.isCorrectAnswer)
        stopTimer()
        view?.showFeedback(correct: answer.isCorrectAnswer)
        view?.reloadAnswers()
    }

    func answerCount() -> Int {
        GameDataService.shared.hourglassLevels[safe: levelIndex]?.answerChoices.count ?? 0
    }

    func answerTitle(at index: Int) -> String {
        guard let choices = GameDataService.shared.hourglassLevels[safe: levelIndex]?.answerChoices,
              choices.indices.contains(index) else { return "" }
        return choices[index].answerValue
    }

    func choiceState(at index: Int) -> AnswerChoiceState {
        guard let selectedIndex else { return .neutral }
        if selectedIndex != index { return .neutral }
        guard let choices = GameDataService.shared.hourglassLevels[safe: levelIndex]?.answerChoices,
              choices.indices.contains(index) else { return .neutral }
        return choices[index].isCorrectAnswer ? .correct : .wrong
    }

    func nextLevel(from viewController: UIViewController) {
        let levels = GameDataService.shared.hourglassLevels
        if levelIndex >= levels.count - 1 {
            let alert = UIAlertController(title: "Mode Complete", message: "You finished every Sum Balance level.", preferredStyle: .alert)
            alert.addAction(UIAlertAction(title: "Back to Arena", style: .default) { _ in
                viewController.navigationController?.popViewController(animated: true)
            })
            viewController.present(alert, animated: true)
            return
        }
        let next = HourglassViewController(levelIndex: levelIndex + 1)
        viewController.navigationController?.pushViewController(next, animated: true)
    }

    private func startTimer() {
        guard timer == nil, selectedIndex == nil, !hasRecordedResult, !isExpired else { return }
        if remaining <= 0 {
            remaining = UserSettingsStore.shared.timerDuration
        }
        view?.updateTimer(text: TimeFormatter.display(for: remaining))
        timer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            guard let self else { return }
            self.remaining -= 1
            if self.remaining <= 0 {
                self.isExpired = true
                self.recordResult(correct: false)
                self.stopTimer()
                self.view?.updateTimer(text: "00:00")
                self.view?.showTimerExpired()
            } else {
                self.view?.updateTimer(text: TimeFormatter.display(for: self.remaining))
            }
        }
    }

    private func recordResult(correct: Bool) {
        guard !hasRecordedResult else { return }
        hasRecordedResult = true
        ProgressStore.shared.recordSession()
        ProgressStore.shared.recordHourglassResult(levelIndex: levelIndex, correct: correct)
    }

    private func stopTimer() {
        timer?.invalidate()
        timer = nil
    }
}

enum AnswerChoiceState {
    case neutral, correct, wrong
}

final class HourglassViewController: UIViewController, HourglassViewProtocol, UICollectionViewDataSource, UICollectionViewDelegateFlowLayout {
    private let presenter: HourglassPresenter
    private let scrollView = UIScrollView()
    private let stack = UIStackView()
    private let timerLabel = UILabel()
    private let rulesHost = UIStackView()
    private let cardsHost = UIView()
    private let hourglassHost = UIView()
    private var collectionView: UICollectionView!
    private var collectionHeightConstraint: NSLayoutConstraint?
    private var currentLevel: HourglassLevel?
    private var lastLaidOutContentWidth: CGFloat = 0
    private var lastLaidOutHourglassSize: CGSize = .zero

    init(levelIndex: Int) {
        presenter = HourglassPresenter(levelIndex: levelIndex)
        super.init(nibName: nil, bundle: nil)
        hidesBottomBarWhenPushed = true
    }

    required init?(coder: NSCoder) { nil }

    override func viewDidLoad() {
        super.viewDidLoad()
        presenter.view = self
        configureUI()
        presenter.viewDidLoad()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        presenter.viewWillAppear()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        presenter.viewWillDisappear()
    }

    override func viewDidDisappear(_ animated: Bool) {
        super.viewDidDisappear(animated)
        presenter.viewDidDisappear()
    }

    func render(level: HourglassLevel, levelIndex: Int, totalLevels: Int) {
        title = "Sum Balance \(levelIndex + 1)/\(totalLevels)"
        currentLevel = level
        lastLaidOutContentWidth = 0
        lastLaidOutHourglassSize = .zero

        renderRules(level)
        renderCurrentLevelLayoutIfNeeded(force: true)
        collectionView.reloadData()
        updateAnswerCollectionHeight()
    }

    func updateTimer(text: String) { timerLabel.text = text }

    func showFeedback(correct: Bool) {
        showBanner(message: correct ? "Correct answer." : "Incorrect. Tap Next Level to continue.", style: correct ? .success : .failure)
    }

    func showTimerExpired() {
        showBanner(message: "Time is up! Tap Next Level to continue.", style: .failure)
    }

    func reloadAnswers() {
        collectionView.reloadData()
        updateAnswerCollectionHeight()
    }

    private func configureUI() {
        view.backgroundColor = AppTheme.background
        navigationItem.rightBarButtonItem = UIBarButtonItem(customView: timerLabel)
        timerLabel.font = .monospacedDigitSystemFont(ofSize: 14, weight: .bold)
        timerLabel.textColor = AppTheme.danger
        timerLabel.backgroundColor = AppTheme.danger.withAlphaComponent(0.12)
        timerLabel.layer.cornerRadius = 12
        timerLabel.layer.masksToBounds = true
        timerLabel.textAlignment = .center
        timerLabel.widthAnchor.constraint(equalToConstant: 72).isActive = true
        timerLabel.heightAnchor.constraint(equalToConstant: 30).isActive = true

        stack.axis = .vertical
        stack.spacing = 16
        rulesHost.axis = .vertical
        rulesHost.spacing = 8
        cardsHost.backgroundColor = AppTheme.surface
        cardsHost.layer.cornerRadius = AppTheme.cornerRadius
        cardsHost.heightAnchor.constraint(equalToConstant: 90).isActive = true
        hourglassHost.backgroundColor = AppTheme.surface
        hourglassHost.layer.cornerRadius = AppTheme.cornerRadius
        hourglassHost.heightAnchor.constraint(equalToConstant: 220).isActive = true

        let layout = UICollectionViewFlowLayout()
        layout.minimumLineSpacing = 10
        layout.minimumInteritemSpacing = 10
        collectionView = UICollectionView(frame: .zero, collectionViewLayout: layout)
        collectionView.backgroundColor = .clear
        collectionView.dataSource = self
        collectionView.delegate = self
        collectionView.register(AnswerChipCell.self, forCellWithReuseIdentifier: AnswerChipCell.reuseID)
        collectionView.isScrollEnabled = false
        collectionHeightConstraint = collectionView.heightAnchor.constraint(equalToConstant: 130)
        collectionHeightConstraint?.isActive = true

        let formula = UILabel()
        formula.text = "Card total value − hourglass total value = ?"
        formula.textColor = AppTheme.textSecondary
        formula.font = .systemFont(ofSize: 14, weight: .medium)
        formula.numberOfLines = 0
        formula.textAlignment = .center

        let next = SecondaryButton(title: "Next Level")
        next.addTarget(self, action: #selector(nextTapped), for: .touchUpInside)

        scrollView.alwaysBounceVertical = true
        view.addSubview(scrollView)
        scrollView.addSubview(stack)
        [PanelTitleView(text: "Card Values"), rulesHost, PanelTitleView(text: "Target Cards"), cardsHost, PanelTitleView(text: "Hourglass Ring"), hourglassHost, formula, collectionView, next].forEach(stack.addArrangedSubview)

        scrollView.translatesAutoresizingMaskIntoConstraints = false
        stack.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor),
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor, constant: 16),
            stack.leadingAnchor.constraint(equalTo: scrollView.leadingAnchor, constant: AppTheme.horizontalPadding),
            stack.trailingAnchor.constraint(equalTo: scrollView.trailingAnchor, constant: -AppTheme.horizontalPadding),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -24),
            stack.widthAnchor.constraint(equalTo: scrollView.widthAnchor, constant: -AppTheme.horizontalPadding * 2)
        ])
    }

    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        renderCurrentLevelLayoutIfNeeded(force: false)
        updateAnswerCollectionHeight()
    }

    private func updateAnswerCollectionHeight() {
        guard collectionView != nil else { return }
        let columns = 3
        let count = presenter.answerCount()
        let rows = max(1, Int(ceil(Double(count) / Double(columns))))
        let itemHeight: CGFloat = 44
        let lineSpacing: CGFloat = 10
        collectionHeightConstraint?.constant = CGFloat(rows) * itemHeight + CGFloat(max(0, rows - 1)) * lineSpacing
        collectionView.collectionViewLayout.invalidateLayout()
    }

    private func renderRules(_ level: HourglassLevel) {
        rulesHost.arrangedSubviews.forEach { $0.removeFromSuperview() }
        let icons = ["dtp_card_spade", "dtp_card_heart", "dtp_card_club"]
        for index in 0..<min(3, level.ruleValues.count) {
            rulesHost.addArrangedSubview(RuleRowView(icon: icons[index], value: level.ruleValues[index]))
        }
    }

    private func renderCurrentLevelLayoutIfNeeded(force: Bool) {
        guard let currentLevel else { return }
        let contentWidth = cardsHost.bounds.width
        let hourglassSize = hourglassHost.bounds.size
        guard contentWidth > 0, hourglassSize.width > 0, hourglassSize.height > 0 else { return }
        guard force || abs(contentWidth - lastLaidOutContentWidth) > 0.5 || hourglassSize != lastLaidOutHourglassSize else { return }

        lastLaidOutContentWidth = contentWidth
        lastLaidOutHourglassSize = hourglassSize
        renderCards(currentLevel.targetCards)
        renderHourglassRing(currentLevel.hourglassRing)
    }

    private func renderCards(_ names: [String]) {
        cardsHost.subviews.forEach { $0.removeFromSuperview() }
        let itemW: CGFloat = 28
        let columns = 6
        let availableWidth = max(cardsHost.bounds.width, itemW * CGFloat(columns))
        let spacing = max(4, (availableWidth - itemW * CGFloat(columns)) / CGFloat(columns - 1))
        for (index, name) in names.enumerated() {
            let row = index / columns
            let col = index % columns
            let imageView = UIImageView(image: UIImage(named: name))
            imageView.frame = CGRect(x: CGFloat(col) * (itemW + spacing), y: CGFloat(row) * 34, width: itemW, height: itemW)
            imageView.contentMode = .scaleAspectFit
            cardsHost.addSubview(imageView)
        }
    }

    private func renderHourglassRing(_ names: [String]) {
        hourglassHost.subviews.forEach { $0.removeFromSuperview() }
        let center = CGPoint(x: hourglassHost.bounds.midX, y: hourglassHost.bounds.midY)
        let radius = max(0, min(hourglassHost.bounds.width, hourglassHost.bounds.height) / 2 - 34)
        for index in names.indices where !names[index].isEmpty {
            let angle = (2 * CGFloat.pi / 18) * CGFloat(index)
            let imageView = UIImageView(image: UIImage(named: "dtp_hourglass"))
            imageView.frame = CGRect(x: 0, y: 0, width: 34, height: 34)
            imageView.center = CGPoint(x: center.x + cos(angle) * radius, y: center.y + sin(angle) * radius)
            hourglassHost.addSubview(imageView)
        }
    }

    func collectionView(_ collectionView: UICollectionView, numberOfItemsInSection section: Int) -> Int {
        presenter.answerCount()
    }

    func collectionView(_ collectionView: UICollectionView, cellForItemAt indexPath: IndexPath) -> UICollectionViewCell {
        let cell = collectionView.dequeueReusableCell(withReuseIdentifier: AnswerChipCell.reuseID, for: indexPath) as! AnswerChipCell
        cell.configure(title: presenter.answerTitle(at: indexPath.item), state: presenter.choiceState(at: indexPath.item))
        return cell
    }

    func collectionView(_ collectionView: UICollectionView, didSelectItemAt indexPath: IndexPath) {
        presenter.selectAnswer(at: indexPath.item)
    }

    func collectionView(_ collectionView: UICollectionView, layout collectionViewLayout: UICollectionViewLayout, sizeForItemAt indexPath: IndexPath) -> CGSize {
        CGSize(width: (collectionView.bounds.width - 20) / 3, height: 44)
    }

    @objc private func nextTapped() {
        presenter.nextLevel(from: self)
    }
}

final class AnswerChipCell: UICollectionViewCell {
    static let reuseID = "AnswerChipCell"
    private let button = UIButton(type: .system)

    override init(frame: CGRect) {
        super.init(frame: frame)
        button.isUserInteractionEnabled = false
        button.layer.cornerRadius = 12
        button.titleLabel?.font = .systemFont(ofSize: 18, weight: .bold)
        contentView.addSubview(button)
        button.translatesAutoresizingMaskIntoConstraints = false
        button.pinToEdges(of: contentView)
    }

    required init?(coder: NSCoder) { nil }

    func configure(title: String, state: AnswerChoiceState) {
        button.setTitle(title, for: .normal)
        switch state {
        case .neutral:
            button.backgroundColor = AppTheme.surfaceElevated
            button.setTitleColor(.white, for: .normal)
        case .correct:
            button.backgroundColor = AppTheme.success
            button.setTitleColor(.white, for: .normal)
        case .wrong:
            button.backgroundColor = AppTheme.danger
            button.setTitleColor(.white, for: .normal)
        }
    }
}

final class RuleRowView: UIView {
    init(icon: String, value: String) {
        super.init(frame: .zero)
        let imageView = UIImageView(image: UIImage(named: icon))
        let label = UILabel()
        label.text = "Value: \(value)"
        label.textColor = AppTheme.accent
        label.font = .systemFont(ofSize: 13, weight: .semibold)
        let stack = UIStackView(arrangedSubviews: [imageView, label])
        stack.spacing = 8
        addSubview(stack)
        stack.translatesAutoresizingMaskIntoConstraints = false
        imageView.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            imageView.widthAnchor.constraint(equalToConstant: 24),
            imageView.heightAnchor.constraint(equalToConstant: 24),
            stack.topAnchor.constraint(equalTo: topAnchor),
            stack.leadingAnchor.constraint(equalTo: leadingAnchor),
            stack.trailingAnchor.constraint(lessThanOrEqualTo: trailingAnchor),
            stack.bottomAnchor.constraint(equalTo: bottomAnchor)
        ])
    }

    required init?(coder: NSCoder) { nil }
}

private extension Array {
    subscript(safe index: Int) -> Element? {
        indices.contains(index) ? self[index] : nil
    }
}
