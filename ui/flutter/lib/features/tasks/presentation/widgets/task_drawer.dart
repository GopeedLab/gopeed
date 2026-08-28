import 'dart:async';

import 'package:flutter/material.dart' show Colors, Icons;
import 'package:flutter/widgets.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_tooltip.dart';
import '../../../../l10n/l10n.dart';
import '../../../../util/util.dart';
import '../../domain/task_record.dart';
import '../task_record_localizations.dart';
import '../../application/task_runtime_status_provider.dart';
import 'task_file_browser_dialog.dart';
import 'task_file_tree.dart';
import 'task_statistics/task_statistics_tab.dart';

class TaskDrawer extends ConsumerStatefulWidget {
  const TaskDrawer({
    super.key,
    required this.task,
    required this.onClose,
    required this.onOpenStorage,
    required this.onUpdateUrl,
  });

  final TaskRecord? task;
  final VoidCallback onClose;
  final VoidCallback onOpenStorage;
  final Future<void> Function(String url) onUpdateUrl;

  @override
  ConsumerState<TaskDrawer> createState() => _TaskDrawerState();
}

class _TaskDrawerState extends ConsumerState<TaskDrawer> {
  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final isOpen = widget.task != null;
    final task = widget.task;
    final drawerWidth = AppDesignTokens.taskDetailsDrawerWidth(MediaQuery.sizeOf(context).width);

