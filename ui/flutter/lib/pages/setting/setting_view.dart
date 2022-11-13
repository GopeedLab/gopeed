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
    final downloaderCfg = appController.downloaderConfig.value;
    final startCfg = appController.startConfig.value;

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

    final basicConfigItems = [
      _ConfigItem(
          'setting.theme'.tr,
          () => _getThemeName(downloaderCfg.extra!.themeMode),
          (Key key) => DropdownButton<String>(
                key: key,
                value: downloaderCfg.extra!.themeMode,
                isDense: true,
                onChanged: (value) async {
                  downloaderCfg.extra!.themeMode = value!;
                  controller.clearTapStatus();
                  Get.changeThemeMode(ThemeMode.values.byName(value));

                  await debounceSave();
                },
                items: ThemeMode.values
                    .map((e) => DropdownMenuItem<String>(
                          value: e.name,
                          child: Text(_getThemeName(e.name)),
                        ))
                    .toList(),
              )),
      _ConfigItem('setting.downloadDir'.tr, () => downloaderCfg.downloadDir,
          (Key key) {
        final downloadDirController =
            TextEditingController(text: downloaderCfg.downloadDir);
        downloadDirController.addListener(() async {
          if (downloadDirController.text != downloaderCfg.downloadDir) {
            downloaderCfg.downloadDir = downloadDirController.text;
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
          () => _getLocaleName(downloaderCfg.extra!.locale),
          (Key key) => DropdownButton<String>(
                key: key,
                value: downloaderCfg.extra!.locale,
                isDense: true,
                onChanged: (value) async {
                  downloaderCfg.extra!.locale = value!;
                  controller.clearTapStatus();
                  Get.updateLocale(toLocale(value));

                  await debounceSave();
                },
                items: _getLocales()
                    .map((e) => DropdownMenuItem<String>(
                          value: e.substring(_settingLocaleKey.length),
                          child: Text(e.tr),
                        ))
                    .toList(),
              )),
    ];
    final basicKeys = basicConfigItems.map((e) => GlobalKey()).toList();

    final advancedConfigItems = [
      _ConfigItem(
        '通信协议',
        () => startCfg.network,
        (Key key) => Row(
          children: [
            RadioListTile<String>(
              contentPadding: EdgeInsets.all(0),
              title: const Text('TCP'),
              value: "tcp",
              groupValue: startCfg.network,
              onChanged: (String? value) {
                startCfg.network = value!;
              },
            ),
            RadioListTile<String>(
              contentPadding: EdgeInsets.all(0),
              title: const Text('Unix Socket'),
              value: "unix",
              groupValue: startCfg.network,
              onChanged: (String? value) {
                startCfg.network = value!;
              },
            ),
          ],
        ),
      ),
    ];
    final advancedKeys = advancedConfigItems.map((e) => GlobalKey()).toList();

    return Scaffold(
        appBar: AppBar(
          title: Text('setting.title'.tr),
          centerTitle: true,
        ),
        body: GestureDetector(
          onTap: () {
            controller.clearTapStatus();
          },
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('setting.basic'.tr, style: Get.textTheme.titleLarge),
                _padding,
                Obx(() => Card(
                        child: Column(
                      children: _buildConfigItems(controller.basicTapStatues,
                          basicKeys, basicConfigItems),
                    ))),
                _padding,
                Text('setting.advanced'.tr, style: Get.textTheme.titleLarge),
                _padding,
                Obx(() => Card(
                        child: Column(
                      children: [
                        ..._buildConfigItems(controller.advancedTapStatues,
                            advancedKeys, advancedConfigItems),
                        const Divider(),
                        DefaultTabController(
                          length: 2,
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            children: <Widget>[
                              Container(
                                child: TabBar(tabs: [
                                  Tab(text: "HTTP"),
                                  Tab(text: "BitTorrent"),
                                ]),
                              ),
                              Container(
                                //Add this to give height
                                height: MediaQuery.of(context).size.height,
                                child: TabBarView(children: [
                                  Container(
                                    child: Text("Home Body"),
                                  ),
                                  Container(
                                    child: Text("Articles Body"),
                                  ),
                                ]),
                              ),
                            ],
                          ),
                        ),
                      ],
                    ))),
              ],
            ).paddingAll(16),
          ),
        ));
  }

  void _tapInputWidget(GlobalKey key) {
    GestureDetector? detector;
    void searchForGestureDetector(BuildContext? element) {
      element?.visitChildElements((element) {
        if (element.widget is GestureDetector) {
          detector = element.widget as GestureDetector?;
        } else {
          searchForGestureDetector(element);
        }
      });
    }

    searchForGestureDetector(key.currentContext);

    detector?.onTap?.call();
  }

  List<Widget> _buildConfigItems(Map<int, bool> tapStatues,
      List<GlobalKey> keys, List<_ConfigItem> buildItems) {
    final result = <Widget>[];
    for (var i = 0; i < buildItems.length; i++) {
      final buildItem = buildItems[i];
      result.add(ListTile(
          title: Text(buildItem.label),
          subtitle: tapStatues[i] ?? false
              ? buildItem.inputItem(keys[i])
              : Text(buildItem.text()),
          onTap: () {
            tapStatues[i] = true;
            // set other false
            for (var j = 0; j < tapStatues.length; j++) {
              if (j != i) {
                tapStatues[j] = false;
              }
            }
            WidgetsBinding.instance.addPostFrameCallback((timeStamp) {
              _tapInputWidget(keys[i]);
            });
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
  late Widget Function(Key key) inputItem;

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
