import 'package:checkable_treeview/checkable_treeview.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:toggle_switch/toggle_switch.dart';

import '../../api/model/resource.dart';
import '../../icon/gopeed_icons.dart';
import '../../util/util.dart';
import 'file_icon.dart';
import 'responsive_builder.dart';
import 'sort_icon_button.dart';

const _toggleSwitchIcons = [
  Gopeed.file_video,
  Gopeed.file_audio,
  Gopeed.file_image,
];
const _sizeGapWidth = 72.0;

class FileTreeView extends StatefulWidget {
  final List<FileInfo> files;
  final List<int> initialValues;
  final Function(List<int>) onSelectionChanged;

  const FileTreeView(
      {Key? key,
      required this.files,
      required this.initialValues,
      required this.onSelectionChanged})
      : super(key: key);

  @override
  State<FileTreeView> createState() => _FileTreeViewState();
}

class _FileTreeViewState extends State<FileTreeView> {
  late GlobalKey<TreeViewState<int>> key;
  late int totalSize;
  int? toggleSwitchIndex;

  @override
  void initState() {
    super.initState();
    key = GlobalKey<TreeViewState<int>>();
    totalSize = widget.files
        .fold(0, (previousValue, element) => previousValue + element.size);
    widget.onSelectionChanged(widget.initialValues);
  }

