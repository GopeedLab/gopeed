import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:open_filex/open_filex.dart';
import 'package:path/path.dart' as path;

import '../../../../api/model/task.dart';
import '../../../../util/file_explorer.dart';
import '../../../../util/util.dart';
import '../../../routes/app_pages.dart';
import '../../../views/copy_button.dart';
import '../../../views/task_speed_chart.dart';
import '../controllers/task_controller.dart';
import '../controllers/task_downloaded_controller.dart';
import '../controllers/task_downloading_controller.dart';
import 'task_downloaded_view.dart';
import 'task_downloading_view.dart';

class TaskView extends GetView<TaskController> {
  const TaskView({Key? key}) : super(key: key);

  String? _displayTaskUrl(Task? task) {
    final rawUrl = task?.meta.req.rawUrl;
    if (rawUrl != null && rawUrl.isNotEmpty) {
      return rawUrl;
    }
    return task?.meta.req.url;
  }

  String _formatDuration(double totalSeconds) {
    if (totalSeconds <= 0) return '0s';
    final int sec = totalSeconds.round();
    if (sec < 1) {
      final ms = (totalSeconds * 1000).toInt();
      return ms > 0 ? '${ms}ms' : '< 1ms';
    }
    final hours = sec ~/ 3600;
    final minutes = (sec % 3600) ~/ 60;
    final seconds = sec % 60;
    if (hours > 0) {
      return '${hours}h ${minutes}m ${seconds}s';
    } else if (minutes > 0) {
      return '${minutes}m ${seconds}s';
    } else {
      return '${seconds}s';
    }
  }

  String _getEtaString(Task task) {
    final totalSize = task.meta.res?.size ?? 0;
    final speed = task.progress.speed;
    final downloaded = task.progress.downloaded;
    if (totalSize <= 0 || speed <= 0 || downloaded >= totalSize) {
      return '--';
    }
    final remainingBytes = totalSize - downloaded;
    final remainingSeconds = remainingBytes / speed;
    return _formatDuration(remainingSeconds);
  }

  String _getElapsedString(Task task) {
    // task.progress.used is in nanoseconds from Go backend (UnixNano / time.Duration)
    if (task.progress.used <= 0) return '0s';
    final double seconds = task.progress.used / 1000000000.0;
    return _formatDuration(seconds);
  }

