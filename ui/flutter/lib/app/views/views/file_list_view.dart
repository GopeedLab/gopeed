import 'dart:ui' as ui;

import 'package:fluent_ui/fluent_ui.dart' as fluent;
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:path/path.dart' as path;

import '../../../util/file_icon.dart';
import '../../../util/icons.dart';
import '../../../util/util.dart';
import '../../modules/app/controllers/app_controller.dart';
import '../../modules/create/controllers/create_controller.dart';

class FileListView extends GetView {
  final List<FileInfo> files;

  FileListView({
    Key? key,
    required this.files,
  }) : super(key: key);
  final parentController = Get.find<CreateController>();
  final appController = Get.find<AppController>();

  List findChildFileIdsR(List children) {
    List res = [];
    for (var k in children) {
      if (parentController.fileInfos[k]['type'] == 'folder') {
        res.addAll(
            findChildFileIdsR(parentController.fileInfos[k]['children']));
      } else {
        res.add(parentController.fileInfos[k]['fileId']);
      }
    }
    return res;
  }

  void clearDescendantFolderSizeR(int id) {
    if (parentController.fileInfos[id]['type'] == 'folder') {
      parentController.fileInfos[id]['size'] = 0;
      for (var childId in parentController.fileInfos[id]['children']) {
        clearDescendantFolderSizeR(childId);
      }
    }
  }

  void reduceAncestorFolderSizeR(int id, int size) {
    int parentId = parentController.fileInfos[id]['parentId'];

    if (parentId != -1) {
      parentController.fileInfos[parentId]['size'] -= size;
      reduceAncestorFolderSizeR(parentId, size);
    }
  }

  int addDescendantFolderSizeR(int id) {
    if (parentController.fileInfos[id]['type'] == 'folder') {
      for (var childId in parentController.fileInfos[id]['children']) {
        parentController.fileInfos[id]['size'] +=
            addDescendantFolderSizeR(childId);
      }
    }
    return parentController.fileInfos[id]['size'];
  }

  void addAncestorFolderSizeR(int id, int size) {
    int parentId = parentController.fileInfos[id]['parentId'];
    if (parentId != -1) {
      parentController.fileInfos[parentId]['size'] += size;
      addAncestorFolderSizeR(parentId, size);
    }
  }

