import 'dart:convert';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:path_provider/path_provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../api/api.dart';
import '../../api/model/downloader_config.dart';
import '../../core/common/start_config.dart';
import '../../i18n/messages.dart';
import '../../util/log_util.dart';
import '../../util/util.dart';

const _startConfigNetwork = "start.network";
const _startConfigAddress = "start.address";
const _startConfigApiToken = "start.apiToken";

const unixSocketPath = 'gopeed.sock';

const allTrackerSubscribeUrls = [
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
];
const allTrackerCdns = [
  // jsdelivr: https://cdn.jsdelivr.net/gh/ngosang/trackerslist/trackers_all.txt
  ["https://cdn.jsdelivr.net/gh", r".*github.com(/.*)/raw/master(/.*)"],
  // nuaa: https://hub.nuaa.cf/ngosang/trackerslist/raw/master/trackers_all.txt
  ["https://hub.nuaa.cf", r".*github.com(/.*)"]
];
final allTrackerSubscribeUrlCdns = Map.fromIterable(allTrackerSubscribeUrls,
    key: (v) => v as String,
    value: (v) {
      final ret = [v as String];
      for (final cdn in allTrackerCdns) {
        final reg = RegExp(cdn[1]);
        final match = reg.firstMatch(v.toString());
        var matchStr = "";
        for (var i = 1; i <= match!.groupCount; i++) {
          matchStr += match.group(i)!;
        }
        ret.add("${cdn[0]}$matchStr");
      }
      return ret;
    });

class AppController extends GetxController {
  static StartConfig? _defaultStartConfig;

  final startConfig = StartConfig().obs;
  final runningPort = 0.obs;
  final downloaderConfig = DownloaderConfig().obs;

  String runningAddress() {
    if (startConfig.value.network == 'unix') {
      return startConfig.value.address;
    }
    return '${startConfig.value.address.split(':').first}:$runningPort';
  }

  Future<StartConfig> _initDefaultStartConfig() async {
    if (_defaultStartConfig != null) {
      return _defaultStartConfig!;
    }
    _defaultStartConfig = StartConfig();
    if (!Util.isUnix()) {
      // not support unix socket, use tcp
      _defaultStartConfig!.network = "tcp";
      _defaultStartConfig!.address = "127.0.0.1:0";
    } else {
      _defaultStartConfig!.network = "unix";
      if (Util.isDesktop()) {
        _defaultStartConfig!.address = unixSocketPath;
      }
      if (Util.isMobile()) {
        _defaultStartConfig!.address =
            "${(await getTemporaryDirectory()).path}/$unixSocketPath";
      }
    }
    _defaultStartConfig!.apiToken = '';
    return _defaultStartConfig!;
  }

  Future<void> loadStartConfig() async {
    final defaultCfg = await _initDefaultStartConfig();
    final prefs = await SharedPreferences.getInstance();
    startConfig.value.network =
        prefs.getString(_startConfigNetwork) ?? defaultCfg.network;
    startConfig.value.address =
        prefs.getString(_startConfigAddress) ?? defaultCfg.address;
    startConfig.value.apiToken =
        prefs.getString(_startConfigApiToken) ?? defaultCfg.apiToken;
  }

  Future<void> loadDownloaderConfig() async {
    try {
      downloaderConfig.value = await getConfig();
    } catch (e) {
      logger.w("load downloader config fail", e);
      downloaderConfig.value = DownloaderConfig();
    }
    await _initDownloaderConfig();
  }

  Future<void> trackerUpdate() async {
    final btExtConfig = downloaderConfig.value.extra.bt;
    final result = <String>[];
    for (var u in btExtConfig.trackerSubscribeUrls) {
      var allFail = true;
      for (var cdn in allTrackerSubscribeUrlCdns[u]!) {
        try {
          result.addAll(await _fetchTrackers(u));
          allFail = false;
          break;
        } catch (e) {
          logger.w("subscribe trackers fail, url: $cdn", e);
        }
      }
      if (allFail) {
        throw Exception('subscribe trackers fail, network error');
      }
    }
    btExtConfig.subscribeTrackers.clear();
    btExtConfig.subscribeTrackers.addAll(result);
    downloaderConfig.update((val) {
      val!.extra.bt.lastTrackerUpdateTime = DateTime.now();
    });
    refreshTrackers();

    await saveConfig();
  }

  refreshTrackers() {
    final btConfig = downloaderConfig.value.protocolConfig.bt;
    final btExtConfig = downloaderConfig.value.extra.bt;
    btConfig.trackers.clear();
    btConfig.trackers.addAll(btExtConfig.subscribeTrackers);
    btConfig.trackers.addAll(btExtConfig.customTrackers);
    // remove duplicate
    btConfig.trackers.toSet().toList();
  }

  Future<void> trackerUpdateOnStart() async {
    final btExtConfig = downloaderConfig.value.extra.bt;
    final lastUpdateTime = btExtConfig.lastTrackerUpdateTime;
    // if last update time is null or more than 1 day, update trackers
    if (lastUpdateTime == null ||
        lastUpdateTime.difference(DateTime.now()).inDays < 0) {
      try {
        await trackerUpdate();
      } catch (e) {
        logger.w("tracker update fail", e);
      }
    }
  }

  Future<List<String>> _fetchTrackers(String subscribeUrl) async {
    final resp = await proxyRequest(subscribeUrl);
    if (resp.statusCode != 200) {
      throw Exception('Failed to get trackers');
    }
    const ls = LineSplitter();
    return ls.convert(resp.data).where((e) => e.isNotEmpty).toList();
  }

  _initDownloaderConfig() async {
    final config = downloaderConfig.value;
    if (config.protocolConfig.http.connections == 0) {
      config.protocolConfig.http.connections = 16;
    }
    final extra = config.extra;
    if (extra.themeMode.isEmpty) {
      extra.themeMode = ThemeMode.system.name;
    }
    if (extra.locale.isEmpty) {
      final systemLocale = getLocaleKey(ui.window.locale);
      extra.locale = messages.keys.containsKey(systemLocale)
          ? systemLocale
          : getLocaleKey(fallbackLocale);
    }
    if (extra.bt.trackerSubscribeUrls.isEmpty) {
      // default select all tracker subscribe urls
      extra.bt.trackerSubscribeUrls.addAll(allTrackerSubscribeUrls);
    }

    if (Util.isDesktop()) {
      config.downloadDir = (await getDownloadsDirectory())?.path ?? "./";
    } else if (Util.isAndroid()) {
      config.downloadDir = (await getExternalStorageDirectory())?.path ??
          (await getApplicationDocumentsDirectory()).path;
      return;
    } else if (Util.isIOS()) {
      config.downloadDir = (await getApplicationDocumentsDirectory()).path;
    } else {
      config.downloadDir = './';
    }
  }

  Future<void> saveConfig() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_startConfigNetwork, startConfig.value.network);
    await prefs.setString(_startConfigAddress, startConfig.value.address);
    await putConfig(downloaderConfig.value);
  }
}
