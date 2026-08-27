import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' hide Column, Expanded, Row;

import '../../../../l10n/l10n.dart';
import '../../../../shared/widgets/app_primary_button.dart';

enum PendingUpdateDecision { updateTask, createTask, cancel }

Future<PendingUpdateDecision?> showPendingUpdateDialog(BuildContext context, {required String taskName}) {
  final dialog = const DialogOverlayHandler().show<PendingUpdateDecision>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (dialogContext) {
      final contentWidth = (MediaQuery.sizeOf(dialogContext).width - 64).clamp(240.0, 420.0);
      return AlertDialog(
        title: Text(dialogContext.l10n.pendingUpdateFound),
        content: SizedBox(
          width: contentWidth,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(dialogContext.l10n.pendingUpdateConfirm(taskName)),
              const SizedBox(height: 20),
              Align(
                alignment: Alignment.centerRight,
                child: Wrap(
                  alignment: WrapAlignment.end,
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    SecondaryButton(
                      onPressed: () => closeOverlay(dialogContext, PendingUpdateDecision.cancel),
                      child: Text(dialogContext.l10n.cancel),
                    ),
                    SecondaryButton(
                      onPressed: () => closeOverlay(dialogContext, PendingUpdateDecision.createTask),
                      child: Text(dialogContext.l10n.pendingUpdateNo),
                    ),
                    AppPrimaryButton(
                      onPressed: () => closeOverlay(dialogContext, PendingUpdateDecision.updateTask),
                      child: Text(dialogContext.l10n.pendingUpdateYes),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      );
    },
  );
  return dialog.future;
}
