import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:path/path.dart' as path;

import '../../../util/file_icon.dart';
import '../../../util/icons.dart';
import '../../../util/util.dart';
import '../../modules/app/controllers/app_controller.dart';
import '../../modules/create/controllers/create_controller.dart';

class FileListView extends GetView {
  FileListView({
    Key? key,
  }) : super(key: key);
  final parentController = Get.find<CreateController>();
  final appController = Get.find<AppController>();

  late final List fileInfos;

  List findChildFileIdsRecursive(children) {
    List res = [];
    for (var i in children) {
      if (fileInfos[i]['type'] == 'folder') {
        findChildFileIdsRecursive(fileInfos[i]['children']);
      } else {
        res.add(i);
      }
    }
    return res;
  }

  List<fluent.TreeViewItem> buildTreeViewItemsRecursive(
      int level, List<fluent.TreeViewItem> treeViewItems) {
    List children = fileInfos.where((e) => e['level'] == level).toList();
    for (int i = 0; i < children.length; i++) {
      Map fileInfo = children[i];
      if (fileInfo['type'] == 'folder') {
        // folder
        treeViewItems.add(fluent.TreeViewItem(
            // expanded: false, bug on init
            // value: fileInfo['id'],
            onInvoked: (item, reason) async {
              debugPrint('onItemInvoked(reason=$reason): $item');
              if (reason.toString() ==
                  'TreeViewItemInvokeReason.selectionToggle') {
                List childrenIds =
                    findChildFileIdsRecursive(fileInfo['children']);
                if (item.selected == true) {
                  parentController.selectedIndexes
                      .addAll(findChildFileIdsRecursive(childrenIds));
                } else {
                  parentController.selectedIndexes
                      .removeWhere((index) => childrenIds.contains(index));
                }
              }
            },
            leading:
                // parentController.openedFolders.contains(fileInfo['id'])
                //     ? const Icon(FaIcons.folder)
                //     : const Icon(FaIcons.folder_open),
                const Icon(FaIcons.folder),
            content: Row(mainAxisAlignment: MainAxisAlignment.end, children: [
              Expanded(
                  child: Text(
                fileInfo['name'],
                overflow: TextOverflow.ellipsis,
                // style: context.textTheme.titleSmall,
              )),
            ]),
            // onExpandToggle: (fluent.TreeViewItem item, bool getExpanded) async {
            //   getExpanded
            //       ? parentController.openedFolders.add(item.value)
            //       : parentController.openedFolders.remove(item.value);
            // },
            children: buildTreeViewItemsRecursive(
                // fileInfos.where((e) => e['level'] > level).toList(),
                level + 1,
                [])));
      } else {
        // file
        // if (fileInfo['selected']) {
        //   parentController.selectedIndexs.add(fileInfo['fileId']);
        // }
        treeViewItems.add(fluent.TreeViewItem(
          value: fileInfo['fileId'],
          selected: fileInfo['selected'],
          onInvoked: (item, reason) async => {
            debugPrint('onItemInvoked(reason=$reason): $item'),
            reason.toString() == 'TreeViewItemInvokeReason.selectionToggle'
                ? item.selected == false
                    ? parentController.selectedIndexes.remove(item.value)
                    : parentController.selectedIndexes.add(item.value)
                : null
          },
          collapsable: false,
          leading: fileInfo['name'].lastIndexOf('.') == -1
              ? const Icon(FaIcons.file)
              : Icon(FaIcons.allIcons[findIcon(fileInfo['name'])]),
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

  List<fluent.TreeViewItem> items(files) {
    List infos = [];
    int idNext = 0;
    List selectedIndexs = [];
    // List openedFolders = [];
    //make fileInfos list
    for (var i = 0; i < files.length; i++) {
      //parentId -1 means path root
      int parentId = -1;
      List folders = path.split(files[i].path);
      for (var folder in folders) {
        int indexInfileInfos = infos.lastIndexWhere((fileInfo) =>
            fileInfo['name'] == folder && fileInfo['parentId'] == parentId);
        //TODO calculate folder size
        if (indexInfileInfos != -1) {
          //  folder exists
          parentId = infos[indexInfileInfos]['id'];
        } else {
          // parent not exist
          // create one and add index to parent's children
          infos.add({
            'id': idNext,
            'type': 'folder',
            'name': folder,
            'parentId': parentId,
            'level': folders.indexOf(folder),
            'children': [],
          });
          if (parentId != -1) {
            infos[parentId]['children'].add(idNext);
          }
          // openedFolders.add(idNext);
          parentId = idNext;
          idNext++;
        }
      }
      //add one file, add index to parent
      infos.add({
        'id': idNext,
        'type': 'file',
        'fileId': i,
        'level': folders.length,
        'name': files[i].name,
        'size': files[i].size,
        'parentId': parentId,
        //TODO add unselected logic
        'selected': true
      });
      selectedIndexs.add(idNext);
      if (parentId != -1) {
        infos[parentId]['children'].add(idNext);
      }
      idNext++;
    }
    fileInfos = infos;
    parentController.selectedIndexes.value = selectedIndexs;
    // parentController.openedFolders.value = openedFolders;
    List<fluent.TreeViewItem> treeItems = buildTreeViewItemsRecursive(0, []);
    return treeItems;
  }

  @override
  Widget build(BuildContext context) {
    final files = parentController.files;
    return fluent.FluentTheme(
        data:
            Get.find<AppController>().downloaderConfig.value.extra.themeMode ==
                    'dark'
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
                        // shrinkWrap: false,
                        // addRepaintBoundaries: false,
                        // usePrototypeItem: true,
                        // onItemInvoked: (item, reason) async => {},
                        onSecondaryTap: (item, details) async {
                          debugPrint(
                              'onSecondaryTap $item at ${details.globalPosition}');
                        },
                        // onSelectionChanged: (selectedItems) async => {},
                        narrowSpacing: true,
                        // scrollPrimary: true,
                        selectionMode: fluent.TreeViewSelectionMode.multiple,
                        items: items(files)))),
          ],
        ));
  }
}
