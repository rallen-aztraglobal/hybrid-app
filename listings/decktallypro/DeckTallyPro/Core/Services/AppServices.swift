import Foundation

final class GameDataService {
    static let shared = GameDataService()

    private(set) var cardCountLevels: [CardCountLevel] = []
    private(set) var hourglassLevels: [HourglassLevel] = []

    private init() {
        reload()
    }

    func reload() {
        cardCountLevels = load(resource: "dtp_game1_levels")
        hourglassLevels = load(resource: "dtp_game2_levels")
    }

    private func load<T: Decodable>(resource: String) -> [T] {
        guard let url = Bundle.main.url(forResource: resource, withExtension: "json") else {
            assertionFailure("Missing bundled resource: \(resource).json")
            return []
        }

        do {
            let data = try Data(contentsOf: url)
            return try JSONDecoder().decode([T].self, from: data)
        } catch {
            assertionFailure("Failed to load or decode \(resource).json: \(error)")
            return []
        }
    }
}

final class UserSettingsStore {
    static let shared = UserSettingsStore()

    private let timerKey = "DTPtimerPresetIndex"
    private let onboardingKey = "dtp.didShowOnboarding"

    private let timerOptions = [120, 110, 100, 90, 80, 70, 60]

    var timerPresetIndex: Int {
        get {
            let stored = UserDefaults.standard.integer(forKey: timerKey)
            return (0...6).contains(stored) ? stored : 0
        }
        set {
            UserDefaults.standard.set(max(0, min(6, newValue)), forKey: timerKey)
        }
    }

    var timerDuration: TimeInterval {
        TimeInterval(timerOptions[timerPresetIndex])
    }

    var timerLabels: [String] {
        timerOptions.map { "\($0)s" }
    }

    var didShowOnboarding: Bool {
        get { UserDefaults.standard.bool(forKey: onboardingKey) }
        set { UserDefaults.standard.set(newValue, forKey: onboardingKey) }
    }

    static let privacyPolicyURL = URL(string: "https://www.privacypolicies.com/live/7375efd6-f47b-4e5d-aa19-e11a1311e803")!
    static let appStoreURL = URL(string: "https://apps.apple.com/app/deck-tally-pro/id6780248860")!
}

final class ProgressStore {
    static let shared = ProgressStore()

    private let key = "dtp.playerProgress"

    private(set) var progress: PlayerProgress

    private init() {
        if let data = UserDefaults.standard.data(forKey: key),
           let decoded = try? JSONDecoder().decode(PlayerProgress.self, from: data) {
            progress = decoded
        } else {
            progress = .empty
        }
    }

    func recordSession() {
        progress.totalSessions += 1
        progress.lastPlayedAt = Date()
        persist()
    }

    func recordCardCountResult(levelIndex: Int, correct: Bool) {
        if correct {
            progress.cardCountCompletedLevels = max(progress.cardCountCompletedLevels, levelIndex + 1)
            progress.cardCountCurrentStreak += 1
            progress.cardCountBestStreak = max(progress.cardCountBestStreak, progress.cardCountCurrentStreak)
        } else {
            progress.cardCountCurrentStreak = 0
        }
        persist()
    }

    func recordHourglassResult(levelIndex: Int, correct: Bool) {
        if correct {
            progress.hourglassCompletedLevels = max(progress.hourglassCompletedLevels, levelIndex + 1)
            progress.hourglassCurrentStreak += 1
            progress.hourglassBestStreak = max(progress.hourglassBestStreak, progress.hourglassCurrentStreak)
        } else {
            progress.hourglassCurrentStreak = 0
        }
        persist()
    }

    private func persist() {
        do {
            let data = try JSONEncoder().encode(progress)
            UserDefaults.standard.set(data, forKey: key)
        } catch {
            assertionFailure("ProgressStore: failed to encode progress — \(error)")
        }
    }
}

// MM:SS display — max game timer is 120 s, HH:MM:SS is never needed.
enum TimeFormatter {
    static func display(for interval: TimeInterval) -> String {
        let seconds = max(0, Int(interval.rounded()))
        let minutes = seconds / 60
        let secs = seconds % 60
        return String(format: "%02d:%02d", minutes, secs)
    }
}
