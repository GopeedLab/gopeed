import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' hide Column, Expanded, Row;

import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_http_headers_editor.dart';
import '../../../../shared/widgets/app_primary_button.dart';
import '../../../../l10n/l10n.dart';
import '../../domain/task_record.dart';

class TaskUrlUpdate {
  const TaskUrlUpdate({required this.url, required this.headers});

  final String url;
  final Map<String, String> headers;
}

Future<TaskUrlUpdate?> showTaskUpdateUrlDialog(BuildContext context, TaskRecord task) async {
  final urlController = TextEditingController(text: task.url);
  final headersController = AppHttpHeadersController(headers: task.requestHeaders);

  final dialog = const DialogOverlayHandler().show<TaskUrlUpdate?>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (dialogContext) {
      String? errorText;
      return StatefulBuilder(
        builder: (dialogContext, setDialogState) {
          final palette = AppPalette.of(dialogContext);
          void submit() {
            final url = urlController.text.trim();
            if (url.isEmpty) {
              setDialogState(() => errorText = dialogContext.l10n.downloadLinkValid);
              return;
            }
            closeOverlay(dialogContext, TaskUrlUpdate(url: url, headers: headersController.toMap()));
          }

          return AlertDialog(
            title: Text(dialogContext.l10n.updateUrl),
            content: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 680, maxHeight: 480),
              child: SingleChildScrollView(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(dialogContext.l10n.downloadLink, style: TextStyle(color: palette.textPrimary, fontSize: 12)),
                    const SizedBox(height: 6),
                    TextField(controller: urlController),
                    const SizedBox(height: 18),
                    AppHttpHeadersEditor(
                      controller: headersController,
                      label: dialogContext.l10n.httpHeader,
                      keyPrefix: 'update-task-http-header',
                    ),
                    if (errorText != null) ...[
                      const SizedBox(height: 10),
                      Text(errorText!, style: TextStyle(color: palette.error, fontSize: 12)),
                    ],
                  ],
                ),
              ),
            ),
            actions: [
              SecondaryButton(onPressed: () => closeOverlay(dialogContext), child: Text(dialogContext.l10n.cancel)),
              AppPrimaryButton(onPressed: submit, child: Text(dialogContext.l10n.updateAndResume)),
            ],
          );
        },
      );
    },
  );

  try {
    return await dialog.future;
  } finally {
    urlController.dispose();
    headersController.dispose();
  }
}
