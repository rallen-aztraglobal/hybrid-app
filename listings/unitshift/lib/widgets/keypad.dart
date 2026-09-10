import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../theme/app_colors.dart';

/// 键盘上的一个键。
enum KeyAction { digit, dot, sign, backspace, clear }

class KeyPress {
  const KeyPress(this.action, [this.digit]);

  final KeyAction action;
  final String? digit;
}

/// 自定义数字键盘。
///
/// 不用系统键盘的理由：一弹一收布局就跳，换算列表被顶得看不见；
/// 各家输入法的数字键位差别很大，还可能塞进全角数字和其他符号。
/// 自己画一块，位置固定、按键大、只可能输出合法字符。
class Keypad extends StatelessWidget {
  const Keypad({super.key, required this.onKey});

  final ValueChanged<KeyPress> onKey;

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        // 退格放在第三行右下角而不是右上角：这是按得最频繁的功能键，
        // 单手握持时拇指够右下比够右上省力得多。
        _row(<Widget>[_digit('7'), _digit('8'), _digit('9'), _clear()]),
        _row(<Widget>[_digit('4'), _digit('5'), _digit('6'), _sign()]),
        _row(<Widget>[_digit('1'), _digit('2'), _digit('3'), _backspace()]),
        // 末行两个键各占一半，把整行铺满 —— 留个空洞会让人以为少了个键。
        _row(<Widget>[_digit('0', flex: 2), _dot(flex: 2)]),
      ],
    );
  }

  Widget _row(List<Widget> children) => Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: Row(children: children),
      );

  Widget _digit(String d, {int flex = 1}) => _KeyButton(
        flex: flex,
        label: d,
        onTap: () => onKey(KeyPress(KeyAction.digit, d)),
      );

  Widget _dot({int flex = 1}) => _KeyButton(
        flex: flex,
        label: '.',
        onTap: () => onKey(const KeyPress(KeyAction.dot)),
      );

  Widget _sign() => _KeyButton(
        label: '±',
        muted: true,
        onTap: () => onKey(const KeyPress(KeyAction.sign)),
      );

  Widget _backspace() => _KeyButton(
        icon: Icons.backspace_outlined,
        muted: true,
        onTap: () => onKey(const KeyPress(KeyAction.backspace)),
      );

  Widget _clear() => _KeyButton(
        label: 'C',
        muted: true,
        onTap: () => onKey(const KeyPress(KeyAction.clear)),
      );

}

/// 一个键。按下当场变色 + 轻触感 —— 数字键盘最要紧的就是「我到底按上了没有」。
class _KeyButton extends StatefulWidget {
  const _KeyButton({
    this.label,
    this.icon,
    required this.onTap,
    this.flex = 1,
    this.muted = false,
  });

  final String? label;
  final IconData? icon;
  final VoidCallback onTap;
  final int flex;
  final bool muted;

  @override
  State<_KeyButton> createState() => _KeyButtonState();
}

class _KeyButtonState extends State<_KeyButton> {
  bool _down = false;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      flex: widget.flex,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 4),
        child: GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTapDown: (_) => setState(() => _down = true),
          onTapCancel: () => setState(() => _down = false),
          onTapUp: (_) {
            setState(() => _down = false);
            HapticFeedback.selectionClick();
            widget.onTap();
          },
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 90),
            height: 54,
            decoration: BoxDecoration(
              color: _down
                  ? AppColors.keyPressed
                  : (widget.muted ? AppColors.keyMuted : AppColors.key),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: AppColors.border),
            ),
            alignment: Alignment.center,
            child: widget.icon != null
                ? Icon(widget.icon, size: 20, color: AppColors.primaryText)
                : Text(
                    widget.label!,
                    style: TextStyle(
                      color: AppColors.primaryText,
                      fontSize: widget.muted ? 17 : 21,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
          ),
        ),
      ),
    );
  }
}
