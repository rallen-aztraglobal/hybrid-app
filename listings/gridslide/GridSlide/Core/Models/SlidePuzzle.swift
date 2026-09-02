import Foundation

/// 滑块拼图的规则内核。**纯 Swift、无 UIKit 依赖、无副作用。**
///
/// 棋盘用一维数组表示，`0` 是空格，复原态是 `[1, 2, ..., n-1, 0]`。
/// 下标与坐标的换算全在 `rowOf` / `colOf` 里，其余代码不直接做除法取模。
///
/// > 这份逻辑在写进本工程之前，先用等价的 Dart 实现跑了 19 项测试
/// > （见 `README_GATE.md`「算法已经验证过」一节），本文件是那份实现的逐行转写。
/// > 注意：那验证的是**算法**，不是这段 Swift 代码本身 —— 转写有没有笔误，
/// > 仍要靠在 Mac 上编译并实跑来确认。
struct SlidePuzzle {
    /// 边长。3 / 4 / 5。
    let size: Int

    private(set) var tiles: [Int]

    init(size: Int) {
        // 2×2 的滑块拼图只有两种局面，不成其为游戏；调用方只会传 3/4/5。
        self.size = max(3, size)
        let count = self.size * self.size
        tiles = (0..<count).map { ($0 + 1) % count }
    }

    var count: Int { size * size }

    /// 空格所在下标。棋盘里恒有且仅有一个 0，取不到时返回最后一格兜底（不应发生）。
    var blankIndex: Int {
        tiles.firstIndex(of: 0) ?? (count - 1)
    }

    func rowOf(_ index: Int) -> Int { index / size }
    func colOf(_ index: Int) -> Int { index % size }

    /// 是否已复原：前 n-1 格依次放着 1...n-1，最后一格是空格。
    var isSolved: Bool {
        for i in 0..<(count - 1) where tiles[i] != i + 1 {
            return false
        }
        return tiles[count - 1] == 0
    }

    /// 第 index 格是不是已经归位（空格不算）。用来给归位的块加高亮。
    func isTileInPlace(_ index: Int) -> Bool {
        guard index >= 0 && index < count else { return false }
        return tiles[index] != 0 && tiles[index] == index + 1
    }

    /// 点击第 index 格能不能动：必须与空格同行或同列，且自己不是空格。
    ///
    /// 允许「整排滑动」—— 点同一行/列上隔着几格的块，中间的块一起挪。
    /// 这是这类游戏的通行手感；不这么做玩家得一格一格点，很费手。
    func canMove(_ index: Int) -> Bool {
        guard index >= 0 && index < count else { return false }
        guard tiles[index] != 0 else { return false }
        let blank = blankIndex
        return rowOf(index) == rowOf(blank) || colOf(index) == colOf(blank)
    }

    /// 执行一次滑动。返回**被挪动过的格子下标**；非法点击返回空数组且不改动状态。
    ///
    /// 行与列只差一个步长（1 与 size），所以合成同一段循环 —— 两份几乎相同的
    /// 代码最容易在某次改动里只改一半。
    @discardableResult
    mutating func move(_ index: Int) -> [Int] {
        guard canMove(index) else { return [] }

        let blank = blankIndex
        let step: Int
        if rowOf(index) == rowOf(blank) {
            step = index > blank ? 1 : -1
        } else {
            step = index > blank ? size : -size
        }

        var moved: [Int] = []
        var cursor = blank
        // 空格朝 index 的方向逐格吞并：把下一格的值挪过来，自己继续前进。
        while cursor != index {
            let next = cursor + step
            tiles[cursor] = tiles[next]
            moved.append(cursor)
            cursor = next
        }
        tiles[cursor] = 0
        moved.append(cursor)

        return moved
    }

    /// 打乱。
    ///
    /// **做法是从复原态开始随机走合法步**，而不是随机排列后再修奇偶性。
    /// 后者要正确处理「空格所在行」的奇偶修正，边界多、极易写错；前者天然只会
    /// 走到可解的局面 —— 每一步都可逆，逆着走回去必然复原。
    ///
    /// （这一点在 Dart 侧被穷举验证过：200 次随机排列有一半以上不可解，
    /// 而 200 次本方法的产出全部可解。）
    /// 用系统随机源打乱（正常游戏走这条）。
    mutating func shuffle(steps: Int) {
        var generator = SystemRandomNumberGenerator()
        shuffle(steps: steps, using: &generator)
    }

    /// 注入随机源的版本，便于将来加测试时复现局面。
    ///
    /// 这里用的是老式泛型而不是 `inout some RandomNumberGenerator` —— 后者是 Swift 5.7
    /// 才有的不透明参数语法，本工程的 `SWIFT_VERSION` 设为 5.0，写成泛型不依赖任何较新语法。
    mutating func shuffle<G: RandomNumberGenerator>(steps: Int, using generator: inout G) {
        reset()
        var previousBlank = -1

        for _ in 0..<steps {
            // 不要立刻把上一步原样退回去，否则等于原地踏步、白打乱。
            let options = movableIndices().filter { $0 != previousBlank }
            guard let pick = options.randomElement(using: &generator) else { continue }
            previousBlank = blankIndex
            move(pick)
        }

        // 步数少时有可能恰好绕回复原态，那样一开局就是「已完成」。多走一轮。
        if isSolved {
            shuffle(steps: steps + size, using: &generator)
        }
    }

    mutating func reset() {
        tiles = (0..<count).map { ($0 + 1) % count }
    }

    /// 当前所有能动的格子（与空格同行或同列的非空格）。
    func movableIndices() -> [Int] {
        let blank = blankIndex
        var result: [Int] = []
        for i in 0..<count where i != blank {
            if rowOf(i) == rowOf(blank) || colOf(i) == colOf(blank) {
                result.append(i)
            }
        }
        return result
    }
}
