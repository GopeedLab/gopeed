import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';

import '../../../../app/application/app_appearance_controller.dart';
import '../../../../core/utils/breakpoints.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/theme/app_theme_color.dart';
import '../../../../l10n/l10n.dart';

class ThemeModeSelector extends StatelessWidget {
  const ThemeModeSelector({super.key, required this.value, required this.accent, required this.onChanged});

  final AppThemeMode value;
  final AppThemeColor accent;
  final ValueChanged<AppThemeMode> onChanged;

  @override
  Widget build(BuildContext context) {
    final mobile = MediaQuery.sizeOf(context).width < Breakpoints.mobile;
    return LayoutBuilder(
      builder: (context, constraints) {
        final available = constraints.maxWidth.isFinite ? constraints.maxWidth : 390.0;
        final optionWidth = ((available - 16) / 3).clamp(84.0, 116.0);
        return Wrap(
          spacing: 8,
          runSpacing: 10,
          alignment: mobile ? WrapAlignment.start : WrapAlignment.end,
          children: [
            _ThemeModeOption(
              mode: AppThemeMode.system,
              label: context.l10n.themeSystem,
              icon: Icons.desktop_windows_outlined,
              width: optionWidth,
              selected: value == AppThemeMode.system,
              accent: accent,
              onPressed: onChanged,
            ),
            _ThemeModeOption(
              mode: AppThemeMode.light,
              label: context.l10n.themeLight,
              icon: Icons.light_mode_outlined,
              width: optionWidth,
              selected: value == AppThemeMode.light,
              accent: accent,
              onPressed: onChanged,
            ),
            _ThemeModeOption(
              mode: AppThemeMode.dark,
              label: context.l10n.themeDark,
              icon: Icons.dark_mode_outlined,
              width: optionWidth,
              selected: value == AppThemeMode.dark,
              accent: accent,
              onPressed: onChanged,
            ),
          ],
        );
      },
    );
  }
}

class ThemeColorSelector extends StatelessWidget {
  const ThemeColorSelector({super.key, required this.value, required this.onChanged});

  final AppThemeColor value;
  final ValueChanged<AppThemeColor> onChanged;

