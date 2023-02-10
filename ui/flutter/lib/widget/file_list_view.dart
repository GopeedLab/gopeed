import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:path/path.dart' as path;

import '../api/model/resource.dart';
import '../pages/app/app_controller.dart';
import '../util/util.dart';

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
  // List<FileInfo> get _files => widget.files;
  // List<int> get _values => widget.values;

  List<fluent.TreeViewItem> buildTreeViewItemsRecursive(
      List fileInfos, int level, List<fluent.TreeViewItem> treeViewItems) {
    List children = fileInfos.where((e) => e['level'] == level).toList();
    for (int i = 0; i < children.length; i++) {
      Map fileInfo = children[i];
      if (fileInfo['size'] == null) {
        // folder
        treeViewItems.add(fluent.TreeViewItem(
            // expanded: false, bug on init
            leading: const Icon(fluent.FluentIcons.open_folder_horizontal),
            content: Row(mainAxisAlignment: MainAxisAlignment.end, children: [
              Expanded(
                  child: Text(
                fileInfo['name'],
                overflow: TextOverflow.ellipsis,
                // style: context.textTheme.titleSmall,
              )),
            ]),
            children: buildTreeViewItemsRecursive(
                fileInfos.where((e) => e['level'] > level).toList(),
                level + 1, [])));
      } else {
        // file
        treeViewItems.add(fluent.TreeViewItem(
          value: fileInfo['fileId'],
          selected: widget.values[fileInfo['fileId']],
          collapsable: false,
          leading: const Icon(Icons.description_outlined),
          content: Row(mainAxisAlignment: MainAxisAlignment.end, children: [
            Expanded(
                child: Text(
              fileInfo['name'],
              overflow: TextOverflow.ellipsis,
              // style: context.textTheme.titleSmall,
            )),
            Text(
              Util.fmtByte(
                fileInfo['size'],
              ),
              // style: context.textTheme.labelMedium,
              overflow: TextOverflow.ellipsis,
            ),
          ]),
        ));
      }
    }
    return treeViewItems;
  }

  List<fluent.TreeViewItem> get items {
    List fileInfos = [];
    int idNext = 0;
    //make fileInfos list
    for (var i = 0; i < widget.files.length; i++) {
      //parentId -1 means path root
      int parentId = -1;
      List folders = path.split(widget.files[i].path);
      for (var folder in folders) {
        int indexInfileInfos = fileInfos.lastIndexWhere((fileInfo) =>
            fileInfo['name'] == folder && fileInfo['parentId'] == parentId);
        //TODO calculate folder size
        if (indexInfileInfos != -1) {
          //  folder exists
          parentId = fileInfos[indexInfileInfos]['id'];
        } else {
          // parent not exist
          // create one and add index to parent's children
          fileInfos.add({
            'id': idNext,
            'type': 'folder',
            'name': folder,
            'parentId': parentId,
            'level': folders.indexOf(folder),
            // 'content': Row(children: [
            //   const Icon(Icons.folder),
            //   Text(folder),
            // ]),
            'children': [],
          });
          if (parentId != -1) {
            fileInfos[parentId]['children'].add(idNext);
          }
          parentId = idNext;
          idNext++;
        }
      }
      //add one file add index to parent
      fileInfos.add({
        'id': idNext,
        'type': 'file',
        'fileId': i,
        'level': folders.length,
        'name': widget.files[i].name,
        'size': widget.files[i].size,
        'parentId': parentId,
        // 'content': Row(children: [
        //   const Icon(Icons.description),
        //   Text(widget.files[i].name),
        // ])
      });
      if (parentId != -1) {
        fileInfos[parentId]['children'].add(idNext);
      }
      idNext++;
    }
    List<fluent.TreeViewItem> treeItems =
        buildTreeViewItemsRecursive(fileInfos, 0, []);
    return treeItems;
  }

  @override
  Widget build(BuildContext context) {
    final appController = Get.find<AppController>();
    return fluent.FluentTheme(
        data: appController.downloaderConfig.value.extra.themeMode == 'dark'
            ? fluent.ThemeData(brightness: Brightness.dark)
            : fluent.ThemeData(brightness: Brightness.light),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Padding(padding: EdgeInsets.only(top: 10)),
            Text(
              'create.selectFile'.tr,
              // style: TextStyle(color: themeData.hintColor),
            ),
            Expanded(
                child: Container(
                    margin: const EdgeInsets.only(top: 10),
                    decoration: BoxDecoration(
                        border: Border.all(color: fluent.Colors.grey, width: 1),
                        borderRadius: BorderRadius.circular(5)),
                    child: fluent.TreeView(
                        onSelectionChanged: (selectedItems) async =>
                            setState(() async {
                              List newValues =
                                  selectedItems.map((j) => j.value).toList();
                              for (var i = 0; i < widget.values.length; i++) {
                                newValues.contains(i)
                                    ? widget.values[i] = true
                                    : widget.values[i] = false;
                              }
                            }),
                        narrowSpacing: true,
                        scrollPrimary: true,
                        selectionMode: fluent.TreeViewSelectionMode.multiple,
                        items: items))),
          ],
        ));
  }
}
