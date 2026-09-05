import 'package:flutter/widgets.dart';

import '../../../../core/utils/breakpoints.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';

class SettingsItem extends StatelessWidget {
  const SettingsItem({super.key, required this.title, required this.child, this.subtitle});

  final String title;
  final String? subtitle;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final desktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    if (desktop) {
      return LayoutBuilder(
        builder: (context, constraints) {
          const leadingSpace = 2.0;
          const columnGap = AppDesignTokens.space24;
          final maxControlWidth =
              (constraints.maxWidth - leadingSpace - columnGap - AppDesignTokens.settingsItemMinLabelWidth).clamp(
                0.0,
                double.infinity,
              );
          return Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const SizedBox(width: leadingSpace),
              Expanded(
                child: _SettingsItemLabel(title: title, subtitle: subtitle),
              ),
              const SizedBox(width: columnGap),
              ConstrainedBox(
                constraints: BoxConstraints(maxWidth: maxControlWidth),
                child: child,
              ),
            ],
          );
        },
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SettingsItemLabel(title: title, subtitle: subtitle),
        const SizedBox(height: 10),
        Align(alignment: Alignment.centerLeft, child: child),
      ],
    );
  }
}

class _SettingsItemLabel extends StatelessWidget {
  const _SettingsItemLabel({required this.title, this.subtitle});

  final String title;
  final String? subtitle;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final subtitleText = subtitle?.trim();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: TextStyle(color: palette.textPrimary, fontSize: 13, fontWeight: FontWeight.w700),
        ),
        if (subtitleText != null && subtitleText.isNotEmpty) ...[
          const SizedBox(height: 4),
          Text(subtitleText, style: TextStyle(color: palette.textSecondary, fontSize: 12, height: 1.3)),
        ],
      ],
    );
  }
}
