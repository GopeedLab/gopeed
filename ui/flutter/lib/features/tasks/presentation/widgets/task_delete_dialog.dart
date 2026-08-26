import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' hide Column, Expanded, Row;

import '../../../../shared/theme/app_palette.dart';
import '../../../../l10n/l10n.dart';

Future<bool?> showTaskDeleteDialog(BuildContext context, {required int taskCount, required bool keepFiles}) async {
  var currentKeepFiles = keepFiles;
  final dialog = const DialogOverlayHandler().show<bool>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (dialogContext) {
      return StatefulBuilder(
        builder: (dialogContext, setDialogState) {
          final palette = AppPalette.of(dialogContext);
          final contentWidth = (MediaQuery.sizeOf(dialogContext).width - 64).clamp(240.0, 360.0);
          return AlertDialog(
            title: Text(dialogContext.l10n.deleteTasksTitle(taskCount)),
            content: SizedBox(
              width: contentWidth,
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(dialogContext.l10n.deleteTaskTip, style: TextStyle(color: palette.textPrimary)),
                        const SizedBox(height: 4),
                        Text(
                          dialogContext.l10n.keepFilesDescription,
                          style: TextStyle(color: palette.textSecondary, fontSize: 12),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 16),
                  Switch(value: currentKeepFiles, onChanged: (value) => setDialogState(() => currentKeepFiles = value)),
                ],
              ),
            ),
            actions: [
              SecondaryButton(onPressed: () => closeOverlay(dialogContext), child: Text(dialogContext.l10n.cancel)),
              DestructiveButton(
                onPressed: () => closeOverlay(dialogContext, currentKeepFiles),
                child: Text(dialogContext.l10n.delete),
              ),
            ],
          );
        },
      );
    },
  );
  return dialog.future;
}
