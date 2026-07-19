import 'package:flutter/material.dart';

import '../navigation/fade_route.dart';
import '../theme/app_gradients.dart';
import 'game_screen.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen>
    with SingleTickerProviderStateMixin {
  late final AnimationController _buttonController;

  @override
  void initState() {
    super.initState();
    _buttonController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 900),
      lowerBound: 0.95,
      upperBound: 1.0,
    )..repeat(reverse: true);
  }

  @override
  void dispose() {
    _buttonController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final size = MediaQuery.sizeOf(context);
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(gradient: AppGradients.home),
        child: SafeArea(
          child: Center(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(
                    Icons.layers_rounded,
                    size: size.width * 0.24,
                    color: Colors.white,
                  ),
                  const SizedBox(height: 16),
                  Text(
                    'COLOR STACK',
                    style: TextStyle(
                      fontSize: size.width * 0.10,
                      letterSpacing: 2,
                      fontWeight: FontWeight.w800,
                      color: Colors.white,
                    ),
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 8),
                  const Text(
                    'Stack same colors together',
                    style: TextStyle(color: Colors.white70, fontSize: 16),
                  ),
                  const SizedBox(height: 44),
                  ScaleTransition(
                    scale: _buttonController,
                    child: FilledButton(
                      onPressed: () {
                        Navigator.of(context).push(
                          createFadeRoute(const GameScreen()),
                        );
                      },
                      style: FilledButton.styleFrom(
                        minimumSize: const Size(220, 56),
                        backgroundColor: Colors.white,
                        foregroundColor: const Color(0xFF3840C2),
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(18),
                        ),
                      ),
                      child: const Text(
                        'PLAY',
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
