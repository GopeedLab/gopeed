import 'package:file_selector/file_selector.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../util/mac_secure_util.dart';

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
          readOnly: true,
          controller: widget.controller,
          decoration: widget.showLabel
              ? const InputDecoration(
                  labelText: "下载目录",
                )
              : null,
          validator: (v) {
            return v!.trim().isNotEmpty ? null : "请选择下载目录";
          },
        )),
        IconButton(
            icon: const Icon(Icons.folder_open),
            onPressed: () async {
              if (GetPlatform.isDesktop) {
                var dir = await getDirectoryPath();
                if (dir != null) {
                  widget.controller.text = dir;
                  MacSecureUtil.saveBookmark(dir);
                }
              }
            })
      ],
    );
  }
}
