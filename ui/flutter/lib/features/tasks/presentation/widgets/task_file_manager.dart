import 'dart:async';

import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:path/path.dart' as path;
import 'package:share_plus/share_plus.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;
import 'package:url_launcher/url_launcher.dart';

import '../../../../api/api.dart' as api;
import '../../../../core/utils/file_explorer.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_toast.dart';
import '../../../../shared/widgets/app_tooltip.dart';
import '../../../../util/browser_download/browser_download.dart';
import '../../../../util/util.dart';
import '../../domain/task_record.dart';

class TaskFileManagerView extends StatefulWidget {
  const TaskFileManagerView({super.key, required this.task, this.webActions, this.desktopActions});

  final TaskRecord task;
  final bool? webActions;
  final bool? desktopActions;

  @override
  State<TaskFileManagerView> createState() => _TaskFileManagerViewState();
}

class _TaskFileManagerViewState extends State<TaskFileManagerView> {
  late _TaskFileDirectoryIndex _index;
  var _directory = '/';

  bool get _webActions => widget.webActions ?? Util.isWeb();
  bool get _desktopActions => widget.desktopActions ?? Util.isDesktop();

  @override
  void initState() {
    super.initState();
    _index = _TaskFileDirectoryIndex(widget.task);
  }

  @override
  void didUpdateWidget(covariant TaskFileManagerView oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.task.id != oldWidget.task.id || !identical(widget.task.files, oldWidget.task.files)) {
      _index = _TaskFileDirectoryIndex(widget.task);
      if (!_index.contains(_directory)) _directory = '/';
    }
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final entries = _index.entries(_directory);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _FileBreadcrumb(directory: _directory, onSelected: (directory) => setState(() => _directory = directory)),
        Expanded(
          child: entries.isEmpty
              ? Center(
                  child: Text(context.l10n.noFiles, style: TextStyle(color: palette.textMuted, fontSize: 13)),
                )
              : ListView.builder(
                  key: PageStorageKey<String>('task-file-manager-${widget.task.id}-$_directory'),
                  padding: const EdgeInsets.fromLTRB(16, 4, 16, 20),
                  itemCount: entries.length,
                  itemBuilder: (context, index) {
                    final entry = entries[index];
                    return _FileManagerRow(
                      key: ValueKey('managed-file-${entry.fullPath}'),
                      entry: entry,
                      childCount: entry.directory ? _index.entries(entry.fullPath).length : 0,
                      webActions: _webActions,
                      desktopActions: _desktopActions,
                      onOpenDirectory: () => setState(() => _directory = entry.fullPath),
                      onOpenSystemDirectory: () => unawaited(_revealEntry(entry)),
                      onOpenFile: () => unawaited(_openFile(entry)),
                      onShareFile: () => unawaited(_shareFile(context, entry)),
                      onDownloadFile: () => _downloadFile(entry),
                    );
                  },
                ),
        ),
      ],
    );
  }

  Future<void> _openFile(_TaskFileEntry entry) async {
    final opened = _webActions
        ? await launchUrl(Uri.parse(_accessUrl(entry)), webOnlyWindowName: '_blank')
        : await FileExplorer.open(_localPath(entry));
    if (!opened && mounted) {
      showAppToast(context, context.l10n.unableOpenPath(entry.name), type: AppToastType.error);
    }
  }

  Future<void> _revealEntry(_TaskFileEntry entry) async {
    final entryPath = _localPath(entry);
    if (!await FileExplorer.reveal(entryPath) && mounted) {
      showAppToast(context, context.l10n.unableLocatePath(entryPath), type: AppToastType.error);
    }
  }

  Future<void> _shareFile(BuildContext actionContext, _TaskFileEntry entry) async {
    try {
      final renderBox = actionContext.findRenderObject();
      final origin = renderBox is RenderBox ? renderBox.localToGlobal(Offset.zero) & renderBox.size : null;
      await Share.shareXFiles([XFile(_localPath(entry))], sharePositionOrigin: origin);
    } catch (error) {
      if (mounted) showAppToast(context, error.toString(), type: AppToastType.error);
    }
  }

  void _downloadFile(_TaskFileEntry entry) {
    download(_accessUrl(entry), entry.name);
  }

  String _localPath(_TaskFileEntry entry) {
    if (!widget.task.isFolder && _index.fileCount == 1) return widget.task.storagePath;
    final segments = entry.fullPath.split('/').where((part) => part.isNotEmpty);
    return path.joinAll([widget.task.storagePath, ...segments]);
  }

  String _accessUrl(_TaskFileEntry entry) {
    final encodedPath = entry.fullPath.split('/').where((part) => part.isNotEmpty).map(Uri.encodeComponent).join('/');
    return api.join('/fs/tasks/${Uri.encodeComponent(widget.task.id)}/$encodedPath');
  }
}

