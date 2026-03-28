import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:open_filex/open_filex.dart';
import 'package:path/path.dart' as path;

import '../../../../api/api.dart';
import '../../../../api/model/stats.dart';
import '../../../../api/model/task.dart';
import '../../../../util/file_explorer.dart';
import '../../../../util/util.dart';
import '../../../routes/app_pages.dart';
import '../../../views/copy_button.dart';
import '../controllers/task_controller.dart';
import '../controllers/task_downloaded_controller.dart';
import '../controllers/task_downloading_controller.dart';
import 'task_downloaded_view.dart';
import 'task_downloading_view.dart';

class TaskView extends GetView<TaskController> {
  const TaskView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final selectTask = controller.selectTask;
    final statsRx = Rxn<Stats>();

    // Load stats when task changes
    Future.microtask(() async {
      if (selectTask.value?.status == Status.done && selectTask.value != null) {
        try {
          final stats = await getTaskStats(selectTask.value!.id);
          statsRx.value = stats;
        } catch (e) {
          // Ignore errors - stats are optional
        }
      }
    });

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
          child: Obx(() => ListView(
                // Important: Remove any padding from the ListView.
                padding: EdgeInsets.zero,
                children: [
                  SizedBox(
                    height: MediaQuery.of(context).padding.top + 65,
                    child: DrawerHeader(
                        child: Text(
                      'taskDetail'.tr,
                      style: Theme.of(context).textTheme.titleLarge,
                    )),
                  ),
                  ListTile(
                      title: Text('taskName'.tr),
                      subtitle: buildTooltipSubtitle(selectTask.value?.name)),
                  ListTile(
                    title: Text('taskUrl'.tr),
                    subtitle:
                        buildTooltipSubtitle(selectTask.value?.meta.req.url),
                    trailing: CopyButton(selectTask.value?.meta.req.url),
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
                  // Integrity verification section - only show for completed tasks
                  if (selectTask.value?.status == Status.done) ...[
                    const Divider(),
                    ListTile(
                      title: Text('integrityVerification'.tr),
                      subtitle: Obx(() => statsRx.value != null
                          ? Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                // File size info
                                Text(
                                  'fileSize'.tr +
                                      ': ${Util.fmtByte(statsRx.value!.fileSize)}' +
                                      (statsRx.value!.expectedSize > 0
                                          ? ' / ${Util.fmtByte(statsRx.value!.expectedSize)}'
                                          : ''),
                                  style: Get.textTheme.bodySmall,
                                ),
                                const SizedBox(height: 4),
                                // SHA256 hash
                                Row(
                                  children: [
                                    const Icon(Icons.security, size: 16),
                                    const SizedBox(width: 4),
                                    Expanded(
                                      child: Text(
                                        'SHA256: ${statsRx.value!.sha256}',
                                        style: Get.textTheme.bodySmall
                                            ?.copyWith(
                                              fontFamily: 'monospace',
                                              color: statsRx.value!
                                                      .integrityVerified
                                                  ? Colors.green
                                                  : Colors.red,
                                            ),
                                        overflow: TextOverflow.ellipsis,
                                      ),
                                    ),
                                    CopyButton(statsRx.value!.sha256),
                                  ],
                                ),
                                const SizedBox(height: 2),
                                // CRC32 hash
                                Row(
                                  children: [
                                    const Icon(Icons.security, size: 16),
                                    const SizedBox(width: 4),
                                    Expanded(
                                      child: Text(
                                        'CRC32: ${statsRx.value!.crc32}',
                                        style: Get.textTheme.bodySmall
                                            ?.copyWith(
                                              fontFamily: 'monospace',
                                              color: statsRx.value!
                                                      .integrityVerified
                                                  ? Colors.green
                                                  : Colors.red,
                                            ),
                                        overflow: TextOverflow.ellipsis,
                                      ),
                                    ),
                                    CopyButton(statsRx.value!.crc32),
                                  ],
                                ),
                                // Verification status
                                if (statsRx.value!.integrityVerified) ...[
                                  const SizedBox(height: 4),
                                  Row(
                                    children: [
                                      Icon(Icons.check_circle,
                                          size: 14, color: Colors.green),
                                      const SizedBox(width: 4),
                                      Text(
                                        'integrityVerified'.tr,
                                        style: Get.textTheme.bodySmall
                                            ?.copyWith(color: Colors.green),
                                      ),
                                    ],
                                  ),
                                ],
                              ],
                            )
                          : Text('loading'.tr)),
                    ),
                  ],
                ],
              )),
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
