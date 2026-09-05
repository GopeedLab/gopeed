import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' hide Column, Expanded, Row;

typedef VirtualTreeItemBuilder<T> = Widget Function(BuildContext context, VirtualTreeEntry<T> entry);
typedef VirtualTreeRowTransitionBuilder<T> =
    Widget Function(BuildContext context, VirtualTreeEntry<T> entry, Widget row);

class VirtualTreeEntry<T> {
  const VirtualTreeEntry({required this.node, required this.depth});

  final TreeItemNode<T> node;
  final List<TreeNodeDepth> depth;
}

class VirtualTreeView<T> extends StatelessWidget {
  const VirtualTreeView({
    super.key,
    required this.nodes,
    required this.itemBuilder,
    this.branchLine = BranchLine.path,
    this.padding = const EdgeInsets.all(8),
    this.rowHeight = 28,
    this.cacheExtent = 280,
    this.controller,
    this.itemPadding,
    this.transitionBuilder,
  });

  final List<TreeNode<T>> nodes;
  final VirtualTreeItemBuilder<T> itemBuilder;
  final BranchLine branchLine;
  final EdgeInsetsGeometry padding;
  final double rowHeight;
  final double cacheExtent;
  final ScrollController? controller;
  final EdgeInsetsGeometry? itemPadding;
  final VirtualTreeRowTransitionBuilder<T>? transitionBuilder;

  @override
  Widget build(BuildContext context) {
    final entries = _flattenVisibleNodes(nodes);
    final theme = Theme.of(context);
    final densityGap = theme.density.baseGap * theme.scaling;

    return ListView.builder(
      controller: controller,
      padding: padding,
      cacheExtent: cacheExtent,
      itemCount: entries.length,
      itemBuilder: (context, index) {
        final entry = entries[index];
        Widget row = SizedBox(
          height: rowHeight,
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              for (var depthIndex = 1; depthIndex < entry.depth.length; depthIndex++) ...[
                SizedBox(width: densityGap),
                SizedBox(width: densityGap * 2, child: branchLine.build(context, entry.depth, depthIndex)),
              ],
              Expanded(
                child: Padding(
                  padding: itemPadding ?? EdgeInsets.symmetric(horizontal: densityGap, vertical: densityGap * 0.5),
                  child: itemBuilder(context, entry),
                ),
              ),
            ],
          ),
        );
        final transition = transitionBuilder;
        if (transition != null) {
          row = transition(context, entry, row);
        }
        return row;
      },
    );
  }
}

List<VirtualTreeEntry<T>> _flattenVisibleNodes<T>(List<TreeNode<T>> nodes) {
  final entries = <VirtualTreeEntry<T>>[];

  void walk(List<TreeNode<T>> currentNodes, List<TreeNodeDepth> depth) {
    for (var index = 0; index < currentNodes.length; index++) {
      final node = currentNodes[index];
      if (node is TreeRootNode<T>) {
        walk(node.children, depth);
        continue;
      }
      if (node is! TreeItemNode<T>) continue;
      final currentDepth = [...depth, TreeNodeDepth(index, currentNodes.length)];
      entries.add(VirtualTreeEntry(node: node, depth: currentDepth));
      if (node.expanded) {
        walk(node.children, currentDepth);
      }
    }
  }

  walk(nodes, const []);
  return entries;
}
