import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:path/path.dart' as path;

import '../../../util/file_icon.dart';
import '../../../util/util.dart';
import '../../modules/app/controllers/app_controller.dart';
import '../../modules/create/controllers/create_controller.dart';

class FileListView extends GetView {
  FileListView({
    Key? key,
  }) : super(key: key);
  final parentController = Get.find<CreateController>();
  final appController = Get.find<AppController>();

  List<fluent.TreeViewItem> buildTreeViewItemsRecursive(
      List fileInfos, int level, List<fluent.TreeViewItem> treeViewItems) {
    List children = fileInfos.where((e) => e['level'] == level).toList();
    for (int i = 0; i < children.length; i++) {
      Map fileInfo = children[i];
      if (fileInfo['size'] == null) {
        // folder
        treeViewItems.add(fluent.TreeViewItem(
            // expanded: false, bug on init
            leading: const Icon(Icons.folder_open),
            content: Row(mainAxisAlignment: MainAxisAlignment.end, children: [
              Expanded(
                  child: Text(
                fileInfo['name'],
                overflow: TextOverflow.ellipsis,
                // style: context.textTheme.titleSmall,
              )),
            ]),
            // onExpandToggle: (Fluent.TreeViewItem item, bool getsExpanded) {
            //TODO
            // item.expanded = getsExpanded;
            // },
            children: buildTreeViewItemsRecursive(
                fileInfos.where((e) => e['level'] > level).toList(),
                level + 1, [])));
      } else {
        // file
        if (fileInfo['selected']) {
          parentController.selectedIndexs.add(fileInfo['fileId']);
        }
        treeViewItems.add(fluent.TreeViewItem(
          value: fileInfo['fileId'],
          selected: fileInfo['selected'],
          collapsable: false,
          leading: fileInfo['name'].lastIndexOf('.') == -1
              ? const Icon(fluent.FluentIcons.document)
              : Icon(fluent.FluentIcons.allIcons[findIcon(fileInfo['name']
                  .substring(fileInfo['name'].lastIndexOf('.') + 1))]),
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
    for (var i = 0; i < parentController.files.length; i++) {
      //parentId -1 means path root
      int parentId = -1;
      List folders = path.split(parentController.files[i].path);
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
            'children': [],
          });
          if (parentId != -1) {
            fileInfos[parentId]['children'].add(idNext);
          }
          parentId = idNext;
          idNext++;
        }
      }
      //add one file, add index to parent
      fileInfos.add({
        'id': idNext,
        'type': 'file',
        'fileId': i,
        'level': folders.length,
        'name': parentController.files[i].name,
        'size': parentController.files[i].size,
        'parentId': parentId,
        //TODO add unselected logic
        'selected': true
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
              'selectFile'.tr,
              // style: TextStyle(color: themeData.hintColor),
            ),
            Expanded(
                child: Container(
                    margin: const EdgeInsets.only(top: 10),
                    decoration: BoxDecoration(
                        border: Border.all(color: Colors.grey, width: 1),
                        borderRadius: BorderRadius.circular(5)),
                    child: fluent.TreeView(
                        //changes all when folder selector is changed
                        onSelectionChanged: (selectedItems) async => {
                              parentController.selectedIndexs.value =
                                  selectedItems.map((j) => j.value).toList()
                              // for (var i = 0; i < parentController.files.length; i++)
                              //   {
                              //     selectedItems
                              //             .map((j) => j.value)
                              //             .toList()
                              //             .contains(i)
                              //         ? values[i] = true
                              //         : values[i] = false
                              //   }
                            },
                        narrowSpacing: true,
                        scrollPrimary: true,
                        selectionMode: fluent.TreeViewSelectionMode.multiple,
                        items: items))),
          ],
        ));
  }
}
