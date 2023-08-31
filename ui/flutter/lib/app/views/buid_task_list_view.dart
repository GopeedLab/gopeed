import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/api/model/meta.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:path/path.dart' as path;

import '../../api/api.dart';
import '../../api/model/task.dart';
import '../../util/file_explorer.dart';
import '../../util/file_icon.dart';
import '../../util/icons.dart';
import '../../util/message.dart';
import '../../util/util.dart';
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

    String buildExplorerUrl(Task task) {
      if (task.meta.res.rootDir.trim().isEmpty) {
        return path.join(
            Util.safeDir(task.meta.opts.path),
            Util.safeDir(task.meta.res.files[0].path),
            task.meta.res.files[0].name);
      } else {
        return path.join(Util.safeDir(task.meta.opts.path),
            Util.safeDir(task.meta.res.rootDir));
      }
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
        if (Util.isDesktop()) {
          list.add(IconButton(
            icon: const Icon(Icons.folder_open),
            onPressed: () async {
              await FileExplorer.openAndSelectFile(buildExplorerUrl(task));
            },
          ));
        }
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
          title: Text(fileName(task.meta),
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: Get.textTheme.titleSmall),
          subtitle: Text(
            "${isDone() ? "" : "${Util.fmtByte(task.progress.downloaded)} / "}${Util.fmtByte(task.size)}",
            style: context.textTheme.bodyLarge
                ?.copyWith(color: Get.theme.disabledColor),
          ),
          leading: (task.meta.res.rootDir.isNotEmpty
              ? const Icon(FaIcons.folder)
              : Icon(FaIcons.allIcons[findIcon(fileName(task.meta))])),

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

  String fileName(Meta meta) {
    if (meta.res.files.length > 1) {
      return meta.res.name;
    }
    return meta.opts.name.isEmpty ? meta.res.files[0].name : meta.opts.name;
  }
}
