import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:go_router/go_router.dart';

import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../l10n/l10n.dart';

enum RailSection { tasks, extensions, settings }

class PrimaryRail extends StatelessWidget {
  const PrimaryRail({super.key, this.activeSection = RailSection.tasks});

  final RailSection activeSection;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);

    return Container(
      width: AppDesignTokens.railWidth,
      color: palette.railBg,
      padding: const EdgeInsets.only(top: AppDesignTokens.windowHeaderHeight),
      child: Column(
        children: [
          SizedBox(
            height: AppDesignTokens.contentHeaderHeight,
            child: Center(
              child: Text(
                'M',
                style: TextStyle(
                  color: palette.textPrimary,
                  fontSize: 18,
                  fontWeight: FontWeight.w800,
                  letterSpacing: -0.8,
                ),
              ),
            ),
          ),
          const SizedBox(height: 6),
          _RailItem(
            icon: Icons.download_rounded,
            active: activeSection == RailSection.tasks,
            onTap: () => context.go('/'),
          ),
          const SizedBox(height: 28),
          _RailItem(
            icon: Icons.extension_outlined,
            active: activeSection == RailSection.extensions,
            onTap: () => context.go('/extensions'),
          ),
          const SizedBox(height: 28),
          _RailItem(
            icon: Icons.settings_outlined,
            active: activeSection == RailSection.settings,
            onTap: () => context.go('/settings'),
          ),
          const Spacer(),
          RotatedBox(
            quarterTurns: 3,
            child: Text(
              'V1.0',
              style: TextStyle(color: palette.textMuted, fontSize: 10, fontWeight: FontWeight.w600, letterSpacing: 1.2),
            ),
          ),
          const SizedBox(height: 20),
        ],
      ),
    );
  }
}

class PrimaryBottomNavigation extends StatelessWidget {
  const PrimaryBottomNavigation({super.key, this.activeSection = RailSection.tasks});

  final RailSection activeSection;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(
      height: 64,
      decoration: BoxDecoration(
        color: palette.railBg,
        border: Border(top: BorderSide(color: palette.border)),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 10),
      child: Row(
        children: [
          Expanded(
            child: _BottomNavItem(
              icon: Icons.download_rounded,
              label: context.l10n.task,
              active: activeSection == RailSection.tasks,
              onTap: () => context.go('/'),
            ),
          ),
          Expanded(
            child: _BottomNavItem(
              icon: Icons.extension_outlined,
              label: context.l10n.extensions,
              active: activeSection == RailSection.extensions,
              onTap: () => context.go('/extensions'),
            ),
          ),
          Expanded(
            child: _BottomNavItem(
              icon: Icons.settings_outlined,
              label: context.l10n.setting,
              active: activeSection == RailSection.settings,
              onTap: () => context.go('/settings'),
            ),
          ),
        ],
      ),
    );
  }
}

class _RailItem extends StatelessWidget {
  const _RailItem({required this.icon, required this.onTap, this.active = false});

  final IconData icon;
  final VoidCallback onTap;
  final bool active;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        onTap: onTap,
        child: SizedBox(
          width: AppDesignTokens.railWidth,
          height: 24,
          child: Stack(
            children: [
              if (active)
                Align(
                  alignment: Alignment.centerLeft,
                  child: Container(
                    key: const ValueKey('primary-rail-active-indicator'),
                    width: 2,
                    height: 24,
                    color: palette.textPrimary,
                  ),
                ),
              Center(child: Icon(icon, size: 18, color: active ? palette.textPrimary : palette.textMuted)),
            ],
          ),
        ),
      ),
    );
  }
}

class _BottomNavItem extends StatelessWidget {
  const _BottomNavItem({required this.icon, required this.label, required this.onTap, this.active = false});

  final IconData icon;
  final String label;
  final VoidCallback onTap;
  final bool active;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: onTap,
        child: SizedBox(
          height: 54,
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(icon, size: 19, color: active ? palette.textPrimary : palette.textMuted),
              const SizedBox(height: 4),
              Text(
                label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  color: active ? palette.textPrimary : palette.textMuted,
                  fontSize: 10,
                  fontWeight: active ? FontWeight.w700 : FontWeight.w600,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
