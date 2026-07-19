import UIKit

protocol CardCountViewProtocol: AnyObject {
    func render(level: CardCountLevel, levelIndex: Int, totalLevels: Int)
    func updateTimer(text: String)
    func showResult(message: String, success: Bool)
}

final class CardCountPresenter {
    weak var view: CardCountViewProtocol?
    private let levelIndex: Int
    private var timer: Timer?
    private var remaining: TimeInterval = UserSettingsStore.shared.timerDuration
    private var isExpired = false
    private var hasRecordedResult = false

    init(levelIndex: Int) {
        self.levelIndex = levelIndex
    }

    func viewDidLoad() {
        let levels = GameDataService.shared.cardCountLevels
        guard levelIndex < levels.count else { return }
        view?.render(level: levels[levelIndex], levelIndex: levelIndex, totalLevels: levels.count)
        remaining = UserSettingsStore.shared.timerDuration
        view?.updateTimer(text: TimeFormatter.display(for: remaining))
    }

    func viewWillAppear() {
        guard !hasRecordedResult, !isExpired else {
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

    func submit(primary: String, secondary: String) {
        guard !isExpired else {
            view?.showResult(message: "Time is up. Tap Next Level to continue.", success: false)
            return
        }
        guard !hasRecordedResult else {
            view?.showResult(message: "Result already recorded. Tap Next Level to continue.", success: false)
            return
        }

        let levels = GameDataService.shared.cardCountLevels
        guard levelIndex < levels.count else { return }
        let level = levels[levelIndex]
        guard let left = Int(primary), let right = Int(secondary) else {
            view?.showResult(message: "Enter numbers for both target cards.", success: false)
            return
        }
        let expectedLeft = level.expectedCount(for: level.targetCardPrimary)
        let expectedRight = level.expectedCount(for: level.targetCardSecondary)
        let leftOK = left == expectedLeft
        let rightOK = right == expectedRight
        let success = leftOK && rightOK
        recordResult(correct: success)
        stopTimer()
        view?.showResult(
            message: "Left: \(leftOK ? "Correct" : "Wrong") (\(left))\nRight: \(rightOK ? "Correct" : "Wrong") (\(right))",
            success: success
        )
    }

    func nextLevel(from viewController: UIViewController) {
        let levels = GameDataService.shared.cardCountLevels
        if levelIndex >= levels.count - 1 {
            let alert = UIAlertController(title: "Mode Complete", message: "You finished every Card Spotter level.", preferredStyle: .alert)
            alert.addAction(UIAlertAction(title: "Back to Arena", style: .default) { _ in
                viewController.navigationController?.popViewController(animated: true)
            })
            viewController.present(alert, animated: true)
            return
        }
        let next = CardCountViewController(levelIndex: levelIndex + 1)
        viewController.navigationController?.pushViewController(next, animated: true)
    }

    private func startTimer() {
        guard timer == nil, !hasRecordedResult, !isExpired else { return }
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
                self.view?.showResult(message: "Time is up! Tap Next Level to continue.", success: false)
            } else {
                self.view?.updateTimer(text: TimeFormatter.display(for: self.remaining))
            }
        }
    }

    private func recordResult(correct: Bool) {
        guard !hasRecordedResult else { return }
        hasRecordedResult = true
        ProgressStore.shared.recordSession()
        ProgressStore.shared.recordCardCountResult(levelIndex: levelIndex, correct: correct)
    }

    private func stopTimer() {
        timer?.invalidate()
        timer = nil
    }
}

final class CardCountViewController: UIViewController, CardCountViewProtocol {
    private let presenter: CardCountPresenter
    private let scrollView = UIScrollView()
    private let stack = UIStackView()
    private let timerLabel = UILabel()
    private let spiralHost = UIView()
    private let matrixHost = UIView()
    private let primaryField = UITextField()
    private let secondaryField = UITextField()
    private let inputRow = UIStackView()
    private var inputImageViews: [UIImageView] = []
    private var currentLevel: CardCountLevel?
    private var lastLaidOutBoardWidth: CGFloat = 0

