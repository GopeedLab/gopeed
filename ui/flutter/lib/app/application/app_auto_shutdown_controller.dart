import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/model/task.dart' as api_task;
import '../../core/capabilities/app_capabilities.dart';
import '../../core/common/task_event.dart';
import '../../core/libgopeed_boot.dart';
import '../../util/log_util.dart';
import '../../util/util.dart';
import 'app_platform_controller.dart';
import 'app_runtime_controller.dart';

final appAutoShutdownControllerProvider = AsyncNotifierProvider<AppAutoShutdownController, AppAutoShutdownState>(
  AppAutoShutdownController.new,
);

/// Extraction states that mean the archive is still being unpacked. Quitting
/// while one of them is set loses the extraction, so it blocks the shutdown.
const _pendingExtractStatuses = {
  api_task.ExtractStatus.queued,
  api_task.ExtractStatus.extracting,
  api_task.ExtractStatus.waitingParts,
};

/// One sample of the task list.
class TaskActivity {
  const TaskActivity({required this.activeDownloads, required this.pendingExtractions});

  /// Tasks in the running or waiting state.
  final int activeDownloads;

  /// Archives that are queued for, or still in, extraction.
  final int pendingExtractions;
}

class AppAutoShutdownState {
  const AppAutoShutdownState({this.armed = false});

  /// Whether a terminal task event can currently trigger the shutdown.
  final bool armed;
}

/// Closes the app once no download is left to finish.
///
/// The controller is driven by terminal task events instead of polling the
/// queue: a done or error event is the only moment the last task can complete.
/// Each event schedules a sample of the task list, and the app exits when that
/// sample finds nothing left to do.
class AppAutoShutdownController extends AsyncNotifier<AppAutoShutdownState> {
  StreamSubscription<TaskEvent>? _subscription;
  Timer? _timer;
  bool _exiting = false;
  int _pollAttempts = 0;

  @override
  Future<AppAutoShutdownState> build() async {
    // Only this switch decides whether the controller listens at all, so an
    // unrelated config save does not drop a pending shutdown check.
    final enabled = ref.watch(
      appRuntimeControllerProvider.select(
        (runtime) => runtime.value?.downloaderConfig.extra.autoCloseWhenAllTasksDone ?? false,
      ),
    );
    ref.onDispose(_stop);
    _stop();
    if (!enabled || kIsWeb || !Util.isDesktop()) {
      return const AppAutoShutdownState();
    }
    _subscription = taskEvents.listen(_onTaskEvent);
    return const AppAutoShutdownState(armed: true);
  }

  void _onTaskEvent(TaskEvent event) {
    if (_exiting) return;
    if (event.type != TaskEventType.done && event.type != TaskEventType.error) return;
    _pollAttempts = 0;
    _schedule(settleDelay);
  }

  void _schedule(Duration delay) {
    _timer?.cancel();
    _timer = Timer(delay, () => unawaited(_check()));
  }

  Future<void> _check() async {
    _timer = null;
    if (_exiting) return;
    final activity = await _sampleActivity();
    if (activity == null) return;
    if (activity.activeDownloads > 0) {
      // Another task is still downloading, its own terminal event re-arms us.
      return;
    }
    if (activity.pendingExtractions > 0) {
      if (_pollAttempts >= maxPollAttempts) {
        logger.w('auto shutdown: extraction did not finish in time, keep the app open');
        return;
      }
      _pollAttempts++;
      _schedule(pollInterval);
      return;
    }
    await shutdown();
  }

  /// Reads the task list, reporting a failed sample as null so a broken
  /// backend never turns into an unwanted exit.
  Future<TaskActivity?> _sampleActivity() async {
    try {
      return await readActivity();
    } catch (error, stackTrace) {
      logger.w('auto shutdown: read task activity failed', error, stackTrace);
      return null;
    }
  }

  /// The task list as the queue sees it right now.
  @visibleForTesting
  Future<TaskActivity> readActivity() async {
    final tasks = await ref.read(gopeedServiceProvider).getTasks(api_task.Status.values);
    var activeDownloads = 0;
    var pendingExtractions = 0;
    for (final task in tasks) {
      if (task.status == api_task.Status.running || task.status == api_task.Status.wait) {
        activeDownloads++;
      }
      if (_pendingExtractStatuses.contains(task.progress.extractStatus)) {
        pendingExtractions++;
      }
    }
    return TaskActivity(activeDownloads: activeDownloads, pendingExtractions: pendingExtractions);
  }

  /// Exits through the same path as the tray menu, which stops the services
  /// before destroying the window.
  @visibleForTesting
  Future<void> shutdown() async {
    if (_exiting) return;
    _exiting = true;
    _stop();
    logger.i('auto shutdown: all download tasks finished, exiting');
    try {
      await ref.read(appPlatformControllerProvider.notifier).exitApp();
    } catch (error, stackTrace) {
      _exiting = false;
      logger.e('auto shutdown: exit failed', error, stackTrace);
    }
  }

  void _stop() {
    unawaited(_subscription?.cancel());
    _subscription = null;
    _timer?.cancel();
    _timer = null;
  }

  @visibleForTesting
  Stream<TaskEvent> get taskEvents => LibgopeedBoot.instance.taskEvents;

  /// Delay after a terminal event, so the backend can queue the follow-up work
  /// it starts once a task is done (extraction, BT task from a .torrent file).
  @visibleForTesting
  Duration get settleDelay => const Duration(seconds: 2);

  /// Re-check interval while an archive is being extracted.
  @visibleForTesting
  Duration get pollInterval => const Duration(seconds: 3);

  /// Upper bound of the extraction wait, about one hour.
  @visibleForTesting
  int get maxPollAttempts => 1200;
}
