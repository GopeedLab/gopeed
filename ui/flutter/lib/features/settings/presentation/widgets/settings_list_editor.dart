import 'package:flutter/material.dart' show Divider, Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../api/model/downloader_config.dart';
import '../../../../core/utils/breakpoints.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_primary_button.dart';
import '../../../../l10n/l10n.dart';

class SettingsListEntry {
  const SettingsListEntry({required this.id, required this.title, this.subtitle});

  final String id;
  final String title;
  final String? subtitle;
}

class SettingsListEditor extends StatelessWidget {
  const SettingsListEditor({
    super.key,
    required this.entries,
    required this.onAdd,
    required this.onEdit,
    required this.onDelete,
    this.addLabel,
    this.addButtonKey,
  });

  final List<SettingsListEntry> entries;
  final VoidCallback onAdd;
  final ValueChanged<int> onEdit;
  final ValueChanged<int> onDelete;
  final String? addLabel;
  final Key? addButtonKey;

  @override
  Widget build(BuildContext context) {
    final desktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    final palette = AppPalette.of(context);
    return SizedBox(
      width: desktop ? AppDesignTokens.settingsFormControlWidth : double.infinity,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          for (var index = 0; index < entries.length; index++) ...[
            if (index > 0)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Divider(color: palette.border),
              ),
            Row(
              key: ValueKey('settings-list-entry-${entries[index].id}'),
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        entries[index].title,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(color: palette.textPrimary, fontSize: 13),
                      ),
                      if (entries[index].subtitle case final subtitle?) ...[
                        const SizedBox(height: 3),
                        Text(
                          subtitle,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(color: palette.textSecondary, fontSize: 11),
                        ),
                      ],
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                shad.GhostButton(
                  key: ValueKey('edit-settings-list-${entries[index].id}'),
                  density: shad.ButtonDensity.icon,
                  onPressed: () => onEdit(index),
                  child: const Icon(Icons.edit_outlined, size: 17),
                ),
                shad.GhostButton(
                  key: ValueKey('delete-settings-list-${entries[index].id}'),
                  density: shad.ButtonDensity.icon,
                  onPressed: () => onDelete(index),
                  child: const Icon(Icons.delete_outline, size: 17),
                ),
              ],
            ),
          ],
          if (entries.isNotEmpty) const SizedBox(height: 10),
          Align(
            alignment: Alignment.centerRight,
            child: shad.SecondaryButton(
              key: addButtonKey,
              onPressed: onAdd,
              leading: const Icon(Icons.add, size: 17),
              child: Text(addLabel ?? context.l10n.add),
            ),
          ),
        ],
      ),
    );
  }
}

Future<String?> showTextSettingDialog(
  BuildContext context, {
  required String title,
  required String fieldLabel,
  String initialValue = '',
  bool requireHttpUrl = false,
  Future<String?> Function()? pickPath,
  Future<void> Function(String value)? onTest,
}) async {
  final controller = TextEditingController(text: initialValue);
  String? validationMessage;
  bool testing = false;
  String? testMessage;
  bool? testSucceeded;
  final overlay = const shad.DialogOverlayHandler().show<String?>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (dialogContext) => StatefulBuilder(
      builder: (dialogContext, setDialogState) {
        final palette = AppPalette.of(dialogContext);

        String? validate() {
          final value = controller.text.trim();
          if (value.isEmpty) return dialogContext.l10n.fieldRequired(fieldLabel);
          if (requireHttpUrl && !(value.startsWith('http://') || value.startsWith('https://'))) {
            return dialogContext.l10n.urlInvalid;
          }
          return null;
        }

        Future<void> test() async {
          final error = validate();
          if (error != null) {
            setDialogState(() => validationMessage = error);
            return;
          }
          setDialogState(() {
            testing = true;
            validationMessage = null;
            testMessage = null;
            testSucceeded = null;
          });
          try {
            await onTest!(controller.text.trim());
            if (dialogContext.mounted) {
              setDialogState(() {
                testMessage = dialogContext.l10n.webhookTestSuccess;
                testSucceeded = true;
              });
            }
          } catch (_) {
            if (dialogContext.mounted) {
              setDialogState(() {
                testMessage = dialogContext.l10n.webhookTestFail;
                testSucceeded = false;
              });
            }
          } finally {
            if (dialogContext.mounted) setDialogState(() => testing = false);
          }
        }

        return shad.AlertDialog(
          title: Text(title),
          content: SizedBox(
            width: (MediaQuery.sizeOf(dialogContext).width - 64).clamp(260.0, 420.0),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(fieldLabel, style: TextStyle(color: palette.textSecondary, fontSize: 12)),
                const SizedBox(height: 6),
                shad.TextField(
                  controller: controller,
                  features: pickPath == null
                      ? const []
                      : [
                          shad.InputFeature.trailing(
                            shad.GhostButton(
                              density: shad.ButtonDensity.icon,
                              onPressed: () async {
                                final path = await pickPath();
                                if (path != null && path.isNotEmpty) controller.text = path;
                              },
                              child: const Icon(Icons.folder_open_outlined, size: 17),
                            ),
                          ),
                        ],
                ),
                if (validationMessage != null || testMessage != null) ...[
                  const SizedBox(height: 8),
                  Text(
                    validationMessage ?? testMessage!,
                    style: TextStyle(
                      color: validationMessage != null
                          ? palette.error
                          : testSucceeded == true
                          ? palette.success
                          : palette.error,
                      fontSize: 12,
                    ),
                  ),
                ],
              ],
            ),
          ),
          actions: [
            if (onTest != null)
              shad.SecondaryButton(
                onPressed: testing ? null : () => test(),
                leading: testing
                    ? const SizedBox.square(dimension: 14, child: shad.CircularProgressIndicator())
                    : const Icon(Icons.send_outlined, size: 16),
                child: Text(dialogContext.l10n.webhookTest),
              ),
            shad.SecondaryButton(
              onPressed: () => shad.closeOverlay(dialogContext),
              child: Text(dialogContext.l10n.cancel),
            ),
            AppPrimaryButton(
              onPressed: () {
                final error = validate();
                if (error != null) {
                  setDialogState(() => validationMessage = error);
                  return;
                }
                shad.closeOverlay(dialogContext, controller.text.trim());
              },
              child: Text(dialogContext.l10n.confirm),
            ),
          ],
        );
      },
    ),
  );
  try {
    return await overlay.future;
  } finally {
    controller.dispose();
  }
}

