import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:get/get.dart';
import 'package:intl/intl.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../generated/locales.g.dart';
import '../../../../i18n/messages.dart';
import '../../../../util/package_info.dart';
import '../../../../util/util.dart';
import '../../../views/views/check_list_view.dart';
import '../../../views/views/directory_selector.dart';
import '../../../views/views/outlined_button_loading.dart';
import '../../app/controllers/app_controller.dart';
import '../controllers/setting_controller.dart';

const _padding = SizedBox(height: 10);
final _divider = const Divider().paddingOnly(left: 10, right: 10);

class SettingView extends GetView<SettingController> {
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
          Get.snackbar('tip'.tr, 'effectAfterRestart'.tr);
        }
      });
      return completer.future;
    }

    // download basic config items start
    final buildDownloadDir = _buildConfigItem(
        'downloadDir', () => downloaderCfg.value.downloadDir, (Key key) {
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
        'connections'.tr, () => httpConfig.connections.toString(), (Key key) {
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
    final btExtConfig = downloaderCfg.value.extra.bt;

    final buildBtTrackerSubscribeUrls = _buildConfigItem(
        'subscribeTracker'.tr,
        () => 'items'.trParams(
            {'count': btExtConfig.trackerSubscribeUrls.length.toString()}),
        (Key key) {
      final trackerUpdateController = OutlinedButtonLoadingController();
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            height: 200,
            child: CheckListView(
              items: allTrackerSubscribeUrls,
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
                    await appController.trackerUpdate();
                  } catch (e) {
                    Get.snackbar('error'.tr, 'subscribeFail'.tr);
                  } finally {
                    trackerUpdateController.stop();
                  }
                },
                controller: trackerUpdateController,
                child: Text('update'.tr),
              ),
              Expanded(
                child: SwitchListTile(
                    controlAffinity: ListTileControlAffinity.leading,
                    value: true,
                    onChanged: (bool value) {},
                    title: Text('updateDaily'.tr)),
              ),
            ],
          ),
          Text('lastUpdate'.trParams({
            'time': btExtConfig.lastTrackerUpdateTime != null
                ? DateFormat('yyyy-MM-dd HH:mm:ss')
                    .format(btExtConfig.lastTrackerUpdateTime!)
                : ''
          })),
        ],
      );
    });
    final buildBtTrackers = _buildConfigItem(
        'addTracker'.tr,
        () => 'items'
            .trParams({'count': btExtConfig.customTrackers.length.toString()}),
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
          hintText: 'addTrackerHit'.tr,
        ),
        onChanged: (value) async {
          btExtConfig.customTrackers = ls.convert(value);
          appController.refreshTrackers();

          await debounceSave();
        },
      );
    });

    // ui config items start
    final buildTheme = _buildConfigItem(
        'theme',
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
        'locale',
        () => AppTranslation
            .translations[downloaderCfg.value.extra.locale]!['label']!,
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
              items: AppTranslation.translations.keys
                  .map((e) => DropdownMenuItem<String>(
                        value: e,
                        child: Text(AppTranslation.translations[e]!['label']!),
                      ))
                  .toList(),
            ));

    // about config items start
    buildHomepage() => ListTile(
          title: Text('homepage'.tr),
          subtitle: const Text('https://github.com/GopeedLab/gopeed'),
          onTap: () {
            launchUrl(Uri.parse('https://github.com/GopeedLab/gopeed'),
                mode: LaunchMode.externalApplication);
          },
        );
    buildVersion() => ListTile(
          title: Text('version'.tr),
          subtitle: Text(packageInfo.version),
        );

    // advanced config items start
    final buildApiProtocol = _buildConfigItem(
      'protocol'.tr,
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
          final arr = startCfg.value.address.split(':');
          var ip = '127.0.0.1';
          var port = '0';
          if (arr.length > 1) {
            ip = arr[0];
            port = arr[1];
          }

          final ipController = TextEditingController(text: ip);
          final portController = TextEditingController(text: port);
          updateAddress() async {
            final newAddress = '${ipController.text}:${portController.text}';
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
                  labelText: 'IP',
                  contentPadding: EdgeInsets.all(0.0),
                ),
                keyboardType: TextInputType.number,
                inputFormatters: [
                  FilteringTextInputFormatter.allow(RegExp('[0-9.]')),
                ],
              ),
            ),
            const Padding(padding: EdgeInsets.only(left: 10)),
            SizedBox(
              width: 200,
              child: TextFormField(
                controller: portController,
                decoration: InputDecoration(
                  labelText: 'port'.tr,
                  contentPadding: const EdgeInsets.all(0.0),
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
    final buildApiToken = _buildConfigItem('apiToken'.tr,
        () => startCfg.value.apiToken.isEmpty ? 'notSet'.tr : 'seted'.tr,
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
                          text: 'basic'.tr,
                        ),
                        Tab(
                          text: 'advanced'.tr,
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
                        Text('general'.tr),
                        Card(
                            child: Column(
                          children: _addDivider([
                            buildDownloadDir(),
                          ]),
                        )),
                        const Text('HTTP'),
                        Card(
                            child: Column(
                          children: _addDivider([
                            buildHttpConnections(),
                          ]),
                        )),
                        const Text('BitTorrent'),
                        Card(
                            child: Column(
                          children: _addDivider([
                            buildBtTrackerSubscribeUrls(),
                            buildBtTrackers(),
                          ]),
                        )),
                        Text('ui'.tr),
                        Card(
                            child: Column(
                          children: _addDivider([
                            buildTheme(),
                            buildLocale(),
                          ]),
                        )),
                        Text('about'.tr),
                        Card(
                            child: Column(
                          children: _addDivider([
                            buildHomepage(),
                            buildVersion(),
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

  String _getThemeName(String? themeMode) {
    switch (ThemeMode.values.byName(themeMode ?? ThemeMode.system.name)) {
      case ThemeMode.light:
        return 'themeLight'.tr;
      case ThemeMode.dark:
        return 'themeDark'.tr;
      default:
        return 'themeSystem'.tr;
    }
  }
}

// class _ConfigItem {
//   late String label;
//   late String Function() text;
//   late Widget Function(Key key) inputItem;
//
//   _ConfigItem(this.label, this.text, this.inputItem);
// }

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
