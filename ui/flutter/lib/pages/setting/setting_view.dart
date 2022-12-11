import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:get/get.dart';

import '../../api/api.dart';
import '../../i18n/messages.dart';
import '../../util/util.dart';
import '../../widget/check_list_view.dart';
import '../../widget/directory_selector.dart';
import '../../widget/outlined_button_loading.dart';
import '../app/app_controller.dart';
import 'setting_controller.dart';

const _padding = SizedBox(height: 10);
final _divider = const Divider().paddingOnly(left: 10, right: 10);

class SettingView extends GetView<SettingController> {
  final _settingLocaleKey = 'setting.locale.';

  const SettingView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final appController = Get.find<AppController>();
    final downloaderCfg = appController.downloaderConfig;
    final startCfg = appController.startConfig;

    Timer? timer;
    debounceSave({bool needRestart = false}) {
      var completer = Completer<void>();
      timer?.cancel();
      timer = Timer(const Duration(milliseconds: 500), () {
        appController
            .saveConfig()
            .then(completer.complete)
            .onError(completer.completeError);
        if (needRestart) {
          Get.snackbar("提示", "此配置项在下次启动时生效");
        }
      });
      return completer.future;
    }

    // download basic config items start
    final buildDownloadDir = _buildConfigItem(
        'setting.downloadDir', () => downloaderCfg.value.downloadDir,
        (Key key) {
      final downloadDirController =
          TextEditingController(text: downloaderCfg.value.downloadDir);
      downloadDirController.addListener(() async {
        if (downloadDirController.text != downloaderCfg.value.downloadDir) {
          downloaderCfg.value.downloadDir = downloadDirController.text;
          if (Util.isDesktop()) {
            controller.clearTap();
          }

          await debounceSave();
        }
      });
      return DirectorySelector(
        controller: downloadDirController,
        showLabel: false,
      );
    });

    // http config items start
    final httpConfig = downloaderCfg.value.protocolConfig.http;
    final buildHttpConnections = _buildConfigItem(
        'setting.connections'.tr, () => httpConfig.connections.toString(),
        (Key key) {
      final connectionsController =
          TextEditingController(text: httpConfig.connections.toString());
      connectionsController.addListener(() async {
        if (connectionsController.text.isNotEmpty &&
            connectionsController.text != httpConfig.connections.toString()) {
          httpConfig.connections = int.parse(connectionsController.text);

          await debounceSave();
        }
      });

      return TextField(
        key: key,
        focusNode: FocusNode(),
        controller: connectionsController,
        keyboardType: TextInputType.number,
        inputFormatters: [
          FilteringTextInputFormatter.digitsOnly,
          _NumericalRangeFormatter(min: 1, max: 256),
        ],
      );
    });

    // bt config items start
    final btConfig = downloaderCfg.value.protocolConfig.bt;
    final btExtConfig = downloaderCfg.value.extra.bt;
    refreshTrackers() {
      btConfig.trackers.clear();
      btConfig.trackers.addAll(btExtConfig.subscribeTrackers);
      btConfig.trackers.addAll(btExtConfig.customTrackers);
    }

