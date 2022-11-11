import 'dart:ui' as ui;

import 'package:external_path/external_path.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:path_provider/path_provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../api/api.dart';
import '../../api/model/downloader_config.dart';
import '../../i18n/messages.dart';
import '../../util/util.dart';

class StartConfig {
  late String network;
  late String address;
  late int runningPort;

  String get runningAddress {
    if (network == 'unix') {
      return address;
    }
    return '${address.split(':').first}:$runningPort';
  }
}

const _startConfigNetwork = "start.network";
const _startConfigAddress = "start.address";

const unixSocketPath = 'gopeed.sock';

class AppController extends GetxController {
  final startConfig = StartConfig().obs;
  final downloaderConfig = DownloaderConfig().obs;

  Future<void> loadStartConfig() async {
    final prefs = await SharedPreferences.getInstance();
    final network = prefs.getString(_startConfigNetwork);
    final address = prefs.getString(_startConfigAddress);

    // loaded from shared preferences
    if (network != null && address != null) {
      startConfig.value.network = network;
      startConfig.value.address = address;
      return;
    }

    // default value
    if (!Util.isUnix()) {
      // not support unix socket, use tcp
      startConfig.value.network = "tcp";
      startConfig.value.address = "127.0.0.1:0";
    } else {
      startConfig.value.network = "unix";
      if (Util.isDesktop()) {
        startConfig.value.address = unixSocketPath;
      }
      if (Util.isMobile()) {
        startConfig.value.address =
            "${(await getTemporaryDirectory()).path}/$unixSocketPath";
      }
    }
  }

  Future<void> loadDownloaderConfig() async {
    downloaderConfig.value = await getConfig();
    final config = downloaderConfig.value;
    config.protocolConfig ??= ProtocolConfig();
    if (config.protocolConfig!.http.connections == 0) {
      config.protocolConfig!.http.connections = 16;
    }
    config.extra ??= ExtraConfig();
    final extra = config.extra!;
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
