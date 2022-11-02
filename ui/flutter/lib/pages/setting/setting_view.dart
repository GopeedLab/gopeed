import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:get/get.dart';
import '../../setting/setting.dart';
import '../../widget/directory_selector.dart';

import 'setting_controller.dart';

class SettingView extends GetView<SettingController> {
  const SettingView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    Timer? timer;
    debounceSave() {
      timer?.cancel();
      timer = Timer(const Duration(milliseconds: 500), Setting.instance.save);
    }

    return Scaffold(
      appBar: AppBar(
        title: const Text('设置'),
        centerTitle: true,
      ),
      body: Column(
        // 左对齐
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text("基本设置", style: Get.textTheme.titleLarge),
          const SizedBox(height: 10),
          Obx(() => Card(
                  child: Column(
                children: _buildConfigItems([
                  _ConfigItem(
                      "主题",
                      () => _getThemeName(controller.setting.value.themeMode),
                      () => DropdownButton<ThemeMode>(
                            value: controller.setting.value.themeMode,
                            isDense: true,
                            onChanged: (value) {
                              controller.setting.value.themeMode = value!;
                              Get.changeThemeMode(value);

                              debounceSave();
                            },
                            items: ThemeMode.values
                                .map((e) => DropdownMenuItem<ThemeMode>(
                                      value: e,
                                      child: Text(_getThemeName(e)),
                                    ))
                                .toList(),
                          )),
                  _ConfigItem(
                      "下载目录", () => controller.setting.value.downloadDir, () {
                    final downloadDirController = TextEditingController(
                        text: controller.setting.value.downloadDir);
                    downloadDirController.addListener(() {
                      if (downloadDirController.text !=
                          controller.setting.value.downloadDir) {
                        controller.setting.value.downloadDir =
                            downloadDirController.text;

                        debounceSave();
                      }
                    });
                    return DirectorySelector(
                      controller: downloadDirController,
                      showLabel: false,
                    );
                  }),
                  _ConfigItem("连接数",
                      () => controller.setting.value.connections.toString(),
                      () {
                    final connectionsController = TextEditingController(
                        text: controller.setting.value.connections.toString());
                    connectionsController.addListener(() {
                      if (connectionsController.text.isNotEmpty &&
                          connectionsController.text !=
                              controller.setting.value.connections.toString()) {
                        controller.setting.value.connections =
                            int.parse(connectionsController.text);

                        debounceSave();
                      }
                    });

                    return TextField(
                      controller: connectionsController,
                      keyboardType: TextInputType.number,
                      inputFormatters: [
                        FilteringTextInputFormatter.digitsOnly,
                        NumericalRangeFormatter(min: 1, max: 256),
                      ],
                    );
                  }),
                ]),
              ))),
        ],
      ).paddingAll(16),
    );
  }

  List<Widget> _buildConfigItems(List<_ConfigItem> buildItems) {
    final result = <Widget>[];
    for (var i = 0; i < buildItems.length; i++) {
      final buildItem = buildItems[i];
      result.add(ListTile(
          title: Text(buildItem.label),
          subtitle: controller.tapStatues[i] ?? false
              ? buildItem.inputItem()
              : Text(buildItem.text()),
          onTap: () {
            controller.tapStatues[i] = true;
            // set other false
            for (var j = 0; j < controller.tapStatues.length; j++) {
              if (j != i) {
                controller.tapStatues[j] = false;
              }
            }
          }));
      if (i != buildItems.length - 1) {
        result.add(const Divider());
      }
    }
    return result;
  }

  String _getThemeName(ThemeMode themeMode) {
    switch (themeMode) {
      case ThemeMode.system:
        return "跟随系统";
      case ThemeMode.light:
        return "明亮主题";
      case ThemeMode.dark:
        return "暗黑主题";
    }
  }
}

class _ConfigItem {
  late String label;
  late String Function() text;
  late Widget Function() inputItem;

  _ConfigItem(this.label, this.text, this.inputItem);
}

class NumericalRangeFormatter extends TextInputFormatter {
  final int min;
  final int max;

  NumericalRangeFormatter({required this.min, required this.max});

  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    if (newValue.text.isEmpty) {
      return newValue;
    }
    var intVal = int.tryParse(newValue.text);
    if (intVal == null) {
      return oldValue;
    }
    if (intVal < min) {
      return newValue.copyWith(text: min.toString());
    } else if (intVal > max) {
      return oldValue.copyWith(text: max.toString());
    } else {
      return newValue;
    }
  }
}
