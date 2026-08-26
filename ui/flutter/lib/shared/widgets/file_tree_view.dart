import 'dart:math' as math;

import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' hide Column, Expanded, Row;

import '../theme/app_palette.dart';
import '../../l10n/l10n.dart';
import '../../util/util.dart';
import 'virtual_tree_view.dart';

class FileTreeItem<T> {
  const FileTreeItem({
    required this.key,
    required this.path,
    required this.name,
    required this.size,
    required this.data,
  });

  final String key;
  final String path;
  final String name;
  final int size;
  final T data;
}

class FileTreeNode<T> {
  const FileTreeNode({
    required this.key,
    required this.label,
    required this.leafItems,
    required this.size,
    required this.originalIndex,
    this.item,
  });

  final String key;
  final String label;
  final List<FileTreeItem<T>> leafItems;
  final int size;
  final int originalIndex;
  final FileTreeItem<T>? item;

  bool get isFolder => item == null;

  @override
  bool operator ==(Object other) => other is FileTreeNode<T> && other.key == key;

  @override
  int get hashCode => key.hashCode;
}

typedef FileTreeNodeWidgetBuilder<T> = Widget Function(BuildContext context, FileTreeNode<T> node);

class FileTreeView<T> extends StatefulWidget {
  const FileTreeView({
    super.key,
    required this.items,
    this.headerLeading,
    this.leadingBuilder,
    this.trailingBuilder,
    this.onNodePressed,
    this.keyPrefix = 'file-tree',
    this.emptyLabel,
    this.showSizeColumn = true,
    this.trailingHeader,
    this.rowHeight = _treeRowHeight,
    this.contentTextStyle,
    this.iconSize = 18,
  });

  final List<FileTreeItem<T>> items;
  final Widget? headerLeading;
  final FileTreeNodeWidgetBuilder<T>? leadingBuilder;
  final FileTreeNodeWidgetBuilder<T>? trailingBuilder;
  final ValueChanged<FileTreeNode<T>>? onNodePressed;
  final String keyPrefix;
  final String? emptyLabel;
  final bool showSizeColumn;
  final Widget? trailingHeader;
  final double rowHeight;
  final TextStyle? contentTextStyle;
  final double iconSize;

  @override
  State<FileTreeView<T>> createState() => _FileTreeViewState<T>();
}

class _FileTreeViewState<T> extends State<FileTreeView<T>> with SingleTickerProviderStateMixin {
  late List<TreeNode<FileTreeNode<T>>> _nodes;
  late final AnimationController _transitionController;
  late final Animation<double> _reveal;
  final _scrollController = ScrollController();
  final _horizontalScrollController = ScrollController();
  final Map<String, double> _labelWidthCache = {};
  TextStyle? _labelMeasurementStyle;
  TextDirection? _labelMeasurementDirection;
  TextScaler? _labelMeasurementScaler;
  List<TreeNode<FileTreeNode<T>>>? _pendingCollapsedNodes;
  Set<String> _transitioningNodeKeys = const {};
  var _transitioning = false;
  var _allFoldersExpanded = false;
  _FileTreeSortColumn? _sortColumn;
  _FileTreeSortDirection _sortDirection = _FileTreeSortDirection.none;

  @override
  void initState() {
    super.initState();
    _nodes = _buildTree(widget.items);
    _allFoldersExpanded = _foldersAreExpanded(_nodes);
    _transitionController = AnimationController(vsync: this, duration: _treeTransitionDuration, value: 1)
      ..addStatusListener(_onTransitionStatus);
    _reveal = CurvedAnimation(parent: _transitionController, curve: Curves.easeOutCubic);
  }

  @override
  void didUpdateWidget(covariant FileTreeView<T> oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (identical(widget.items, oldWidget.items)) return;

    final expandedKeys = _expandedFolderKeys(_nodes);
    var nodes = _restoreExpandedFolders(_buildTree(widget.items), expandedKeys);
    nodes = _sortNodes(nodes, _sortColumn, _sortDirection);
    _nodes = nodes;
    _allFoldersExpanded = _foldersAreExpanded(nodes);
    _pendingCollapsedNodes = null;
    _transitioningNodeKeys = const {};
    _transitioning = false;
    _transitionController.value = 1;
  }

