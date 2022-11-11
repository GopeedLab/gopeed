import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:get/get.dart';

import '../../i18n/messages.dart';
import '../../util/util.dart';
import '../../widget/directory_selector.dart';
import '../app/app_controller.dart';
import 'setting_controller.dart';

const _padding = SizedBox(height: 10);

class SettingView extends GetView<SettingController> {
  final _settingLocaleKey = 'setting.locale.';

  const SettingView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final appController = Get.find<AppController>();
    final setting = appController.downloaderConfig.value;

    Timer? timer;
    debounceSave() {
      var completer = Completer<void>();
      timer?.cancel();
      timer = Timer(const Duration(milliseconds: 500), () {
        appController
            .saveConfig()
            .then(completer.complete)
            .onError(completer.completeError);
      });
      return completer.future;
    }

    return Scaffold(
      appBar: AppBar(
        title: Text('setting.title'.tr),
        centerTitle: true,
      ),
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('setting.basic'.tr, style: Get.textTheme.titleLarge),
          _padding,
          Obx(() => Card(
                  child: Column(
                children: _buildConfigItems([
                  _ConfigItem(
                      'setting.theme'.tr,
                      () => _getThemeName(setting.extra!.themeMode),
                      () => DropdownButton<String>(
                            value: setting.extra!.themeMode,
                            isDense: true,
                            onChanged: (value) async {
                              setting.extra!.themeMode = value!;
                              controller.clearTapStatus();
                              Get.changeThemeMode(
                                  ThemeMode.values.byName(value));

                              await debounceSave();
                            },
                            items: ThemeMode.values
                                .map((e) => DropdownMenuItem<String>(
                                      value: e.name,
                                      child: Text(_getThemeName(e.name)),
                                    ))
                                .toList(),
                          )),
                  _ConfigItem(
                      'setting.downloadDir'.tr, () => setting.downloadDir, () {
                    final downloadDirController =
                        TextEditingController(text: setting.downloadDir);
                    downloadDirController.addListener(() async {
                      if (downloadDirController.text != setting.downloadDir) {
                        setting.downloadDir = downloadDirController.text;
                        if (Util.isDesktop()) {
                          controller.clearTapStatus();
                        }

                        await debounceSave();
                      }
                    });
                    return DirectorySelector(
                      controller: downloadDirController,
                      showLabel: false,
                    );
                  }),
                  // _ConfigItem('setting.connections'.tr,
                  //     () => controller.setting.value.connections.toString(),
                  //     () {
                  //   final connectionsController = TextEditingController(
                  //       text: controller.setting.value.connections.toString());
                  //   connectionsController.addListener(() async {
                  //     if (connectionsController.text.isNotEmpty &&
                  //         connectionsController.text !=
                  //             controller.setting.value.connections.toString()) {
                  //       controller.setting.value.connections =
                  //           int.parse(connectionsController.text);

                  //       await debounceSave();
                  //     }
                  //   });

                  //   return TextField(
                  //     controller: connectionsController,
                  //     keyboardType: TextInputType.number,
                  //     inputFormatters: [
                  //       FilteringTextInputFormatter.digitsOnly,
                  //       NumericalRangeFormatter(min: 1, max: 256),
                  //     ],
                  //   );
                  // }),
                  _ConfigItem(
                      'setting.locale'.tr,
                      () => _getLocaleName(setting.extra!.locale),
                      () => DropdownButton<String>(
                            value: setting.extra!.locale,
                            isDense: true,
                            onChanged: (value) async {
                              setting.extra!.locale = value!;
                              controller.clearTapStatus();
                              Get.updateLocale(toLocale(value));

                              await debounceSave();
                            },
                            items: _getLocales()
                                .map((e) => DropdownMenuItem<String>(
                                      value:
                                          e.substring(_settingLocaleKey.length),
                                      child: Text(e.tr),
                                    ))
                                .toList(),
                          )),
                ]),
              ))),
          _padding,
          Text('setting.advanced'.tr, style: Get.textTheme.titleLarge),
          _padding,
          Obx(() => Card(
                  child: Column(
                children: _buildConfigItems([
                  _ConfigItem(
                      'setting.theme'.tr,
                      () => _getThemeName(setting.extra!.themeMode),
                      () => DropdownButton<String>(
                            value: setting.extra!.themeMode,
                            isDense: true,
                            onChanged: (value) async {
                              setting.extra!.themeMode = value!;
                              controller.clearTapStatus();
                              Get.changeThemeMode(
                                  ThemeMode.values.byName(value));

                              await debounceSave();
                            },
                            items: ThemeMode.values
                                .map((e) => DropdownMenuItem<String>(
                                      value: e.name,
                                      child: Text(_getThemeName(e.name)),
                                    ))
                                .toList(),
                          )),
                  _ConfigItem(
                      'setting.downloadDir'.tr, () => setting.downloadDir, () {
                    final downloadDirController =
                        TextEditingController(text: setting.downloadDir);
                    downloadDirController.addListener(() async {
                      if (downloadDirController.text != setting.downloadDir) {
                        setting.downloadDir = downloadDirController.text;
                        if (Util.isDesktop()) {
                          controller.clearTapStatus();
                        }

                        await debounceSave();
                      }
                    });
                    return DirectorySelector(
                      controller: downloadDirController,
                      showLabel: false,
                    );
                  }),
                  // _ConfigItem('setting.connections'.tr,
                  //     () => controller.setting.value.connections.toString(),
                  //     () {
                  //   final connectionsController = TextEditingController(
                  //       text: controller.setting.value.connections.toString());
                  //   connectionsController.addListener(() async {
                  //     if (connectionsController.text.isNotEmpty &&
                  //         connectionsController.text !=
                  //             controller.setting.value.connections.toString()) {
                  //       controller.setting.value.connections =
                  //           int.parse(connectionsController.text);

                  //       await debounceSave();
                  //     }
                  //   });

                  //   return TextField(
                  //     controller: connectionsController,
                  //     keyboardType: TextInputType.number,
                  //     inputFormatters: [
                  //       FilteringTextInputFormatter.digitsOnly,
                  //       NumericalRangeFormatter(min: 1, max: 256),
                  //     ],
                  //   );
                  // }),
                  _ConfigItem(
                      'setting.locale'.tr,
                      () => _getLocaleName(setting.extra!.locale),
                      () => DropdownButton<String>(
                            value: setting.extra!.locale,
                            isDense: true,
                            onChanged: (value) async {
                              setting.extra!.locale = value!;
                              controller.clearTapStatus();
                              Get.updateLocale(toLocale(value));

                              await debounceSave();
                            },
                            items: _getLocales()
                                .map((e) => DropdownMenuItem<String>(
                                      value:
                                          e.substring(_settingLocaleKey.length),
                                      child: Text(e.tr),
                                    ))
                                .toList(),
                          )),
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

  String _getLocaleName(String locale) {
    final localeKey = '$_settingLocaleKey${locale.toString()}';
    if (messages.keys[locale]?.containsKey(localeKey) ?? false) {
      return localeKey.tr;
    }
    return '$_settingLocaleKey$locale'.tr;
  }

  List<String> _getLocales() {
    return messages.keys[getLocaleKey(fallbackLocale)]!.entries
        .where((e) => e.key.startsWith(_settingLocaleKey))
        .map((e) => e.key)
        .toList();
  }

  String _getThemeName(String themeMode) {
    switch (ThemeMode.values.byName(themeMode)) {
      case ThemeMode.system:
        return 'setting.themeSystem'.tr;
      case ThemeMode.light:
        return 'setting.themeLight'.tr;
      case ThemeMode.dark:
        return 'setting.themeDark'.tr;
    }
  }
}

class _ConfigItem {
  late String label;
  late String Function() text;
  late Widget Function() inputItem;

  _ConfigItem(this.label, this.text, this.inputItem);
}

class _NumericalRangeFormatter extends TextInputFormatter {
  final int min;
  final int max;

  _NumericalRangeFormatter({required this.min, required this.max});

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
