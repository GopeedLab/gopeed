import 'dart:core';
import 'package:checkable_treeview/checkable_treeview.dart';
import 'package:flutter/material.dart';
import '../api/model/resource.dart';
import '../app/views/file_icon.dart';
import 'util.dart';

extension ListFileInfoExtension on List<FileInfo> {
  List<TreeNode<int>> toTreeNodes() {
    final List<TreeNode<int>> rootNodes = [];
    final Map<String, TreeNode<int>> dirNodes = {};
    var nodeIndex = 0;

    for (var i = 0; i < length; i++) {
      final file = this[i];
      final parts = file.path.split('/');
      String currentPath = '';
      TreeNode<int>? parentNode;

      // Create or get directory nodes
      for (final part in parts) {
        if (part.isEmpty) continue;

        currentPath += '/$part';
        if (!dirNodes.containsKey(currentPath)) {
          final node = TreeNode<int>(
            value: nodeIndex++,
            label: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(file.name),
                Text(Util.fmtByte(file.size)),
              ],
            ),
            icon: Icon(
              fileIcon(part, isFolder: true),
              size: 18,
            ),
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
        value: nodeIndex++,
        label: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(file.name),
            Text(Util.fmtByte(file.size)),
          ],
        ),
        icon: Icon(fileIcon(file.name, isFolder: false), size: 18),
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
