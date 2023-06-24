import 'package:flutter/material.dart';
import 'package:get/get.dart';
import '../controllers/extension_controller.dart';

class ExtensionView extends GetView<ExtensionController> {
  const ExtensionView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SizedBox(height: 20),
          Row(
            children: [
              const Expanded(
                child: TextField(
                  decoration: InputDecoration(
                    labelText: '扩展安装地址',
                  ),
                ),
              ),
              const SizedBox(width: 10),
              IconButton(
                  onPressed: () {
                    Get.snackbar('提示', '安装成功');
                  },
                  icon: const Icon(Icons.download))
            ],
          ),
          const SizedBox(height: 32),
          const Text('已安装的扩展'),
          Expanded(
              child: ListView.builder(
            itemCount: 5,
            itemBuilder: (context, index) => SizedBox(
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
                        title: Text('扩展名称'),
                        subtitle: Text('扩展描述'),
                      ),
                    ),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.end,
                      children: [
                        IconButton(
                            onPressed: null, icon: const Icon(Icons.settings)),
                        IconButton(
                            onPressed: null, icon: const Icon(Icons.home)),
                        IconButton(
                            onPressed: null, icon: const Icon(Icons.code)),
                        IconButton(
                            onPressed: null, icon: const Icon(Icons.delete)),
                      ],
                    ).paddingOnly(right: 16)
                  ],
                ),
              ),
            ).paddingOnly(top: 8),
          )),
        ],
      ).paddingAll(32),
    );
  }
}
