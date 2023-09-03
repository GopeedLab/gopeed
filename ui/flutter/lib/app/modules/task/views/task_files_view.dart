import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/util/util.dart';
import '../../../../util/file_icon.dart';
import '../../../../util/icons.dart';
import '../../../views/breadcrumb_view.dart';
import '../controllers/task_files_controller.dart';

class TaskFilesView extends GetView<TaskFilesController> {
  const TaskFilesView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
        appBar: AppBar(
          leading: IconButton(
              icon: const Icon(Icons.arrow_back),
              onPressed: () => Get.rootDelegate.popRoute()),
          // actions: [],
          title: Obx(() => Text(controller.task.value?.meta.res.name ?? "")),
        ),
        body: Obx(() {
          final fileList = controller.fileList;
          final breadcrumbItems = ["/"];
          if (fileList.isNotEmpty) {
            final file = fileList.first;
            final path = file.path.substring(1);
            if (path.isNotEmpty) {
              final pathArr = path.split("/");
              for (int i = 0; i < pathArr.length; i++) {
                breadcrumbItems.add(pathArr[i]);
              }
            }
          }
          return Column(
            children: [
              Breadcrumb(
                  items: breadcrumbItems,
                  onItemTap: (index) {
                    final targetDirArr = <String>[];
                    for (int i = 0; i <= index; i++) {
                      targetDirArr.add(breadcrumbItems[i]);
                    }
                    controller
                        .toDir(targetDirArr.join("/").replaceFirst('//', '/'));
                  }).paddingOnly(left: 16, top: 16, bottom: 8),
              Expanded(
                child: ListView.builder(
                  itemBuilder: (context, index) {
                    final file = fileList[index];
                    return ListTile(
                      leading: file.isDirectory
                          ? const Icon(Icons.folder)
                          : Icon(FaIcons.allIcons[findIcon(file.name)]),
                      title: Text(fileList[index].name),
                      subtitle: file.isDirectory
                          ? Text(
                              "${controller.dirItemCount(file.fullPath)} items")
                          : Text(Util.fmtByte(file.size)),
                      onTap: () {
                        if (file.isDirectory) {
                          controller.toDir(file.fullPath);
                        }
                      },
                    );
                  },
                  itemCount: controller.fileList.length,
                ),
              )
            ],
          );
        }));
  }
}