class GithubMirrorDraft {
  const GithubMirrorDraft({required this.type, required this.url});

  final GithubMirrorType type;
  final String url;
}

Future<GithubMirrorDraft?> showGithubMirrorDialog(BuildContext context, {GithubMirror? mirror}) async {
  var selectedType = mirror?.type ?? GithubMirrorType.jsdelivr;
  final controller = TextEditingController(text: mirror?.url ?? '');
  String? validationMessage;
  final overlay = const shad.DialogOverlayHandler().show<GithubMirrorDraft?>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (dialogContext) => StatefulBuilder(
      builder: (dialogContext, setDialogState) {
        final palette = AppPalette.of(dialogContext);
        return shad.AlertDialog(
          title: Text(mirror == null ? dialogContext.l10n.addGithubMirror : dialogContext.l10n.editGithubMirror),
          content: SizedBox(
            width: (MediaQuery.sizeOf(dialogContext).width - 64).clamp(260.0, 420.0),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(dialogContext.l10n.githubMirrorType, style: TextStyle(color: palette.textSecondary, fontSize: 12)),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  children: [
                    for (final type in GithubMirrorType.values)
                      shad.SecondaryButton(
                        onPressed: () => setDialogState(() => selectedType = type),
                        leading: selectedType == type ? const Icon(Icons.check, size: 15) : null,
                        child: Text(type.name),
                      ),
                  ],
                ),
                const SizedBox(height: 14),
                Text(dialogContext.l10n.githubMirrorUrl, style: TextStyle(color: palette.textSecondary, fontSize: 12)),
                const SizedBox(height: 6),
                shad.TextField(controller: controller),
                if (validationMessage != null) ...[
                  const SizedBox(height: 8),
                  Text(validationMessage!, style: TextStyle(color: palette.error, fontSize: 12)),
                ],
              ],
            ),
          ),
          actions: [
            shad.SecondaryButton(
              onPressed: () => shad.closeOverlay(dialogContext),
              child: Text(dialogContext.l10n.cancel),
            ),
            AppPrimaryButton(
              onPressed: () {
                final value = controller.text.trim();
                if (value.isEmpty || !(value.startsWith('http://') || value.startsWith('https://'))) {
                  setDialogState(() => validationMessage = dialogContext.l10n.urlInvalid);
                  return;
                }
                shad.closeOverlay(dialogContext, GithubMirrorDraft(type: selectedType, url: value));
              },
              child: Text(dialogContext.l10n.confirm),
            ),
          ],
        );
      },
    ),
  );
  try {
    return await overlay.future;
  } finally {
    controller.dispose();
  }
}

Future<bool> showDeleteSettingsEntryDialog(BuildContext context, String name) async {
  final overlay = const shad.DialogOverlayHandler().show<bool>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (dialogContext) => shad.AlertDialog(
      title: Text(dialogContext.l10n.confirmDeleteTitle),
      content: Text(dialogContext.l10n.confirmDeleteNamed(name)),
      actions: [
        shad.SecondaryButton(
          onPressed: () => shad.closeOverlay(dialogContext, false),
          child: Text(dialogContext.l10n.cancel),
        ),
        shad.DestructiveButton(
          onPressed: () => shad.closeOverlay(dialogContext, true),
          child: Text(dialogContext.l10n.delete),
        ),
      ],
    ),
  );
  return await overlay.future ?? false;
}
