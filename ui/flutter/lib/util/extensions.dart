import 'dart:core';
import 'package:checkable_treeview/checkable_treeview.dart';
import 'package:flutter/material.dart';
import '../api/model/resource.dart';
import '../app/views/file_icon.dart';

extension ListFileInfoExtension on List<FileInfo> {
  List<TreeNode<int>> toTreeNodes() {
    final rootNodes = <String, TreeNode<int>>{};

    for (var i = 0; i < length; i++) {
      _addFileToTree(rootNodes, this[i], i);
    }

    return rootNodes.values.toList();
  }

  void _addFileToTree(
      Map<String, TreeNode<int>> rootNodes, FileInfo file, int index) {
    final pathParts = _getPathParts(file);
    var currentNodes = rootNodes;

    final fullPath = <String>[];

    for (var i = 0; i < pathParts.length; i++) {
      final part = pathParts[i];
      fullPath.add(part);
      final isLastPart = i == pathParts.length - 1;

      if (!currentNodes.containsKey(part)) {
        currentNodes[part] = TreeNode(
          label: Text(part),
          value: isLastPart ? index : -1,
          icon: isLastPart ? fileIcon(part) : folderIcon,
          isSelected: true,
          children: [],
        );
      }

      if (!isLastPart) {
        final nodeChildren = currentNodes[part]!.children;
        currentNodes = Map.fromEntries(
          nodeChildren.map((node) => MapEntry(
                (node.label as Text).data!,
                node,
              )),
        );
      }
    }
  }

  List<String> _getPathParts(FileInfo file) {
    final fullPath =
        file.path.isEmpty ? file.name : '${file.path}/${file.name}';
    return fullPath.split('/')..removeWhere((part) => part.isEmpty);
  }
}