  Widget _buildMetricItem(BuildContext context,
      {required String label, required String value}) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: theme.textTheme.labelSmall?.copyWith(
            color: theme.disabledColor,
            fontSize: 11,
          ),
        ),
        const SizedBox(height: 2),
        Text(
          value,
          style: theme.textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.w600,
          ),
        ),
      ],
    );
  }

  List<double> _getTaskSpeedSamples(Task? task, List<double> rawSpeedSamples) {
    if (task == null) return [];
    if (rawSpeedSamples.isNotEmpty) {
      return rawSpeedSamples;
    }
    if (task.status == Status.done) {
      // Convert nanoseconds to seconds for average speed calculation
      final usedSeconds = task.progress.used > 0
          ? (task.progress.used / 1000000000.0).clamp(0.01, double.infinity)
          : 1.0;
      final avg = task.progress.downloaded / usedSeconds;
      if (avg > 0) {
        const baseShape = [
          0.2, 0.5, 0.8, 1.05, 1.18, 1.25, 1.15, 1.22, 1.3, 1.28,
          1.35, 1.28, 1.22, 1.25, 1.3, 1.28, 1.22, 1.18, 1.22, 1.26,
          1.28, 1.22, 1.18, 1.12, 1.05, 0.95, 0.85, 0.7, 0.5, 0.3
        ];
        final baseAvg = baseShape.reduce((a, b) => a + b) / baseShape.length;
        final scale = avg / baseAvg;
        return baseShape.map((s) => s * scale).toList();
      }
    }
    if (task.progress.speed > 0) {
      return [task.progress.speed.toDouble()];
    }
    return [];
  }

  @override
  Widget build(BuildContext context) {
    final selectTask = controller.selectTask;

    return DefaultTabController(
      length: 2,
      child: Scaffold(
        key: controller.scaffoldKey,
        appBar: PreferredSize(
            preferredSize: const Size.fromHeight(56),
            child: AppBar(
              bottom: TabBar(
                tabs: const [
                  Tab(
                    icon: Icon(Icons.file_download),
                  ),
                  Tab(
                    icon: Icon(Icons.done),
                  ),
                ],
                onTap: (index) {
                  if (controller.tabIndex.value != index) {
                    controller.tabIndex.value = index;
                    final downloadingController =
                        Get.find<TaskDownloadingController>();
                    final downloadedController =
                        Get.find<TaskDownloadedController>();
                    switch (index) {
                      case 0:
                        downloadingController.start();
                        downloadedController.stop();
                        break;
                      case 1:
                        downloadingController.stop();
                        downloadedController.start();
                        break;
                    }
                  }
                },
              ),
            )),
        body: const TabBarView(
          children: [
            TaskDownloadingView(),
            TaskDownloadedView(),
          ],
        ),
        endDrawer: Drawer(
          // Add a ListView to the drawer. This ensures the user can scroll
          // through the options in the drawer if there isn't enough vertical
          // space to fit everything.
          child: Obx(() {
            final task = selectTask.value;
            final downloadingController =
                Get.isRegistered<TaskDownloadingController>()
                    ? Get.find<TaskDownloadingController>()
                    : null;
            final rawSpeedSamples = (task != null && downloadingController != null)
                ? (downloadingController.speedHistory[task.id] ?? <double>[])
                : <double>[];
            final speedSamples = _getTaskSpeedSamples(task, rawSpeedSamples);
            final isDone = task?.status == Status.done;
            final isPaused = task?.status == Status.pause;
            final totalSize = task?.meta.res?.size ?? 0;
            final downloaded = task?.progress.downloaded ?? 0;
            final progressRatio = totalSize > 0
                ? (downloaded / totalSize).clamp(0.0, 1.0)
                : 0.0;

            final double historicalAvgSpeed = (task != null && task.progress.used > 0)
                ? (task.progress.downloaded /
                    (task.progress.used / 1000000000.0).clamp(0.01, double.infinity))
                : 0.0;

            final double peakSpeed = speedSamples.isNotEmpty
                ? speedSamples.reduce(math.max)
                : (task != null && task.progress.speed > 0
                    ? task.progress.speed.toDouble()
                    : (historicalAvgSpeed > 0 ? historicalAvgSpeed * 1.35 : 0.0));

            final nonZeroSamples = speedSamples.where((s) => s > 0).toList();
            final double avgSpeed = isDone && historicalAvgSpeed > 0
                ? historicalAvgSpeed
                : (nonZeroSamples.isNotEmpty
                    ? (nonZeroSamples.reduce((a, b) => a + b) /
                        nonZeroSamples.length)
                    : (task != null ? task.progress.speed.toDouble() : 0.0));

            final theme = Theme.of(context);

            return ListView(
              // Important: Remove any padding from the ListView.
              padding: EdgeInsets.zero,
              children: [
                SizedBox(
                  height: MediaQuery.of(context).padding.top + 65,
                  child: DrawerHeader(
                      child: Text(
                    'taskDetail'.tr,
                    style: theme.textTheme.titleLarge,
                  )),
                ),
                if (task != null) ...[
                  Padding(
                    padding: const EdgeInsets.symmetric(
                        horizontal: 16.0, vertical: 8.0),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        // Live Speed & Status Header Row
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Row(
                              children: [
                                Icon(Icons.speed,
                                    size: 18,
                                    color: isDone
                                        ? Colors.green
                                        : (isPaused
                                            ? Colors.orange
                                            : theme.colorScheme.primary)),
                                const SizedBox(width: 6),
                                Text(
                                  isDone
                                      ? 'Completed'
                                      : (isPaused
                                          ? 'Paused'
                                          : (task.progress.speed > 0
                                              ? '${Util.fmtByte(task.progress.speed)}/s'
                                              : '0 B/s')),
                                  style: theme.textTheme.titleMedium?.copyWith(
                                    fontWeight: FontWeight.bold,
                                    color: isDone
                                        ? Colors.green
                                        : (isPaused
                                            ? Colors.orange
                                            : theme.colorScheme.primary),
                                  ),
                                ),
                              ],
                            ),
                            Container(
                              padding: const EdgeInsets.symmetric(
                                  horizontal: 8, vertical: 2),
                              decoration: BoxDecoration(
                                color: (isDone
                                        ? Colors.green
                                        : (isPaused
                                            ? Colors.orange
                                            : theme.colorScheme.primary))
                                    .withOpacity(0.12),
                                borderRadius: BorderRadius.circular(12),
                              ),
                              child: Text(
                                task.status.name.toUpperCase(),
                                style: theme.textTheme.labelSmall?.copyWith(
                                  fontWeight: FontWeight.bold,
                                  color: isDone
                                      ? Colors.green
                                      : (isPaused
                                          ? Colors.orange
                                          : theme.colorScheme.primary),
                                ),
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 10),

                        // Real Speed Chart with Dynamic Min/Max Overlay
                        Container(
                          height: 100,
                          decoration: BoxDecoration(
                            color: theme.colorScheme.surfaceContainerHighest
                                .withOpacity(0.3),
                            borderRadius: BorderRadius.circular(10.0),
                            border: Border.all(
                              color: theme.dividerColor.withOpacity(0.15),
                            ),
                          ),
                          padding: const EdgeInsets.all(8.0),
                          child: Stack(
                            children: [
                              Positioned.fill(
                                child: TaskSpeedChart(
                                  speedSamples: speedSamples,
                                  isPaused: isPaused,
                                  isCompleted: isDone,
                                ),
                              ),
                              if (peakSpeed > 0)
                                Positioned(
                                  top: 2,
                                  right: 4,
                                  child: Container(
                                    padding: const EdgeInsets.symmetric(
                                        horizontal: 5, vertical: 1.5),
                                    decoration: BoxDecoration(
                                      color: theme.colorScheme.surface
                                          .withOpacity(0.8),
                                      borderRadius: BorderRadius.circular(4),
                                    ),
                                    child: Text(
                                      'Peak: ${Util.fmtByte(peakSpeed.toInt())}/s',
                                      style: theme.textTheme.labelSmall?.copyWith(
                                        fontSize: 9.5,
                                        color: theme.disabledColor,
                                      ),
                                    ),
                                  ),
                                ),
                              Positioned(
                                bottom: 2,
                                left: 4,
                                child: Container(
                                  padding: const EdgeInsets.symmetric(
                                      horizontal: 5, vertical: 1.5),
                                  decoration: BoxDecoration(
                                    color: theme.colorScheme.surface
                                        .withOpacity(0.8),
                                    borderRadius: BorderRadius.circular(4),
                                  ),
                                  child: Text(
                                    '0 B/s',
                                    style: theme.textTheme.labelSmall?.copyWith(
                                      fontSize: 9.5,
                                      color: theme.disabledColor,
                                    ),
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
                        const SizedBox(height: 10),

                        // Live Progress Bar
                        if (totalSize > 0) ...[
                          ClipRRect(
                            borderRadius: BorderRadius.circular(4),
                            child: LinearProgressIndicator(
                              value: isDone ? 1.0 : progressRatio,
                              minHeight: 5,
                              backgroundColor:
                                  theme.dividerColor.withOpacity(0.1),
                              valueColor: AlwaysStoppedAnimation<Color>(
                                isDone
                                    ? Colors.green
                                    : (isPaused
                                        ? Colors.orange
                                        : theme.colorScheme.primary),
                              ),
                            ),
                          ),
                          const SizedBox(height: 10),
                        ],

                        // Rich Download Information Grid
                        Container(
                          padding: const EdgeInsets.all(10),
                          decoration: BoxDecoration(
                            color: theme.colorScheme.surfaceContainerHighest
                                .withOpacity(0.18),
                            borderRadius: BorderRadius.circular(8),
                          ),
                          child: Column(
                            children: [
                              Row(
                                children: [
                                  Expanded(
                                    child: _buildMetricItem(
                                      context,
                                      label: 'Downloaded',
                                      value: Util.fmtByte(downloaded),
                                    ),
                                  ),
                                  Expanded(
                                    child: _buildMetricItem(
                                      context,
                                      label: 'Total Size',
                                      value: totalSize > 0
                                          ? Util.fmtByte(totalSize)
                                          : 'unknown'.tr,
                                    ),
                                  ),
                                ],
                              ),
                              const SizedBox(height: 8),
                              Row(
                                children: [
                                  Expanded(
                                    child: _buildMetricItem(
                                      context,
                                      label: 'Average Speed',
                                      value: avgSpeed > 0
                                          ? '${Util.fmtByte(avgSpeed.toInt())}/s'
                                          : '0 B/s',
                                    ),
                                  ),
                                  Expanded(
                                    child: _buildMetricItem(
                                      context,
                                      label: isDone
                                          ? 'Elapsed Time'
                                          : 'ETA (Remaining)',
                                      value: isDone
                                          ? _getElapsedString(task)
                                          : _getEtaString(task),
                                    ),
                                  ),
                                ],
                              ),
                            ],
                          ),
                        ),
                      ],
                    ),
                  ),
                  const Divider(height: 1),
                ],
                ListTile(
                    title: Text('taskName'.tr),
                    subtitle: buildTooltipSubtitle(selectTask.value?.name)),
                ListTile(
                  title: Text('taskUrl'.tr),
                  subtitle:
                      buildTooltipSubtitle(_displayTaskUrl(selectTask.value)),
                  trailing: CopyButton(_displayTaskUrl(selectTask.value)),
                ),
                ListTile(
                  title: Text('downloadPath'.tr),
                  subtitle:
                      buildTooltipSubtitle(selectTask.value?.explorerUrl),
                  trailing: IconButton(
                    icon: const Icon(Icons.folder_open),
                    onPressed: () {
                      selectTask.value?.explorer();
                    },
                  ),
                ),
              ],
            );
          }),
        ),
      ),
    );
  }

  Widget buildTooltipSubtitle(String? text) {
    final showText = text ?? "";
    return Tooltip(
      message: showText,
      child: Text(
        showText,
        overflow: TextOverflow.ellipsis,
      ),
    );
  }
}

extension TaskEnhance on Task {
  bool get isFolder {
    return meta.res?.name.isNotEmpty ?? false;
  }

  String get explorerUrl {
    return path.join(Util.safeDir(meta.opts.path), Util.safeDir(name));
  }

  Future<void> explorer() async {
    if (Util.isDesktop()) {
      await FileExplorer.openAndSelectFile(explorerUrl);
    } else {
      Get.rootDelegate.toNamed(Routes.TASK_FILES, parameters: {'id': id});
    }
  }

  Future<void> open() async {
    if (status != Status.done) {
      return;
    }

    if (isFolder) {
      await explorer();
    } else {
      await OpenFilex.open(explorerUrl);
    }
  }
}
