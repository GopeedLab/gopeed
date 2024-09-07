import 'dart:io';

import 'package:device_info_plus/device_info_plus.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:lecle_downloads_path_provider/lecle_downloads_path_provider.dart';
import 'package:path_provider/path_provider.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:toggle_switch/toggle_switch.dart';

import '../../util/mac_secure_util.dart';
import '../../util/message.dart';
import '../../util/util.dart';

final deviceInfo = DeviceInfoPlugin();

class DirectorySelector extends StatefulWidget {
  final TextEditingController controller;
  final bool showLabel;
  final bool showAndoirdToggle;

  const DirectorySelector(
      {Key? key,
      required this.controller,
      this.showLabel = true,
      this.showAndoirdToggle = false})
      : super(key: key);

  @override
  State<DirectorySelector> createState() => _DirectorySelectorState();
}

class _DirectorySelectorState extends State<DirectorySelector> {
  @override
  Widget build(BuildContext context) {
    Widget? buildSelectWidget() {
      if (Util.isDesktop()) {
        return IconButton(
            icon: const Icon(Icons.folder_open),
            onPressed: () async {
              var dir = await FilePicker.platform.getDirectoryPath();
              if (dir != null) {
                widget.controller.text = dir;
                MacSecureUtil.saveBookmark(dir);
              }
            });
      }
      // After Android 11, access to external storage is increasingly restricted, so it no longer supports selecting the download directory. However, if you do not download in external storage, all downloaded files will be deleted after the application is uninstalled.
      // Fortunately, so far, most Android devices can still access the system download directory.
      // For the sake of user experience, it is decided to only support selecting the application's internal directory and the system download directory. Also, a test for file write permission is performed when selecting the system download directory. If it cannot be written, selection is not allowed.
      if (Util.isAndroid() && widget.showAndoirdToggle) {
        final isSwitchToDownloadDir =
            widget.controller.text.endsWith('/Gopeed');

        return ToggleSwitch(
          initialLabelIndex: isSwitchToDownloadDir ? 1 : 0,
          totalSwitches: 2,
          icons: const [Icons.home, Icons.download],
          customWidths: const [50, 50],
          onToggle: (index) async {
            if (index == 0) {
              widget.controller.text =
                  (await getExternalStorageDirectory())?.path ??
                      (await getApplicationDocumentsDirectory()).path;
            } else {
              widget.controller.text =
                  '${(await DownloadsPath.downloadsDirectory())!.path}/Gopeed';
            }
          },
          cancelToggle: (index) async {
            if (index == 0) {
              return false;
            }

            final downloadDir =
                (await DownloadsPath.downloadsDirectory())?.path;
            if (downloadDir == null) {
              return true;
            }

            // Check and request external storage permission when sdk version < 30 (android 11)
            if ((await deviceInfo.androidInfo).version.sdkInt < 30) {
              var status = await Permission.storage.status;
              if (!status.isGranted) {
                status = await Permission.storage.request();
                if (!status.isGranted) {
                  showErrorMessage('noStoragePermission'.tr);
                  return true;
                }
              }
            }

            // Check write permission
            final fileRandomeName =
                "test_${DateTime.now().millisecondsSinceEpoch}.tmp";
            final testFile = File('$downloadDir/Gopeed/$fileRandomeName');
            try {
              await testFile.create(recursive: true);
              await testFile.writeAsString('test');
              await testFile.delete();
              return false;
            } catch (e) {
              showErrorMessage(e);
              return true;
            }
          },
        ).marginOnly(left: 10);
      }
      return null;
    }

    return Row(
      children: [
        Expanded(
            child: TextFormField(
          readOnly: Util.isWeb() ? false : true,
          controller: widget.controller,
          decoration: widget.showLabel
              ? InputDecoration(
                  labelText: 'downloadDir'.tr,
                )
              : null,
          validator: (v) {
            return v!.trim().isNotEmpty ? null : 'downloadDirValid'.tr;
          },
        )),
        buildSelectWidget()
      ].where((e) => e != null).map((e) => e!).toList(),
    );
  }
}
