import 'dart:convert';
import 'dart:ui' as ui;

import 'package:external_path/external_path.dart';
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
      result.addAll(await _fetchTrackers(u));
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
  }

  Future<void> trackerUpdateOnStart() async {
    final btExtConfig = downloaderConfig.value.extra.bt;
    final lastUpdateTime = btExtConfig.lastTrackerUpdateTime;
    // if last update time is null or more than 1 day, update trackers
    if (lastUpdateTime == null ||
        lastUpdateTime.difference(DateTime.now()).inDays < 0) {
      await trackerUpdate();
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
      extra.locale = getLocaleKey(ui.window.locale);
    }

    if (Util.isAndroid()) {
      config.downloadDir =
          '${await ExternalPath.getExternalStoragePublicDirectory(ExternalPath.DIRECTORY_DOWNLOADS)}/com.gopeed';
      return;
    }

    if (config.downloadDir.isEmpty) {
      if (Util.isWeb()) {
        config.downloadDir = './';
      }
      if (Util.isDesktop()) {
        config.downloadDir = (await getDownloadsDirectory())?.path ?? "./";
      }
    }
  }

  Future<void> saveConfig() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_startConfigNetwork, startConfig.value.network);
    await prefs.setString(_startConfigAddress, startConfig.value.address);
    await putConfig(downloaderConfig.value);
  }
}