  @override
  Widget build(BuildContext context) {
    final dark = AppPalette.of(context).bg.computeLuminance() < 0.2;
    final mobile = MediaQuery.sizeOf(context).width < Breakpoints.mobile;
    final swatchSize = mobile ? 38.0 : 30.0;
    return Wrap(
      spacing: mobile ? 14 : 10,
      runSpacing: 12,
      alignment: mobile ? WrapAlignment.start : WrapAlignment.end,
      children: [
        for (final color in AppThemeColor.values)
          Semantics(
            button: true,
            selected: value == color,
            label: _colorLabel(context, color),
            child: GestureDetector(
              key: ValueKey('theme-color-${color.key}'),
              behavior: HitTestBehavior.opaque,
              onTap: () => onChanged(color),
              child: SizedBox.square(
                dimension: swatchSize,
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    border: Border.all(
                      color: value == color
                          ? AppPalette.of(context).textPrimary
                          : AppPalette.of(context).border.withValues(alpha: 0),
                      width: 2,
                    ),
                  ),
                  child: Padding(
                    padding: const EdgeInsets.all(4),
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        color: dark ? color.dark : color.light,
                        shape: BoxShape.circle,
                        border: Border.all(color: AppPalette.of(context).border),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
      ],
    );
  }
}

String _colorLabel(BuildContext context, AppThemeColor color) => switch (color) {
  AppThemeColor.green => context.l10n.colorGreen,
  AppThemeColor.red => context.l10n.colorRed,
  AppThemeColor.rose => context.l10n.colorRose,
  AppThemeColor.orange => context.l10n.colorOrange,
  AppThemeColor.cyan => context.l10n.colorCyan,
  AppThemeColor.blue => context.l10n.colorBlue,
  AppThemeColor.amber => context.l10n.colorAmber,
  AppThemeColor.purple => context.l10n.colorPurple,
};

class _ThemeModeOption extends StatefulWidget {
  const _ThemeModeOption({
    required this.mode,
    required this.label,
    required this.icon,
    required this.width,
    required this.selected,
    required this.accent,
    required this.onPressed,
  });

  final AppThemeMode mode;
  final String label;
  final IconData icon;
  final double width;
  final bool selected;
  final AppThemeColor accent;
  final ValueChanged<AppThemeMode> onPressed;

  @override
  State<_ThemeModeOption> createState() => _ThemeModeOptionState();
}

class _ThemeModeOptionState extends State<_ThemeModeOption> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Semantics(
      button: true,
      selected: widget.selected,
      label: widget.label,
      child: MouseRegion(
        onEnter: (_) => setState(() => _hovered = true),
        onExit: (_) => setState(() => _hovered = false),
        cursor: SystemMouseCursors.click,
        child: GestureDetector(
          key: ValueKey('theme-mode-${widget.mode.key}'),
          behavior: HitTestBehavior.opaque,
          onTap: () => widget.onPressed(widget.mode),
          child: SizedBox(
            width: widget.width,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                AnimatedContainer(
                  duration: const Duration(milliseconds: 140),
                  height: 52,
                  padding: const EdgeInsets.all(4),
                  decoration: BoxDecoration(
                    color: _hovered ? palette.cardHoverBg : palette.cardBg,
                    borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius + 2),
                    border: Border.all(
                      color: widget.selected ? palette.brand : palette.border,
                      width: widget.selected ? 2 : 1,
                    ),
                  ),
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
                    child: _ThemePreview(mode: widget.mode, accent: widget.accent),
                  ),
                ),
                const SizedBox(height: 7),
                Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(widget.icon, size: 14, color: palette.textSecondary),
                    const SizedBox(width: 5),
                    Flexible(
                      child: Text(
                        widget.label,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          color: widget.selected ? palette.textPrimary : palette.textSecondary,
                          fontSize: 12,
                          fontWeight: widget.selected ? FontWeight.w700 : FontWeight.w600,
                        ),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _ThemePreview extends StatelessWidget {
  const _ThemePreview({required this.mode, required this.accent});

  final AppThemeMode mode;
  final AppThemeColor accent;

  @override
  Widget build(BuildContext context) {
    return switch (mode) {
      AppThemeMode.light => _PreviewSurface(
        background: AppThemePreviewColors.lightBackground,
        side: AppThemePreviewColors.lightSide,
        line: AppThemePreviewColors.lightLine,
        accent: accent.light,
      ),
      AppThemeMode.dark => _PreviewSurface(
        background: AppThemePreviewColors.darkBackground,
        side: AppThemePreviewColors.darkSide,
        line: AppThemePreviewColors.darkLine,
        accent: accent.dark,
      ),
      AppThemeMode.system => Stack(
        fit: StackFit.expand,
        children: [
          const ColoredBox(color: AppThemePreviewColors.lightBackground),
          Align(
            alignment: Alignment.centerRight,
            child: FractionallySizedBox(
              widthFactor: 0.5,
              heightFactor: 1,
              child: const ColoredBox(color: AppThemePreviewColors.darkBackground),
            ),
          ),
          _PreviewSurface(
            background: AppThemePreviewColors.transparent,
            side: AppThemePreviewColors.systemSide,
            line: AppThemePreviewColors.systemLine,
            accent: accent.light,
          ),
        ],
      ),
    };
  }
}

class _PreviewSurface extends StatelessWidget {
  const _PreviewSurface({required this.background, required this.side, required this.line, required this.accent});

  final Color background;
  final Color side;
  final Color line;
  final Color accent;

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: background,
      child: Row(
        children: [
          SizedBox(width: 14, child: ColoredBox(color: side)),
          Expanded(
            child: Padding(
              padding: const EdgeInsets.fromLTRB(8, 8, 6, 6),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    height: 3,
                    width: 32,
                    decoration: BoxDecoration(color: line, borderRadius: BorderRadius.circular(2)),
                  ),
                  const SizedBox(height: 5),
                  Container(
                    height: 3,
                    width: 22,
                    decoration: BoxDecoration(
                      color: line.withValues(alpha: 0.65),
                      borderRadius: BorderRadius.circular(2),
                    ),
                  ),
                  const Spacer(),
                  Container(
                    height: 3,
                    decoration: BoxDecoration(color: accent, borderRadius: BorderRadius.circular(2)),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
