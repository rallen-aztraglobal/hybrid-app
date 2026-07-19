import UIKit

final class GuideViewController: UIViewController {
    var onDismiss: (() -> Void)?

    private let scrollView = UIScrollView()
    private let stack = UIStackView()

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "How To Play"
        view.backgroundColor = AppTheme.background
        navigationItem.rightBarButtonItem = UIBarButtonItem(barButtonSystemItem: .close, target: self, action: #selector(closeTapped))
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
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor, constant: 20),
            stack.leadingAnchor.constraint(equalTo: scrollView.leadingAnchor, constant: AppTheme.horizontalPadding),
            stack.trailingAnchor.constraint(equalTo: scrollView.trailingAnchor, constant: -AppTheme.horizontalPadding),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -24),
            stack.widthAnchor.constraint(equalTo: scrollView.widthAnchor, constant: -AppTheme.horizontalPadding * 2)
        ])

        stack.addArrangedSubview(InfoCardView(title: "Card Spotter", body: "Study the spiral and matrix boards, then count how many times each target card appears. Enter both counts before checking your answer."))
        stack.addArrangedSubview(InfoCardView(title: "Sum Balance", body: "Each card color has a value shown on the left. Use the ring of hourglasses and card matrix to calculate: card total minus hourglass total."))
        stack.addArrangedSubview(InfoCardView(title: "Timer & Progress", body: "Adjust round length in Settings. Completed levels and streaks are tracked in the Progress tab so you can resume where you left off."))
        stack.addArrangedSubview(InfoCardView(title: "Training Goal", body: "Deck Tally Pro is designed for short, repeatable focus drills. Play a few rounds daily to improve speed and accuracy."))
    }

    @objc private func closeTapped() {
        onDismiss?()
        if let nav = navigationController, nav.viewControllers.first !== self {
            navigationController?.popViewController(animated: true)
        } else {
            dismiss(animated: true)
        }
    }
}
