import 'package:flutter/material.dart';
import 'package:get/get.dart';
import '../util/util.dart';

import '../api/model/resource.dart';

class FileListView extends StatefulWidget {
  final List<FileInfo> files;
  final List<bool> values;

  const FileListView({
    Key? key,
    required this.files,
    required this.values,
  }) : super(key: key);

  @override
  State<FileListView> createState() => _FileListViewState();
}

class _FileListViewState extends State<FileListView> {
  List<FileInfo> get _files => widget.files;

  List<bool> get _values => widget.values;

  @override
  Widget build(BuildContext context) {
    final themeData = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Padding(padding: EdgeInsets.only(top: 10)),
        Text('create.selectDir'.tr,
            style: TextStyle(color: themeData.hintColor)),
        Expanded(
            child: Container(
                margin: const EdgeInsets.only(top: 10),
                decoration: BoxDecoration(
                    border: Border.all(color: Colors.grey, width: 1),
                    borderRadius: BorderRadius.circular(5)),
                child: ListView.builder(
                    itemCount: _files.length,
                    itemBuilder: (context, index) {
                      var file = _files[index];
                      return CheckboxListTile(
                        value: _values[index],
                        onChanged: (value) {
                          setState(() {
                            _values[index] = value!;
                          });
                        },
                        title: Text(Util.buildPath(file.path, file.name)),
                        subtitle: Text(Util.fmtByte(file.size)),
                        secondary: const Icon(Icons.description),
                      );
                    }))),
      ],
    );
  }
}
