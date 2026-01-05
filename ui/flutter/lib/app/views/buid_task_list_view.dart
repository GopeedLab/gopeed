import 'package:contextmenu_plus/contextmenu_plus.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:styled_widget/styled_widget.dart';

import '../../api/api.dart';
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

    Widget buildContextItem(IconData icon, String label, Function() onTap,
        {bool enabled = true}) {
      return ListTile(
        dense: true,
        visualDensity: const VisualDensity(vertical: -1),
        minLeadingWidth: 12,
        leading: Icon(icon, size: 18),
        title: Text(label,
            style: const TextStyle(
              fontWeight: FontWeight.bold, // Make the text bold
            )),
        onTap: () async {
          Get.back();
          try {
            await onTap();
          } catch (e) {
            showErrorMessage(e);
          }
        },
        enabled: enabled,
      );
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

    final taskController = Get.find<TaskController>();
    final taskListController = taskController.tabIndex.value == 0
        ? Get.find<TaskDownloadingController>()
        : Get.find<TaskDownloadedController>();

    // Filter selected task ids that are still in the task list
    filterSelectedTaskIds(Iterable<String> selectedTaskIds) => selectedTaskIds
        .where((id) => tasks.any((task) => task.id == id))
        .toList();

    return ContextMenuArea(
      width: 140,
      builder: (context) => [
        buildContextItem(Icons.checklist, 'selectAll'.tr, () {
          if (tasks.isEmpty) return;

          if (selectedTaskIds.isNotEmpty) {
            taskListController.selectedTaskIds([]);
          } else {
            taskListController.selectedTaskIds(tasks.map((e) => e.id).toList());
          }
        }),
        buildContextItem(Icons.check, 'select'.tr, () {
          if (isSelect()) {
            taskListController.selectedTaskIds(taskListController
                .selectedTaskIds
                .where((element) => element != task.id)
                .toList());
          } else {
            taskListController.selectedTaskIds(
                [...taskListController.selectedTaskIds, task.id]);
          }
        }),
        const Divider(
          indent: 8,
          endIndent: 8,
        ),
        buildContextItem(Icons.play_arrow, 'continue'.tr, () async {
          try {
            await continueAllTasks(filterSelectedTaskIds(
                {...taskListController.selectedTaskIds, task.id}));
          } finally {
            taskListController.selectedTaskIds([]);
          }
        }, enabled: !isDone() && !isRunning()),
        buildContextItem(Icons.pause, 'pause'.tr, () async {
          try {
            await pauseAllTasks(filterSelectedTaskIds(
                {...taskListController.selectedTaskIds, task.id}));
          } finally {
            taskListController.selectedTaskIds([]);
          }
        }, enabled: !isDone() && isRunning()),
        buildContextItem(Icons.delete, 'delete'.tr, () async {
          try {
            await showDeleteDialog(filterSelectedTaskIds(
                {...taskListController.selectedTaskIds, task.id}));
          } finally {
            taskListController.selectedTaskIds([]);
          }
        }),
      ],
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
                      title: Text(task.name),
                      leading: Icon(
                        fileIcon(task.name,
                            isFolder: isFolderTask(),
                            isBitTorrent: task.protocol == Protocol.bt),
                      )),
                  Row(
                    children: [
                      Expanded(
                          flex: 1,
                          child: Text(
                            getProgressText(),
                            style: Get.textTheme.bodyLarge
                                ?.copyWith(color: Get.theme.disabledColor),
                          ).padding(left: 18)),
                      Expanded(
                          flex: 1,
                          child: Row(
                            mainAxisAlignment: MainAxisAlignment.end,
                            children: [
                              Text("${Util.fmtByte(task.progress.speed)} / s",
                                  style: Get.textTheme.titleSmall),
                              ...buildActions()
                            ],
                          )),
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
