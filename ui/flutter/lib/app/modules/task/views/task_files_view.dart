import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:open_filex/open_filex.dart';
import 'package:path/path.dart';
import 'package:share_plus/share_plus.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../api/api.dart' as api;
import '../../../../util/browser_download/browser_download.dart';
import '../../../../util/util.dart';
import '../../../views/breadcrumb_view.dart';
import '../../../views/file_icon.dart';
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
          title: Obx(() => Text(controller.task.value?.meta.res?.name ?? "")),
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
                    final meta = controller.task.value!.meta;
                    final file = fileList[index];
                    // if resource is single file, use opts.name as file name
                    final realFileName = meta.res!.name.isEmpty
                        ? (meta.opts.name.isEmpty ? file.name : meta.opts.name)
                        : "";
                    final fileRelativePath = file.filePath(realFileName);
                    final filePath = Util.safePathJoin(
                        [meta.opts.path, meta.res!.name, fileRelativePath]);
                    final fileName = basename(filePath);
                    return ListTile(
                      leading:
                          Icon(fileIcon(fileName, isFolder: file.isDirectory)),
                      title: Text(fileName),
                      subtitle: file.isDirectory
                          ? Text('items'.trParams({
                              'count': controller
                                  .dirItemCount(file.fullPath)
                                  .toString()
                            }))
                          : Text(Util.fmtByte(file.size)),
                      trailing: file.isDirectory
                          ? null
                          : SizedBox(
                              width: 100,
                              child: Row(
                                mainAxisAlignment: MainAxisAlignment.end,
                                children: Util.isWeb()
                                    ? () {
                                        final accessUrl = api.join(
                                            "/fs/tasks/${controller.task.value!.id}$fileRelativePath");
                                        return [
                                          IconButton(
                                              icon:
                                                  const Icon(Icons.open_in_new),
                                              onPressed: () {
                                                launchUrl(Uri.parse(accessUrl),
                                                    webOnlyWindowName:
                                                        "_blank");
                                              }),
                                          IconButton(
                                              icon: const Icon(Icons.download),
                                              onPressed: () {
                                                download(accessUrl, fileName);
                                              })
                                        ];
                                      }()
                                    : [
                                        IconButton(
                                            icon: const Icon(Icons.open_in_new),
                                            onPressed: () async {
                                              await OpenFilex.open(filePath);
                                            }),
                                        IconButton(
                                            icon: const Icon(Icons.share),
                                            onPressed: () {
                                              final xfile = XFile(filePath);
                                              Share.shareXFiles([xfile]);
                                            })
                                      ],
                              ),
                            ),
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
