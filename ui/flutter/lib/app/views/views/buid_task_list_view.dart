import 'dart:io';

import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../api/api.dart';
import '../../../api/model/task.dart';
import '../../../util/file_icon.dart';
import '../../../util/icons.dart';
import '../../../util/util.dart';
import '../../routes/app_pages.dart';

class BuildTaskListView extends GetView {
  final List<Task> tasks;

  const BuildTaskListView({
    Key? key,
    required this.tasks,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
        appBar: AppBar(
          actions: <Widget>[
            IconButton(
              icon: const Icon(Icons.add),
              tooltip: 'create'.tr,
              onPressed: () {
                Get.rootDelegate.toNamed(Routes.CREATE);
              },
            ),
            //TODO appBar toggleALl/start selected/delete selected/
            // IconButton(
            //   icon: const Icon(Icons.pause),
            //   tooltip: 'title'.tr,
            //   onPressed: () {
            //     // pause all
            //   },
            // ),
            // IconButton(
            //   icon: const Icon(Icons.delete),
            //   tooltip: 'title'.tr,
            //   onPressed: () {
            //     // delete all
            //   },
            // )
          ],
        ),
        body: Obx(() {
          return buildTaskList(context, tasks);
        }));
  }

  Widget buildTaskList(BuildContext context, tasks) {
    return ListView.builder(
      itemCount: tasks.length,
      itemBuilder: (context, index) {
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

    Future<void> showDeleteDialog(String id) {
      final keep = true.obs;

      final context = Get.context!;

      return showDialog<void>(
          context: context,
          barrierDismissible: false,
          builder: (_) => AlertDialog(
                title: Text('deleteTask'.tr),
                content: Obx(() => CheckboxListTile(
                    value: keep.value,
                    title: Text('deleteTaskTip'.tr,
                        style: context.textTheme.bodyLarge),
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
                      'delete'.tr,
                      style: const TextStyle(color: Colors.redAccent),
                    ),
                    onPressed: () async {
                      try {
                        await deleteTask(id, !keep.value);
                        Get.back();
                      } catch (e) {
                        Get.snackbar('error'.tr, e.toString());
                      }
                    },
                  ),
                ],
              ));
    }

    List<Widget> buildActions() {
      final list = <Widget>[];
      if (isDone()) {
        if (Util.isDesktop()) {
          list.add(IconButton(
            icon: const Icon(Icons.folder_open),
            onPressed: () async {
              final file = File(Util.buildAbsPath(task.meta.opts.path,
                  task.meta.res.files[0].path, task.meta.res.files[0].name));
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
          showDeleteDialog(task.id);
        },
      ));
      return list;
    }

    double getProgress() {
      return task.size <= 0 ? 1 : task.progress.downloaded / task.size;
    }

    Color pickColor() {
      switch (task.status) {
        // ready, running, pause, error, done
        case Status.running:
          return Get.theme.colorScheme.primary;
        // case Status.pause:
        //   return Get.theme.colorScheme.secondary;
        case Status.error:
          return Get.theme.colorScheme.error;
        default:
          return Get.theme.colorScheme.primary;
      }
    }

    return InkWell(
        // onTap: () {},
        child: Column(mainAxisSize: MainAxisSize.min, children: [
      Stack(children: [
        !isDone()
            ? Positioned(
                left: 0,
                right: 0,
                top: 0,
                bottom: 0,
                child: Opacity(
                  opacity: 0.6,
                  child: LinearProgressIndicator(
                    backgroundColor: Colors.transparent,
                    color: pickColor(),
                    // minHeight: 76,
                    value: getProgress(),
                  ),
                ))
            : const SizedBox.shrink(),
        ListTile(
          // isThreeLine: true,
          title: Text(task.meta.res.name,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: Get.textTheme.titleSmall),
          subtitle: Text(
            "${isDone() ? "" : "${Util.fmtByte(task.progress.downloaded)} / "}${Util.fmtByte(task.size)}",
            style: context.textTheme.bodyLarge
                ?.copyWith(color: Get.theme.disabledColor),
          ),
          leading: task.meta.res.name.lastIndexOf('.') == -1
              ? const Icon(FaIcons.file)
              : Icon(FaIcons.allIcons[findIcon(task.meta.res.name)]),

          trailing: SizedBox(
            width: 180,
            child: Row(
              // crossAxisAlignment: CrossAxisAlignment.baseline,
              // textBaseline: DefaultTextStyle.of(context).style.textBaseline,
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                Text("${Util.fmtByte(task.progress.speed)} / s",
                    style: context.textTheme.titleSmall),
                ...buildActions()
              ],
            ),
          ),
        ),
      ])
    ]));
  }
}
