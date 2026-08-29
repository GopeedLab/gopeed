import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart' show Colors, Icons;
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;
import 'package:window_manager/window_manager.dart';

import '../../../../app/application/app_runtime_controller.dart';
import '../../../../core/capabilities/app_capabilities.dart';
import '../../../../core/utils/breakpoints.dart';
import '../../../../core/utils/file_explorer.dart';
import '../../../../core/window/app_window_launcher.dart';
import '../../../../core/window/app_window_chrome.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_toast.dart';
import '../../../../shared/widgets/responsive_navigation_layout.dart';
import '../../../../l10n/l10n.dart';
import '../../../../util/util.dart';
import '../../../tasks/application/tasks_controller.dart';
import '../../../tasks/application/pending_update_task.dart';
import '../../../tasks/application/task_batch_selection_controller.dart';
import '../../../tasks/domain/task_record.dart';
import '../../../tasks/presentation/widgets/speed_monitor_card.dart';
import '../../../tasks/presentation/widgets/task_batch_selection_builder.dart';
import '../../../tasks/presentation/widgets/task_card/task_card.dart';
import '../../../tasks/presentation/widgets/task_drawer.dart';
import '../../../tasks/presentation/widgets/task_context_menu.dart';
import '../../../tasks/presentation/widgets/task_delete_dialog.dart';
import '../../../tasks/presentation/widgets/task_empty_state.dart';
import '../../../tasks/presentation/widgets/task_file_browser_dialog.dart';
import '../../../tasks/presentation/widgets/task_update_url_dialog.dart';
import '../widgets/primary_rail.dart';
import '../widgets/tasks_top_bar.dart';

enum _TaskFilter { downloading, completed, failed }

class _BatchActionAvailability {
  const _BatchActionAvailability({required this.canPause, required this.canResume});

  final bool canPause;
  final bool canResume;
}

_BatchActionAvailability _batchActionAvailability(Iterable<TaskRecord> tasks, TaskBatchSelectionController selection) {
  var canPause = false;
  var canResume = false;
  for (final task in tasks) {
    if (!selection.contains(task.id)) continue;
    canPause = canPause || task.status == TaskStatus.downloading;
    canResume = canResume || task.status == TaskStatus.paused || task.status == TaskStatus.failed;
    if (canPause && canResume) break;
  }
  return _BatchActionAvailability(canPause: canPause, canResume: canResume);
}

class HomePage extends ConsumerStatefulWidget {
  const HomePage({super.key});

  @override
  ConsumerState<HomePage> createState() => _HomePageState();
}

class _HomePageState extends ConsumerState<HomePage> {
  final TextEditingController _searchController = TextEditingController();
  _TaskFilter _activeFilter = _TaskFilter.downloading;
  TaskRecord? _selectedTask;
  bool _batchMode = false;
  final TaskBatchSelectionController _batchSelection = TaskBatchSelectionController();

