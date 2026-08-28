import 'dart:async';

import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_loading_button.dart';
import '../../../../util/package_info.dart';
import '../../../../util/updater.dart';
import '../../../../util/util.dart';
import '../../../../l10n/l10n.dart';

typedef StartAppUpdate = Future<void> Function(VersionInfo versionInfo, UpdateProgressCallback onProgress);

Future<void> showAppUpdateDialog(
  BuildContext context, {
  required VersionInfo versionInfo,
  required StartAppUpdate onUpdate,
}) async {
  final overlay = const shad.DialogOverlayHandler().show<void>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (dialogContext) {
      var updating = false;
      var received = 0;
      var total = 0;
      String? errorMessage;

      return StatefulBuilder(
        builder: (dialogContext, setDialogState) {
          final palette = AppPalette.of(dialogContext);
          final availableWidth = MediaQuery.sizeOf(dialogContext).width - 104;
          final dialogWidth = availableWidth.clamp(260.0, 380.0);
          final releaseNotes = localizedReleaseNotes(
            versionInfo.changeLog,
            Localizations.localeOf(dialogContext).languageCode,
          );
          final progress = total > 0 ? (received / total).clamp(0.0, 1.0) : null;

          Future<void> update() async {
            setDialogState(() {
              updating = true;
              errorMessage = null;
            });
            try {
              await onUpdate(versionInfo, (nextReceived, nextTotal) {
                if (!dialogContext.mounted) return;
                setDialogState(() {
                  received = nextReceived;
                  total = nextTotal;
                });
              });
            } catch (error) {
              if (!dialogContext.mounted) return;
              setDialogState(() {
                updating = false;
                errorMessage = dialogContext.l10n.updateFailed;
              });
            }
          }

          return shad.AlertDialog(
            padding: const EdgeInsets.all(20),
            title: SizedBox(
              width: dialogWidth,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  Text(
                    dialogContext.l10n.newVersionAvailable,
                    key: const ValueKey('app-update-heading'),
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 3),
                  Text(
                    'v${packageInfo.version}  →  v${versionInfo.version}',
                    textAlign: TextAlign.center,
                    style: TextStyle(color: palette.textSecondary, fontSize: 12, fontWeight: FontWeight.w400),
                  ),
                ],
              ),
            ),
            content: SizedBox(
              key: const ValueKey('app-update-dialog'),
              width: dialogWidth,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    dialogContext.l10n.releaseNotes,
                    style: TextStyle(color: palette.textPrimary, fontSize: 12, fontWeight: FontWeight.w700),
                  ),
                  const SizedBox(height: 10),
                  ConstrainedBox(
                    constraints: const BoxConstraints(maxHeight: 210),
                    child: SingleChildScrollView(
                      padding: const EdgeInsets.symmetric(horizontal: 4),
                      child: _ReleaseNotes(
                        notes: releaseNotes.isEmpty ? dialogContext.l10n.noReleaseNotes : releaseNotes,
                      ),
                    ),
                  ),
                  if (updating) ...[
                    const SizedBox(height: 16),
                    shad.LinearProgressIndicator(
                      value: progress,
                      minHeight: 5,
                      borderRadius: BorderRadius.circular(999),
                      color: palette.brand,
                      backgroundColor: palette.progressTrack,
                    ),
                    const SizedBox(height: 7),
                    Row(
                      children: [
                        Text(
                          progress == null
                              ? dialogContext.l10n.preparingDownload
                              : '${(progress * 100).toStringAsFixed(0)}%',
                          style: TextStyle(color: palette.textSecondary, fontSize: 11),
                        ),
                        const Spacer(),
                        if (total > 0)
                          Text(
                            '${Util.fmtByte(received)} / ${Util.fmtByte(total)}',
                            style: TextStyle(color: palette.textSecondary, fontSize: 11),
                          ),
                      ],
                    ),
                  ],
                  if (errorMessage != null) ...[
                    const SizedBox(height: 12),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(Icons.error_outline, size: 15, color: palette.error),
                        const SizedBox(width: 6),
                        Text(errorMessage!, style: TextStyle(color: palette.error, fontSize: 12)),
                      ],
                    ),
                  ],
                  const SizedBox(height: 18),
                  Wrap(
                    key: const ValueKey('app-update-actions'),
                    alignment: WrapAlignment.center,
                    spacing: 8,
                    runSpacing: 8,
                    children: [
                      shad.SecondaryButton(
                        onPressed: updating ? null : () => unawaited(shad.closeOverlay(dialogContext)),
                        child: Text(dialogContext.l10n.newVersionLater),
                      ),
                      AppLoadingButton(
                        onPressed: () => unawaited(update()),
                        loading: updating,
                        variant: AppLoadingButtonVariant.primary,
                        icon: const Icon(Icons.system_update_alt_outlined, size: 17),
                        child: Text(dialogContext.l10n.newVersionUpdate),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          );
        },
      );
    },
  );
  await overlay.future;
}

class _ReleaseNotes extends StatelessWidget {
  const _ReleaseNotes({required this.notes});

  final String notes;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final lines = notes
        .split('\n')
        .map((line) => line.trimRight())
        .where((line) => line.trim().isNotEmpty)
        .where((line) => !RegExp(r'^#\s+(Release notes|更新日志)\s*$', caseSensitive: false).hasMatch(line.trim()));
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final line in lines) ...[
          if (line.startsWith('#'))
            Text(
              _plainMarkdown(line.replaceFirst(RegExp(r'^#+\s*'), '')),
              style: TextStyle(color: palette.textPrimary, fontSize: 13, fontWeight: FontWeight.w700),
            )
          else if (line.trimLeft().startsWith('- '))
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('•  ', style: TextStyle(color: palette.textSecondary, fontSize: 12, height: 1.45)),
                Expanded(
                  child: Text(
                    _plainMarkdown(line.trimLeft().substring(2).replaceFirst(RegExp(r'\s+@[\w-]+\s+\(#\d+\)\s*$'), '')),
                    style: TextStyle(color: palette.textSecondary, fontSize: 12, height: 1.45),
                  ),
                ),
              ],
            )
          else
            Text(_plainMarkdown(line), style: TextStyle(color: palette.textSecondary, fontSize: 12, height: 1.45)),
          const SizedBox(height: 7),
        ],
      ],
    );
  }
}

String _plainMarkdown(String value) {
  return value
      .replaceAllMapped(RegExp(r'\[([^\]]+)\]\([^\)]+\)'), (match) => match.group(1) ?? '')
      .replaceAll(RegExp(r'\*\*|__|`'), '');
}
