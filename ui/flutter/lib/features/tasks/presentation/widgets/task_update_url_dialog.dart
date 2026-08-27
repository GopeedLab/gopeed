import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' hide Column, Expanded, Row;

import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_primary_button.dart';
import '../../../../l10n/l10n.dart';
import '../../domain/task_record.dart';

class TaskUrlUpdate {
  const TaskUrlUpdate({required this.url, required this.headers});

  final String url;
  final Map<String, String> headers;
}

class _HeaderControllers {
  _HeaderControllers({String name = '', String value = ''})
    : name = TextEditingController(text: name),
      value = TextEditingController(text: value);

  final TextEditingController name;
  final TextEditingController value;

  void dispose() {
    name.dispose();
    value.dispose();
  }
}

Future<TaskUrlUpdate?> showTaskUpdateUrlDialog(BuildContext context, TaskRecord task) async {
  final urlController = TextEditingController(text: task.url);
  final headerControllers = task.requestHeaders.entries
      .map((entry) => _HeaderControllers(name: entry.key, value: entry.value))
      .toList(growable: true);
  if (headerControllers.isEmpty) {
    headerControllers.add(_HeaderControllers());
  }

  final dialog = const DialogOverlayHandler().show<TaskUrlUpdate>(
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
            final headers = <String, String>{};
            for (final controllers in headerControllers) {
              final name = controllers.name.text.trim();
              if (name.isNotEmpty) {
                headers[name] = controllers.value.text.trim();
              }
            }
            closeOverlay(dialogContext, TaskUrlUpdate(url: url, headers: headers));
          }

          return AlertDialog(
            title: Text(dialogContext.l10n.updateUrl),
            content: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 520, maxHeight: 480),
              child: SingleChildScrollView(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(dialogContext.l10n.downloadLink, style: TextStyle(color: palette.textPrimary, fontSize: 12)),
                    const SizedBox(height: 6),
                    TextField(controller: urlController),
                    const SizedBox(height: 18),
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            dialogContext.l10n.httpHeaders,
                            style: TextStyle(color: palette.textPrimary, fontSize: 12),
                          ),
                        ),
                        Tooltip(
                          tooltip: (_) => Text(dialogContext.l10n.add),
                          child: IconButton.ghost(
                            onPressed: () => setDialogState(() => headerControllers.add(_HeaderControllers())),
                            icon: const Icon(Icons.add, size: 18),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 6),
                    for (var index = 0; index < headerControllers.length; index++) ...[
                      Row(
                        children: [
                          Expanded(
                            child: TextField(
                              controller: headerControllers[index].name,
                              placeholder: Text(dialogContext.l10n.httpHeaderName),
                            ),
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            child: TextField(
                              controller: headerControllers[index].value,
                              placeholder: Text(dialogContext.l10n.httpHeaderValue),
                            ),
                          ),
                          const SizedBox(width: 4),
                          Tooltip(
                            tooltip: (_) => Text(dialogContext.l10n.delete),
                            child: IconButton.ghost(
                              onPressed: headerControllers.length == 1
                                  ? null
                                  : () {
                                      final removed = headerControllers.removeAt(index);
                                      removed.dispose();
                                      setDialogState(() {});
                                    },
                              icon: const Icon(Icons.remove, size: 18),
                            ),
                          ),
                        ],
                      ),
                      if (index != headerControllers.length - 1) const SizedBox(height: 8),
                    ],
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
    for (final controllers in headerControllers) {
      controllers.dispose();
    }
  }
}