  @override
  void initState() {
    super.initState();
    _searchController.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _searchController.dispose();
    _batchSelection.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final tasksAsync = ref.watch(tasksControllerProvider);
    final tasksState = tasksAsync.value;
    final tasks = tasksState?.tasks ?? const <TaskRecord>[];
    final filteredTasks = _filteredTasks(tasks);
    final emptyStateMessage = _searchController.text.trim().isEmpty
        ? context.l10n.emptyTaskList
        : context.l10n.noMatchingTasks;
    final selectedTask = _latestSelectedTask(tasks);
    final visibleTaskIds = filteredTasks.map((task) => task.id).toList(growable: false);
    final pendingUpdateTask = ref.watch(pendingUpdateTaskProvider);
    final navigationItems = _taskNavigationItems(tasks);
    final desktopBody = shad.Scaffold(
      backgroundColor: palette.bg,
      child: Row(
        children: [
          const PrimaryRail(),
          Expanded(
            child: ResponsiveNavigationLayout<_TaskFilter>(
              desktopTitle: context.l10n.task.toUpperCase(),
              items: navigationItems,
              selectedValue: _activeFilter,
              onSelected: _setActiveFilter,
              desktopFooter: SpeedMonitorCard(
                downloadBytesPerSecond: tasksState?.downloadSpeedBytesPerSecond ?? 0,
                uploadBytesPerSecond: tasksState?.uploadSpeedBytesPerSecond ?? 0,
              ),
              child: Padding(
                padding: EdgeInsets.only(
                  top: AppWindowChrome.reservesHeaderInset ? AppDesignTokens.windowHeaderHeight : 0,
                ),
                child: Column(
                  children: [
                    ListenableBuilder(
                      listenable: _batchSelection,
                      builder: (context, _) {
                        final availability = _batchActionAvailability(tasks, _batchSelection);
                        return TasksTopBar(
                          searchController: _searchController,
                          onAddTask: _handleAddTask,
                          batchMode: _batchMode,
                          selectedBatchCount: _batchSelection.count,
                          canPauseSelected: availability.canPause,
                          canResumeSelected: availability.canResume,
                          onToggleBatchMode: _toggleBatchMode,
                          onPauseSelected: _pauseSelectedTasks,
                          onResumeSelected: _resumeSelectedTasks,
                          onDeleteSelected: _deleteSelectedTasks,
                        );
                      },
                    ),
                    Expanded(
                      child: Stack(
                        children: [
                          Padding(
                            padding: const EdgeInsets.fromLTRB(32, AppDesignTokens.space8, 32, 0),
                            child: Column(
                              key: const ValueKey('tasks-content-start'),
                              children: [
                                AnimatedSize(
                                  duration: const Duration(milliseconds: 180),
                                  curve: Curves.easeOutCubic,
                                  child: _batchMode
                                      ? ListenableBuilder(
                                          listenable: _batchSelection,
                                          builder: (context, _) => _TaskBatchRow(
                                            selectedCount: _batchSelection.count,
                                            totalCount: filteredTasks.length,
                                            allSelected: _batchSelection.areAllSelected(visibleTaskIds),
                                            someSelected:
                                                _batchSelection.isNotEmpty &&
                                                !_batchSelection.areAllSelected(visibleTaskIds),
                                            onToggleSelectAll: _toggleSelectAll,
                                          ),
                                        )
                                      : const SizedBox.shrink(),
                                ),
                                Expanded(
                                  child: tasksAsync.when(
                                    loading: () => const Center(child: shad.CircularProgressIndicator()),
                                    error: (error, _) => const TaskEmptyState(),
                                    data: (_) => filteredTasks.isEmpty
                                        ? TaskEmptyState(message: emptyStateMessage)
                                        : ListView.separated(
                                            padding: const EdgeInsets.only(bottom: 16),
                                            itemCount: filteredTasks.length,
                                            separatorBuilder: (_, _) => const SizedBox(height: 8),
                                            itemBuilder: (context, index) {
                                              final task = filteredTasks[index];
                                              return TaskBatchSelectionBuilder(
                                                controller: _batchSelection,
                                                taskId: task.id,
                                                visibleTaskIds: visibleTaskIds,
                                                builder: (context, selectedInBatch, allSelected) => TaskCard(
                                                  task: task,
                                                  selected: _selectedTask?.id == task.id,
                                                  batchMode: _batchMode,
                                                  selectedInBatch: selectedInBatch,
                                                  onPressed: () => _handleTaskPressed(task),
                                                  onToggleBatch: () => _toggleBatchTask(task.id),
                                                  onPause: () => _runTaskAction(
                                                    () => ref.read(tasksControllerProvider.notifier).pause(task.id),
                                                  ),
                                                  onResume: () => _runTaskAction(
                                                    () => ref.read(tasksControllerProvider.notifier).resume(task.id),
                                                  ),
                                                  onDelete: () => unawaited(_deleteTasks([task.id])),
                                                  onOpen: () => _openTask(task),
                                                  onReveal: () => _revealTask(task),
                                                  contextActions: _taskContextActions(
                                                    task,
                                                    selected: selectedInBatch,
                                                    allSelected: allSelected,
                                                  ),
                                                  listeningForUpdate: pendingUpdateTask?.id == task.id,
                                                ),
                                              );
                                            },
                                          ),
                                  ),
                                ),
                              ],
                            ),
                          ),
                          TaskDrawer(
                            task: selectedTask,
                            onClose: () => setState(() => _selectedTask = null),
                            onOpenStorage: selectedTask == null ? () {} : () => unawaited(_revealTask(selectedTask)),
                            onUpdateUrl: selectedTask == null
                                ? (_) async {}
                                : (url) => _updateTaskUrl(selectedTask.id, url, headers: selectedTask.requestHeaders),
                          ),
                        ],
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

    if (MediaQuery.sizeOf(context).width < Breakpoints.mobile) {
      final mobileBody = tasksAsync.when(
        loading: () => _MobileTasksView(
          tasks: const [],
          batchMode: _batchMode,
          batchSelection: _batchSelection,
          loading: true,
          onToggleSelectAll: _toggleSelectAll,
          onPauseSelected: _pauseSelectedTasks,
          onResumeSelected: _resumeSelectedTasks,
          onDeleteSelected: _deleteSelectedTasks,
          onTaskPressed: _handleTaskPressed,
          onToggleBatch: _toggleBatchTask,
          onTaskPause: (task) => _runTaskAction(() => ref.read(tasksControllerProvider.notifier).pause(task.id)),
          onTaskResume: (task) => _runTaskAction(() => ref.read(tasksControllerProvider.notifier).resume(task.id)),
          onTaskDelete: (task) => unawaited(_deleteTasks([task.id])),
          onTaskOpen: _openTask,
          onTaskReveal: _revealTask,
          contextActionsForTask: (task, selected, allSelected) =>
              _taskContextActions(task, selected: selected, allSelected: allSelected),
          pendingUpdateTaskId: pendingUpdateTask?.id,
        ),
        error: (error, _) => ColoredBox(color: palette.bg, child: const TaskEmptyState()),
        data: (_) => _MobileTasksView(
          tasks: filteredTasks,
          emptyStateMessage: emptyStateMessage,
          batchMode: _batchMode,
          batchSelection: _batchSelection,
          onToggleSelectAll: _toggleSelectAll,
          onPauseSelected: _pauseSelectedTasks,
          onResumeSelected: _resumeSelectedTasks,
          onDeleteSelected: _deleteSelectedTasks,
          onTaskPressed: _handleTaskPressed,
          onToggleBatch: _toggleBatchTask,
          onTaskPause: (task) => _runTaskAction(() => ref.read(tasksControllerProvider.notifier).pause(task.id)),
          onTaskResume: (task) => _runTaskAction(() => ref.read(tasksControllerProvider.notifier).resume(task.id)),
          onTaskDelete: (task) => unawaited(_deleteTasks([task.id])),
          onTaskOpen: _openTask,
          onTaskReveal: _revealTask,
          contextActionsForTask: (task, selected, allSelected) =>
              _taskContextActions(task, selected: selected, allSelected: allSelected),
          pendingUpdateTaskId: pendingUpdateTask?.id,
        ),
      );
      return shad.Scaffold(
        backgroundColor: palette.bg,
        child: Column(
          children: [
            Expanded(
              child: ResponsiveNavigationLayout<_TaskFilter>(
                desktopTitle: context.l10n.task.toUpperCase(),
                items: navigationItems,
                selectedValue: _activeFilter,
                onSelected: _setActiveFilter,
                mobileHeader: ListenableBuilder(
                  listenable: _batchSelection,
                  builder: (context, _) => _MobileTasksHeader(
                    batchMode: _batchMode,
                    selectedCount: _batchSelection.count,
                    onAddTask: _handleAddTask,
                    onToggleBatchMode: _toggleBatchMode,
                  ),
                ),
                child: mobileBody,
              ),
            ),
            const PrimaryBottomNavigation(),
          ],
        ),
      );
    }

    if (kIsWeb || defaultTargetPlatform == TargetPlatform.macOS) {
      return desktopBody;
    }

    return DragToResizeArea(resizeEdgeSize: 6, child: desktopBody);
  }

  void _setActiveFilter(_TaskFilter filter) {
    _batchSelection.clear();
    setState(() {
      _activeFilter = filter;
      _selectedTask = null;
    });
  }

  List<TaskRecord> _filteredTasks(List<TaskRecord> tasks) {
    final query = _searchController.text.trim().toLowerCase();
    return tasks
        .where((task) {
          final passesFilter = switch (_activeFilter) {
            _TaskFilter.downloading => task.status == TaskStatus.downloading || task.status == TaskStatus.paused,
            _TaskFilter.completed => task.status == TaskStatus.completed,
            _TaskFilter.failed => task.status == TaskStatus.failed,
          };
          final passesSearch =
              query.isEmpty || task.name.toLowerCase().contains(query) || task.url.toLowerCase().contains(query);
          return passesFilter && passesSearch;
        })
        .toList(growable: false);
  }

  TaskRecord? _latestSelectedTask(List<TaskRecord> tasks) {
    final selectedTask = _selectedTask;
    if (selectedTask == null) return null;
    for (final task in tasks) {
      if (task.id == selectedTask.id) return task;
    }
    return selectedTask;
  }

  List<ResponsiveNavigationItem<_TaskFilter>> _taskNavigationItems(List<TaskRecord> tasks) {
    int count(bool Function(TaskRecord task) test) => tasks.where(test).length;

    return [
      ResponsiveNavigationItem<_TaskFilter>(
        value: _TaskFilter.downloading,
        label: context.l10n.downloading,
        icon: Icons.south_rounded,
        count: count((task) => task.status == TaskStatus.downloading || task.status == TaskStatus.paused),
      ),
      ResponsiveNavigationItem<_TaskFilter>(
        value: _TaskFilter.completed,
        label: context.l10n.completed,
        icon: Icons.check_circle_outline,
        count: count((task) => task.status == TaskStatus.completed),
      ),
      ResponsiveNavigationItem<_TaskFilter>(
        value: _TaskFilter.failed,
        label: context.l10n.failed,
        icon: Icons.error_outline,
        count: count((task) => task.status == TaskStatus.failed),
      ),
    ];
  }

  void _handleTaskPressed(TaskRecord task) {
    if (_batchMode) {
      _toggleBatchTask(task.id);
    } else {
      _showTaskDetails(task);
    }
  }

  void _showTaskDetails(TaskRecord task) {
    if (MediaQuery.sizeOf(context).width < Breakpoints.mobile) {
      context.push('/tasks/${Uri.encodeComponent(task.id)}', extra: task);
    } else {
      setState(() {
        _selectedTask = task;
      });
    }
  }

  void _toggleBatchMode() {
    final enteringBatchMode = !_batchMode;
    if (!enteringBatchMode) {
      _batchSelection.clear();
    }
    setState(() {
      _batchMode = enteringBatchMode;
      if (_batchMode) {
        _selectedTask = null;
      }
    });
  }

  void _toggleBatchTask(String taskId) => _batchSelection.toggle(taskId);

  void _toggleSelectAll() {
    final visibleIds = _filteredTasks(
      ref.read(tasksControllerProvider).value?.tasks ?? const [],
    ).map((task) => task.id).toList(growable: false);
    if (_batchSelection.areAllSelected(visibleIds)) {
      _batchSelection.clear();
      return;
    }
    if (!_batchMode) {
      setState(() {
        _batchMode = true;
        _selectedTask = null;
      });
    }
    _batchSelection.replaceWith(visibleIds);
  }

  void _deleteSelectedTasks() {
    if (_batchSelection.isEmpty) return;
    final ids = _batchSelection.toList();
    unawaited(_deleteTasks(ids));
  }

  void _pauseSelectedTasks() {
    if (_batchMode && _batchSelection.isEmpty) return;
    final ids = _batchMode ? _selectedTaskIdsFor((task) => task.status == TaskStatus.downloading) : null;
    if (_batchMode && ids!.isEmpty) return;
    unawaited(_runTaskAction(() => ref.read(tasksControllerProvider.notifier).pauseAll(ids)));
  }

  void _resumeSelectedTasks() {
    if (_batchMode && _batchSelection.isEmpty) return;
    final ids = _batchMode
        ? _selectedTaskIdsFor((task) => task.status == TaskStatus.paused || task.status == TaskStatus.failed)
        : null;
    if (_batchMode && ids!.isEmpty) return;
    unawaited(_runTaskAction(() => ref.read(tasksControllerProvider.notifier).resumeAll(ids)));
  }

  List<String> _selectedTaskIdsFor(bool Function(TaskRecord task) test) {
    final tasks = ref.read(tasksControllerProvider).value?.tasks ?? const <TaskRecord>[];
    return tasks.where((task) => _batchSelection.contains(task.id) && test(task)).map((task) => task.id).toList();
  }

  TaskContextActions _taskContextActions(TaskRecord task, {required bool selected, required bool allSelected}) {
    final listeningForUpdate = ref.read(pendingUpdateTaskProvider)?.id == task.id;
    return TaskContextActions(
      selected: selected,
      allSelected: allSelected,
      listeningForUpdate: listeningForUpdate,
      onShowDetails: () => _showTaskDetails(task),
      onBrowseFiles: () =>
          unawaited(browseTaskFiles(context, task, mobile: MediaQuery.sizeOf(context).width < Breakpoints.mobile)),
      onToggleSelectAll: _toggleSelectAll,
      onToggleSelected: () {
        if (!_batchMode) {
          setState(() {
            _batchMode = true;
            _selectedTask = null;
          });
        }
        _toggleBatchTask(task.id);
      },
      onPause: () => unawaited(_runContextTaskAction(task, pause: true)),
      onResume: () => unawaited(_runContextTaskAction(task, pause: false)),
      onDelete: () => unawaited(_deleteTasks(_contextTaskIds(task))),
      onUpdateUrl: () => unawaited(_showUpdateUrlDialog(task)),
      onToggleUpdateListener: () => _toggleUpdateListener(task),
    );
  }

  List<String> _contextTaskIds(TaskRecord task) {
    final visibleIds = _filteredTasks(
      ref.read(tasksControllerProvider).value?.tasks ?? const [],
    ).map((item) => item.id).toSet();
    return <String>{..._batchSelection.where(visibleIds.contains), task.id}.toList(growable: false);
  }

  Future<void> _runContextTaskAction(TaskRecord task, {required bool pause}) async {
    final ids = _contextTaskIds(task);
    await _runTaskAction(() {
      final controller = ref.read(tasksControllerProvider.notifier);
      return pause ? controller.pauseAll(ids) : controller.resumeAll(ids);
    });
    if (!mounted) return;
    _batchSelection.clear();
    setState(() {
      _batchMode = false;
    });
  }

  Future<void> _deleteTasks(List<String> ids) async {
    if (ids.isEmpty) return;
    final runtime = ref.read(appRuntimeControllerProvider).value;
    final config = runtime?.downloaderConfig;
    final keepFiles = await showTaskDeleteDialog(
      context,
      taskCount: ids.length,
      keepFiles: config?.extra.lastDeleteTaskKeep ?? false,
    );
    if (keepFiles == null || !mounted) return;

    await _runTaskAction(() async {
      if (config != null) {
        config.extra.lastDeleteTaskKeep = keepFiles;
        await ref.read(gopeedServiceProvider).putConfig(config);
      }
      await ref.read(tasksControllerProvider.notifier).deleteSelected(ids, force: !keepFiles);
      await ref.read(appRuntimeControllerProvider.notifier).reloadConfig();
    });
    if (!mounted) return;
    _batchSelection.removeAll(ids);
    setState(() {
      _batchMode = false;
      if (_selectedTask != null && ids.contains(_selectedTask!.id)) {
        _selectedTask = null;
      }
    });
  }

  Future<void> _showUpdateUrlDialog(TaskRecord task) async {
    final update = await showTaskUpdateUrlDialog(context, task);
    if (update == null || !mounted) return;
    await _updateTaskUrl(task.id, update.url, headers: update.headers);
  }

  void _toggleUpdateListener(TaskRecord task) {
    if (!task.canUpdateUrl) return;
    final notifier = ref.read(pendingUpdateTaskProvider.notifier);
    if (ref.read(pendingUpdateTaskProvider)?.id == task.id) {
      notifier.clear();
    } else {
      notifier.set(PendingUpdateTask(id: task.id, name: task.name));
    }
  }

  void _handleAddTask() {
    if (!Util.isDesktop()) {
      context.go('/create');
      return;
    }
    unawaited(
      AppWindowLauncher.openCreateTaskWindow().then((opened) {
        if (!opened && mounted) {
          context.go('/create');
        }
      }),
    );
  }

  Future<void> _runTaskAction(Future<void> Function() action) async {
    try {
      await action();
    } catch (error) {
      if (!mounted) return;
      _showToast(error.toString());
    }
  }

  Future<void> _updateTaskUrl(String id, String url, {Map<String, String> headers = const {}}) async {
    try {
      await ref.read(tasksControllerProvider.notifier).updateUrl(id, url, headers: headers);
      final updatedTasks = ref.read(tasksControllerProvider).value?.tasks ?? const <TaskRecord>[];
      for (final task in updatedTasks) {
        if (task.id == id) {
          if (mounted) {
            setState(() => _selectedTask = task);
          }
          break;
        }
      }
    } catch (error) {
      if (!mounted) rethrow;
      _showToast(error.toString());
      rethrow;
    }
  }

  Future<void> _openTask(TaskRecord task) async {
    final opened = task.isFolder
        ? await FileExplorer.reveal(task.storagePath)
        : await FileExplorer.open(task.storagePath);
    if (!opened && mounted) {
      _showToast(context.l10n.unableOpenPath(task.storagePath));
    }
  }

  Future<void> _revealTask(TaskRecord task) async {
    if (!await FileExplorer.reveal(task.storagePath) && mounted) {
      _showToast(context.l10n.unableLocatePath(task.storagePath));
    }
  }

  void _showToast(String message) {
    showAppToast(context, message, type: AppToastType.error);
  }
}

class _TaskBatchRow extends StatelessWidget {
  const _TaskBatchRow({
    required this.selectedCount,
    required this.totalCount,
    required this.allSelected,
    required this.someSelected,
    required this.onToggleSelectAll,
  });

  final int selectedCount;
  final int totalCount;
  final bool allSelected;
  final bool someSelected;
  final VoidCallback onToggleSelectAll;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);

    return Container(
      height: 48,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        children: [
          GestureDetector(
            onTap: onToggleSelectAll,
            child: Row(
              children: [
                shad.Checkbox(
                  state: allSelected
                      ? shad.CheckboxState.checked
                      : someSelected
                      ? shad.CheckboxState.indeterminate
                      : shad.CheckboxState.unchecked,
                  onChanged: (_) => onToggleSelectAll(),
                  borderColor: palette.border,
                  backgroundColor: palette.cardBg,
                  size: 20,
                ),
                const SizedBox(width: 14),
                Text(
                  context.l10n.selectAll.toUpperCase(),
                  style: TextStyle(
                    color: palette.textMuted,
                    fontSize: 11,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 1.1,
                  ),
                ),
              ],
            ),
          ),
          const Spacer(),
          Text(
            context.l10n.selectedCount(selectedCount, totalCount).toUpperCase(),
            style: TextStyle(color: palette.textMuted, fontSize: 11, fontWeight: FontWeight.w700, letterSpacing: 1.1),
          ),
        ],
      ),
    );
  }
}

class _MobileTasksView extends StatelessWidget {
  const _MobileTasksView({
    required this.tasks,
    required this.batchMode,
    required this.batchSelection,
    required this.onToggleSelectAll,
    required this.onPauseSelected,
    required this.onResumeSelected,
    required this.onDeleteSelected,
    required this.onTaskPressed,
    required this.onToggleBatch,
    required this.onTaskPause,
    required this.onTaskResume,
    required this.onTaskDelete,
    required this.onTaskOpen,
    required this.onTaskReveal,
    required this.contextActionsForTask,
    required this.pendingUpdateTaskId,
    this.emptyStateMessage,
    this.loading = false,
  });

  final List<TaskRecord> tasks;
  final bool batchMode;
  final TaskBatchSelectionController batchSelection;
  final VoidCallback onToggleSelectAll;
  final VoidCallback onPauseSelected;
  final VoidCallback onResumeSelected;
  final VoidCallback onDeleteSelected;
  final ValueChanged<TaskRecord> onTaskPressed;
  final ValueChanged<String> onToggleBatch;
  final ValueChanged<TaskRecord> onTaskPause;
  final ValueChanged<TaskRecord> onTaskResume;
  final ValueChanged<TaskRecord> onTaskDelete;
  final ValueChanged<TaskRecord> onTaskOpen;
  final ValueChanged<TaskRecord> onTaskReveal;
  final TaskContextActions Function(TaskRecord task, bool selected, bool allSelected) contextActionsForTask;
  final String? pendingUpdateTaskId;
  final String? emptyStateMessage;
  final bool loading;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final visibleTaskIds = tasks.map((task) => task.id).toList(growable: false);

    return ColoredBox(
      color: palette.bg,
      child: Column(
        children: [
          AnimatedSize(
            duration: const Duration(milliseconds: 180),
            curve: Curves.easeOutCubic,
            child: batchMode
                ? ListenableBuilder(
                    listenable: batchSelection,
                    builder: (context, _) {
                      final allSelected = batchSelection.areAllSelected(visibleTaskIds);
                      final availability = _batchActionAvailability(tasks, batchSelection);
                      return _MobileBatchToolbar(
                        selectedCount: batchSelection.count,
                        visibleCount: tasks.length,
                        allSelected: allSelected,
                        someSelected: batchSelection.isNotEmpty && !allSelected,
                        canPauseSelected: availability.canPause,
                        canResumeSelected: availability.canResume,
                        onToggleSelectAll: onToggleSelectAll,
                        onPauseSelected: onPauseSelected,
                        onResumeSelected: onResumeSelected,
                        onDeleteSelected: onDeleteSelected,
                      );
                    },
                  )
                : const SizedBox.shrink(),
          ),
          Expanded(
            child: loading
                ? const Center(child: shad.CircularProgressIndicator())
                : tasks.isEmpty
                ? TaskEmptyState(message: emptyStateMessage)
                : ListView.separated(
                    padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
                    itemCount: tasks.length,
                    separatorBuilder: (_, _) => const SizedBox(height: 6),
                    itemBuilder: (context, index) {
                      final task = tasks[index];
                      return TaskBatchSelectionBuilder(
                        controller: batchSelection,
                        taskId: task.id,
                        visibleTaskIds: visibleTaskIds,
                        builder: (context, selectedInBatch, allSelected) => TaskCard(
                          task: task,
                          selected: false,
                          batchMode: batchMode,
                          selectedInBatch: selectedInBatch,
                          onPressed: () => onTaskPressed(task),
                          onToggleBatch: () => onToggleBatch(task.id),
                          onPause: () => onTaskPause(task),
                          onResume: () => onTaskResume(task),
                          onDelete: () => onTaskDelete(task),
                          onOpen: () => onTaskOpen(task),
                          onReveal: () => onTaskReveal(task),
                          contextActions: contextActionsForTask(task, selectedInBatch, allSelected),
                          listeningForUpdate: pendingUpdateTaskId == task.id,
                        ),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}

class _MobileTasksHeader extends StatelessWidget {
  const _MobileTasksHeader({
    required this.batchMode,
    required this.selectedCount,
    required this.onAddTask,
    required this.onToggleBatchMode,
  });

  final bool batchMode;
  final int selectedCount;
  final VoidCallback onAddTask;
  final VoidCallback onToggleBatchMode;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(
      decoration: BoxDecoration(
        color: palette.bg,
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: SafeArea(
        bottom: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  context.l10n.taskList,
                  style: TextStyle(color: palette.textPrimary, fontSize: 22, fontWeight: FontWeight.w800),
                ),
              ),
              _MobileHeaderIconButton(icon: Icons.add_rounded, onPressed: onAddTask),
              const SizedBox(width: 8),
              _MobileHeaderIconButton(
                icon: Icons.checklist_rtl_outlined,
                active: batchMode,
                badge: batchMode && selectedCount > 0 ? selectedCount.toString() : null,
                onPressed: onToggleBatchMode,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _MobileBatchToolbar extends StatelessWidget {
  const _MobileBatchToolbar({
    required this.selectedCount,
    required this.visibleCount,
    required this.allSelected,
    required this.someSelected,
    required this.canPauseSelected,
    required this.canResumeSelected,
    required this.onToggleSelectAll,
    required this.onPauseSelected,
    required this.onResumeSelected,
    required this.onDeleteSelected,
  });

  final int selectedCount;
  final int visibleCount;
  final bool allSelected;
  final bool someSelected;
  final bool canPauseSelected;
  final bool canResumeSelected;
  final VoidCallback onToggleSelectAll;
  final VoidCallback onPauseSelected;
  final VoidCallback onResumeSelected;
  final VoidCallback onDeleteSelected;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(
      height: 48,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      decoration: BoxDecoration(
        color: palette.sideBg,
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        children: [
          GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: onToggleSelectAll,
            child: Row(
              children: [
                shad.Checkbox(
                  state: allSelected
                      ? shad.CheckboxState.checked
                      : someSelected
                      ? shad.CheckboxState.indeterminate
                      : shad.CheckboxState.unchecked,
                  onChanged: (_) => onToggleSelectAll(),
                  borderColor: palette.border,
                  backgroundColor: palette.cardBg,
                ),
                const SizedBox(width: AppDesignTokens.checkboxLabelGap),
                Text(
                  '$selectedCount / $visibleCount',
                  style: TextStyle(color: palette.textSecondary, fontSize: 12, fontWeight: FontWeight.w700),
                ),
              ],
            ),
          ),
          const Spacer(),
          _MobileIconAction(icon: Icons.pause_outlined, onPressed: onPauseSelected, enabled: canPauseSelected),
          const SizedBox(width: 6),
          _MobileIconAction(icon: Icons.play_arrow_outlined, onPressed: onResumeSelected, enabled: canResumeSelected),
          const SizedBox(width: 6),
          _MobileIconAction(icon: Icons.delete_outline, onPressed: onDeleteSelected, enabled: selectedCount > 0),
        ],
      ),
    );
  }
}

class _MobileHeaderIconButton extends StatelessWidget {
  const _MobileHeaderIconButton({required this.icon, required this.onPressed, this.active = false, this.badge});

  final IconData icon;
  final VoidCallback onPressed;
  final bool active;
  final String? badge;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTap: onPressed,
      child: Container(
        width: 34,
        height: 34,
        decoration: BoxDecoration(
          color: active ? palette.itemActiveBg : palette.cardBg,
          borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
          border: Border.all(color: active ? palette.textPrimary : palette.border),
        ),
        child: Stack(
          clipBehavior: Clip.none,
          children: [
            Center(child: Icon(icon, size: 18, color: active ? palette.textPrimary : palette.textSecondary)),
            if (badge != null)
              Positioned(
                right: -5,
                top: -5,
                child: Container(
                  constraints: const BoxConstraints(minWidth: 16),
                  height: 16,
                  padding: const EdgeInsets.symmetric(horizontal: 4),
                  decoration: BoxDecoration(color: palette.brand, borderRadius: BorderRadius.circular(999)),
                  child: Center(
                    child: Text(
                      badge!,
                      style: TextStyle(color: palette.brandForeground, fontSize: 9, fontWeight: FontWeight.w800),
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _MobileIconAction extends StatelessWidget {
  const _MobileIconAction({required this.icon, required this.onPressed, required this.enabled});

  final IconData icon;
  final VoidCallback onPressed;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTap: enabled ? onPressed : null,
      child: Container(
        width: 32,
        height: 32,
        decoration: BoxDecoration(
          color: enabled ? palette.cardBg : Colors.transparent,
          borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
          border: Border.all(color: enabled ? palette.border : Colors.transparent),
        ),
        child: Icon(icon, size: 18, color: enabled ? palette.textSecondary : palette.textMuted),
      ),
    );
  }
}
