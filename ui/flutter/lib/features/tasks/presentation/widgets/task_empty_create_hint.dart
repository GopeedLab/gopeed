import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_tooltip.dart';

/// Keeps the action inside the localized sentence, including RTL word order.
class TaskEmptyCreateHint extends StatelessWidget {
  const TaskEmptyCreateHint({super.key, required this.onCreateTask});

  final VoidCallback onCreateTask;

  @override
  Widget build(BuildContext context) {
    const actionMarker = '\uFFFC';
    final parts = context.l10n.emptyTaskCreateHint(actionMarker).split(actionMarker);
    return Text.rich(
      TextSpan(
        children: [
          TextSpan(text: parts.first.trimRight()),
          WidgetSpan(
            alignment: PlaceholderAlignment.middle,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8),
              child: AppTooltip(
                message: context.l10n.create,
                child: Semantics(
                  label: context.l10n.create,
                  child: SizedBox.square(
                    dimension: 30,
                    child: IconButton.outline(
                      key: const ValueKey('tasks-empty-create-button'),
                      density: ButtonDensity.compact,
                      onPressed: onCreateTask,
                      icon: const Icon(Icons.add, size: 16),
                    ),
                  ),
                ),
              ),
            ),
          ),
          TextSpan(text: parts.last.trimLeft()),
        ],
      ),
      textAlign: TextAlign.center,
      style: TextStyle(color: AppPalette.of(context).textMuted, fontSize: 13),
    );
  }
}
