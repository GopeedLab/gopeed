import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../l10n/l10n.dart';
import '../theme/app_design_tokens.dart';
import '../theme/app_palette.dart';

/// Explains supported date variables and inserts one at the path's selection.
class AppPathPlaceholderButton extends StatelessWidget {
  const AppPathPlaceholderButton({super.key, required this.controller});

  final TextEditingController controller;

  void _showPlaceholders(BuildContext context) {
    final selection = controller.selection;
    final now = DateTime.now();
    final year = now.year.toString();
    final month = now.month.toString().padLeft(2, '0');
    final day = now.day.toString().padLeft(2, '0');
    final l10n = context.l10n;
    final placeholders = [
      ('%year%', l10n.placeholderYear, year),
      ('%month%', l10n.placeholderMonth, month),
      ('%day%', l10n.placeholderDay, day),
      ('%date%', l10n.placeholderDate, '$year-$month-$day'),
    ];
    const PopoverOverlayHandler().show<void>(
      context: context,
      alignment: Alignment.topLeft,
      anchorAlignment: Alignment.bottomLeft,
      offset: const Offset(0, AppDesignTokens.space4),
      builder: (popoverContext) => Card(
        padding: const EdgeInsets.all(AppDesignTokens.space8),
        child: SizedBox(
          width: (MediaQuery.sizeOf(context).width - 80).clamp(180.0, 320.0),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 280),
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  for (final (variable, description, example) in placeholders)
                    GhostButton(
                      key: ValueKey('path-placeholder-$variable'),
                      onPressed: () {
                        final text = controller.text;
                        final valid = selection.isValid && selection.end <= text.length;
                        final start = valid ? selection.start : text.length;
                        final end = valid ? selection.end : text.length;
                        controller.value = TextEditingValue(
                          text: text.replaceRange(start, end, variable),
                          selection: TextSelection.collapsed(offset: start + variable.length),
                        );
                        closeOverlay(popoverContext);
                      },
                      child: SizedBox(
                        width: double.infinity,
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('$variable — $description', style: const TextStyle(fontSize: 12)),
                            Text(
                              l10n.example.replaceAll('@value', example),
                              style: TextStyle(color: AppPalette.of(context).textMuted, fontSize: 12),
                            ),
                          ],
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

  @override
  Widget build(BuildContext context) => Builder(
    builder: (anchorContext) =>
        GhostButton(onPressed: () => _showPlaceholders(anchorContext), child: Text(context.l10n.insertPlaceholder)),
  );
}
