import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/window/app_window_chrome.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/gopeed_app_mark.dart';
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
      padding: EdgeInsets.only(top: AppWindowChrome.reservesHeaderInset ? AppDesignTokens.windowHeaderHeight : 0),
      child: Column(
        children: [
          SizedBox(
            height: AppDesignTokens.contentHeaderHeight,
            child: const Center(child: GopeedAppMark(key: ValueKey('primary-rail-app-mark'))),
          ),
          const SizedBox(height: AppDesignTokens.space8),
          _RailItem(
            key: const ValueKey('primary-rail-tasks-item'),
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
      decoration: BoxDecoration(
        color: palette.railBg,
        border: Border(top: BorderSide(color: palette.border)),
      ),
      child: SafeArea(
        top: false,
        child: SizedBox(
          height: 64,
          child: Padding(
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
          ),
        ),
      ),
    );
  }
}

class _RailItem extends StatelessWidget {
  const _RailItem({super.key, required this.icon, required this.onTap, this.active = false});

  final IconData icon;
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