    init(levelIndex: Int) {
        presenter = CardCountPresenter(levelIndex: levelIndex)
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

    func render(level: CardCountLevel, levelIndex: Int, totalLevels: Int) {
        title = "Card Spotter \(levelIndex + 1)/\(totalLevels)"
        currentLevel = level
        lastLaidOutBoardWidth = 0
        renderCurrentLevelBoardsIfNeeded(force: true)

        let targets = [level.targetCardPrimary, level.targetCardSecondary]
        for (index, name) in targets.enumerated() where index < inputImageViews.count {
            inputImageViews[index].image = UIImage(named: name)
        }
    }

    func updateTimer(text: String) { timerLabel.text = text }

    func showResult(message: String, success: Bool) {
        showBanner(message: message, style: success ? .success : .failure)
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
        scrollView.alwaysBounceVertical = true
        view.addSubview(scrollView)
        scrollView.addSubview(stack)
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

        spiralHost.heightAnchor.constraint(equalToConstant: 280).isActive = true
        matrixHost.heightAnchor.constraint(equalToConstant: 180).isActive = true
        [PanelTitleView(text: "Spiral Board"), spiralHost, PanelTitleView(text: "Matrix Board"), matrixHost].forEach(stack.addArrangedSubview)

        let hint = UILabel()
        hint.text = "How many times does each target card appear across both boards?"
        hint.textColor = AppTheme.textSecondary
        hint.font = .systemFont(ofSize: 14, weight: .medium)
        hint.numberOfLines = 0
        stack.addArrangedSubview(hint)

        inputRow.axis = .horizontal
        inputRow.spacing = 12
        inputRow.distribution = .fillEqually
        let left = makeInputCard(field: primaryField, accessibilityLabel: "Primary target card count")
        let right = makeInputCard(field: secondaryField, accessibilityLabel: "Secondary target card count")
        inputRow.addArrangedSubview(left)
        inputRow.addArrangedSubview(right)
        stack.addArrangedSubview(inputRow)

        let confirm = PrimaryButton(title: "Check Answers")
        confirm.addTarget(self, action: #selector(confirmTapped), for: .touchUpInside)
        let next = SecondaryButton(title: "Next Level")
        next.addTarget(self, action: #selector(nextTapped), for: .touchUpInside)
        stack.addArrangedSubview(confirm)
        stack.addArrangedSubview(next)
    }

    private func makeInputCard(field: UITextField, accessibilityLabel: String) -> UIView {
        let container = UIView()
        container.backgroundColor = AppTheme.surface
        container.layer.cornerRadius = AppTheme.cornerRadius
        let imageView = UIImageView()
        imageView.contentMode = .scaleAspectFit
        inputImageViews.append(imageView)
        field.keyboardType = .numberPad
        field.textAlignment = .center
        field.placeholder = "Count"
        field.accessibilityLabel = accessibilityLabel
        field.backgroundColor = UIColor(white: 1, alpha: 0.08)
        field.textColor = .white
        field.layer.cornerRadius = 10
        field.heightAnchor.constraint(equalToConstant: 42).isActive = true
        let inner = UIStackView(arrangedSubviews: [imageView, field])
        inner.axis = .vertical
        inner.spacing = 10
        container.addSubview(inner)
        inner.translatesAutoresizingMaskIntoConstraints = false
        imageView.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            imageView.heightAnchor.constraint(equalToConstant: 56),
            inner.topAnchor.constraint(equalTo: container.topAnchor, constant: 14),
            inner.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: 14),
            inner.trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -14),
            inner.bottomAnchor.constraint(equalTo: container.bottomAnchor, constant: -14)
        ])
        return container
    }

    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        renderCurrentLevelBoardsIfNeeded(force: false)
    }

    private func renderCurrentLevelBoardsIfNeeded(force: Bool) {
        guard let currentLevel else { return }
        let boardWidth = spiralHost.bounds.width
        guard boardWidth > 0 else { return }
        guard force || abs(boardWidth - lastLaidOutBoardWidth) > 0.5 else { return }

        lastLaidOutBoardWidth = boardWidth
        layoutCards(currentLevel.spiralCards, in: spiralHost, spiral: true)
        layoutCards(currentLevel.matrixCards, in: matrixHost, spiral: false)
    }

    private func layoutCards(_ names: [String], in host: UIView, spiral: Bool) {
        host.subviews.forEach { $0.removeFromSuperview() }
        host.backgroundColor = AppTheme.surface
        host.layer.cornerRadius = AppTheme.cornerRadius
        let bounds = CGRect(x: 0, y: 0, width: max(host.bounds.width, 1), height: spiral ? 280 : 180)
        if spiral {
            let center = CGPoint(x: bounds.width / 2, y: bounds.height / 2)
            let maxInnerCount = 12
            let innerCount = min(names.count, maxInnerCount)
            let outerCount = max(0, names.count - innerCount)
            for (index, name) in names.enumerated() {
                let angle: CGFloat
                let radius: CGFloat
                if index < innerCount {
                    angle = (2 * .pi / CGFloat(max(innerCount, 1))) * CGFloat(index)
                    radius = 52
                } else {
                    angle = (2 * .pi / CGFloat(max(outerCount, 1))) * CGFloat(index - innerCount) + 0.3
                    radius = 105
                }
                let imageView = UIImageView(image: UIImage(named: name))
                imageView.frame = CGRect(x: 0, y: 0, width: 30, height: 42)
                imageView.center = CGPoint(x: center.x + cos(angle) * radius, y: center.y + sin(angle) * radius)
                imageView.transform = CGAffineTransform(rotationAngle: angle + .pi / 2)
                host.addSubview(imageView)
            }
        } else {
            let itemW: CGFloat = 40
            let itemH: CGFloat = 55
            let padding = (bounds.width - itemW * 5) / 6
            for (index, name) in names.enumerated() where !name.isEmpty {
                let row = index / 5
                let col = index % 5
                let imageView = UIImageView(image: UIImage(named: name))
                imageView.frame = CGRect(
                    x: padding + (itemW + padding) * CGFloat(col),
                    y: 12 + (itemH + 5) * CGFloat(row),
                    width: itemW,
                    height: itemH
                )
                imageView.contentMode = .scaleAspectFit
                host.addSubview(imageView)
            }
        }
    }

    @objc private func confirmTapped() {
        view.endEditing(true)
        presenter.submit(primary: primaryField.text ?? "", secondary: secondaryField.text ?? "")
    }

    @objc private func nextTapped() {
        presenter.nextLevel(from: self)
    }
}
