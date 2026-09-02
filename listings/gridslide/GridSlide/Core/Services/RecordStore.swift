import Foundation

extension Notification.Name {
    /// 成绩有变动（打破纪录或清空）时广播，Records 页据此刷新。
    static let recordsDidChange = Notification.Name("GridSlide.recordsDidChange")
}

/// 某个棋盘尺寸下的最好成绩。
struct BoardRecord: Codable, Equatable {
    /// 最少步数。nil = 还没通关过。
    var bestMoves: Int?
    /// 最短用时（秒）。nil = 还没通关过。
    var bestSeconds: Int?
    /// 通关次数。
    var completions: Int

    static let empty = BoardRecord(bestMoves: nil, bestSeconds: nil, completions: 0)
}

/// 一次通关之后，成绩有没有刷新纪录。
struct RecordOutcome {
    let isNewMovesBest: Bool
    let isNewTimeBest: Bool

    var isAnyNewBest: Bool { isNewMovesBest || isNewTimeBest }
}

/// 各棋盘尺寸的最好成绩。数据量极小，放 UserDefaults 正合适。
///
/// **注意 `init` 里只读、不发通知。** 单例的惰性初始化由 `swift_once` 保护，
/// 而那把锁不可重入 —— 若在 init 里发通知，观察者回头访问 `shared` 就会同线程
/// 二次进入还没走完的 once，直接崩。（这是同仓库 pocketledger 上真踩过的坑。）
final class RecordStore {
    static let shared = RecordStore()

    private let storageKey = "gsl.records"
    private var records: [String: BoardRecord]

    private init() {
        if let data = UserDefaults.standard.data(forKey: storageKey),
           let decoded = try? JSONDecoder().decode([String: BoardRecord].self, from: data) {
            records = decoded
        } else {
            records = [:]
        }
    }

    /// 字典键用字符串而不是 Int：`[Int: T]` 经 JSONEncoder 会被编成
    /// 「键值交替的数组」而不是对象，存档文件既不可读、将来也难手工修。
    private func recordKey(for size: Int) -> String { "\(size)" }

    func record(forSize size: Int) -> BoardRecord {
        records[recordKey(for: size)] ?? .empty
    }

    /// 提交一次通关成绩，返回是否刷新了纪录。
    ///
    /// 步数与时间**各自独立**记纪录：有人追求最少步、有人追求最快，
    /// 把两者绑在一起（只记「最少步那局的时间」）会让另一半玩家的努力看不见。
    @discardableResult
    func submit(size: Int, moves: Int, seconds: Int) -> RecordOutcome {
        var record = record(forSize: size)
        let newMovesBest = record.bestMoves.map { moves < $0 } ?? true
        let newTimeBest = record.bestSeconds.map { seconds < $0 } ?? true

        if newMovesBest { record.bestMoves = moves }
        if newTimeBest { record.bestSeconds = seconds }
        record.completions += 1

        records[recordKey(for: size)] = record
        persist()

        return RecordOutcome(isNewMovesBest: newMovesBest, isNewTimeBest: newTimeBest)
    }

    func reset() {
        records = [:]
        persist()
    }

    private func persist() {
        if let data = try? JSONEncoder().encode(records) {
            UserDefaults.standard.set(data, forKey: storageKey)
        }
        NotificationCenter.default.post(name: .recordsDidChange, object: nil)
    }
}

/// 用时显示。棋盘再难也不会打到一小时以上，mm:ss 够用。
enum TimeDisplay {
    static func clock(_ seconds: Int) -> String {
        let safe = max(0, seconds)
        return String(format: "%02d:%02d", safe / 60, safe % 60)
    }
}
