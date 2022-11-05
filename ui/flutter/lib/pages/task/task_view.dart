import 'dart:io';

import 'package:flutter/material.dart';
import 'package:get/get.dart';
import '../../api/api.dart';
import '../../api/model/task.dart';
import 'package:styled_widget/styled_widget.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../util/util.dart';
import 'task_controller.dart';

class TaskView extends GetView<TaskController> {
  const TaskView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 2,
      child: Scaffold(
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
                  controller.tabIndex.value = index;
                  if (index == 1) {
                    controller.loadTab();
                  }
                },
              ),
            )),
        body: TabBarView(
          children: [
            Obx(() => buildTaskList(controller.unDoneTasks)),
            Obx(() => buildTaskList(controller.doneTasks)),
          ],
        ),
      ),
    );
  }

  Widget buildTaskList(List<Task> tasks) {
    return ListView.builder(
      itemCount: tasks.length,
      itemBuilder: (context, index) {
        return Item(task: tasks[index]);
      },
    );
  }
}

class Item extends StatelessWidget {
  final Task task;

  const Item({Key? key, required this.task}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Card(
        elevation: 4.0,
        child: InkWell(
          onTap: () {},
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              ListTile(
                title: Text(task.res.name),
                leading: const Icon(Icons.insert_drive_file),
              ),
              Row(
                children: [
                  Expanded(
                      flex: 1,
                      child: Text(
                        "${isDone() ? "" : "${Util.fmtByte(task.progress.downloaded)} / "}${Util.fmtByte(task.size)}",
                        style: Get.textTheme.bodyText1
                            ?.copyWith(color: Get.theme.disabledColor),
                      ).padding(left: 18)),
                  Expanded(
                      flex: 1,
                      child: Row(
                        mainAxisAlignment: MainAxisAlignment.end,
                        children: [
                          Text("${Util.fmtByte(task.progress.speed)} / s",
                              style: Get.textTheme.subtitle2),
                          ...buildActions()
                        ],
                      )),
                ],
              ),
              LinearProgressIndicator(
                value: getProgress(),
              ),
            ],
          ),
        )).padding(horizontal: 8, top: 8);
  }

  List<Widget> buildActions() {
    final list = <Widget>[];
    if (isDone()) {
      if (Util.isDesktop()) {
        list.add(IconButton(
          icon: const Icon(Icons.folder_open),
          onPressed: () async {
            final file = File(Util.buildAbsPath(task.opts.path,
                task.res.files[0].path, task.res.files[0].name));
            await launchUrl(file.parent.uri);
          },
        ));
      }
    } else {
      if (isRunning()) {
        list.add(IconButton(
          icon: const Icon(Icons.pause),
          onPressed: () async {
            await pauseTask(task.id);
          },
        ));
      } else {
        list.add(IconButton(
          icon: const Icon(Icons.play_arrow),
          onPressed: () async {
            await continueTask(task.id);
          },
        ));
      }
    }
    list.add(IconButton(
      icon: const Icon(Icons.delete),
      onPressed: () {
        _showDeleteDialog(task.id);
      },
    ));
    return list;
  }

  Future<void> _showDeleteDialog(String id) {
    final keep = true.obs;

    return showDialog<void>(
        context: Get.context!,
        barrierDismissible: false,
        builder: (_) => AlertDialog(
              title: Text('task.deleteTask'.tr),
              content: Obx(() => CheckboxListTile(
                  value: keep.value,
                  title: Text('task.deleteTaskTip'.tr,
                      style: Get.textTheme.bodyText1),
                  onChanged: (v) {
                    keep.value = v!;
                  })),
              actions: [
                TextButton(
                  child: Text('cancel'.tr),
                  onPressed: () => Get.back(),
                ),
                TextButton(
                  child: Text(
                    'task.delete'.tr,
                    style: const TextStyle(color: Colors.redAccent),
                  ),
                  onPressed: () async {
                    try {
                      await deleteTask(id, !keep.value);
                      Get.find<TaskController>().loadTab();
                      Get.back();
                    } catch (e) {
                      Get.snackbar('error'.tr, e.toString());
                    }
                  },
                ),
              ],
            ));
  }

  double getProgress() {
    return task.size <= 0 ? 1 : task.progress.downloaded / task.size;
  }

  bool isRunning() {
    return task.status == Status.running;
  }

  bool isDone() {
    return task.status == Status.done;
  }
}
