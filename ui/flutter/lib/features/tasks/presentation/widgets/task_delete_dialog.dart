import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' hide Column, Expanded, Row;

import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/theme/app_design_tokens.dart';
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
          final contentWidth = (MediaQuery.sizeOf(dialogContext).width - 128).clamp(180.0, 380.0);
          return AlertDialog(
            leading: Container(
              key: const ValueKey('task-delete-dialog-icon'),
              width: 40,
              height: 40,
              decoration: BoxDecoration(
                color: palette.error.withValues(alpha: 0.08),
                border: Border.all(color: palette.error.withValues(alpha: 0.16)),
                borderRadius: BorderRadius.circular(10),
              ),
              alignment: Alignment.center,
              child: Icon(Icons.delete_outline, size: 20, color: palette.error),
            ),
            title: Text(dialogContext.l10n.deleteTasksTitle(taskCount)),
            content: SizedBox(
              width: contentWidth,
              child: AnimatedContainer(
                key: const ValueKey('task-delete-keep-files-option'),
                duration: const Duration(milliseconds: 160),
                padding: const EdgeInsets.symmetric(
                  horizontal: AppDesignTokens.space16,
                  vertical: AppDesignTokens.space12,
                ),
                decoration: BoxDecoration(
                  color: currentKeepFiles ? palette.brandSoft : palette.surfaceSoft,
                  border: Border.all(color: currentKeepFiles ? palette.brand.withValues(alpha: 0.45) : palette.border),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        dialogContext.l10n.deleteTaskTip,
                        style: TextStyle(color: palette.textPrimary, fontSize: 14, fontWeight: FontWeight.w500),
                      ),
                    ),
                    const SizedBox(width: AppDesignTokens.space16),
                    Switch(
                      key: const ValueKey('task-delete-keep-files-switch'),
                      value: currentKeepFiles,
                      onChanged: (value) => setDialogState(() => currentKeepFiles = value),
                    ),
                  ],
                ),
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
