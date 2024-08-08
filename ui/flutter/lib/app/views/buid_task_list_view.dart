import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:styled_widget/styled_widget.dart';

import '../../api/api.dart';
import '../../api/model/task.dart';
import '../../util/file_icon.dart';
import '../../util/icons.dart';
import '../../util/message.dart';
import '../../util/util.dart';
import '../modules/app/controllers/app_controller.dart';
import '../modules/task/controllers/task_controller.dart';
import '../modules/task/views/task_view.dart';
import '../routes/app_pages.dart';

class BuildTaskListView extends GetView {
  final List<Task> tasks;

  const BuildTaskListView({
    Key? key,
    required this.tasks,
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

    bool isFolderTask() {
      return task.isFolder;
    }

    Future<void> showDeleteDialog(String id) {
      final appController = Get.find<AppController>();

      final context = Get.context!;

      return showDialog<void>(
          context: context,
          barrierDismissible: false,
          builder: (_) => AlertDialog(
                title: Text('deleteTask'.tr),
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
                        await deleteTask(id, force);
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
          showDeleteDialog(task.id);
        },
      ));
      return list;
    }

    double getProgress() {
      final totalSize = task.meta.res?.size ?? 0;
      return totalSize <= 0 ? 0 : task.progress.downloaded / totalSize;
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

    return Card(
        elevation: 4.0,
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
                  title: Text(task.showName),
                  leading: isFolderTask()
                      ? const Icon(FaIcons.folder)
                      : Icon(FaIcons.allIcons[findIcon(task.showName)])),
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
            ],
          ),
        )).padding(horizontal: 14, top: 8);
  }
}
