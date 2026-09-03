import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../domain/task_record.dart';
import '../../../../l10n/l10n.dart';
import '../../../../util/util.dart';

class TaskContextActions {
  const TaskContextActions({
    required this.selected,
    required this.allSelected,
    required this.listeningForUpdate,
    required this.onShowDetails,
    required this.onOpenFile,
    required this.onOpenDirectory,
    required this.onBrowseFiles,
    required this.onToggleSelectAll,
    required this.onToggleSelected,
    required this.onPause,
    required this.onResume,
    required this.onDelete,
    required this.onUpdateUrl,
    required this.onToggleUpdateListener,
  });

  final bool selected;
  final bool allSelected;
  final bool listeningForUpdate;
  final VoidCallback onShowDetails;
  final VoidCallback onOpenFile;
  final VoidCallback onOpenDirectory;
  final VoidCallback onBrowseFiles;
  final VoidCallback onToggleSelectAll;
  final VoidCallback onToggleSelected;
  final VoidCallback onPause;
  final VoidCallback onResume;
  final VoidCallback onDelete;
  final VoidCallback onUpdateUrl;
  final VoidCallback onToggleUpdateListener;
}

class TaskContextMenu extends StatelessWidget {
  const TaskContextMenu({super.key, required this.task, required this.actions, required this.child});

  final TaskRecord task;
  final TaskContextActions actions;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return shad.ContextMenu(
      items: [
        shad.MenuButton(
          key: const ValueKey('task-context-select-all'),
          leading: Icon(actions.allSelected ? Icons.deselect : Icons.select_all, size: 16),
          onPressed: (_) => actions.onToggleSelectAll(),
          child: Text(actions.allSelected ? context.l10n.clearSelection : context.l10n.selectAll),
        ),
        shad.MenuButton(
          key: const ValueKey('task-context-select'),
          leading: Icon(actions.selected ? Icons.check_box : Icons.check_box_outline_blank, size: 16),
          onPressed: (_) => actions.onToggleSelected(),
          child: Text(actions.selected ? context.l10n.deselectTask : context.l10n.selectTask),
        ),
        const shad.MenuDivider(),
        shad.MenuButton(
          key: const ValueKey('task-context-resume'),
          leading: const Icon(Icons.play_arrow, size: 16),
          enabled: task.status == TaskStatus.paused || task.status == TaskStatus.failed,
          onPressed: (_) => actions.onResume(),
          child: Text(context.l10n.continueAction),
        ),
        shad.MenuButton(
          key: const ValueKey('task-context-pause'),
          leading: const Icon(Icons.pause, size: 16),
          enabled: task.status == TaskStatus.downloading,
          onPressed: (_) => actions.onPause(),
          child: Text(context.l10n.pause),
        ),
        shad.MenuButton(
          key: const ValueKey('task-context-delete'),
          leading: const Icon(Icons.delete_outline, size: 16),
          onPressed: (_) => actions.onDelete(),
          child: Text(context.l10n.delete),
        ),
        const shad.MenuDivider(),
        shad.MenuButton(
          key: const ValueKey('task-context-update-url'),
          leading: const Icon(Icons.link, size: 16),
          enabled: task.canUpdateUrl,
          subMenu: [
            shad.MenuButton(
              leading: const Icon(Icons.edit_note, size: 16),
              onPressed: (_) => actions.onUpdateUrl(),
              child: Text(context.l10n.updateManually),
            ),
            shad.MenuButton(
              leading: Icon(actions.listeningForUpdate ? Icons.hearing_disabled : Icons.hearing, size: 16),
              onPressed: (_) => actions.onToggleUpdateListener(),
              child: Text(actions.listeningForUpdate ? context.l10n.stopListening : context.l10n.listenForUrl),
            ),
          ],
          child: Text(context.l10n.updateUrl),
        ),
        const shad.MenuDivider(),
        shad.MenuButton(
          key: const ValueKey('task-context-details'),
          leading: const Icon(Icons.info_outline, size: 16),
          onPressed: (_) => actions.onShowDetails(),
          child: Text(context.l10n.taskDetailsInfoTab),
        ),
        if (shouldShowTaskOpenFileAction(task))
          shad.MenuButton(
            key: const ValueKey('task-context-open-file'),
            leading: const Icon(Icons.open_in_new, size: 16),
            onPressed: (_) => actions.onOpenFile(),
            child: Text(context.l10n.openFile),
          ),
        if (shouldShowTaskOpenDirectoryAction())
          shad.MenuButton(
            key: const ValueKey('task-context-open-directory'),
            leading: const Icon(Icons.folder_open_outlined, size: 16),
            onPressed: (_) => actions.onOpenDirectory(),
            child: Text(context.l10n.openDirectory),
          ),
        shad.MenuButton(
          key: const ValueKey('task-context-browse-files'),
          leading: const Icon(Icons.snippet_folder_outlined, size: 16),
          onPressed: (_) => actions.onBrowseFiles(),
          child: Text(context.l10n.browseFiles),
        ),
      ],
      child: child,
    );
  }
}

@visibleForTesting
bool shouldShowTaskOpenFileAction(TaskRecord task, {bool? web}) {
  return !(web ?? kIsWeb) && !task.isFolder && task.status == TaskStatus.completed;
}

@visibleForTesting
bool shouldShowTaskOpenDirectoryAction({bool? web, bool? desktop}) {
  return !(web ?? kIsWeb) && (desktop ?? Util.isDesktop());
}
