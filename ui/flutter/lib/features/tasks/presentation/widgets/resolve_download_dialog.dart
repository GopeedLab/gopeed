import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter/widgets.dart' as flutter show Flexible;

import '../../../../core/capabilities/gopeed_capability.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../application/pending_download.dart';
import 'create_task_action_buttons.dart';
import 'resolve_file_tree.dart';

Future<bool> showResolveDownloadsDialog(
  BuildContext context, {
  required List<PendingDownload> sources,
  required GopeedService service,
  required bool directoryManuallySet,
}) async {
  var submitting = false;
  String? errorText;
  final dialog = const DialogOverlayHandler().show<bool>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (dialogContext) => StatefulBuilder(
      builder: (dialogContext, setDialogState) {
        final palette = AppPalette.of(dialogContext);
        final screen = MediaQuery.sizeOf(dialogContext);
        Future<void> submit() async {
          if (submitting) return;
          if (!sources.any((source) => source.result != null && source.hasSelection)) {
            if (sources.any((source) => source.created || source.createdFiles.isNotEmpty) &&
                !sources.any((source) => source.included && source.resolveError != null)) {
              await closeOverlay(dialogContext, true);
              return;
            }
            setDialogState(() => errorText = dialogContext.l10n.selectAtLeastOneFile);
            return;
          }
          setDialogState(() {
            submitting = true;
            errorText = null;
          });
          for (final source in sources) {
            await source.submit(service);
          }
          if (!dialogContext.mounted) return;
          if (sources.every(
            (source) => !source.included || (source.resolveError == null && source.createError == null),
          )) {
            await closeOverlay(dialogContext, true);
          } else {
            setDialogState(() => submitting = false);
          }
        }

        return AlertDialog(
          padding: const EdgeInsets.all(18),
          title: Text(
            sources.length == 1 && sources.single.result?.res.name.trim().isNotEmpty == true
                ? sources.single.result!.res.name
                : dialogContext.l10n.resolvedFiles,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          content: SizedBox(
            width: screen.width < 760 ? screen.width - 68 : 720,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  dialogContext.l10n.resolvedFileCount(sources.fold(0, (count, source) => count + source.files.length)),
                  style: TextStyle(color: palette.textSecondary, fontSize: 12),
                ),
                if (sources.length == 1) ...[
                  const SizedBox(height: 4),
                  Text(
                    dialogContext.l10n.resolvedCategoryDirectory(
                      directoryManuallySet
                          ? sources.single.options.path
                          : sources.single.result?.suggestedDownloadDir.isNotEmpty == true
                          ? sources.single.result!.suggestedDownloadDir
                          : sources.single.options.path,
                    ),
                    key: const ValueKey('resolve-category-directory'),
                    style: TextStyle(color: palette.textSecondary, fontSize: 12),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
                const SizedBox(height: 12),
                SizedBox(
                  height: (screen.height * 0.48).clamp(120.0, 440.0),
                  child: IgnorePointer(
                    ignoring: submitting,
                    child: ResolveFileTree(
                      sources: List.of(sources),
                      onSelectionChanged: (selection) {
                        for (final source in sources) {
                          if (!source.included || source.result == null || source.created) continue;
                          source.selected
                            ..clear()
                            ..addAll(selection.where((key) => key.$1 == source.id).map((key) => key.$2));
                        }
                      },
                    ),
                  ),
                ),
                if (sources.any(
                  (source) => !source.included || source.resolveError != null || source.createError != null,
                ))
                  ConstrainedBox(
                    constraints: BoxConstraints(maxHeight: screen.height * 0.15),
                    child: SingleChildScrollView(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          for (final source in sources)
                            if (!source.included || source.resolveError != null || source.createError != null)
                              Row(
                                children: [
                                  Checkbox(
                                    state: source.included ? CheckboxState.checked : CheckboxState.unchecked,
                                    onChanged: submitting
                                        ? null
                                        : (_) => setDialogState(() => source.included = !source.included),
                                  ),
                                  const SizedBox(width: 8),
                                  Expanded(
                                    child: Text(
                                      source.resolveError != null || source.createError != null
                                          ? '${source.name}: ${source.resolveError ?? source.createError}'
                                          : source.name,
                                      style: TextStyle(color: palette.error, fontSize: 12),
                                    ),
                                  ),
                                  if (source.resolveError != null)
                                    GhostButton(
                                      onPressed: submitting
                                          ? null
                                          : () async {
                                              setDialogState(() => submitting = true);
                                              await source.resolve(service);
                                              if (dialogContext.mounted) setDialogState(() => submitting = false);
                                            },
                                      child: Text(dialogContext.l10n.retry),
                                    ),
                                ],
                              ),
                        ],
                      ),
                    ),
                  ),
                if (errorText != null) Text(errorText!, style: TextStyle(color: palette.error, fontSize: 12)),
              ],
            ),
          ),
          actions: [
            flutter.Flexible(
              child: CreateTaskActionButtons(
                submitting: submitting,
                onCancel: () => closeOverlay(dialogContext, false),
                onSubmit: submit,
                cancelLabel: dialogContext.l10n.cancel,
                submitLabel: dialogContext.l10n.createAction,
                cancelButtonKey: const ValueKey('resolve-cancel-button'),
                submitButtonKey: const ValueKey('resolve-create-button'),
              ),
            ),
          ],
        );
      },
    ),
  );
  return await dialog.future == true;
}
