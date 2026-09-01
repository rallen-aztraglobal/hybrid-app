import UIKit

/// 会跟着账本变动自动刷新的页面基类。
///
/// 四个标签页都要在「别处改了账本」之后重画（比如在流水页记了一笔，账户页的余额得跟着变）。
/// 与其在每个页面里各写一遍订阅与注销，不如收在这里，子类只实现 `reloadContent()`。
///
/// 同时在 `viewWillAppear` 再刷一次：页面在后台时也会收到通知并刷新，但标签页切换回来时
/// 再拉一遍最省心 —— 免得纠结「不可见时到底要不要刷」。数据量很小，重画不值一提。
class LedgerObservingViewController: UIViewController {
    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = AppTheme.background
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(handleLedgerDidChange),
            name: .ledgerDidChange,
            object: nil
        )
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        reloadContent()
    }

    @objc private func handleLedgerDidChange() {
        reloadContent()
    }

    /// 子类重写：从 LedgerStore 重新取数并更新界面。
    func reloadContent() {}

    /// 当前币种。四个页面都要用，收在基类里免得各处重复取。
    var currencyCode: String {
        UserSettingsStore.shared.currencyCode
    }
}
