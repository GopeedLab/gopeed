import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../api/model/create_task.dart';
import '../../application/app_deep_link_controller.dart';
import '../../application/app_notification_controller.dart';
import '../../application/app_runtime_controller.dart';
import '../../application/app_platform_controller.dart';
import '../../../features/settings/presentation/widgets/app_update_dialog.dart';
import '../../../features/tasks/application/pending_create_task.dart';
import '../../../features/tasks/application/pending_update_task.dart';
import '../../../features/tasks/application/tasks_controller.dart';
import '../../../features/tasks/presentation/widgets/pending_update_dialog.dart';
import '../../../core/window/app_window_launcher.dart';
import '../../../shared/widgets/app_toast.dart';

class MainShell extends ConsumerStatefulWidget {
  const MainShell({super.key, required this.child});

  final Widget child;

  @override
  ConsumerState<MainShell> createState() => _MainShellState();
}

class _MainShellState extends ConsumerState<MainShell> {
  String? _shownUpdateVersion;
  bool _handlingPendingUpdateRequest = false;

  @override
  Widget build(BuildContext context) {
    ref.listen(appPlatformControllerProvider, (_, next) {
      if (!kReleaseMode) return;
      final update = next.value;
      final version = update?.latestVersion;
      if (update?.updateStatus != AppUpdateStatus.available ||
          version == null ||
          _shownUpdateVersion == version.version) {
        return;
      }
      _shownUpdateVersion = version.version;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        unawaited(
          showAppUpdateDialog(
            context,
            versionInfo: version,
            onUpdate: ref.read(appPlatformControllerProvider.notifier).startUpdate,
          ),
        );
      });
    });
    ref.listen(pendingUpdateRequestProvider, (_, next) {
      if (next != null) {
        _schedulePendingUpdateRequest(next);
      }
    });
    final runtime = ref.watch(appRuntimeControllerProvider);
    if (runtime.hasValue) {
      ref.watch(appPlatformControllerProvider);
      ref.watch(appDeepLinkControllerProvider);
      ref.watch(appNotificationControllerProvider);
    }
    // Keep the nested Navigator mounted even while the runtime is starting.
    return widget.child;
  }

  void _schedulePendingUpdateRequest(PendingUpdateRequest request) {
    if (_handlingPendingUpdateRequest) return;
    _handlingPendingUpdateRequest = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) {
        _handlingPendingUpdateRequest = false;
        return;
      }
      unawaited(_handlePendingUpdateRequest(request));
    });
  }

  Future<void> _handlePendingUpdateRequest(PendingUpdateRequest request) async {
    try {
      final decision = await showPendingUpdateDialog(context, taskName: request.task.name);
      if (!mounted) return;
      _clearPendingUpdateRequest(request);
      switch (decision) {
        case PendingUpdateDecision.updateTask:
          final updateRequest = request.createTask.req;
          if (updateRequest == null) return;
          await ref.read(tasksControllerProvider.notifier).updateRequest(request.task.id, updateRequest);
          final listeningTask = ref.read(pendingUpdateTaskProvider);
          if (listeningTask?.id == request.task.id) {
            ref.read(pendingUpdateTaskProvider.notifier).clear();
          }
          break;
        case PendingUpdateDecision.createTask:
          await _openCapturedCreateTask(request.createTask);
          break;
        case PendingUpdateDecision.cancel || null:
          break;
      }
    } catch (error) {
      if (mounted) {
        showAppToast(context, error.toString(), type: AppToastType.error);
      }
    } finally {
      _handlingPendingUpdateRequest = false;
      if (mounted) {
        _clearPendingUpdateRequest(request);
        final nextRequest = ref.read(pendingUpdateRequestProvider);
        if (nextRequest != null) {
          _schedulePendingUpdateRequest(nextRequest);
        }
      }
    }
  }

  void _clearPendingUpdateRequest(PendingUpdateRequest request) {
    if (identical(ref.read(pendingUpdateRequestProvider), request)) {
      ref.read(pendingUpdateRequestProvider.notifier).clear();
    }
  }

  Future<void> _openCapturedCreateTask(CreateTask createTask) async {
    final opened = await AppWindowLauncher.openCreateTaskWindow(createTask: createTask);
    if (!opened && mounted) {
      ref.read(pendingCreateTaskProvider.notifier).set(createTask);
      context.go('/create');
    }
  }
}
