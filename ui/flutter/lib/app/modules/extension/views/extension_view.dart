import 'package:flutter/material.dart';
import 'package:get/get.dart';
import '../../../../api/api.dart';
import '../../../../api/model/install_extension.dart';
import '../controllers/extension_controller.dart';

class ExtensionView extends GetView<ExtensionController> {
  ExtensionView({Key? key}) : super(key: key);

  final _installUrlController = TextEditingController();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SizedBox(height: 20),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: _installUrlController,
                  decoration: const InputDecoration(
                    labelText: '扩展安装地址',
                  ),
                ),
              ),
              const SizedBox(width: 10),
              IconButton(
                  onPressed: () async {
                    try {
                      await installExtension(
                          InstallExtension(url: _installUrlController.text));
                      Get.snackbar('提示', '安装成功');
                    } catch (e) {
                      Get.snackbar('提示', '安装失败');
                    }
                  },
                  icon: const Icon(Icons.download))
            ],
          ),
          const SizedBox(height: 32),
          const Text('已安装的扩展'),
          Expanded(
              child: Obx(() => ListView.builder(
                    itemCount: controller.extensions.length,
                    itemBuilder: (context, index) {
                      final extension = controller.extensions[index];
                      return SizedBox(
                        height: 112,
                        child: Card(
                          elevation: 4.0,
                          child: Column(
                            children: [
                              Expanded(
                                child: ListTile(
                                  leading: Image.asset(
                                    "assets/tray_icon/icon.png",
                                    width: 32,
                                    height: 32,
                                  ),
                                  trailing: Switch(
                                    value: true,
                                    onChanged: (value) {},
                                  ),
                                  title: Text(extension.title),
                                  subtitle: Text(extension.description),
                                ),
                              ),
                              Row(
                                mainAxisAlignment: MainAxisAlignment.end,
                                children: [
                                  extension.settings?.isNotEmpty == true
                                      ? const IconButton(
                                          onPressed: null,
                                          icon: Icon(Icons.settings))
                                      : null,
                                  extension.homepage.isNotEmpty == true
                                      ? const IconButton(
                                          onPressed: null,
                                          icon: Icon(Icons.home))
                                      : null,
                                  extension.repository.isNotEmpty == true
                                      ? const IconButton(
                                          onPressed: null,
                                          icon: Icon(Icons.code))
                                      : null,
                                  const IconButton(
                                      onPressed: null,
                                      icon: Icon(Icons.delete)),
                                ]
                                    .where((e) => e != null)
                                    .map((e) => e!)
                                    .toList(),
                              ).paddingOnly(right: 16)
                            ],
                          ),
                        ),
                      ).paddingOnly(top: 8);
                    },
                  ))),
        ],
      ).paddingAll(32),
    );
  }
}
