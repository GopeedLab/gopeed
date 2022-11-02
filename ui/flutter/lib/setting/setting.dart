import 'package:external_path/external_path.dart';
import 'package:flutter/material.dart';
import '../api/api.dart';
import 'package:path_provider/path_provider.dart';

import '../api/model/server_config.dart';
import '../util/util.dart';

class Setting {
  String host = "127.0.0.1";
  int port = 0;
  int connections = 0;
  String downloadDir = "";
  ThemeMode themeMode = ThemeMode.system;

  Setting({
    required this.host,
    required this.port,
    required this.connections,
    required this.downloadDir,
    required this.themeMode,
  });

  // singleton pattern
  static Setting? _instance;

  static Setting get instance {
    _instance ??= Setting._internal();
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
    if (themeMode == null) {
      this.themeMode = ThemeMode.system;
    } else {
      this.themeMode = ThemeMode.values.byName(themeMode);
    }

    if (Util.isWeb()) {
      downloadDir = "./";
      return;
    }
    if (Util.isAndroid()) {
      downloadDir =
          '${await ExternalPath.getExternalStoragePublicDirectory(ExternalPath.DIRECTORY_DOWNLOADS)}/com.gopeed';
      return;
    }

    if (downloadDir.isEmpty) {
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
      },
    );
    await putConfig(config);
  }
}
