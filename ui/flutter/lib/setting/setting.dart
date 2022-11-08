import 'package:external_path/external_path.dart';
import 'package:flutter/material.dart';
import 'package:gopeed/i18n/messages.dart';
import '../api/api.dart';
import 'package:path_provider/path_provider.dart';

import 'dart:ui' as ui;

import '../api/model/server_config.dart';
import '../util/util.dart';

class Setting {
  String host = "127.0.0.1";
  int port = 0;
  int connections = 0;
  late String downloadDir;
  late ThemeMode themeMode;
  late String locale;

  // singleton pattern
  static Setting? _instance;

  static Setting get instance {
    _instance ??= Setting._internal();
    _instance!.themeMode = ThemeMode.system;
    _instance!.locale = getLocaleKey(ui.window.locale);
    return _instance!;
  }

  Setting._internal();

  Future<void> load() async {
    final config = await getConfig();
    host = config.host;
    port = config.port;
    connections = config.connections;
    downloadDir = config.downloadDir;
    final themeMode = config.extra?['themeMode'];
    if (themeMode != null) {
      this.themeMode = ThemeMode.values.byName(themeMode);
    }
    final locale = config.extra?['locale'];
    if (locale != null) {
      this.locale = locale;
    }

    if (Util.isAndroid()) {
      downloadDir =
          '${await ExternalPath.getExternalStoragePublicDirectory(ExternalPath.DIRECTORY_DOWNLOADS)}/com.gopeed';
      return;
    }

    if (downloadDir.isEmpty) {
      if (Util.isWeb()) {
        downloadDir = './';
      }
      if (Util.isDesktop()) {
        downloadDir = (await getDownloadsDirectory())!.path;
      }
    }
  }

  Future<void> save() async {
    final config = ServerConfig(
      host: host,
      port: port,
      connections: connections,
      downloadDir: downloadDir,
      extra: {
        'themeMode': themeMode.name,
        'locale': locale,
      },
    );
    await putConfig(config);
  }
}
