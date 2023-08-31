import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../util/mac_secure_util.dart';
import '../../util/util.dart';

class DirectorySelector extends StatefulWidget {
  final TextEditingController controller;
  final bool showLabel;

  const DirectorySelector(
      {Key? key, required this.controller, this.showLabel = true})
      : super(key: key);

  @override
  State<DirectorySelector> createState() => _DirectorySelectorState();
}

class _DirectorySelectorState extends State<DirectorySelector> {
  @override
  Widget build(BuildContext context) {
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
        Util.isDesktop() || Util.isAndroid()
            ? IconButton(
                icon: const Icon(Icons.folder_open),
                onPressed: () async {
                  var dir = await FilePicker.platform.getDirectoryPath();
                  if (dir != null) {
                    widget.controller.text = dir;
                    MacSecureUtil.saveBookmark(dir);
                  }
                })
            : null
      ].where((e) => e != null).map((e) => e!).toList(),
    );
  }
}