class _FileBreadcrumb extends StatelessWidget {
  const _FileBreadcrumb({required this.directory, required this.onSelected});

  final String directory;
  final ValueChanged<String> onSelected;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final segments = directory.split('/').where((part) => part.isNotEmpty).toList(growable: false);
    final crumbs = <({String label, String path})>[(label: '/', path: '/')];
    var current = '';
    for (final segment in segments) {
      current = '$current/$segment';
      crumbs.add((label: segment, path: current));
    }

    return SizedBox(
      height: 48,
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 16),
        child: Row(
          children: [
            for (var index = 0; index < crumbs.length; index++) ...[
              if (index > 0) Icon(Icons.chevron_right, size: 16, color: palette.textMuted),
              GestureDetector(
                key: ValueKey('file-breadcrumb-${crumbs[index].path}'),
                onTap: () => onSelected(crumbs[index].path),
                behavior: HitTestBehavior.opaque,
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 12),
                  child: Text(
                    crumbs[index].label,
                    style: TextStyle(
                      color: index == crumbs.length - 1 ? palette.textPrimary : palette.textSecondary,
                      fontSize: 12,
                      fontWeight: index == crumbs.length - 1 ? FontWeight.w600 : FontWeight.w400,
                    ),
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _FileManagerRow extends StatelessWidget {
  const _FileManagerRow({
    super.key,
    required this.entry,
    required this.childCount,
    required this.webActions,
    required this.desktopActions,
    required this.onOpenDirectory,
    required this.onOpenSystemDirectory,
    required this.onOpenFile,
    required this.onShareFile,
    required this.onDownloadFile,
  });

  final _TaskFileEntry entry;
  final int childCount;
  final bool webActions;
  final bool desktopActions;
  final VoidCallback onOpenDirectory;
  final VoidCallback onOpenSystemDirectory;
  final VoidCallback onOpenFile;
  final VoidCallback onShareFile;
  final VoidCallback onDownloadFile;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return GestureDetector(
      onTap: entry.directory ? onOpenDirectory : null,
      behavior: HitTestBehavior.opaque,
      child: Container(
        constraints: const BoxConstraints(minHeight: 54),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
        decoration: BoxDecoration(
          border: Border(bottom: BorderSide(color: palette.border)),
        ),
        child: Row(
          children: [
            Icon(
              taskFileTypeIcon(entry.name, isFolder: entry.directory),
              size: 20,
              color: entry.directory ? palette.brand : palette.textSecondary,
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    entry.name,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(color: palette.textPrimary, fontSize: 13, fontWeight: FontWeight.w500),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    entry.directory ? context.l10n.items(childCount) : Util.fmtByte(entry.size),
                    style: TextStyle(color: palette.textMuted, fontSize: 11),
                  ),
                ],
              ),
            ),
            if (entry.directory) ...[
              if (desktopActions) ...[
                _FileActionButton(
                  label: context.l10n.openDirectory,
                  icon: Icons.folder_open_outlined,
                  onPressed: onOpenSystemDirectory,
                ),
                const SizedBox(width: 4),
              ],
              Icon(Icons.chevron_right, size: 18, color: palette.textMuted),
            ] else ...[
              _FileActionButton(label: context.l10n.openFile, icon: Icons.open_in_new, onPressed: onOpenFile),
              if (desktopActions) ...[
                const SizedBox(width: 4),
                _FileActionButton(
                  label: context.l10n.openDirectory,
                  icon: Icons.folder_open_outlined,
                  onPressed: onOpenSystemDirectory,
                ),
              ] else ...[
                const SizedBox(width: 4),
                if (webActions)
                  _FileActionButton(
                    label: context.l10n.download,
                    icon: Icons.download_outlined,
                    onPressed: onDownloadFile,
                  )
                else
                  _FileActionButton(label: context.l10n.share, icon: Icons.share_outlined, onPressed: onShareFile),
              ],
            ],
          ],
        ),
      ),
    );
  }
}

class _FileActionButton extends StatelessWidget {
  const _FileActionButton({required this.label, required this.icon, required this.onPressed});

