import 'package:flutter/material.dart';

import '../core/constants/app_strings.dart';
import '../core/constants/game_constants.dart';
import '../core/theme/app_theme.dart';
import 'home_screen.dart';

class SplashScreen extends StatefulWidget {
  const SplashScreen({super.key});

  @override
  State<SplashScreen> createState() => _SplashScreenState();
}

class _SplashScreenState extends State<SplashScreen>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: GameConstants.splashDuration,
    )..forward();
    _controller.addStatusListener((status) {
      if (status == AnimationStatus.completed) {
        _goHome();
      }
    });
  }

  void _goHome() {
    if (!mounted) return;
    Navigator.of(
      context,
    ).pushReplacement(MaterialPageRoute(builder: (_) => const HomeScreen()));
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(gradient: AppTheme.backgroundGradient),
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              SizedBox(
                width: 140,
                height: 140,
                child: AnimatedBuilder(
                  animation: _controller,
                  builder: (context, _) {
                    return Stack(
                      alignment: Alignment.bottomCenter,
                      clipBehavior: Clip.none,
                      children: [
                        for (var i = 0; i < kColorPalette.length; i++)
                          _buildBlock(i),
                      ],
                    );
                  },
                ),
              ),
              const SizedBox(height: 28),
              const Text(
                AppStrings.appName,
                style: TextStyle(
                  fontSize: 28,
                  fontWeight: FontWeight.bold,
                  color: AppTheme.textPrimary,
                  letterSpacing: 0.5,
                ),
              ),
              const SizedBox(height: 6),
              Text(
                AppStrings.tagline,
                style: TextStyle(fontSize: 14, color: AppTheme.textSecondary),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildBlock(int index) {
    final total = kColorPalette.length;
    final start = index / total * 0.7;
    final end = start + 0.3;
    final progress = Curves.easeOutBack.transform(
      ((_controller.value - start) / (end - start)).clamp(0.0, 1.0),
    );
    final style = kColorPalette[index];
    return Positioned(
      bottom: index * 20.0 * progress,
      child: Opacity(
        opacity: progress.clamp(0.0, 1.0),
        child: Container(
          width: 70,
          height: 26,
          decoration: BoxDecoration(
            color: style.color,
            borderRadius: BorderRadius.circular(10),
            boxShadow: const [
              BoxShadow(
                color: Colors.black38,
                blurRadius: 4,
                offset: Offset(0, 2),
              ),
            ],
          ),
          alignment: Alignment.center,
          child: Icon(style.icon, size: 14, color: Colors.white70),
        ),
      ),
    );
  }
}
