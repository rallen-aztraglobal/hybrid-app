import Foundation

struct CardCountLevel: Decodable {
    let targetCardPrimary: String
    let targetCardSecondary: String
    let spiralCards: [String]
    let matrixCards: [String]

    func expectedCount(for target: String) -> Int {
        let spiral = spiralCards.filter { $0 == target }.count
        let matrix = matrixCards.filter { !$0.isEmpty && $0 == target }.count
        return spiral + matrix
    }
}

struct HourglassLevel: Decodable {
    let ruleValues: [String]
    let targetCards: [String]
    let hourglassRing: [String]
    let answerChoices: [HourglassAnswer]

}

struct HourglassAnswer: Decodable {
    let answerValue: String
    let isCorrect: String

    var isCorrectAnswer: Bool { isCorrect == "1" }
}

enum GameMode: String, CaseIterable {
    case cardCount
    case hourglass

    var title: String {
        switch self {
        case .cardCount: return "Card Spotter"
        case .hourglass: return "Sum Balance"
        }
    }

    var subtitle: String {
        switch self {
        case .cardCount: return "Count target cards across two layouts."
        case .hourglass: return "Balance card totals against hourglass values."
        }
    }

    var symbolName: String {
        switch self {
        case .cardCount: return "rectangle.on.rectangle.angled"
        case .hourglass: return "hourglass"
        }
    }
}

struct PlayerProgress: Codable {
    var cardCountCompletedLevels: Int
    var hourglassCompletedLevels: Int
    var cardCountCurrentStreak: Int
    var hourglassCurrentStreak: Int
    var cardCountBestStreak: Int
    var hourglassBestStreak: Int
    var totalSessions: Int
    var lastPlayedAt: Date?

    static let empty = PlayerProgress(
        cardCountCompletedLevels: 0,
        hourglassCompletedLevels: 0,
        cardCountCurrentStreak: 0,
        hourglassCurrentStreak: 0,
        cardCountBestStreak: 0,
        hourglassBestStreak: 0,
        totalSessions: 0,
        lastPlayedAt: nil
    )

    init(
        cardCountCompletedLevels: Int,
        hourglassCompletedLevels: Int,
        cardCountCurrentStreak: Int,
        hourglassCurrentStreak: Int,
        cardCountBestStreak: Int,
        hourglassBestStreak: Int,
        totalSessions: Int,
        lastPlayedAt: Date?
    ) {
        self.cardCountCompletedLevels = cardCountCompletedLevels
        self.hourglassCompletedLevels = hourglassCompletedLevels
        self.cardCountCurrentStreak = cardCountCurrentStreak
        self.hourglassCurrentStreak = hourglassCurrentStreak
        self.cardCountBestStreak = cardCountBestStreak
        self.hourglassBestStreak = hourglassBestStreak
        self.totalSessions = totalSessions
        self.lastPlayedAt = lastPlayedAt
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        cardCountCompletedLevels = try container.decodeIfPresent(Int.self, forKey: .cardCountCompletedLevels) ?? 0
        hourglassCompletedLevels = try container.decodeIfPresent(Int.self, forKey: .hourglassCompletedLevels) ?? 0
        cardCountCurrentStreak = try container.decodeIfPresent(Int.self, forKey: .cardCountCurrentStreak) ?? 0
        hourglassCurrentStreak = try container.decodeIfPresent(Int.self, forKey: .hourglassCurrentStreak) ?? 0
        cardCountBestStreak = try container.decodeIfPresent(Int.self, forKey: .cardCountBestStreak) ?? 0
        hourglassBestStreak = try container.decodeIfPresent(Int.self, forKey: .hourglassBestStreak) ?? 0
        totalSessions = try container.decodeIfPresent(Int.self, forKey: .totalSessions) ?? 0
        lastPlayedAt = try container.decodeIfPresent(Date.self, forKey: .lastPlayedAt)
    }
}
