import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:install_plugin/install_plugin.dart';
import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../api/api.dart';
import 'log_util.dart';
import 'package_info.dart';
import 'util.dart';

enum Channel {
  windows,
  macos,
  linuxFlathub,
  linuxSnap,
  linuxDeb,
  linuxAppImage,
  androidApk,
  iosIpa,
  docker,
}

const _channelEnv = String.fromEnvironment("UPDATE_CHANNEL");
final _channel =
    Channel.values.where((e) => e.name == _channelEnv).firstOrNull ??
        () {
          if (Util.isWindows()) {
            return Channel.windows;
          } else if (Util.isMacos()) {
            return Channel.macos;
          } else if (Util.isAndroid()) {
            return Channel.androidApk;
          } else if (Util.isIOS()) {
            return Channel.iosIpa;
          } else {
            return null;
          }
        }();
final _updaterBin = Util.isWindows() ? "updater.exe" : "updater";

Future<void> installUpdater() async {
  await Util.installAsset(
      'assets/exec/$_updaterBin', Util.homePathJoin(_updaterBin),
      executable: true);
}

Future<void> checkUpdate() async {
  String? releaseDataStr;
  try {
    releaseDataStr = (await proxyRequest(
            "https://api.github.com/repos/GopeedLab/gopeed/releases/latest"))
        .data;
  } catch (e) {
    releaseDataStr =
        (await proxyRequest("https://gopeed.com/api/release")).data;
  }
  if (releaseDataStr == null) {
    return;
  }
  final releaseData = jsonDecode(releaseDataStr);
  final tagName = releaseData["tag_name"];
  if (tagName == null) {
    return;
  }
  final latestVersion = releaseData["tag_name"].substring(1);

  // compare version x.y.z to x.y.z
  final currentVersion = packageInfo.version;
  var isNewVersion = false;
  if (latestVersion != currentVersion) {
    final currentVersionList = currentVersion.split(".");
    final latestVersionList = latestVersion.split(".");
    for (var i = 0; i < currentVersionList.length; i++) {
      if (int.parse(latestVersionList[i]) > int.parse(currentVersionList[i])) {
        isNewVersion = true;
        break;
      }
    }
  }
  if (isNewVersion) {
    await showDialog(
      context: Get.context!,
      builder: (context) {
        bool isDownloading = false;
        double progress = 0;

        return StatefulBuilder(
          builder: (context, setState) {
            return AlertDialog(
              title:
                  Text('newVersionTitle'.trParams({'version': latestVersion})),
              content: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('更新日志：asdasda'),
                  if (isDownloading)
                    LinearProgressIndicator(value: progress.toDouble())
                  else
                    Container(),
                ],
              ),
              actions: [
                TextButton(
                  onPressed: () {
                    Get.back();
                  },
                  child: Text('newVersionLater'.tr),
                ),
                TextButton(
                  onPressed: () async {
                    setState(() {
                      isDownloading = true;
                    });
                    await update(latestVersion, (received, total) {
                      setState(() {
                        progress = received / total;
                        print("progress: $progress");
                      });
                    });
                  },
                  child: Text('newVersionUpdate'.tr),
                ),
              ],
            );
          },
        );
      },
    );
  }
}

Future<void> update(String version, Function(int, int) onProgress) async {
  final downloadUrl =
      'https://github.com/GopeedLab/gopeed/releases/download/v$version/${_getAssetName(version)}';
  final newVersionAssetPath = await _getAssetPath(version);
  if (downloadUrl.isNotEmpty) {
    final downloadClient = Dio();
    await downloadClient.download(downloadUrl, newVersionAssetPath,
        onReceiveProgress: onProgress);
  }
  switch (_channel) {
    case Channel.windows:
    case Channel.macos:
    case Channel.linuxDeb:
      // updater <pid> <asset> [log]
      await Process.start(Util.homePathJoin(_updaterBin), [
        pid.toString(),
        newVersionAssetPath,
        path.join(logsDir(), "updater.log")
      ]);
      break;
    case Channel.androidApk:
      await InstallPlugin.installApk(newVersionAssetPath);
      break;
    default:
      await launchUrl(
          Uri.parse(
              'https://github.com/GopeedLab/gopeed/releases/tag/v$version'),
          mode: LaunchMode.externalApplication);
      break;
  }
}

String _getAssetName(String version) {
  switch (_channel) {
    case Channel.windows:
      return 'Gopeed-v$version-windows-amd64-portable.zip';
    case Channel.macos:
      return 'Gopeed-v$version-macos.dmg';
    case Channel.linuxDeb:
      return 'Gopeed-v$version-linux-amd64.deb';
    case Channel.androidApk:
      return 'Gopeed-v$version-android.apk';
    default:
      return "";
  }
}

Future<String> _getAssetPath(String version) async {
  return path.join(
      (await getTemporaryDirectory()).path, _getAssetName(version));
}
