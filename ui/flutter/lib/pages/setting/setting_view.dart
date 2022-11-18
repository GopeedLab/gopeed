import 'dart:async';

import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:get/get.dart';

import '../../i18n/messages.dart';
import '../../util/util.dart';
import '../../widget/directory_selector.dart';
import '../app/app_controller.dart';
import 'setting_controller.dart';

const _padding = SizedBox(height: 10);
const _configWidth = 250.0;

class SettingView extends GetView<SettingController> {
  final _settingLocaleKey = 'setting.locale.';

  const SettingView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final appController = Get.find<AppController>();
    final downloaderCfg = appController.downloaderConfig;
    final startCfg = appController.startConfig;

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
      _buildConfigItem(
          'setting.theme'.tr,
          () => _getThemeName(downloaderCfg.value.extra?.themeMode),
          (Key key) => DropdownButton<String>(
                key: key,
                value: downloaderCfg.value.extra?.themeMode,
                onChanged: (value) async {
                  downloaderCfg.update((val) {
                    val?.extra?.themeMode = value!;
                  });
                  Get.changeThemeMode(ThemeMode.values.byName(value!));
                  controller.clearTapStatus();

                  await debounceSave();
                },
                items: ThemeMode.values
                    .map((e) => DropdownMenuItem<String>(
                          value: e.name,
                          child: Text(_getThemeName(e.name)),
                        ))
                    .toList(),
              )),
      _buildConfigItem(
          'setting.downloadDir'.tr, () => downloaderCfg.value.downloadDir,
          (Key key) {
        final downloadDirController =
            TextEditingController(text: downloaderCfg.value.downloadDir);
        downloadDirController.addListener(() async {
          if (downloadDirController.text != downloaderCfg.value.downloadDir) {
            downloaderCfg.value.downloadDir = downloadDirController.text;
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
      _buildConfigItem(
          'setting.locale'.tr,
          () => _getLocaleName(downloaderCfg.value.extra?.locale),
          (Key key) => DropdownButton<String>(
                key: key,
                value: downloaderCfg.value.extra?.locale,
                isDense: true,
                onChanged: (value) async {
                  downloaderCfg.update((val) {
                    val!.extra!.locale = value!;
                  });
                  Get.updateLocale(toLocale(value!));
                  controller.clearTapStatus();

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

    final advancedConfigItems = [
      _buildConfigItem(
        '接口协议',
        () => startCfg.value.network == 'tcp'
            ? 'TCP ${startCfg.value.address}'
            : 'Unix',
        (Key key) {
          final items = <Widget>[
            SizedBox(
              width: 150,
              child: DropdownButtonFormField<String>(
                value: startCfg.value.network,
                onChanged: (value) async {
                  startCfg.update((val) {
                    val!.network = value!;
                  });

                  await debounceSave();
                },
                items: [
                  !Util.isMobile()
                      ? const DropdownMenuItem<String>(
                          value: 'tcp',
                          child: Text('TCP'),
                        )
                      : null,
                  Util.isUnix()
                      ? const DropdownMenuItem<String>(
                          value: 'unix',
                          child: Text('Unix'),
                        )
                      : null,
                ].where((e) => e != null).map((e) => e!).toList(),
              ),
            )
          ];
          if (Util.isDesktop() && startCfg.value.network == 'tcp') {
            final arr = startCfg.value.address.split(":");
            var ip = "127.0.0.1";
            var port = "0";
            if (arr.length > 1) {
              ip = arr[0];
              port = arr[1];
            }

            final ipController = TextEditingController(text: ip);
            final portController = TextEditingController(text: port);
            updateAddress() async {
              final newAddress = "${ipController.text}:${portController.text}";
              if (newAddress != startCfg.value.address) {
                startCfg.value.address = newAddress;

                await debounceSave();
              }
            }

            ipController.addListener(updateAddress);
            portController.addListener(updateAddress);
            items.addAll([
              const Padding(padding: EdgeInsets.only(left: 20)),
              SizedBox(
                width: 200,
                child: TextFormField(
                  controller: ipController,
                  decoration: const InputDecoration(
                    labelText: "IP",
                    contentPadding: EdgeInsets.all(0.0),
                  ),
                  keyboardType: TextInputType.number,
                  inputFormatters: [
                    FilteringTextInputFormatter.allow(RegExp("[0-9.]")),
                  ],
                ),
              ),
              const Padding(padding: EdgeInsets.only(left: 10)),
              SizedBox(
                width: 200,
                child: TextFormField(
                  controller: portController,
                  decoration: const InputDecoration(
                    labelText: '端口',
                    contentPadding: EdgeInsets.all(0.0),
                  ),
                  keyboardType: TextInputType.number,
                  inputFormatters: [
                    FilteringTextInputFormatter.digitsOnly,
                    _NumericalRangeFormatter(min: 0, max: 65535),
                  ],
                ),
              ),
            ]);
          }

          return Form(
            child: Row(
              children: items,
            ),
          );
        },
      ),
    ];

    if (Util.isDesktop() && startCfg.value.network == 'tcp') {
      advancedConfigItems.add(_buildConfigItem(
          '接口令牌', () => startCfg.value.apiToken.isEmpty ? "未设置" : '已设置',
          (Key key) {
        final apiTokenController =
            TextEditingController(text: startCfg.value.apiToken);
        apiTokenController.addListener(() async {
          if (apiTokenController.text != startCfg.value.apiToken) {
            startCfg.value.apiToken = apiTokenController.text;

            await debounceSave();
          }
        });
        apiTokenController.addListener(() async {});
        return TextField(
          key: key,
          obscureText: true,
          controller: apiTokenController,
          focusNode: FocusNode(),
        );
      }));
    }

    return Scaffold(
        appBar: AppBar(
          title: Text('setting.title'.tr),
          centerTitle: true,
        ),
        body: GestureDetector(
          onTap: () {
            controller.clearTapStatus();
          },
          child: Container(
            alignment: Alignment.topLeft,
            child: SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('setting.basic'.tr, style: context.textTheme.titleLarge),
                  _padding,
                  Obx(() => Card(
                          child: Column(
                        children: basicConfigItems.map((e) => e()).toList(),
                      ))),
                  _padding,
                  Text('setting.advanced'.tr,
                      style: context.textTheme.titleLarge),
                  _padding,
                  Obx(() => Card(
                          child: Column(
                        children: [
                          ...advancedConfigItems.map((e) => e()).toList(),
                          // DefaultTabController(
                          //   length: 2,
                          //   child: Column(
                          //     mainAxisSize: MainAxisSize.min,
                          //     children: <Widget>[
                          //       Container(
                          //         child: TabBar(tabs: [
                          //           Tab(text: "HTTP"),
                          //           Tab(text: "BitTorrent"),
                          //         ]),
                          //       ),
                          //       Container(
                          //         //Add this to give height
                          //         height: MediaQuery.of(context).size.height,
                          //         child: TabBarView(children: [
                          //           Container(
                          //             child: Text("Home Body"),
                          //           ),
                          //           Container(
                          //             child: Text("Articles Body"),
                          //           ),
                          //         ]),
                          //       ),
                          //     ],
                          //   ),
                          // ),
                        ],
                      ))),
                ],
              ).paddingAll(16),
            ),
          ),
        ));
  }

  void _tapInputWidget(GlobalKey key) {
    if (key.currentContext == null) {
      return;
    }

    if (key.currentContext?.widget is TextField) {
      final textField = key.currentContext?.widget as TextField;
      textField.focusNode?.requestFocus();
      return;
    }

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

  // void _tapInputWidget(GlobalKey key) {
  //   dynamic detector;
  //   void searchForGestureDetector(BuildContext? element) {
  //     element?.visitChildElements((element) {
  //       if (element.widget is GestureDetector ||
  //           element.widget is TextSelectionGestureDetector) {
  //         detector = element.widget;
  //         return;
  //       } else {
  //         searchForGestureDetector(element);
  //       }
  //     });
  //   }

  //   searchForGestureDetector(key.currentContext);

  //   if (detector != null) {
  //     if (detector is GestureDetector) {
  //       detector.onTap?.call();
  //     } else if (detector is TextSelectionGestureDetector) {
  //       detector.onTapDown?.call(TapDownDetails());
  //     }
  //   }
  // }

  Widget Function() _buildConfigItem(
      String label, String Function() text, Widget Function(Key key) input) {
    final tapStatues = controller.tapStatues;
    tapStatues.add(false);
    final i = controller.tapStatues.length - 1;
    final key = GlobalKey();
    return () => ListTile(
        title: Text(label),
        subtitle: tapStatues[i] ? input(key) : Text(text()),
        onTap: () {
          tapStatues[i] = true;
          // set other false
          for (var j = 0; j < tapStatues.length; j++) {
            if (j != i) {
              tapStatues[j] = false;
            }
          }

          WidgetsBinding.instance.addPostFrameCallback((timeStamp) {
            _tapInputWidget(key);
          });
        });
  }

  String _getLocaleName(String? locale) {
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

  String _getThemeName(String? themeMode) {
    switch (ThemeMode.values.byName(themeMode ?? ThemeMode.system.name)) {
      case ThemeMode.light:
        return 'setting.themeLight'.tr;
      case ThemeMode.dark:
        return 'setting.themeDark'.tr;
      default:
        return 'setting.themeSystem'.tr;
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
