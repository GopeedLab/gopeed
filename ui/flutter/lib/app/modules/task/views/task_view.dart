import 'package:desktop_drop/desktop_drop.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/util/util.dart';

import '../../../routes/app_pages.dart';
import '../../app/controllers/app_controller.dart';
import '../controllers/task_controller.dart';
import '../controllers/task_downloaded_controller.dart';
import '../controllers/task_downloading_controller.dart';
import 'task_downloaded_view.dart';
import 'task_downloading_view.dart';

class TaskView extends GetView<TaskController> {
  const TaskView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final appController = Get.find<AppController>();

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
            body: DropTarget(
              onDragDone: (details) async {
                if (!Util.isWeb()) {
                  appController.urlController.text = details.files[0].path;
                  Get.rootDelegate.toNamed(Routes.CREATE);
                  return;
                }
                appController.urlController.text = details.files[0].name;
                final bytes = await details.files[0].readAsBytes();
                appController.setFileDataUri(bytes);
                Get.rootDelegate.toNamed(Routes.CREATE);
              },
              child: GestureDetector(
                behavior: HitTestBehavior.opaque,
                onTap: () {
                  FocusScope.of(context).requestFocus(FocusNode());
                },
                child: const TabBarView(
                  children: [
                    TaskDownloadingView(),
                    TaskDownloadedView(),
                  ],
                ),
              ),
            )));
  }
}