  List<fluent.TreeViewItem> buildTreeViewItemsR(int level, int parentId) {
    List<fluent.TreeViewItem> res = [];
    List children = parentController.fileInfos
        .where((e) => e['level'] == level && e['parentId'] == parentId)
        .toList();
    for (int j = 0; j < children.length; j++) {
      Map fileInfo = children[j];

      if (fileInfo['type'] == 'folder') {
        // folder
        res.add(fluent.TreeViewItem(
            // expanded: false, bug on init
            value: fileInfo['id'],
            onInvoked: (item, reason) async {
              if (reason == fluent.TreeViewItemInvokeReason.selectionToggle) {
                List childrenIds = findChildFileIdsR(fileInfo['children']);
                if (item.selected == true) {
                  parentController.selectedIndexes.addAll(childrenIds);
                  parentController.fileInfos[fileInfo['id']]['size'] = 0;
                  addDescendantFolderSizeR(item.value);
                  addAncestorFolderSizeR(item.value, fileInfo['size']);
                  parentController.fileInfos.refresh();
                } else {
                  parentController.selectedIndexes
                      .removeWhere((index) => childrenIds.contains(index));
                  reduceAncestorFolderSizeR(item.value, fileInfo['size']);
                  clearDescendantFolderSizeR(item.value);

                  parentController.fileInfos.refresh();
                }
              }
            },
            leading: Obx(() {
              return parentController.openedFolders.contains(fileInfo['id'])
                  ? const Icon(FaIcons.folder_open,
                      color: Colors.lightBlueAccent)
                  : const Icon(FaIcons.folder, color: Colors.lightBlueAccent);
            }),
            onExpandToggle: (fluent.TreeViewItem item, bool getExpanded) async {
              getExpanded
                  ? parentController.openedFolders.add(item.value)
                  : parentController.openedFolders.remove(item.value);
            },
            content: Row(mainAxisAlignment: MainAxisAlignment.end, children: [
              Expanded(
                child: Text(
                  fileInfo['name'],
                  overflow: TextOverflow.ellipsis,
                  // style: context.textTheme.titleSmall,
                ),
              ),
              Obx(() {
                return parentController.fileInfos[fileInfo['id']]['size'] != 0
                    ? Text(
                        Util.fmtByte(
                            parentController.fileInfos[fileInfo['id']]['size']),
                        // style: context.textTheme.labelMedium,
                        overflow: TextOverflow.ellipsis,
                      )
                    : const SizedBox.shrink();
              })
            ]),
            children: buildTreeViewItemsR(
                // parentController.fileInfos.where((e) => e['level'] > level).toList(),
                level + 1,
                fileInfo['id'])));
      } else {
        // file
        res.add(fluent.TreeViewItem(
          value: {'id': fileInfo['id'], 'fileId': fileInfo['fileId']},
          selected: fileInfo['selected'],
          onInvoked: (item, reason) async {
            if (reason == fluent.TreeViewItemInvokeReason.selectionToggle) {
              if (item.selected == true) {
                parentController.selectedIndexes.add(item.value['fileId']);
                addAncestorFolderSizeR(item.value['id'], fileInfo['size']);
                parentController.fileInfos.refresh();
              } else {
                parentController.selectedIndexes.remove(item.value['fileId']);
                reduceAncestorFolderSizeR(item.value['id'], fileInfo['size']);
                parentController.fileInfos.refresh();
              }
            }
          },
          collapsable: false,
          leading: fileInfo['name'].lastIndexOf('.') == -1
              ? Icon(
                  FaIcons.doc,
                  color: appController.downloaderConfig.value.extra.themeMode ==
                          'dark'
                      ? Colors.white
                      : Colors.black45,
                )
              : Icon(
                  FaIcons.allIcons[findIcon(fileInfo['name'])],
                  color: appController.downloaderConfig.value.extra.themeMode ==
                          'dark'
                      ? Colors.white
                      : Colors.black45,
                ),
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

    return res;
  }

  List<fluent.TreeViewItem> get items {
    List infos = [];
    int idNext = 0;
    List selectedFileIds = [];
    List openedFolders = [];
    //make fileInfos list
    for (var i = 0; i < files.length; i++) {
      //parentId -1 means path root
      int parentId = -1;
      List folders = path.split(files[i].path);
      for (var folder in folders) {
        int indexInInfos = infos.lastIndexWhere((fileInfo) =>
            fileInfo['name'] == folder && fileInfo['parentId'] == parentId);

        if (indexInInfos != -1) {
          //  folder exists
          parentId = infos[indexInInfos]['id'];
          // add size
          infos[indexInInfos]['size'] += files[i].size;
        } else {
          // parent not exist
          // create one and add index to parent's children
          infos.add({
            'level': folders.indexOf(folder),
            'id': idNext,
            'type': 'folder',
            'name': folder,
            'size': files[i].size,
            'parentId': parentId,
            'children': [],
          });
          openedFolders.add(idNext);
          if (parentId != -1) {
            infos[parentId]['children'].add(idNext);
          }
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
      selectedFileIds.add(i);
      if (parentId != -1) {
        infos[parentId]['children'].add(idNext);
      }
      idNext++;
    }
    parentController.fileInfos.value = infos;
    parentController.selectedIndexes.value = selectedFileIds;
    parentController.openedFolders.value = openedFolders;
    List<fluent.TreeViewItem> treeItems = buildTreeViewItemsR(0, -1);
    return treeItems;
  }

  @override
  Widget build(BuildContext context) {
    return fluent.FluentTheme(
        data: appController.downloaderConfig.value.extra.themeMode == 'system'
            ? fluent.FluentThemeData(brightness: ui.window.platformBrightness)
            : appController.downloaderConfig.value.extra.themeMode == 'light'
                ? fluent.FluentThemeData(brightness: Brightness.light)
                : fluent.FluentThemeData(brightness: Brightness.dark),
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
                      selectionMode: fluent.TreeViewSelectionMode.multiple,
                      shrinkWrap: false,
                      // addRepaintBoundaries: false,
                      usePrototypeItem: true,
                      // cacheExtent: 20,
                      // onItemInvoked: (item, reason) async => {},
                      onSecondaryTap: (item, details) async {
                        debugPrint(
                            'onSecondaryTap $item at ${details.globalPosition}');
                      },
                      // onSelectionChanged: (selectedItems) async => {},
                      narrowSpacing: true,
                      items: items,
                      // scrollPrimary: true,
                    ))),
          ],
        ));
  }
}