    return IgnorePointer(
      ignoring: !isOpen,
      child: Stack(
        children: [
          AnimatedOpacity(
            opacity: isOpen ? 1 : 0,
            duration: const Duration(milliseconds: 180),
            child: GestureDetector(
              onTap: widget.onClose,
              child: ColoredBox(color: Colors.black.withValues(alpha: 0.35), child: const SizedBox.expand()),
            ),
          ),
          Align(
            alignment: Alignment.centerRight,
            child: AnimatedSlide(
              duration: const Duration(milliseconds: 260),
              curve: Curves.easeOutCubic,
              offset: isOpen ? Offset.zero : const Offset(1, 0),
              child: Container(
                key: const ValueKey('task-details-drawer'),
                width: drawerWidth,
                decoration: BoxDecoration(
                  color: palette.bg,
                  border: Border(left: BorderSide(color: palette.border)),
                ),
                child: task == null
                    ? const SizedBox.shrink()
                    : Column(
                        children: [
                          SizedBox(
                            height: AppDesignTokens.contentHeaderHeight,
                            child: Padding(
                              padding: const EdgeInsets.symmetric(horizontal: 24),
                              child: Row(
                                children: [
                                  Expanded(
                                    child: Text(
                                      context.l10n.taskDetails,
                                      style: TextStyle(
                                        color: palette.textPrimary,
                                        fontSize: 16,
                                        fontWeight: FontWeight.w700,
                                      ),
                                    ),
                                  ),
                                  _TaskDetailIconButton(
                                    key: const ValueKey('task-details-close'),
                                    label: context.l10n.close,
                                    icon: const Icon(Icons.close, size: 18),
                                    onPressed: widget.onClose,
                                  ),
                                ],
                              ),
                            ),
                          ),
                          Expanded(
                            child: TaskDetailsView(
                              task: task,
                              mobile: false,
                              onOpenStorage: widget.onOpenStorage,
                              onUpdateUrl: widget.onUpdateUrl,
                            ),
                          ),
                        ],
                      ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class TaskDetailsView extends ConsumerStatefulWidget {
  const TaskDetailsView({
    super.key,
    required this.task,
    required this.mobile,
    required this.onOpenStorage,
    required this.onUpdateUrl,
  });

  final TaskRecord task;
  final bool mobile;
  final VoidCallback onOpenStorage;
  final Future<void> Function(String url) onUpdateUrl;

  @override
  ConsumerState<TaskDetailsView> createState() => _TaskDetailsViewState();
}

class _TaskDetailsViewState extends ConsumerState<TaskDetailsView> {
  final _urlController = TextEditingController();
  int _tabIndex = 0;
  bool _copied = false;
  bool _editingUrl = false;
  bool _updatingUrl = false;

  @override
  void initState() {
    super.initState();
    _urlController.text = widget.task.url;
  }

  @override
  void didUpdateWidget(covariant TaskDetailsView oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.task.id != oldWidget.task.id) {
      _tabIndex = 0;
      _copied = false;
      _editingUrl = false;
      _updatingUrl = false;
      _urlController.text = widget.task.url;
    } else if (!_editingUrl && widget.task.url != oldWidget.task.url) {
      _urlController.text = widget.task.url;
    }
    if (!widget.task.canUpdateUrl && _editingUrl) {
      _editingUrl = false;
      _updatingUrl = false;
      _urlController.text = widget.task.url;
    }
  }

  @override
  void dispose() {
    _urlController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final contentPadding = widget.mobile
        ? AppDesignTokens.taskDetailsMobilePadding
        : AppDesignTokens.taskDetailsDesktopPadding;
    final runtimeStatus = _tabIndex == 1 ? ref.watch(taskRuntimeStatusProvider(widget.task.id)).value : null;
    return Column(
      children: [
        Container(
          height: 48,
          padding: EdgeInsets.symmetric(horizontal: contentPadding),
          decoration: BoxDecoration(
            border: Border(
              top: BorderSide(color: palette.border),
              bottom: BorderSide(color: palette.border),
            ),
          ),
          child: Row(
            children: [
              Expanded(
                child: SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  child: Row(
                    children: [
                      _DrawerTabButton(
                        label: context.l10n.taskDetailsInfoTab,
                        active: _tabIndex == 0,
                        onTap: () => setState(() => _tabIndex = 0),
                      ),
                      const SizedBox(width: 24),
                      _DrawerTabButton(
                        label: context.l10n.taskDetailsFilesTab,
                        active: _tabIndex == 1,
                        onTap: () => setState(() => _tabIndex = 1),
                      ),
                      const SizedBox(width: 24),
                      _DrawerTabButton(
                        label: context.l10n.taskDetailsStatsTab,
                        active: _tabIndex == 2,
                        onTap: () => setState(() => _tabIndex = 2),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
        Expanded(
          child: switch (_tabIndex) {
            0 => SingleChildScrollView(
              key: PageStorageKey<String>('task-info-${widget.task.id}'),
              padding: EdgeInsets.all(contentPadding),
              child: _TaskInfoTab(
                task: widget.task,
                copied: _copied,
                editingUrl: _editingUrl,
                updatingUrl: _updatingUrl,
                urlController: _urlController,
                onOpenStorage: widget.onOpenStorage,
                onEditUrl: widget.task.canUpdateUrl
                    ? () => setState(() {
                        _urlController.text = widget.task.url;
                        _editingUrl = true;
                      })
                    : null,
                onCancelEditUrl: () => setState(() {
                  _urlController.text = widget.task.url;
                  _editingUrl = false;
                }),
                onSaveUrl: () => unawaited(_saveUrl()),
                onCopy: () async {
                  await Clipboard.setData(ClipboardData(text: widget.task.url));
                  if (mounted) setState(() => _copied = true);
                },
              ),
            ),
            1 => Padding(
              padding: EdgeInsets.all(contentPadding),
              child: TaskFileTree(
                task: widget.task,
                runtimeStatus: runtimeStatus,
                progressAction: _TaskDetailIconButton(
                  key: const ValueKey('task-files-browse'),
                  label: context.l10n.browseFiles,
                  icon: const Icon(Icons.snippet_folder_outlined, size: 18),
                  onPressed: _browseFiles,
                ),
              ),
            ),
            _ => TaskStatisticsTab(task: widget.task, mobile: widget.mobile),
          },
        ),
      ],
    );
  }

  void _browseFiles() {
    if (widget.mobile && !Util.isWeb()) {
      context.push('/tasks/${Uri.encodeComponent(widget.task.id)}/files', extra: widget.task);
      return;
    }
    unawaited(showTaskFileBrowserDialog(context, widget.task));
  }

  Future<void> _saveUrl() async {
    final value = _urlController.text.trim();
    if (value.isEmpty || _updatingUrl || !widget.task.canUpdateUrl) return;
    setState(() => _updatingUrl = true);
    try {
      await widget.onUpdateUrl(value);
      if (!mounted) return;
      setState(() {
        _editingUrl = false;
        _updatingUrl = false;
      });
    } catch (_) {
      if (mounted) setState(() => _updatingUrl = false);
    }
  }
}

class _DrawerTabButton extends StatelessWidget {
  const _DrawerTabButton({required this.label, required this.active, required this.onTap});

  final String label;
  final bool active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 12),
        decoration: BoxDecoration(
          border: Border(bottom: BorderSide(color: active ? palette.textPrimary : Colors.transparent, width: 2)),
        ),
        child: Text(
          label,
          style: TextStyle(
            color: active ? palette.textPrimary : palette.textSecondary,
            fontSize: 13,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
    );
  }
}

class _TaskInfoTab extends StatelessWidget {
  const _TaskInfoTab({
    required this.task,
    required this.copied,
    required this.editingUrl,
    required this.updatingUrl,
    required this.urlController,
    required this.onCopy,
    required this.onOpenStorage,
    required this.onEditUrl,
    required this.onCancelEditUrl,
    required this.onSaveUrl,
  });

  final TaskRecord task;
  final bool copied;
  final bool editingUrl;
  final bool updatingUrl;
  final TextEditingController urlController;
  final VoidCallback onCopy;
  final VoidCallback onOpenStorage;
  final VoidCallback? onEditUrl;
  final VoidCallback onCancelEditUrl;
  final VoidCallback onSaveUrl;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _InfoBlock(label: context.l10n.taskName, value: task.name),
        _InfoBlock(
          label: context.l10n.status,
          value: task.localizedStatus(context.l10n),
          error: task.status == TaskStatus.failed,
        ),
        _InfoBlock(
          label: context.l10n.size,
          value: task.status == TaskStatus.completed
              ? task.total ?? task.downloaded
              : task.total != null
              ? '${task.downloaded} / ${task.total}'
              : task.downloaded,
        ),
        if (task.speed != null) _InfoBlock(label: context.l10n.speed, value: task.speed!),
        if (task.uploading) _InfoBlock(label: context.l10n.uploadSpeed, value: task.uploadSpeed!),
        if (task.status == TaskStatus.completed)
          _InfoBlock(label: context.l10n.downloadDuration, value: task.localizedDownloadDuration(context.l10n) ?? '—')
        else if (task.localizedRemaining(context.l10n) case final remaining?)
          _InfoBlock(label: context.l10n.remaining, value: remaining),
        if (task.createdAt case final createdAt?)
          _InfoBlock(label: context.l10n.createdAt, value: _formatTaskDateTime(createdAt)),
        Container(
          key: const ValueKey('task-details-info-divider'),
          height: 1,
          margin: const EdgeInsets.symmetric(vertical: 4),
          color: palette.border,
        ),
        if (editingUrl)
          _EditableUrlBlock(
            controller: urlController,
            updating: updatingUrl,
            onCancel: onCancelEditUrl,
            onSave: onSaveUrl,
          )
        else
          _CopyableInfoBlock(
            label: context.l10n.downloadLink,
            value: task.url,
            valueKey: const ValueKey('task-details-url-value'),
            actionKey: const ValueKey('task-details-copy-url'),
            actionLabel: copied ? context.l10n.copied : context.l10n.copy,
            actionIcon: copied ? Icons.check : Icons.copy_outlined,
            secondaryActionLabel: onEditUrl == null ? null : context.l10n.edit,
            secondaryActionIcon: Icons.edit_outlined,
            onSecondaryTap: onEditUrl,
            onTap: onCopy,
          ),
        _CopyableInfoBlock(
          label: context.l10n.storagePath,
          value: task.storagePath,
          valueKey: const ValueKey('task-details-storage-value'),
          actionKey: const ValueKey('task-details-open-storage'),
          actionLabel: context.l10n.open,
          actionIcon: Icons.folder_open_outlined,
          onTap: onOpenStorage,
        ),
      ],
    );
  }
}

String _formatTaskDateTime(DateTime value) {
  final local = value.toLocal();
  String two(int number) => number.toString().padLeft(2, '0');
  return '${local.year}-${two(local.month)}-${two(local.day)} '
      '${two(local.hour)}:${two(local.minute)}:${two(local.second)}';
}

class _InfoBlock extends StatelessWidget {
  const _InfoBlock({required this.label, required this.value, this.error = false});

  final String label;
  final String value;
  final bool error;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 14),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _InfoLabel(label),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              value,
              style: TextStyle(color: error ? palette.error : palette.textPrimary, fontSize: 12, height: 1.35),
            ),
          ),
        ],
      ),
    );
  }
}

class _CopyableInfoBlock extends StatelessWidget {
  const _CopyableInfoBlock({
    required this.label,
    required this.value,
    required this.valueKey,
    required this.actionKey,
    required this.actionLabel,
    required this.actionIcon,
    required this.onTap,
    this.secondaryActionLabel,
    this.secondaryActionIcon,
    this.onSecondaryTap,
  });

  final String label;
  final String value;
  final Key valueKey;
  final Key actionKey;
  final String actionLabel;
  final IconData actionIcon;
  final VoidCallback onTap;
  final String? secondaryActionLabel;
  final IconData? secondaryActionIcon;
  final VoidCallback? onSecondaryTap;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 14),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _InfoLabel(label),
          const SizedBox(width: 12),
          Expanded(
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Text(
                    _softWrapAnywhere(value),
                    key: valueKey,
                    semanticsLabel: value,
                    softWrap: true,
                    style: TextStyle(color: palette.textSecondary, fontSize: 11, height: 1.35),
                  ),
                ),
                const SizedBox(width: 8),
                if (secondaryActionLabel != null && secondaryActionIcon != null && onSecondaryTap != null) ...[
                  _TaskDetailIconButton(
                    key: const ValueKey('task-details-edit-url'),
                    label: secondaryActionLabel!,
                    icon: Icon(secondaryActionIcon!, size: 16),
                    onPressed: onSecondaryTap,
                  ),
                  const SizedBox(width: 4),
                ],
                _TaskDetailIconButton(
                  key: actionKey,
                  label: actionLabel,
                  icon: Icon(actionIcon, size: 16),
                  onPressed: onTap,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

String _softWrapAnywhere(String value) => value.characters.join('\u200B');

class _InfoLabel extends StatelessWidget {
  const _InfoLabel(this.label);

  final String label;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return SizedBox(
      width: 88,
      child: Padding(
        padding: const EdgeInsets.only(top: 2),
        child: Text(
          label,
          style: TextStyle(
            color: palette.textMuted,
            fontSize: 10,
            fontWeight: FontWeight.w700,
            letterSpacing: 0.6,
            height: 1.3,
          ),
        ),
      ),
    );
  }
}

class _EditableUrlBlock extends StatelessWidget {
  const _EditableUrlBlock({
    required this.controller,
    required this.updating,
    required this.onCancel,
    required this.onSave,
  });

  final TextEditingController controller;
  final bool updating;
  final VoidCallback onCancel;
  final VoidCallback onSave;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 14),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _InfoLabel(context.l10n.downloadLink),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                SizedBox(
                  height: 76,
                  child: shad.TextField(
                    controller: controller,
                    minLines: null,
                    maxLines: null,
                    expands: true,
                    textAlignVertical: TextAlignVertical.top,
                    style: TextStyle(color: palette.textPrimary, fontSize: 11, height: 1.3),
                    padding: const EdgeInsets.all(8),
                    decoration: BoxDecoration(
                      color: palette.inputBg,
                      borderRadius: BorderRadius.circular(4),
                      border: Border.all(color: palette.border),
                    ),
                  ),
                ),
                const SizedBox(height: 8),
                Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    _TaskDetailIconButton(
                      key: const ValueKey('task-details-cancel-url'),
                      label: context.l10n.cancel,
                      icon: const Icon(Icons.close, size: 17),
                      onPressed: updating ? null : onCancel,
                    ),
                    const SizedBox(width: 8),
                    _TaskDetailIconButton(
                      key: const ValueKey('task-details-save-url'),
                      label: context.l10n.updateAndResume,
                      icon: updating
                          ? const SizedBox.square(dimension: 14, child: shad.CircularProgressIndicator())
                          : const Icon(Icons.check, size: 17),
                      onPressed: updating ? null : onSave,
                      primary: true,
                    ),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _TaskDetailIconButton extends StatelessWidget {
  const _TaskDetailIconButton({
    super.key,
    required this.label,
    required this.icon,
    required this.onPressed,
    this.primary = false,
  });

  final String label;
  final Widget icon;
  final VoidCallback? onPressed;
  final bool primary;

  @override
  Widget build(BuildContext context) {
    final button = primary
        ? shad.IconButton.primary(size: shad.ButtonSize.xSmall, onPressed: onPressed, icon: icon)
        : shad.IconButton.ghost(size: shad.ButtonSize.xSmall, onPressed: onPressed, icon: icon);
    return AppTooltip(
      message: label,
      child: SizedBox.square(dimension: 28, child: button),
    );
  }
}