  final String label;
  final IconData icon;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return AppTooltip(
      message: label,
      child: shad.IconButton.ghost(
        size: shad.ButtonSize.small,
        onPressed: onPressed,
        icon: Icon(icon, size: 17, color: palette.textSecondary),
      ),
    );
  }
}

class _TaskFileDirectoryIndex {
  _TaskFileDirectoryIndex(TaskRecord task) {
    final sourceFiles = task.files.isEmpty
        ? [TaskFileNode(path: '/', name: task.name, sizeBytes: task.totalBytes ?? 0)]
        : task.files;
    fileCount = sourceFiles.length;
    final singleFile = !task.isFolder && fileCount == 1;
    for (final file in sourceFiles) {
      final directory = _normalizeDirectory(file.path);
      final name = singleFile ? path.basename(task.storagePath) : file.name;
      final fullPath = directory == '/' ? '/$name' : '$directory/$name';
      _add(directory, _TaskFileEntry(directory: false, fullPath: fullPath, name: name, size: file.sizeBytes));
      _addParentDirectories(directory);
    }
    for (final entries in _entries.values) {
      entries.sort((left, right) {
        if (left.directory != right.directory) return left.directory ? -1 : 1;
        return left.name.toLowerCase().compareTo(right.name.toLowerCase());
      });
    }
  }

  final Map<String, List<_TaskFileEntry>> _entries = {'/': []};
  late final int fileCount;

  bool contains(String directory) => _entries.containsKey(directory);

  List<_TaskFileEntry> entries(String directory) => _entries[directory] ?? const [];

  void _addParentDirectories(String directory) {
    var current = directory;
    while (current != '/') {
      final parent = path.posix.dirname(current);
      final normalizedParent = parent == '.' ? '/' : parent;
      final name = path.posix.basename(current);
      _add(normalizedParent, _TaskFileEntry(directory: true, fullPath: current, name: name, size: 0));
      _entries.putIfAbsent(current, () => []);
      current = normalizedParent;
    }
  }

  void _add(String directory, _TaskFileEntry entry) {
    final entries = _entries.putIfAbsent(directory, () => []);
    if (!entries.any((item) => item.fullPath == entry.fullPath)) entries.add(entry);
  }
}

class _TaskFileEntry {
  const _TaskFileEntry({required this.directory, required this.fullPath, required this.name, required this.size});

  final bool directory;
  final String fullPath;
  final String name;
  final int size;
}

String _normalizeDirectory(String value) {
  var normalized = value.trim().replaceAll('\\', '/');
  if (normalized.isEmpty || normalized == '.') return '/';
  if (!normalized.startsWith('/')) normalized = '/$normalized';
  while (normalized.length > 1 && normalized.endsWith('/')) {
    normalized = normalized.substring(0, normalized.length - 1);
  }
  return normalized;
}