  @override
  Widget build(BuildContext context) {
    final selectedFileCount =
        key.currentState?.getSelectedValues().where((e) => e != null).length ??
            widget.files.length;
    final selectedFileSize = calcSelectedSize(null);

    final filterRow = InkWell(
      onTap: () {},
      child: ToggleSwitch(
        minHeight: 32,
        cornerRadius: 8,
        doubleTapDisable: true,
        inactiveBgColor: Theme.of(context).dividerColor,
        activeBgColor: [Theme.of(context).colorScheme.primary],
        initialLabelIndex: toggleSwitchIndex,
        icons: _toggleSwitchIcons,
        onToggle: (index) {
          toggleSwitchIndex = index;
          if (index == null) {
            key.currentState?.setSelectedValues(List.empty());
            return;
          }

          final iconFileExtArr = iconConfigMap[_toggleSwitchIcons[index]] ?? [];
          final selectedFileIndexes = widget.files
              .asMap()
              .entries
              .where((e) => iconFileExtArr.contains(fileExt(e.value.name)))
              .map((e) => e.key)
              .toList();
          key.currentState?.setSelectedValues(selectedFileIndexes);
        },
      ),
    );
    final countRow = Row(
      children: [
        Text('fileSelectedCount'.tr),
        Text(
          selectedFileCount.toString(),
          style: Theme.of(context).textTheme.bodySmall,
        ),
        const SizedBox(width: 12),
        Text('fileSelectedSize'.tr),
        Text(
          selectedFileCount > 0 && selectedFileSize == 0
              ? 'unknown'.tr
              : Util.fmtByte(selectedFileSize),
          style: Theme.of(context).textTheme.bodySmall,
        ),
      ],
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          child: Container(
            decoration: BoxDecoration(
              border: Border.all(color: Theme.of(context).dividerColor),
              borderRadius: BorderRadius.circular(4),
            ),
            child: TreeView(
              key: key,
              nodes: buildTreeNodes(),
              showExpandCollapseButton: true,
              showSelectAll: true,
              onSelectionChanged: (selectedValues) {
                setState(() {});
                widget.onSelectionChanged(selectedValues
                    .where((e) => e != null)
                    .map((e) => e!)
                    .toList());
              },
              selectAllTrailing: (context) {
                return Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.end,
                      children: [
                        Text('name'.tr),
                        SortIconButton(
                          onStateChanged: (state) {
                            switch (state) {
                              case SortState.asc:
                                key.currentState?.sort((p0, p1) {
                                  return (p0.label as Text)
                                      .data!
                                      .compareTo((p1.label as Text).data!);
                                });
                                break;
                              case SortState.desc:
                                key.currentState?.sort((p0, p1) {
                                  return (p1.label as Text)
                                      .data!
                                      .compareTo((p0.label as Text).data!);
                                });
                                break;
                              default:
                                key.currentState?.sort(null);
                                break;
                            }
                          },
                        ),
                      ],
                    ),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.end,
                      children: [
                        Text('size'.tr),
                        SortIconButton(
                          onStateChanged: (state) {
                            switch (state) {
                              case SortState.asc:
                                key.currentState?.sort((p0, p1) {
                                  return calcSelectedSize(p0)
                                      .compareTo(calcSelectedSize(p1));
                                });
                                break;
                              case SortState.desc:
                                key.currentState?.sort((p0, p1) {
                                  return calcSelectedSize(p1)
                                      .compareTo(calcSelectedSize(p0));
                                });
                                break;
                              default:
                                key.currentState?.sort(null);
                                break;
                            }
                          },
                        ),
                      ],
                    )
                  ],
                );
              },
            ),
          ),
        ),
        const SizedBox(height: 12),
        !ResponsiveBuilder.isNarrow(context)
            ? Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  filterRow,
                  countRow,
                ],
              )
            : Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  filterRow,
                  const SizedBox(height: 8),
                  countRow,
                ],
              ),
      ],
    );
  }

  int calcSelectedSize(TreeNode<int>? node) {
    if (key.currentState == null) {
      return widget.files
          .fold(0, (previousValue, element) => previousValue + element.size);
    }

    final selectedFileIndexes = node == null
        ? key.currentState?.getSelectedValues()
        : (node.value != null
            ? [node.value]
            : key.currentState?.getChildSelectedValues(node));

    if (selectedFileIndexes == null) return 0;
    return selectedFileIndexes.where((e) => e != null).map((e) => e!).fold(0,
        (previousValue, element) => previousValue + widget.files[element].size);
  }

  List<TreeNode<int>> buildTreeNodes() {
    final List<TreeNode<int>> rootNodes = [];
    final Map<String, TreeNode<int>> dirNodes = {};

    for (var i = 0; i < widget.files.length; i++) {
      final file = widget.files[i];
      final parts = file.path.split('/');
      String currentPath = '';
      TreeNode<int>? parentNode;

      // Create or get directory nodes
      for (final part in parts) {
        if (part.isEmpty) continue;

        currentPath += '/$part';
        if (!dirNodes.containsKey(currentPath)) {
          final node = TreeNode<int>(
            label: Text(part),
            icon: Icon(
              fileIcon(part, isFolder: true),
              size: 18,
            ),
            trailing: (context, node) {
              final size = calcSelectedSize(node);
              return size > 0
                  ? Text(Util.fmtByte(calcSelectedSize(node)),
                      style: Theme.of(context).textTheme.bodySmall)
                  : const SizedBox(width: _sizeGapWidth);
            },
            children: [],
          );
          dirNodes[currentPath] = node;

          if (parentNode == null) {
            rootNodes.add(node);
          } else {
            parentNode.children.add(node);
          }
        }
        parentNode = dirNodes[currentPath];
      }

      // Create file node using file.name
      final fileNode = TreeNode<int>(
        label: Text(file.name),
        value: i,
        icon: Icon(fileIcon(file.name, isFolder: false), size: 18),
        trailing: (context, node) {
          return file.size > 0
              ? Text(Util.fmtByte(file.size),
                  style: Theme.of(context).textTheme.bodySmall)
              : const SizedBox(width: _sizeGapWidth);
        },
        isSelected: widget.initialValues.contains(i),
        children: [],
      );

      // Add file node to parent or root
      if (parentNode != null) {
        parentNode.children.add(fileNode);
      } else {
        rootNodes.add(fileNode);
      }
    }

    return rootNodes;
  }
}
