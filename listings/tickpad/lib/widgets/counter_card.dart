import 'package:flutter/material.dart';

import '../logic/counter.dart';
import '../theme/app_colors.dart';

/// 一张计数卡片。
///
/// 布局的一条硬要求：**整张卡片都是加一的按钮**。计数器是不看屏幕也要能按的东西，
/// 按钮做小了就得瞄准。减一是个小按钮靠边放，误触的代价也小（还有撤销兜底）。
class CounterCard extends StatefulWidget {
  const CounterCard({
    super.key,
    required this.counter,
    required this.active,
    required this.onIncrement,
    required this.onDecrement,
    required this.onEdit,
  });

  final Counter counter;

  /// 是否是「当前」计数器 —— 音量键作用在它身上，所以必须一眼看得出是哪张。
  final bool active;

  final VoidCallback onIncrement;
  final VoidCallback onDecrement;
  final VoidCallback onEdit;

  @override
  State<CounterCard> createState() => _CounterCardState();
}

class _CounterCardState extends State<CounterCard> {
  bool _down = false;

  @override
  Widget build(BuildContext context) {
    final counter = widget.counter;
    final tag = AppColors.tag(counter.colorIndex);

    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTapDown: (_) => setState(() => _down = true),
      onTapCancel: () => setState(() => _down = false),
      onTapUp: (_) {
        setState(() => _down = false);
        widget.onIncrement();
      },
      onLongPress: widget.onEdit,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 90),
        margin: const EdgeInsets.only(bottom: 10),
        decoration: BoxDecoration(
          color: _down ? AppColors.surfacePressed : AppColors.surface,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: widget.active ? tag : AppColors.border,
            width: widget.active ? 1.6 : 1,
          ),
        ),
        clipBehavior: Clip.antiAlias,
        child: Row(
          children: <Widget>[
            // 左侧色条。比色点更容易在余光里认出来，也不占正文的位置。
            Container(width: 5, height: 92, color: tag),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(13, 12, 12, 12),
                child: Row(
                  children: <Widget>[
                    Expanded(child: _readout(counter, tag)),
                    _RoundButton(
                      icon: Icons.remove_rounded,
                      onTap: widget.onDecrement,
                      enabled: counter.value > 0,
                    ),
                    const SizedBox(width: 10),
                    // 加一：只是个视觉提示 —— 真正的热区是整张卡片
                    Container(
                      width: 52,
                      height: 52,
                      decoration: BoxDecoration(
                        color: tag,
                        borderRadius: BorderRadius.circular(14),
                      ),
                      child: const Icon(
                        Icons.add_rounded,
                        size: 26,
                        color: AppColors.onTag,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _readout(Counter counter, Color tag) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        Row(
          children: <Widget>[
            Flexible(
              child: Text(
                counter.label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  color: AppColors.mutedText,
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            if (counter.isComplete) ...<Widget>[
              const SizedBox(width: 6),
              Icon(Icons.check_circle_rounded, size: 14, color: tag),
            ],
          ],
        ),
        const SizedBox(height: 2),
        Row(
          crossAxisAlignment: CrossAxisAlignment.baseline,
          textBaseline: TextBaseline.alphabetic,
          children: <Widget>[
            Text(
              '${counter.value}',
              style: TextStyle(
                color: counter.isComplete ? tag : AppColors.primaryText,
                fontSize: 34,
                fontWeight: FontWeight.w700,
                height: 1.0,
                // 等宽字形：数字跳动时卡片不会跟着左右抖
                fontFeatures: const <FontFeature>[FontFeature.tabularFigures()],
              ),
            ),
            if (counter.hasTarget)
              Text(
                ' / ${counter.target}',
                style: const TextStyle(
                  color: AppColors.mutedText,
                  fontSize: 15,
                  fontWeight: FontWeight.w600,
                  fontFeatures: <FontFeature>[FontFeature.tabularFigures()],
                ),
              ),
            if (counter.step != 1) ...<Widget>[
              const SizedBox(width: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
                decoration: BoxDecoration(
                  color: AppColors.surfacePressed,
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Text(
                  '+${counter.step}',
                  style: TextStyle(
                    color: tag,
                    fontSize: 11.5,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            ],
          ],
        ),
        if (counter.hasTarget) ...<Widget>[
          const SizedBox(height: 8),
          ClipRRect(
            borderRadius: BorderRadius.circular(3),
            child: TweenAnimationBuilder<double>(
              tween: Tween<double>(begin: 0, end: counter.progress),
              duration: const Duration(milliseconds: 200),
              curve: Curves.easeOut,
              builder: (context, value, _) => LinearProgressIndicator(
                value: value,
                minHeight: 4,
                backgroundColor: AppColors.surfacePressed,
                valueColor: AlwaysStoppedAnimation<Color>(tag),
              ),
            ),
          ),
        ],
      ],
    );
  }
}

class _RoundButton extends StatelessWidget {
  const _RoundButton({
    required this.icon,
    required this.onTap,
    required this.enabled,
  });

  final IconData icon;
  final VoidCallback onTap;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTap: enabled ? onTap : null,
      child: Container(
        width: 44,
        height: 44,
        decoration: BoxDecoration(
          color: AppColors.minus,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Icon(
          icon,
          size: 20,
          // 到 0 就把减号淡掉：告诉用户「这里减不动了」，而不是按下去没反应
          color: enabled ? AppColors.primaryText : AppColors.mutedText,
        ),
      ),
    );
  }
}