    final buildBtTrackerSubscribeUrls = _buildConfigItem(
        '订阅 tracker'.tr, () => '${btExtConfig.trackerSubscribeUrls.length}条',
        (Key key) {
      final trackerUpdateController = OutlinedButtonLoadingController();
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            height: 200,
            child: CheckListView(
              items: const [
                'https://github.com/ngosang/trackerslist/raw/master/trackers_all.txt',
                'https://github.com/ngosang/trackerslist/raw/master/trackers_all_http.txt',
                'https://github.com/ngosang/trackerslist/raw/master/trackers_all_https.txt',
                'https://github.com/ngosang/trackerslist/raw/master/trackers_all_ip.txt',
                'https://github.com/ngosang/trackerslist/raw/master/trackers_all_udp.txt',
                'https://github.com/ngosang/trackerslist/raw/master/trackers_all_ws.txt',
                'https://github.com/ngosang/trackerslist/raw/master/trackers_best.txt',
                'https://github.com/ngosang/trackerslist/raw/master/trackers_best_ip.txt',
                'https://github.com/XIU2/TrackersListCollection/raw/master/all.txt',
                'https://github.com/XIU2/TrackersListCollection/raw/master/best.txt',
                'https://github.com/XIU2/TrackersListCollection/raw/master/http.txt',
              ],
              checked: btExtConfig.trackerSubscribeUrls,
              onChanged: (value) {
                btExtConfig.trackerSubscribeUrls = value;

                debounceSave();
              },
            ),
          ),
          _padding,
          Row(
            children: [
              OutlinedButtonLoading(
                onPressed: () async {
                  trackerUpdateController.start();
                  try {
                    final result = <String>[];
                    for (var u in btExtConfig.trackerSubscribeUrls) {
                      result.addAll(await getTrackers(u));
                    }
                    btExtConfig.subscribeTrackers.clear();
                    btExtConfig.subscribeTrackers.addAll(result);
                    refreshTrackers();
                    downloaderCfg.update((val) {
                      val!.extra.bt.lastTrackerUpdateTime = DateTime.now();
                    });

                    await debounceSave();
                  } catch (e) {
                    Get.snackbar("错误", "更新失败");
                  } finally {
                    trackerUpdateController.stop();
                  }
                },
                controller: trackerUpdateController,
                child: Text("更新"),
              ),
              Flexible(
                child: SizedBox(
                  width: 200,
                  child: SwitchListTile(
                      // contentPadding: EdgeInsets.zero,
                      controlAffinity: ListTileControlAffinity.leading,
                      value: true,
                      onChanged: (bool value) {},
                      title: Text("每天自动更新")), // TODO 自动更新and国际化调整
                ),
              ),
            ],
          ),
          Text("上次更新：${btExtConfig.lastTrackerUpdateTime?.toLocal() ?? ''}"),
        ],
      );
    });
    final buildBtTrackers = _buildConfigItem(
        '添加 tracker'.tr, () => '${btExtConfig.customTrackers.length}条',
        (Key key) {
      final trackersController = TextEditingController(
          text: btExtConfig.customTrackers.join('\r\n').toString());
      const ls = LineSplitter();
      return TextField(
        key: key,
        focusNode: FocusNode(),
        controller: trackersController,
        keyboardType: TextInputType.multiline,
        maxLines: 5,
        decoration: InputDecoration(
          hintText: '请输入 tracker 地址，每行一条'.tr,
        ),
        onChanged: (value) async {
          btExtConfig.customTrackers = ls.convert(value);
          refreshTrackers();

          await debounceSave();
        },
      );
    });

    // ui config items start
    final buildTheme = _buildConfigItem(
        'setting.theme',
        () => _getThemeName(downloaderCfg.value.extra.themeMode),
        (Key key) => DropdownButton<String>(
              key: key,
              value: downloaderCfg.value.extra.themeMode,
              onChanged: (value) async {
                downloaderCfg.update((val) {
                  val?.extra.themeMode = value!;
                });
                Get.changeThemeMode(ThemeMode.values.byName(value!));
                controller.clearTap();

                await debounceSave();
              },
              items: ThemeMode.values
                  .map((e) => DropdownMenuItem<String>(
                        value: e.name,
                        child: Text(_getThemeName(e.name)),
                      ))
                  .toList(),
            ));
    final buildLocale = _buildConfigItem(
        'setting.locale',
        () => _getLocaleName(downloaderCfg.value.extra.locale),
        (Key key) => DropdownButton<String>(
              key: key,
              value: downloaderCfg.value.extra.locale,
              isDense: true,
              onChanged: (value) async {
                downloaderCfg.update((val) {
                  val!.extra.locale = value!;
                });
                Get.updateLocale(toLocale(value!));
                controller.clearTap();

                await debounceSave();
              },
              items: _getLocales()
                  .map((e) => DropdownMenuItem<String>(
                        value: e.substring(_settingLocaleKey.length),
                        child: Text(e.tr),
                      ))
                  .toList(),
            ));

    // advanced config items start
    final buildApiProtocol = _buildConfigItem(
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

                await debounceSave(needRestart: true);
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

              await debounceSave(needRestart: true);
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
    );
    final buildApiToken = _buildConfigItem(
        '接口令牌', () => startCfg.value.apiToken.isEmpty ? "未设置" : '已设置',
        (Key key) {
      final apiTokenController =
          TextEditingController(text: startCfg.value.apiToken);
      apiTokenController.addListener(() async {
        if (apiTokenController.text != startCfg.value.apiToken) {
          startCfg.value.apiToken = apiTokenController.text;

          await debounceSave(needRestart: true);
        }
      });
      apiTokenController.addListener(() async {});
      return TextField(
        key: key,
        obscureText: true,
        controller: apiTokenController,
        focusNode: FocusNode(),
      );
    });

    return Obx(() {
      return GestureDetector(
        onTap: () {
          controller.clearTap();
        },
        child: DefaultTabController(
          length: 2,
          child: Scaffold(
              appBar: PreferredSize(
                  preferredSize: Size.fromHeight(56),
                  child: AppBar(
                    bottom: TabBar(
                      tabs: [
                        Tab(
                          text: '通用',
                        ),
                        Tab(
                          text: '高级',
                        ),
                      ],
                    ),
                  )),
              body: TabBarView(
                children: [
                  SingleChildScrollView(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: _addPadding([
                        Text('基础'),
                        Card(
                            child: Column(
                          children: _addDivider([
                            buildDownloadDir(),
                          ]),
                        )),
                        Text('HTTP'),
                        Card(
                            child: Column(
                          children: _addDivider([
                            buildHttpConnections(),
                          ]),
                        )),
                        Text('BitTorrent'),
                        Card(
                            child: Column(
                          children: _addDivider([
                            buildBtTrackerSubscribeUrls(),
                            buildBtTrackers(),
                          ]),
                        )),
                        Text('界面'),
                        Card(
                            child: Column(
                          children: _addDivider([
                            buildTheme(),
                            buildLocale(),
                          ]),
                        )),
                      ]),
                    ),
                  ),
                  Column(
                    children: [
                      Card(
                          child: Column(
                        children: [
                          ..._addDivider([
                            buildApiProtocol(),
                            Util.isDesktop() && startCfg.value.network == 'tcp'
                                ? buildApiToken()
                                : null,
                          ]),
                        ],
                      )),
                    ],
                  ),
                ],
              ).paddingOnly(left: 16, right: 16, top: 16, bottom: 64)),
        ),
      );
    });
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

    /* GestureDetector? detector;
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
    detector?.onTap?.call(); */
  }

  Widget Function() _buildConfigItem(
      String label, String Function() text, Widget Function(Key key) input) {
    final tapStatues = controller.tapStatues;
    final inputKey = GlobalKey();
    return () => ListTile(
        title: Text(label.tr),
        subtitle: tapStatues[label] ?? false ? input(inputKey) : Text(text()),
        onTap: () {
          controller.onTap(label);
          WidgetsBinding.instance.addPostFrameCallback((timeStamp) {
            _tapInputWidget(inputKey);
          });
        });
  }

  List<Widget> _addPadding(List<Widget> widgets) {
    final result = <Widget>[];
    for (var i = 0; i < widgets.length; i++) {
      result.add(widgets[i]);
      result.add(_padding);
    }
    return result;
  }

  List<Widget> _addDivider(List<Widget?> widgets) {
    final result = <Widget>[];
    final newArr = widgets.where((e) => e != null).map((e) => e!).toList();
    for (var i = 0; i < newArr.length; i++) {
      result.add(newArr[i]);
      if (i != newArr.length - 1) {
        result.add(_divider);
      }
    }
    return result;
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

  Future<List<String>> getTrackers(String subscribeUrl) async {
    final resp = await proxyRequest(subscribeUrl);
    if (resp.statusCode != 200) {
      throw Exception('Failed to get trackers');
    }
    const ls = LineSplitter();
    return ls.convert(resp.data).where((e) => e.isNotEmpty).toList();
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
