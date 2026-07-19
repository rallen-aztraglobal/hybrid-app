import UIKit

protocol HomeViewProtocol: AnyObject {
    func render(viewModel: HomeViewModel)
}

struct HomeViewModel {
    let headline: String
    let sessions: Int
    let cardProgress: String
    let hourglassProgress: String
    let recommendation: String
}

final class HomePresenter {
    weak var view: HomeViewProtocol?

    private let dataService: GameDataService
    private let progressStore: ProgressStore

    init(
        dataService: GameDataService = .shared,
        progressStore: ProgressStore = .shared
    ) {
        self.dataService = dataService
        self.progressStore = progressStore
    }

    func viewDidLoad() {
        reload()
    }

    func reload() {
        let progress = progressStore.progress
        let cardTotal = dataService.cardCountLevels.count
        let hourglassTotal = dataService.hourglassLevels.count
        let recommendation: String
        if progress.cardCountCompletedLevels < cardTotal {
            recommendation = "Continue Card Spotter at level \(progress.cardCountCompletedLevels + 1)."
        } else if progress.hourglassCompletedLevels < hourglassTotal {
            recommendation = "Try Sum Balance level \(progress.hourglassCompletedLevels + 1)."
        } else {
            recommendation = "All levels cleared. Replay for a faster time."
        }

        view?.render(viewModel: HomeViewModel(
            headline: "Train focus with fast card math.",
            sessions: progress.totalSessions,
            cardProgress: "\(min(progress.cardCountCompletedLevels, cardTotal))/\(cardTotal)",
            hourglassProgress: "\(min(progress.hourglassCompletedLevels, hourglassTotal))/\(hourglassTotal)",
            recommendation: recommendation
        ))
    }
}

final class HomeViewController: UIViewController, HomeViewProtocol {
    private let presenter = HomePresenter()
    private let scrollView = UIScrollView()
    private let contentStack = UIStackView()
    private let headlineLabel = UILabel()
    private let statsStack = UIStackView()
    private let sessionsTile = MetricTileView(title: "Sessions", value: "-", symbol: "bolt.fill")
    private let cardTile = MetricTileView(title: "Card Spotter", value: "-", symbol: "suit.spade.fill")
    private let hourglassTile = MetricTileView(title: "Sum Balance", value: "-", symbol: "hourglass")
    private let recommendationCard = InfoCardView()

    override func viewDidLoad() {
        super.viewDidLoad()
        presenter.view = self
        configureUI()
        presenter.viewDidLoad()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        presenter.reload()
    }

    func render(viewModel: HomeViewModel) {
        headlineLabel.text = viewModel.headline
        sessionsTile.update(value: "\(viewModel.sessions)")
        cardTile.update(value: viewModel.cardProgress)
        hourglassTile.update(value: viewModel.hourglassProgress)
        recommendationCard.configure(title: "Suggested Next Step", body: viewModel.recommendation)
    }

    private func configureUI() {
        view.backgroundColor = AppTheme.background
        navigationItem.rightBarButtonItem = UIBarButtonItem(
            image: UIImage(systemName: "questionmark.circle"),
            style: .plain,
            target: self,
            action: #selector(openGuide)
        )

        scrollView.alwaysBounceVertical = true
        contentStack.axis = .vertical
        contentStack.spacing = 18

        let hero = HeroHeaderView(
            title: "Deck Tally Pro",
            subtitle: "A focused training studio for card counting and quick arithmetic."
        )

        headlineLabel.font = .systemFont(ofSize: 18, weight: .semibold)
        headlineLabel.textColor = AppTheme.textPrimary
        headlineLabel.numberOfLines = 0

        statsStack.axis = .vertical
        statsStack.spacing = 12
        [sessionsTile, cardTile, hourglassTile].forEach(statsStack.addArrangedSubview)

        let quickStart = PrimaryButton(title: "Open Arena")
        quickStart.addTarget(self, action: #selector(openArena), for: .touchUpInside)

        view.addSubview(scrollView)
        scrollView.addSubview(contentStack)
        [hero, headlineLabel, statsStack, recommendationCard, quickStart].forEach(contentStack.addArrangedSubview)

        scrollView.translatesAutoresizingMaskIntoConstraints = false
        contentStack.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor),
            contentStack.topAnchor.constraint(equalTo: scrollView.topAnchor, constant: 12),
            contentStack.leadingAnchor.constraint(equalTo: scrollView.leadingAnchor, constant: AppTheme.horizontalPadding),
            contentStack.trailingAnchor.constraint(equalTo: scrollView.trailingAnchor, constant: -AppTheme.horizontalPadding),
            contentStack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -24),
            contentStack.widthAnchor.constraint(equalTo: scrollView.widthAnchor, constant: -AppTheme.horizontalPadding * 2)
        ])
    }

    @objc private func openArena() {
        guard let tabBarController else { return }
        if let arenaIndex = tabBarController.viewControllers?.firstIndex(where: { viewController in
            if let nav = viewController as? UINavigationController {
                return nav.viewControllers.first is ArenaViewController
            }
            return viewController is ArenaViewController
        }) {
            tabBarController.selectedIndex = arenaIndex
        }
    }

    @objc private func openGuide() {
        let guide = GuideViewController()
        let nav = UINavigationController(rootViewController: guide)
        AppTheme.applyNavigationAppearance(nav.navigationBar)
        present(nav, animated: true)
    }
}
