import 'package:flutter/material.dart';
import 'package:flutter_context_menu/flutter_context_menu.dart';
import 'package:get/get.dart';
import 'package:styled_widget/styled_widget.dart';

import '../../api/api.dart';
import '../../api/model/request.dart';
import '../../api/model/resolve_task.dart';
import '../../api/model/task.dart';
import '../../util/message.dart';
import '../../util/util.dart';
import '../modules/app/controllers/app_controller.dart';
import '../modules/task/controllers/task_controller.dart';
import '../modules/task/controllers/task_downloaded_controller.dart';
import '../modules/task/controllers/task_downloading_controller.dart';
import '../modules/task/views/task_view.dart';
import '../routes/app_pages.dart';
import 'file_icon.dart';

class BuildTaskListView extends GetView {
  final List<Task> tasks;
  final List<String> selectedTaskIds;

  const BuildTaskListView({
    Key? key,
    required this.tasks,
    required this.selectedTaskIds,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
        floatingActionButton: FloatingActionButton(
          onPressed: () {
            Get.rootDelegate.toNamed(Routes.CREATE);
          },
          tooltip: 'create'.tr,
          child: const Icon(Icons.add),
        ),
        body: Obx(() {
          return buildTaskList(context, tasks);
        }));
  }

  Widget buildTaskList(BuildContext context, tasks) {
    return ListView.builder(
      itemCount: tasks.length + 1,
      itemBuilder: (context, index) {
        if (index == tasks.length) {
          return const SizedBox(height: 75);
        }
        return item(context, tasks[index]);
      },
    );
  }

