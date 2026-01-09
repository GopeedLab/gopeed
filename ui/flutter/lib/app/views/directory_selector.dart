import 'dart:io';

import 'package:device_info_plus/device_info_plus.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:lecle_downloads_path_provider/lecle_downloads_path_provider.dart';
import 'package:path_provider/path_provider.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:toggle_switch/toggle_switch.dart';

import '../../util/message.dart';
import '../../util/util.dart';

final deviceInfo = DeviceInfoPlugin();

// Placeholder information for download directory
class PathPlaceholder {
  final String placeholder;
  final String description;
  final String example;

  const PathPlaceholder({
    required this.placeholder,
    required this.description,
    required this.example,
  });
}

// Available placeholders for download directory
List<PathPlaceholder> getPathPlaceholders() {
  final now = DateTime.now();
  final year = now.year.toString();
  final month = now.month.toString().padLeft(2, '0');
  final day = now.day.toString().padLeft(2, '0');

  return [
    PathPlaceholder(
      placeholder: '%year%',
      description: 'placeholderYear'.tr,
      example: year,
    ),
    PathPlaceholder(
      placeholder: '%month%',
      description: 'placeholderMonth'.tr,
      example: month,
    ),
    PathPlaceholder(
      placeholder: '%day%',
      description: 'placeholderDay'.tr,
      example: day,
    ),
    PathPlaceholder(
      placeholder: '%date%',
      description: 'placeholderDate'.tr,
      example: '$year-$month-$day',
    ),
  ];
}

// Render placeholders in a path with actual values
String renderPathPlaceholders(String path) {
  if (path.isEmpty) return path;

  final now = DateTime.now();
  final year = now.year.toString();
  final month = now.month.toString().padLeft(2, '0');
  final day = now.day.toString().padLeft(2, '0');
  final date = '$year-$month-$day';

  return path
      .replaceAll('%year%', year)
      .replaceAll('%month%', month)
      .replaceAll('%day%', day)
      .replaceAll('%date%', date);
}

class DirectorySelector extends StatefulWidget {
  final TextEditingController controller;
  final bool showLabel;
  final bool showAndoirdToggle;
  final bool allowEdit;
  final bool showPlaceholderButton;
  final VoidCallback? onEditComplete;
  final bool showRenderedPlaceholders;

  const DirectorySelector({
    Key? key,
    required this.controller,
    this.showLabel = true,
    this.showAndoirdToggle = false,
    this.allowEdit = false,
    this.showPlaceholderButton = false,
    this.onEditComplete,
    this.showRenderedPlaceholders = false,
  }) : super(key: key);

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

    Widget? buildPlaceholderButton() {
      if (!widget.showPlaceholderButton) return null;

      return PopupMenuButton<String>(
        icon: const Icon(Icons.data_object),
        tooltip: 'insertPlaceholder'.tr,
        onSelected: (String placeholder) {
          final currentText = widget.controller.text;
          final selection = widget.controller.selection;
          final cursorPosition = selection.baseOffset >= 0
              ? selection.baseOffset
              : currentText.length;

          final newText = currentText.substring(0, cursorPosition) +
              placeholder +
              currentText.substring(cursorPosition);
          widget.controller.text = newText;
          widget.controller.selection = TextSelection.fromPosition(
            TextPosition(offset: cursorPosition + placeholder.length),
          );
        },
        itemBuilder: (BuildContext context) {
          final placeholders = getPathPlaceholders();
          return placeholders.map((p) {
            return PopupMenuItem<String>(
              value: p.placeholder,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '${p.placeholder} - ${p.description}',
                    style: const TextStyle(fontWeight: FontWeight.bold),
                  ),
                  Text(
                    'example'.trParams({'value': p.example}),
                    style: TextStyle(
                      fontSize: 12,
                      color: Theme.of(context).hintColor,
                    ),
                  ),
                ],
              ),
            );
          }).toList();
        },
      );
    }

    return Row(
      children: [
        Expanded(
            child: ValueListenableBuilder<TextEditingValue>(
          valueListenable: widget.controller,
          builder: (context, value, child) {
            Widget? suffix;
            if (widget.showRenderedPlaceholders && value.text.contains('%')) {
              final renderedPath = renderPathPlaceholders(value.text);
              // Show rendered path as a chip/badge in the input field
              suffix = Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                margin: const EdgeInsets.only(right: 8),
                decoration: BoxDecoration(
                  color: Colors.blue[50],
                  borderRadius: BorderRadius.circular(4),
                  border: Border.all(color: Colors.blue[200]!),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(Icons.arrow_forward,
                        size: 14, color: Colors.blue[700]),
                    const SizedBox(width: 4),
                    Flexible(
                      child: Text(
                        renderedPath,
                        style: TextStyle(
                          color: Colors.blue[700],
                          fontSize: 12,
                          fontWeight: FontWeight.w500,
                        ),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                  ],
                ),
              );
            }

            return TextFormField(
              readOnly:
                  widget.allowEdit ? false : (Util.isWeb() ? false : true),
              controller: widget.controller,
              decoration: widget.showLabel
                  ? InputDecoration(
                      labelText: 'downloadDir'.tr,
                      suffix: suffix,
                    )
                  : InputDecoration(
                      suffix: suffix,
                    ),
              validator: (v) {
                return v!.trim().isNotEmpty ? null : 'downloadDirValid'.tr;
              },
              onEditingComplete: widget.onEditComplete,
              onTapOutside: (event) {
                // Call onEditComplete when user taps outside the field
                widget.onEditComplete?.call();
              },
            );
          },
        )),
        buildSelectWidget(),
        buildPlaceholderButton(),
      ].where((e) => e != null).map((e) => e!).toList(),
    );
  }
}
