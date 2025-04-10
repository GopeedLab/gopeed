import 'package:flutter/material.dart';
import 'sidebar.dart';

class HomeScreen extends StatelessWidget {
  final Widget child;

  const HomeScreen({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Row(
        children: [
          const Sidebar(),
          Expanded(
            child: Container(color: const Color(0xFF131956), child: child),
          ),
        ],
      ),
    );
  }
}