  Widget item(BuildContext context, Task task) {
    bool isDone() {
      return task.status == Status.done;
    }

    bool isRunning() {
      return task.status == Status.running;
    }

    bool isSelect() {
      return selectedTaskIds.contains(task.id);
    }

    bool isFolderTask() {
      return task.isFolder;
    }

    Future<void> showDeleteDialog(List<String> ids) {
      final appController = Get.find<AppController>();

      final context = Get.context!;

      return showDialog<void>(
          context: context,
          barrierDismissible: false,
          builder: (_) => AlertDialog(
                title: Text(
                    'deleteTask'.trParams({'count': ids.length.toString()})),
                content: Obx(() => CheckboxListTile(
                    value: appController
                        .downloaderConfig.value.extra.lastDeleteTaskKeep,
                    title: Text('deleteTaskTip'.tr,
                        style: context.textTheme.bodyLarge),
                    onChanged: (v) {
                      appController.downloaderConfig.update((val) {
                        val!.extra.lastDeleteTaskKeep = v!;
                      });
                    })),
                actions: [
                  TextButton(
                    child: Text('cancel'.tr),
                    onPressed: () => Get.back(),
                  ),
                  TextButton(
                    child: Text(
                      'confirm'.tr,
                      style: const TextStyle(color: Colors.redAccent),
                    ),
                    onPressed: () async {
                      try {
                        final force = !appController
                            .downloaderConfig.value.extra.lastDeleteTaskKeep;
                        await appController.saveConfig();
                        await deleteTasks(ids, force);
                        Get.back();
                      } catch (e) {
                        showErrorMessage(e);
                      }
                    },
                  ),
                ],
              ));
    }

    Future<void> showUpdateUrlDialog(BuildContext context, Task task) async {
      final urlController = TextEditingController(text: task.meta.req.url);
      final headerControllers =
          <MapEntry<TextEditingController, TextEditingController>>[];

      // Initialize with existing headers if available
      if (task.meta.req.extra != null && task.meta.req.extra is Map) {
        final extra = task.meta.req.extra as Map<String, dynamic>;
        if (extra.containsKey('header') && extra['header'] is Map) {
          final headers = extra['header'] as Map<String, dynamic>;
          for (final entry in headers.entries) {
            headerControllers.add(MapEntry(
              TextEditingController(text: entry.key),
              TextEditingController(text: entry.value.toString()),
            ));
          }
        }
      }

      // Add one empty header row by default if none exists
      if (headerControllers.isEmpty) {
        headerControllers.add(MapEntry(
          TextEditingController(),
          TextEditingController(),
        ));
      }

      return showDialog<void>(
        context: context,
        barrierDismissible: false,
        builder: (_) => StatefulBuilder(
          builder: (context, setState) {
            return AlertDialog(
              title: Text('updateUrl'.tr),
              content: SizedBox(
                width: 400,
                child: SingleChildScrollView(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      TextField(
                        controller: urlController,
                        decoration: InputDecoration(
                          labelText: 'downloadLink'.tr,
                          hintText: 'updateUrlDialogHint'.tr,
                          icon: const Icon(Icons.link),
                        ),
                      ),
                      const SizedBox(height: 16),
                      Text('httpHeader'.tr,
                          style: Theme.of(context).textTheme.titleMedium),
                      const SizedBox(height: 8),
                      ...headerControllers.asMap().entries.map((entry) {
                        final index = entry.key;
                        final controllers = entry.value;
                        return Padding(
                          padding: const EdgeInsets.only(bottom: 8),
                          child: Row(
                            children: [
                              Expanded(
                                child: TextField(
                                  controller: controllers.key,
                                  decoration: InputDecoration(
                                    hintText: 'httpHeaderName'.tr,
                                    isDense: true,
                                  ),
                                ),
                              ),
                              const SizedBox(width: 8),
                              Expanded(
                                child: TextField(
                                  controller: controllers.value,
                                  decoration: InputDecoration(
                                    hintText: 'httpHeaderValue'.tr,
                                    isDense: true,
                                  ),
                                ),
                              ),
                              IconButton(
                                icon: const Icon(Icons.add),
                                onPressed: () {
                                  setState(() {
                                    headerControllers.add(MapEntry(
                                      TextEditingController(),
                                      TextEditingController(),
                                    ));
                                  });
                                },
                                iconSize: 20,
                              ),
                              IconButton(
                                icon: const Icon(Icons.remove),
                                onPressed: () {
                                  if (headerControllers.length <= 1) {
                                    return;
                                  }
                                  setState(() {
                                    headerControllers.removeAt(index);
                                  });
                                },
                                iconSize: 20,
                              ),
                            ],
                          ),
                        );
                      }),
                    ],
                  ),
                ),
              ),
              actions: [
                TextButton(
                  child: Text('cancel'.tr),
                  onPressed: () => Get.back(),
                ),
                TextButton(
                  child: Text('confirm'.tr),
                  onPressed: () async {
                    try {
                      // Build headers map
                      final headers = <String, String>{};
                      for (final entry in headerControllers) {
                        final key = entry.key.text.trim();
                        final value = entry.value.text.trim();
                        if (key.isNotEmpty) {
                          headers[key] = value;
                        }
                      }

                      // Build ReqExtraHttp
                      final reqExtra = ReqExtraHttp(header: headers);

                      // Create patch request
                      final patchData = ResolveTask(
                        req: Request(
                          url: urlController.text.trim(),
                          extra: reqExtra.toJson(),
                        ),
                      );

                      await patchTask(task.id, patchData);
                      await continueTask(task.id);
                      Get.back();
                    } catch (e) {
                      showErrorMessage(e);
                    }
                  },
                ),
              ],
            );
          },
        ),
      );
    }

    List<Widget> buildActions() {
      final list = <Widget>[];
      if (isDone()) {
        list.add(IconButton(
          icon: const Icon(Icons.folder_open),
          onPressed: () {
            task.explorer();
          },
        ));
      } else {
        if (isRunning()) {
          list.add(IconButton(
            icon: const Icon(Icons.pause),
            onPressed: () async {
              try {
                await pauseTask(task.id);
              } catch (e) {
                showErrorMessage(e);
              }
            },
          ));
        } else {
          list.add(IconButton(
            icon: const Icon(Icons.play_arrow),
            onPressed: () async {
              try {
                await continueTask(task.id);
              } catch (e) {
                showErrorMessage(e);
              }
            },
          ));
        }
      }
      list.add(IconButton(
        icon: const Icon(Icons.delete),
        onPressed: () {
          showDeleteDialog([task.id]);
        },
      ));
      return list;
    }

    double getProgress() {
      final totalSize = task.meta.res?.size ?? 0;
      return totalSize <= 0 ? 0 : task.progress.downloaded / totalSize;
    }

    String getExtractionStatusText() {
      switch (task.progress.extractStatus) {
        case ExtractStatus.extracting:
          return '${'extracting'.tr} ${task.progress.extractProgress}%';
        case ExtractStatus.done:
          return 'extractDone'.tr;
        case ExtractStatus.error:
          return 'extractError'.tr;
        case ExtractStatus.waitingParts:
          return 'waitingParts'.tr;
        default:
          return '';
      }
    }

    String getProgressText() {
      if (isDone()) {
        return Util.fmtByte(task.meta.res!.size);
      }
      if (task.meta.res == null) {
        return "";
      }
      final total = task.meta.res!.size;
      return Util.fmtByte(task.progress.downloaded) +
          (total > 0 ? " / ${Util.fmtByte(total)}" : "");
    }

    // Get percentage text, e.g. " (50.5%)"
    String getPercentText() {
      final total = task.meta.res?.size ?? 0;
      if (total <= 0 || isDone()) return "";
      double p = getProgress();
      return "(${(p * 100).toStringAsFixed(1)}%)";
    }

    // Get ETA text, e.g. "00:05:30"
    String getEtaText() {
      if (isDone()) return "";
      if (!isRunning()) return "";

      final total = task.meta.res?.size ?? 0;
      final downloaded = task.progress.downloaded;
      final speed = task.progress.speed;

      // If speed is 0 or total unknown, don't show time
      if (total <= 0 || speed <= 0) {
        return "";
      }

      final remainingBytes = total - downloaded;
      // If remaining bytes <= 0, download is essentially complete
      if (remainingBytes <= 0) {
        return "";
      }

      // Use ceiling division to avoid showing 0 seconds when there's still data remaining
      final remainingSeconds = (remainingBytes + speed - 1) ~/ speed;

      // If time is too long (e.g. > 1 day), return > 1d
      if (remainingSeconds > 86400) return "> 1d";

      Duration duration = Duration(seconds: remainingSeconds);
      String twoDigits(int n) => n.toString().padLeft(2, "0");

      if (duration.inHours > 0) {
        return "${twoDigits(duration.inHours)}:${twoDigits(duration.inMinutes.remainder(60))}:${twoDigits(duration.inSeconds.remainder(60))}";
      } else {
        return "${twoDigits(duration.inMinutes.remainder(60))}:${twoDigits(duration.inSeconds.remainder(60))}";
      }
    }

    final appController = Get.find<AppController>();
    final taskController = Get.find<TaskController>();
    final taskListController = taskController.tabIndex.value == 0
        ? Get.find<TaskDownloadingController>()
        : Get.find<TaskDownloadedController>();

    // Filter selected task ids that are still in the task list
    filterSelectedTaskIds(Iterable<String> selectedTaskIds) => selectedTaskIds
        .where((id) => tasks.any((task) => task.id == id))
        .toList();

    // Build context menu entries
    final contextMenuEntries = <ContextMenuEntry>[
      MenuItem(
        icon: const Icon(Icons.checklist),
        label: Text('selectAll'.tr),
        onSelected: (_) {
          if (tasks.isEmpty) return;
          if (selectedTaskIds.isNotEmpty) {
            taskListController.selectedTaskIds([]);
          } else {
            taskListController.selectedTaskIds(tasks.map((e) => e.id).toList());
          }
        },
      ),
      MenuItem(
        icon: const Icon(Icons.check),
        label: Text('select'.tr),
        onSelected: (_) {
          if (isSelect()) {
            taskListController.selectedTaskIds(taskListController
                .selectedTaskIds
                .where((element) => element != task.id)
                .toList());
          } else {
            taskListController.selectedTaskIds(
                [...taskListController.selectedTaskIds, task.id]);
          }
        },
      ),
      const MenuDivider(),
      MenuItem(
        icon: const Icon(Icons.play_arrow),
        label: Text('continue'.tr),
        enabled: !isDone() && !isRunning(),
        onSelected: (_) async {
          try {
            await continueAllTasks(filterSelectedTaskIds(
                {...taskListController.selectedTaskIds, task.id}));
          } finally {
            taskListController.selectedTaskIds([]);
          }
        },
      ),
      MenuItem(
        icon: const Icon(Icons.pause),
        label: Text('pause'.tr),
        enabled: !isDone() && isRunning(),
        onSelected: (_) async {
          try {
            await pauseAllTasks(filterSelectedTaskIds(
                {...taskListController.selectedTaskIds, task.id}));
          } finally {
            taskListController.selectedTaskIds([]);
          }
        },
      ),
      MenuItem(
        icon: const Icon(Icons.delete),
        label: Text('delete'.tr),
        onSelected: (_) async {
          try {
            await showDeleteDialog(filterSelectedTaskIds(
                {...taskListController.selectedTaskIds, task.id}));
          } finally {
            taskListController.selectedTaskIds([]);
          }
        },
      ),
      // Update URL submenu - only enabled for HTTP tasks in pause or error status
      const MenuDivider(),
      MenuItem.submenu(
        icon: const Icon(Icons.link),
        label: Text('updateUrl'.tr),
        enabled: task.protocol == Protocol.http &&
            (task.status == Status.pause || task.status == Status.error),
        items: [
          MenuItem(
            icon: const Icon(Icons.edit_note),
            label: Text('updateUrlManual'.tr),
            onSelected: (_) async {
              await showUpdateUrlDialog(context, task);
            },
          ),
          MenuItem(
            icon: Icon(appController.pendingUpdateTask.value?.id == task.id
                ? Icons.cancel
                : Icons.sensors),
            label: Text(appController.pendingUpdateTask.value?.id == task.id
                ? 'updateUrlCancelListen'.tr
                : 'updateUrlListen'.tr),
            onSelected: (_) {
              if (appController.pendingUpdateTask.value?.id == task.id) {
                appController.pendingUpdateTask.value = null;
              } else {
                appController.pendingUpdateTask.value =
                    PendingUpdateTask(id: task.id, name: task.name);
              }
            },
          ),
        ],
      ),
    ];

    final contextMenu = ContextMenu(
      entries: contextMenuEntries,
      padding: const EdgeInsets.all(8.0),
    );

    return ContextMenuRegion(
      contextMenu: contextMenu,
      child: Obx(
        () => Card(
            elevation: 4.0,
            shape: isSelect()
                ? RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8.0),
                    side: BorderSide(
                      color: Theme.of(context).colorScheme.primary,
                      width: 2.0,
                    ),
                  )
                : null,
            child: InkWell(
              onTap: () {
                taskController.scaffoldKey.currentState?.openEndDrawer();
                taskController.selectTask.value = task;
              },
              onDoubleTap: () {
                task.open();
              },
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  ListTile(
                      title: Row(
                        children: [
                          Expanded(child: Text(task.name)),
                          // Show pending update indicator
                          if (appController.pendingUpdateTask.value?.id ==
                              task.id)
                            Tooltip(
                              message: 'updateUrlListeningTip'.tr,
                              child: Padding(
                                padding: const EdgeInsets.only(left: 8),
                                child: Icon(Icons.hearing,
                                    size: 16,
                                    color:
                                        Theme.of(context).colorScheme.primary),
                              ),
                            ),
                        ],
                      ),
                      leading: Icon(
                        fileIcon(task.name,
                            isFolder: isFolderTask(),
                            isBitTorrent: task.protocol == Protocol.bt),
                      )),
                  Row(
                    children: [
                      // Left side: Progress text + Percentage
                      Expanded(
                          child: Row(
                        children: [
                          Flexible(
                            child: Text(
                              getProgressText(),
                              style: Get.textTheme.bodyLarge
                                  ?.copyWith(color: Get.theme.disabledColor),
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                          // Hide percentage on mobile
                          if (!Util.isMobile() &&
                              getPercentText().isNotEmpty) ...[
                            const SizedBox(width: 4),
                            Text(
                              getPercentText(),
                              style: Get.textTheme.bodyLarge
                                  ?.copyWith(color: Get.theme.disabledColor),
                            ),
                          ],
                        ],
                      ).padding(left: 18)),
                      // Right side: ETA + Speed + Actions
                      Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          // Only show ETA on wider screens
                          if (!Util.isMobile() && getEtaText().isNotEmpty) ...[
                            Text(
                              getEtaText(),
                              style: Get.textTheme.titleSmall,
                            ),
                            Text(
                              " | ",
                              style: Get.textTheme.titleSmall?.copyWith(
                                color: Get.theme.disabledColor,
                                fontWeight: FontWeight.w300,
                              ),
                            ).padding(horizontal: 4),
                          ],
                          Text("${Util.fmtByte(task.progress.speed)}/s",
                              style: Get.textTheme.titleSmall),
                          ...buildActions()
                        ],
                      ),
                    ],
                  ),
                  isDone()
                      ? Container()
                      : LinearProgressIndicator(
                          value: getProgress(),
                        ),
                  // Extraction status row
                  if (task.progress.extractStatus != ExtractStatus.none)
                    Builder(builder: (context) {
                      final isExtracting = task.progress.extractStatus ==
                          ExtractStatus.extracting;
                      final isExtractDone =
                          task.progress.extractStatus == ExtractStatus.done;
                      final isWaitingParts = task.progress.extractStatus ==
                          ExtractStatus.waitingParts;
                      final statusColor = isExtracting
                          ? Get.theme.colorScheme.primary
                          : (isExtractDone
                              ? Colors.green
                              : isWaitingParts
                                  ? Colors.orange
                                  : Colors.red);
                      return Column(
                        children: [
                          Row(
                            children: [
                              Expanded(
                                child: Row(
                                  children: [
                                    Icon(
                                      isExtracting
                                          ? Icons.unarchive
                                          : (isExtractDone
                                              ? Icons.check_circle
                                              : Icons.error),
                                      size: 16,
                                      color: statusColor,
                                    ),
                                    const SizedBox(width: 4),
                                    Text(
                                      getExtractionStatusText(),
                                      style: Get.textTheme.bodySmall?.copyWith(
                                        color: statusColor,
                                      ),
                                    ),
                                  ],
                                ).padding(left: 18),
                              ),
                            ],
                          ).padding(top: 4, bottom: 8),
                          // Extraction progress bar
                          if (isExtracting)
                            LinearProgressIndicator(
                              value: task.progress.extractProgress / 100.0,
                              backgroundColor:
                                  Get.theme.colorScheme.surfaceContainerHighest,
                              valueColor: AlwaysStoppedAnimation<Color>(
                                  Get.theme.colorScheme.secondary),
                            ),
                        ],
                      );
                    }),
                ],
              ),
            )).padding(horizontal: 14, top: 8),
      ),
    );
  }
}