  @override
  void dispose() {
    _transitionController.removeStatusListener(_onTransitionStatus);
    _transitionController.dispose();
    _scrollController.dispose();
    _horizontalScrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final treeItemInset = theme.density.baseGap * theme.scaling;
    final hasTrailingColumn = widget.trailingBuilder != null || widget.trailingHeader != null || widget.showSizeColumn;
    final trailingColumnWidth = hasTrailingColumn
        ? (widget.trailingBuilder == null ? _treeSizeColumnWidth : _treeCustomTrailingColumnWidth)
        : 0.0;
    final labelStyle = DefaultTextStyle.of(context).style.merge(widget.contentTextStyle);
    final textDirection = Directionality.of(context);
    final textScaler = MediaQuery.textScalerOf(context);
    if (_labelMeasurementStyle != labelStyle ||
        _labelMeasurementDirection != textDirection ||
        _labelMeasurementScaler != textScaler) {
      _labelWidthCache.clear();
      _labelMeasurementStyle = labelStyle;
      _labelMeasurementDirection = textDirection;
      _labelMeasurementScaler = textScaler;
    }
    double measureLabel(String label) => _labelWidthCache.putIfAbsent(
      label,
      () => _measureLabelWidth(label, style: labelStyle, textDirection: textDirection, textScaler: textScaler),
    );
    final nameContentWidth = _widestLabelWidth(widget.items, measureLabel);
    final fixedNameChrome =
        _treeNodeExpandChrome + widget.iconSize + 4 + (widget.leadingBuilder == null ? 0 : _treeLeadingChrome);

    return LayoutBuilder(
      builder: (context, constraints) {
        final treeWidth = constraints.hasBoundedWidth ? constraints.maxWidth : _treeFallbackWidth;
        final horizontalOverflow = _horizontalOverflow(
          widget.items,
          baseNameViewport:
              treeWidth - _treeHorizontalPadding - trailingColumnWidth - _treeTrailingGap - fixedNameChrome,
          depthIndent: treeItemInset * 3,
          measureLabel: measureLabel,
        );

        return OutlinedContainer(
          child: Column(
            children: [
              _FileTreeToolbar(
                keyPrefix: widget.keyPrefix,
                headerLeading: widget.headerLeading,
                allExpanded: _allFoldersExpanded,
                nameSortDirection: _sortColumn == _FileTreeSortColumn.name
                    ? _sortDirection
                    : _FileTreeSortDirection.none,
                sizeSortDirection: _sortColumn == _FileTreeSortColumn.size
                    ? _sortDirection
                    : _FileTreeSortDirection.none,
                showSizeColumn: widget.showSizeColumn,
                trailingHeader: widget.trailingHeader,
                trailingColumnWidth: trailingColumnWidth,
                onNameSort: () => _toggleSort(_FileTreeSortColumn.name),
                onSizeSort: () => _toggleSort(_FileTreeSortColumn.size),
                onToggleExpanded: _allFoldersExpanded ? _collapseAll : _expandAll,
              ),
              Expanded(
                child: _nodes.isEmpty
                    ? _FileTreeEmptyState(label: widget.emptyLabel ?? context.l10n.noFiles)
                    : VirtualTreeView<FileTreeNode<T>>(
                        nodes: _nodes,
                        controller: _scrollController,
                        branchLine: BranchLine.path,
                        rowHeight: widget.rowHeight,
                        padding: const EdgeInsets.fromLTRB(_treeContentOffset, 4, 8, 4),
                        itemPadding: EdgeInsets.symmetric(vertical: treeItemInset * 0.5),
                        itemBuilder: (context, entry) => _buildTreeItem(
                          context,
                          entry,
                          nameContentWidth: nameContentWidth,
                          trailingColumnWidth: trailingColumnWidth,
                          hasTrailingColumn: hasTrailingColumn,
                          horizontalScrollEnabled: horizontalOverflow > 0.5,
                          contentTextStyle: labelStyle,
                        ),
                        transitionBuilder: _buildTreeTransition,
                      ),
              ),
              if (_nodes.isNotEmpty && horizontalOverflow > 0.5)
                _FileNameHorizontalScrollbar(
                  keyPrefix: widget.keyPrefix,
                  controller: _horizontalScrollController,
                  overflow: horizontalOverflow,
                  trailingColumnWidth: trailingColumnWidth,
                ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildTreeItem(
    BuildContext context,
    VirtualTreeEntry<FileTreeNode<T>> entry, {
    required double nameContentWidth,
    required double trailingColumnWidth,
    required bool hasTrailingColumn,
    required bool horizontalScrollEnabled,
    required TextStyle contentTextStyle,
  }) {
    final palette = AppPalette.of(context);
    final node = entry.node;
    final data = node.data;
    final pendingNode = _pendingCollapsedNodes == null ? null : _findTreeNode(_pendingCollapsedNodes!, data.key);
    final visuallyExpanded = pendingNode?.expanded ?? node.expanded;
    final leading = widget.leadingBuilder?.call(context, data);
    final trailing = widget.trailingBuilder?.call(context, data);
    final trailingContent =
        trailing ??
        (data.size > 0
            ? Text(Util.fmtByte(data.size), style: contentTextStyle.copyWith(color: palette.textSecondary))
            : const SizedBox.shrink());

    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        _FileTreeNodeExpandButton(
          key: ValueKey('${widget.keyPrefix}-node-expand-${data.key}'),
          expandable: node.children.isNotEmpty,
          expanded: visuallyExpanded,
          rowHeight: widget.rowHeight,
          onPressed: () => _toggleFolder(data.key),
        ),
        const SizedBox(width: _treeExpandLeadingGap),
        Expanded(
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: widget.onNodePressed == null ? null : () => widget.onNodePressed!(data),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.center,
              children: [
                if (leading != null) ...[leading, const SizedBox(width: 8)],
                Icon(
                  data.isFolder
                      ? (visuallyExpanded ? BootstrapIcons.folder2Open : BootstrapIcons.folder2)
                      : _fileIcon(data.label),
                  size: widget.iconSize,
                  color: palette.textSecondary,
                ),
                const SizedBox(width: 4),
                Expanded(
                  child: ClipRect(
                    child: AnimatedBuilder(
                      animation: _horizontalScrollController,
                      builder: (context, child) {
                        final offset = horizontalScrollEnabled && _horizontalScrollController.hasClients
                            ? _horizontalScrollController.offset
                            : 0.0;
                        return Transform.translate(offset: Offset(-offset, 0), child: child);
                      },
                      child: OverflowBox(
                        alignment: Alignment.centerLeft,
                        minWidth: 0,
                        maxWidth: double.infinity,
                        child: SizedBox(
                          width: nameContentWidth,
                          child: Text(
                            data.label,
                            maxLines: 1,
                            softWrap: false,
                            style: contentTextStyle.copyWith(color: palette.textPrimary),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: 4),
                if (hasTrailingColumn)
                  SizedBox(
                    key: ValueKey('${widget.keyPrefix}-trailing-${data.key}'),
                    width: trailingColumnWidth,
                    child: Align(alignment: Alignment.centerRight, child: trailingContent),
                  ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildTreeTransition(BuildContext context, VirtualTreeEntry<FileTreeNode<T>> entry, Widget row) {
    if (_transitioningNodeKeys.contains(entry.node.data.key)) {
      return SizeTransition(
        sizeFactor: _reveal,
        axisAlignment: -1,
        child: FadeTransition(opacity: _reveal, child: row),
      );
    }
    return row;
  }

  void _onTransitionStatus(AnimationStatus status) {
    if (!mounted) return;
    if (status == AnimationStatus.completed && _pendingCollapsedNodes == null && _transitioning) {
      setState(() {
        _transitioningNodeKeys = const {};
        _transitioning = false;
      });
      return;
    }
    if (status == AnimationStatus.dismissed && _pendingCollapsedNodes != null) {
      final nodes = _pendingCollapsedNodes!;
      setState(() {
        _nodes = nodes;
        _allFoldersExpanded = _foldersAreExpanded(nodes);
        _pendingCollapsedNodes = null;
        _transitioningNodeKeys = const {};
        _transitioning = false;
      });
      _transitionController.value = 1;
    }
  }

  void _toggleFolder(String key) {
    final node = _findTreeNode(_nodes, key);
    if (node == null) return;
    _transitionToTree(node.expanded ? _nodes.collapseNode(node) : _nodes.expandNode(node));
  }

  void _expandAll() => _transitionToTree(_nodes.expandAll());

  void _collapseAll() => _transitionToTree(_nodes.collapseAll());

  void _transitionToTree(List<TreeNode<FileTreeNode<T>>> nodes) {
    if (_transitioning) return;
    final currentKeyOrder = _visibleNodeKeysInOrder(_nodes);
    final nextKeyOrder = _visibleNodeKeysInOrder(nodes);
    final currentKeys = currentKeyOrder.toSet();
    final nextKeys = nextKeyOrder.toSet();
    final enteringKeys = nextKeys.difference(currentKeys);
    final leavingKeys = currentKeys.difference(nextKeys);

    if (enteringKeys.isNotEmpty) {
      setState(() {
        _nodes = nodes;
        _allFoldersExpanded = _foldersAreExpanded(nodes);
        _transitioningNodeKeys = _viewportTransitionKeys(nextKeyOrder, enteringKeys);
        _transitioning = true;
      });
      _transitionController.forward(from: 0);
      return;
    }
    if (leavingKeys.isNotEmpty) {
      setState(() {
        _pendingCollapsedNodes = nodes;
        _transitioningNodeKeys = _viewportTransitionKeys(currentKeyOrder, leavingKeys);
        _transitioning = true;
      });
      _transitionController.reverse(from: 1);
      return;
    }
    setState(() {
      _nodes = nodes;
      _allFoldersExpanded = _foldersAreExpanded(nodes);
    });
  }

  Set<String> _viewportTransitionKeys(List<String> keyOrder, Set<String> changedKeys) {
    if (changedKeys.isEmpty) return const {};
    var firstIndex = 0;
    var rowCount = _fallbackAnimatedRows;
    if (_scrollController.hasClients) {
      final position = _scrollController.position;
      firstIndex = (position.pixels / widget.rowHeight).floor() - _treeAnimationOverscanRows;
      rowCount = (position.viewportDimension / widget.rowHeight).ceil() + (_treeAnimationOverscanRows * 2);
    }
    firstIndex = firstIndex.clamp(0, keyOrder.length);
    final endIndex = (firstIndex + rowCount).clamp(firstIndex, keyOrder.length);
    return keyOrder.sublist(firstIndex, endIndex).where(changedKeys.contains).toSet();
  }

  void _toggleSort(_FileTreeSortColumn column) {
    setState(() {
      if (_sortColumn != column) {
        _sortColumn = column;
        _sortDirection = _FileTreeSortDirection.asc;
      } else {
        _sortDirection = switch (_sortDirection) {
          _FileTreeSortDirection.none => _FileTreeSortDirection.asc,
          _FileTreeSortDirection.asc => _FileTreeSortDirection.desc,
          _FileTreeSortDirection.desc => _FileTreeSortDirection.none,
        };
        if (_sortDirection == _FileTreeSortDirection.none) _sortColumn = null;
      }
      _nodes = _sortNodes(_nodes, _sortColumn, _sortDirection);
    });
  }
}

class _FileTreeToolbar extends StatelessWidget {
  const _FileTreeToolbar({
    required this.keyPrefix,
    required this.headerLeading,
    required this.allExpanded,
    required this.nameSortDirection,
    required this.sizeSortDirection,
    required this.showSizeColumn,
    required this.trailingHeader,
    required this.trailingColumnWidth,
    required this.onNameSort,
    required this.onSizeSort,
    required this.onToggleExpanded,
  });

  final String keyPrefix;
  final Widget? headerLeading;
  final bool allExpanded;
  final _FileTreeSortDirection nameSortDirection;
  final _FileTreeSortDirection sizeSortDirection;
  final bool showSizeColumn;
  final Widget? trailingHeader;
  final double trailingColumnWidth;
  final VoidCallback onNameSort;
  final VoidCallback onSizeSort;
  final VoidCallback onToggleExpanded;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(8, 4, 8, 2),
      child: SizedBox(
        height: _treeHeaderControlHeight,
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            _FileTreeExpandToggle(keyPrefix: keyPrefix, expanded: allExpanded, onPressed: onToggleExpanded),
            const SizedBox(width: _treeExpandLeadingGap),
            if (headerLeading != null) ...[headerLeading!, const SizedBox(width: 8)],
            _FileTreeSortButton(
              keyPrefix: keyPrefix,
              sortKey: 'name',
              label: context.l10n.name,
              direction: nameSortDirection,
              onPressed: onNameSort,
            ),
            const Spacer(),
            if (trailingColumnWidth > 0)
              SizedBox(
                width: trailingColumnWidth,
                child: Align(
                  alignment: Alignment.centerRight,
                  child: trailingHeader != null
                      ? trailingHeader!
                      : showSizeColumn
                      ? _FileTreeSortButton(
                          keyPrefix: keyPrefix,
                          sortKey: 'size',
                          label: context.l10n.size,
                          direction: sizeSortDirection,
                          onPressed: onSizeSort,
                        )
                      : const SizedBox.shrink(),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _FileNameHorizontalScrollbar extends StatelessWidget {
  const _FileNameHorizontalScrollbar({
    required this.keyPrefix,
    required this.controller,
    required this.overflow,
    required this.trailingColumnWidth,
  });

  final String keyPrefix;
  final ScrollController controller;
  final double overflow;
  final double trailingColumnWidth;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(8, 0, 8, 4),
      child: Row(
        children: [
          Expanded(
            child: SizedBox(
              height: _treeHorizontalScrollbarHeight,
              child: LayoutBuilder(
                builder: (context, constraints) {
                  final viewportWidth = constraints.hasBoundedWidth ? constraints.maxWidth : _treeFallbackWidth;
                  return Scrollbar(
                    key: ValueKey('$keyPrefix-horizontal-scrollbar'),
                    controller: controller,
                    thumbVisibility: true,
                    trackVisibility: true,
                    interactive: true,
                    thickness: 6,
                    scrollbarOrientation: ScrollbarOrientation.bottom,
                    child: SingleChildScrollView(
                      key: ValueKey('$keyPrefix-horizontal-scroll'),
                      controller: controller,
                      scrollDirection: Axis.horizontal,
                      child: SizedBox(width: viewportWidth + overflow, height: 1),
                    ),
                  );
                },
              ),
            ),
          ),
          if (trailingColumnWidth > 0) ...[
            const SizedBox(width: _treeTrailingGap),
            SizedBox(width: trailingColumnWidth),
          ],
        ],
      ),
    );
  }
}

class _FileTreeExpandToggle extends StatelessWidget {
  const _FileTreeExpandToggle({required this.keyPrefix, required this.expanded, required this.onPressed});

  final String keyPrefix;
  final bool expanded;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return SizedBox(
      key: ValueKey('$keyPrefix-expand-toggle'),
      width: _treeExpandButtonWidth,
      height: _treeHeaderControlHeight,
      child: GhostButton(
        density: ButtonDensity.compact,
        onPressed: onPressed,
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              _DirectionChevron(key: ValueKey('$keyPrefix-expand-top'), up: !expanded, color: palette.textPrimary),
              _DirectionChevron(key: ValueKey('$keyPrefix-expand-bottom'), up: expanded, color: palette.textPrimary),
            ],
          ),
        ),
      ),
    );
  }
}

class _FileTreeNodeExpandButton extends StatelessWidget {
  const _FileTreeNodeExpandButton({
    super.key,
    required this.expandable,
    required this.expanded,
    required this.rowHeight,
    required this.onPressed,
  });

  final bool expandable;
  final bool expanded;
  final double rowHeight;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    if (!expandable) {
      return SizedBox(width: _treeExpandButtonWidth, height: rowHeight);
    }
    final palette = AppPalette.of(context);
    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTap: onPressed,
      child: SizedBox(
        width: _treeExpandButtonWidth,
        height: rowHeight,
        child: Center(
          child: AnimatedRotation(
            turns: expanded ? 0.25 : 0,
            duration: _treeTransitionDuration,
            curve: Curves.easeOutCubic,
            child: Icon(BootstrapIcons.caretRightFill, size: 10, color: palette.textSecondary),
          ),
        ),
      ),
    );
  }
}

class _DirectionChevron extends StatelessWidget {
  const _DirectionChevron({super.key, required this.up, required this.color});

  final bool up;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 14,
      height: 8,
      child: CustomPaint(
        painter: _DirectionChevronPainter(up: up, color: color),
      ),
    );
  }
}

class _DirectionChevronPainter extends CustomPainter {
  const _DirectionChevronPainter({required this.up, required this.color});

  final bool up;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final path = Path();
    if (up) {
      path
        ..moveTo(2.5, 5.75)
        ..lineTo(size.width / 2, 1.75)
        ..lineTo(size.width - 2.5, 5.75);
    } else {
      path
        ..moveTo(2.5, 2.25)
        ..lineTo(size.width / 2, 6.25)
        ..lineTo(size.width - 2.5, 2.25);
    }
    canvas.drawPath(
      path,
      Paint()
        ..color = color
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.5
        ..strokeCap = StrokeCap.round
        ..strokeJoin = StrokeJoin.round,
    );
  }

  @override
  bool shouldRepaint(covariant _DirectionChevronPainter oldDelegate) {
    return oldDelegate.up != up || oldDelegate.color != color;
  }
}

class _FileTreeSortButton extends StatelessWidget {
  const _FileTreeSortButton({
    required this.keyPrefix,
    required this.sortKey,
    required this.label,
    required this.direction,
    required this.onPressed,
  });

  final String keyPrefix;
  final String sortKey;
  final String label;
  final _FileTreeSortDirection direction;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return GhostButton(
      density: ButtonDensity.compact,
      onPressed: onPressed,
      child: SizedBox(
        height: _treeHeaderControlHeight,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            Text(label, style: const TextStyle(fontSize: 12)),
            const SizedBox(width: 2),
            SizedBox(
              width: 14,
              height: 16,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  _SortTriangle(
                    key: ValueKey('$keyPrefix-sort-$sortKey-asc'),
                    up: true,
                    active: direction == _FileTreeSortDirection.asc,
                    palette: palette,
                  ),
                  _SortTriangle(
                    key: ValueKey('$keyPrefix-sort-$sortKey-desc'),
                    up: false,
                    active: direction == _FileTreeSortDirection.desc,
                    palette: palette,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _SortTriangle extends StatelessWidget {
  const _SortTriangle({super.key, required this.up, required this.active, required this.palette});

  final bool up;
  final bool active;
  final AppPalette palette;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(color: active ? palette.itemActiveBg : null, borderRadius: BorderRadius.circular(2)),
      child: SizedBox(
        width: 14,
        height: 8,
        child: CustomPaint(
          painter: _SortTrianglePainter(up: up, color: active ? palette.textPrimary : palette.textMuted),
        ),
      ),
    );
  }
}

class _SortTrianglePainter extends CustomPainter {
  const _SortTrianglePainter({required this.up, required this.color});

  final bool up;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final centerX = size.width / 2;
    const horizontalInset = 2.5;
    const verticalInset = 1.75;
    final left = horizontalInset;
    final right = size.width - horizontalInset;
    final top = verticalInset;
    final bottom = size.height - verticalInset;
    final path = Path();
    if (up) {
      path
        ..moveTo(centerX, top)
        ..lineTo(right, bottom)
        ..lineTo(left, bottom);
    } else {
      path
        ..moveTo(left, top)
        ..lineTo(right, top)
        ..lineTo(centerX, bottom);
    }
    canvas.drawPath(path..close(), Paint()..color = color);
  }

  @override
  bool shouldRepaint(covariant _SortTrianglePainter oldDelegate) {
    return oldDelegate.up != up || oldDelegate.color != color;
  }
}

class _FileTreeEmptyState extends StatelessWidget {
  const _FileTreeEmptyState({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Center(
      child: Text(
        label,
        style: TextStyle(color: palette.textMuted, fontSize: 13, fontWeight: FontWeight.w600),
      ),
    );
  }
}

class _BuildNode<T> {
  _BuildNode({required this.key, required this.label, required this.originalIndex, this.item});

  final String key;
  final String label;
  final int originalIndex;
  final FileTreeItem<T>? item;
  final List<_BuildNode<T>> children = [];

  int get size => item?.size ?? children.fold<int>(0, (sum, child) => sum + child.size);

  TreeItemNode<FileTreeNode<T>> toTreeItem() {
    final childItems = children.map((child) => child.toTreeItem()).toList(growable: false);
    final leaves = item == null ? [for (final child in childItems) ...child.data.leafItems] : <FileTreeItem<T>>[item!];
    return TreeItemNode(
      data: FileTreeNode(
        key: key,
        label: label,
        leafItems: List.unmodifiable(leaves),
        size: size,
        originalIndex: originalIndex,
        item: item,
      ),
      expanded: true,
      children: childItems,
    );
  }
}

List<TreeNode<FileTreeNode<T>>> _buildTree<T>(List<FileTreeItem<T>> items) {
  final root = <_BuildNode<T>>[];
  final folders = <String, _BuildNode<T>>{};
  var originalIndex = 0;

  for (final item in items) {
    final parts = item.path.split(RegExp(r'[/\\]+')).where((part) => part.trim().isNotEmpty).toList();
    var currentPath = '';
    _BuildNode<T>? parent;

    for (final part in parts) {
      currentPath = currentPath.isEmpty ? part : '$currentPath/$part';
      final folder = folders.putIfAbsent(currentPath, () {
        final node = _BuildNode<T>(key: 'folder:$currentPath', label: part, originalIndex: originalIndex++);
        if (parent == null) {
          root.add(node);
        } else {
          parent.children.add(node);
        }
        return node;
      });
      parent = folder;
    }

    final fileNode = _BuildNode<T>(
      key: 'file:${item.key}',
      label: item.name,
      originalIndex: originalIndex++,
      item: item,
    );
    if (parent == null) {
      root.add(fileNode);
    } else {
      parent.children.add(fileNode);
    }
  }

  return root.map((node) => node.toTreeItem()).toList(growable: false);
}

List<String> _visibleNodeKeysInOrder<T>(List<TreeNode<FileTreeNode<T>>> nodes) {
  final keys = <String>[];

  void collect(TreeNode<FileTreeNode<T>> node) {
    final item = node as TreeItemNode<FileTreeNode<T>>;
    keys.add(item.data.key);
    if (!item.expanded) return;
    for (final child in item.children) {
      collect(child);
    }
  }

  for (final node in nodes) {
    collect(node);
  }
  return keys;
}

List<TreeNode<FileTreeNode<T>>> _sortNodes<T>(
  List<TreeNode<FileTreeNode<T>>> nodes,
  _FileTreeSortColumn? column,
  _FileTreeSortDirection direction,
) {
  final sorted = nodes
      .map((node) {
        final item = node as TreeItemNode<FileTreeNode<T>>;
        return item.updateChildren(_sortNodes(item.children, column, direction));
      })
      .toList(growable: true);
  sorted.sort((a, b) {
    final left = a.data;
    final right = b.data;
    if (left.isFolder != right.isFolder) return left.isFolder ? -1 : 1;
    if (column == null || direction == _FileTreeSortDirection.none) {
      return left.originalIndex.compareTo(right.originalIndex);
    }
    final comparison = switch (column) {
      _FileTreeSortColumn.name => left.label.toLowerCase().compareTo(right.label.toLowerCase()),
      _FileTreeSortColumn.size => left.size.compareTo(right.size),
    };
    if (comparison == 0) return left.originalIndex.compareTo(right.originalIndex);
    return direction == _FileTreeSortDirection.asc ? comparison : -comparison;
  });
  return sorted;
}

TreeItemNode<FileTreeNode<T>>? _findTreeNode<T>(Iterable<TreeNode<FileTreeNode<T>>> nodes, String key) {
  for (final node in nodes) {
    final item = node as TreeItemNode<FileTreeNode<T>>;
    if (item.data.key == key) return item;
    final child = _findTreeNode(item.children, key);
    if (child != null) return child;
  }
  return null;
}

bool _foldersAreExpanded<T>(Iterable<TreeNode<FileTreeNode<T>>> nodes) {
  var hasFolder = false;
  var allExpanded = true;

  void inspect(TreeNode<FileTreeNode<T>> node) {
    final item = node as TreeItemNode<FileTreeNode<T>>;
    if (item.children.isNotEmpty) {
      hasFolder = true;
      allExpanded = allExpanded && item.expanded;
    }
    for (final child in item.children) {
      inspect(child);
    }
  }

  for (final node in nodes) {
    inspect(node);
  }
  return hasFolder && allExpanded;
}

Set<String> _expandedFolderKeys<T>(Iterable<TreeNode<FileTreeNode<T>>> nodes) {
  final keys = <String>{};
  void collect(TreeNode<FileTreeNode<T>> node) {
    final item = node as TreeItemNode<FileTreeNode<T>>;
    if (item.children.isNotEmpty && item.expanded) keys.add(item.data.key);
    for (final child in item.children) {
      collect(child);
    }
  }

  for (final node in nodes) {
    collect(node);
  }
  return keys;
}

List<TreeNode<FileTreeNode<T>>> _restoreExpandedFolders<T>(
  List<TreeNode<FileTreeNode<T>>> nodes,
  Set<String> expandedKeys,
) {
  return nodes
      .map((node) {
        final item = node as TreeItemNode<FileTreeNode<T>>;
        final children = _restoreExpandedFolders(item.children, expandedKeys);
        return item
            .updateChildren(children)
            .updateState(expanded: item.children.isNotEmpty && expandedKeys.contains(item.data.key));
      })
      .toList(growable: false);
}

IconData _fileIcon(String name) {
  final ext = _extensionOf(name);
  if (_videoExts.contains(ext)) return BootstrapIcons.fileEarmarkPlay;
  if (_audioExts.contains(ext)) return BootstrapIcons.fileEarmarkMusic;
  if (_imageExts.contains(ext)) return BootstrapIcons.fileEarmarkImage;
  if (_archiveExts.contains(ext)) return BootstrapIcons.fileEarmarkZip;
  return BootstrapIcons.fileEarmark;
}

String _extensionOf(String name) {
  final index = name.lastIndexOf('.');
  if (index < 0 || index == name.length - 1) return '';
  return name.substring(index + 1).toLowerCase();
}

double _widestLabelWidth<T>(List<FileTreeItem<T>> items, double Function(String label) measureLabel) {
  var widest = _treeMinimumNameContentWidth;
  for (final item in items) {
    widest = math.max(widest, measureLabel(item.name));
    for (final segment in item.path.split(RegExp(r'[/\\]+'))) {
      if (segment.isNotEmpty) widest = math.max(widest, measureLabel(segment));
    }
  }
  return widest;
}

double _measureLabelWidth(
  String label, {
  required TextStyle style,
  required TextDirection textDirection,
  required TextScaler textScaler,
}) {
  final painter = TextPainter(
    text: TextSpan(text: label, style: style),
    maxLines: 1,
    textDirection: textDirection,
    textScaler: textScaler,
  )..layout();
  return painter.width + _treeNameTextPadding;
}

double _horizontalOverflow<T>(
  List<FileTreeItem<T>> items, {
  required double baseNameViewport,
  required double depthIndent,
  required double Function(String label) measureLabel,
}) {
  var overflow = 0.0;
  for (final item in items) {
    final segments = item.path.split(RegExp(r'[/\\]+')).where((segment) => segment.isNotEmpty).toList();
    for (var depth = 0; depth < segments.length; depth++) {
      final viewport = math.max(_treeMinimumNameViewport, baseNameViewport - (depth * depthIndent));
      overflow = math.max(overflow, measureLabel(segments[depth]) - viewport);
    }
    final fileViewport = math.max(_treeMinimumNameViewport, baseNameViewport - (segments.length * depthIndent));
    overflow = math.max(overflow, measureLabel(item.name) - fileViewport);
  }
  return math.max(0, overflow);
}

enum _FileTreeSortColumn { name, size }

enum _FileTreeSortDirection { none, asc, desc }

const _videoExts = <String>{'mp4', 'mkv', 'avi', 'mov', 'wmv', 'flv', 'webm', 'm4v'};
const _audioExts = <String>{'mp3', 'flac', 'wav', 'aac', 'm4a', 'ogg', 'ape'};
const _imageExts = <String>{'jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg'};
const _archiveExts = <String>{'zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz'};
const _treeExpandButtonWidth = 18.0;
const _treeExpandLeadingGap = 4.0;
const _treeContentOffset = 8.0;
const _treeHeaderControlHeight = 24.0;
const _treeRowHeight = 28.0;
const _treeSizeColumnWidth = 72.0;
const _treeCustomTrailingColumnWidth = 116.0;
const _treeTrailingGap = 4.0;
const _treeHorizontalPadding = 16.0;
const _treeNodeExpandChrome = _treeExpandButtonWidth + _treeExpandLeadingGap;
const _treeLeadingChrome = 24.0;
const _treeMinimumNameViewport = 48.0;
const _treeMinimumNameContentWidth = 48.0;
const _treeNameTextPadding = 8.0;
const _treeFallbackWidth = 600.0;
const _treeHorizontalScrollbarHeight = 12.0;
const _treeTransitionDuration = Duration(milliseconds: 150);
const _treeAnimationOverscanRows = 3;
const _fallbackAnimatedRows = 32;
