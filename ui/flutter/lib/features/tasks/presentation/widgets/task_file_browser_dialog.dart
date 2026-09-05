import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:go_router/go_router.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../l10n/l10n.dart';
import '../../../../shared/widgets/app_tooltip.dart';
import '../../../../util/util.dart';
import '../../domain/task_record.dart';
import 'task_file_manager.dart';

Future<void> browseTaskFiles(BuildContext context, TaskRecord task, {required bool mobile}) async {
  if (mobile && !Util.isWeb()) {
    await context.push('/tasks/${Uri.encodeComponent(task.id)}/files', extra: task);
    return;
  }
  await showTaskFileBrowserDialog(context, task);
}

Future<void> showTaskFileBrowserDialog(BuildContext context, TaskRecord task) async {
  final overlay = const shad.DialogOverlayHandler().show<void>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (dialogContext) {
      final viewport = MediaQuery.sizeOf(dialogContext);
      final dialogWidth = (viewport.width - 64).clamp(240.0, 820.0);
      final dialogHeight = (viewport.height - 160).clamp(180.0, 620.0);

      return shad.AlertDialog(
        key: const ValueKey('task-file-browser-dialog'),
        padding: const EdgeInsets.all(20),
        title: SizedBox(
          width: dialogWidth,
          child: Row(
            children: [
              Expanded(child: Text(dialogContext.l10n.browseFiles, maxLines: 1, overflow: TextOverflow.ellipsis)),
              AppTooltip(
                message: dialogContext.l10n.close,
                child: shad.IconButton.ghost(
                  size: shad.ButtonSize.xSmall,
                  onPressed: () => shad.closeOverlay(dialogContext),
                  icon: const Icon(Icons.close, size: 18),
                ),
              ),
            ],
          ),
        ),
        content: SizedBox(
          width: dialogWidth,
          height: dialogHeight,
          child: TaskFileManagerView(task: task),
        ),
      );
    },
  );
  await overlay.future;
}
