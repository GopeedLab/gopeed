import 'package:flutter/material.dart';
import 'package:get/get.dart';

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
                    height: 65,
                    child: DrawerHeader(
                        child: Text(
                      '任务详情',
                      style: Theme.of(context).textTheme.titleLarge,
                    )),
                  ),
                  ListTile(
                    title: const Text('任务名称'),
                    subtitle: Text(
                      '${selectTask.value?.status.name}',
                      overflow: TextOverflow.ellipsis,
                    ),
                    trailing: IconButton(
                      icon: const Icon(Icons.copy),
                      onPressed: () {
                        Get.snackbar('复制成功', '任务链接已复制到剪贴板');
                      },
                    ),
                  ),
                  ListTile(
                      title: Text('任务状态'),
                      subtitle: Text('${selectTask.value?.status.name}')),
                  ListTile(
                    title: const Text('任务链接'),
                    subtitle: const Text(
                      "https://www.baidu.com/asssssssssssssssssssssssssssssssssssssssss",
                      overflow: TextOverflow.ellipsis,
                    ),
                    trailing: IconButton(
                      icon: const Icon(Icons.copy),
                      onPressed: () {
                        Get.snackbar('复制成功', '任务链接已复制到剪贴板');
                      },
                    ),
                  ),
                  ListTile(
                    title: const Text('下载目录'),
                    subtitle: const Text(
                      "index.html",
                      overflow: TextOverflow.ellipsis,
                    ),
                    trailing: IconButton(
                      icon: const Icon(Icons.folder_open),
                      onPressed: () {
                        Get.snackbar('复制成功', '任务链接已复制到剪贴板');
                      },
                    ),
                  ),
                ],
              )),
        ),
      ),
    );
  }
}
