import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' hide Column, Expanded, Row;

import '../../../../api/model/resource.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/file_tree_view.dart';
import '../../../../util/util.dart';

class ResolveFileTree extends StatefulWidget {
  const ResolveFileTree({
    super.key,
    required this.files,
    required this.initialSelection,
    required this.onSelectionChanged,
  });

  final List<FileInfo> files;
  final List<int> initialSelection;
  final ValueChanged<List<int>> onSelectionChanged;

  @override
  State<ResolveFileTree> createState() => _ResolveFileTreeState();
}

class _ResolveFileTreeState extends State<ResolveFileTree> {
  late List<FileTreeItem<int>> _treeItems;
  late List<int> _allFileIndexes;
  late final Set<int> _selectedIndexes;
  final Set<_FileTypeFilter> _activeTypeFilters = {};

  @override
  void initState() {
    super.initState();
    _selectedIndexes = widget.initialSelection.where(_isValidIndex).toSet();
    _rebuildTreeItems();
    WidgetsBinding.instance.addPostFrameCallback((_) => _notifySelection());
  }

  @override
  void didUpdateWidget(covariant ResolveFileTree oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (identical(widget.files, oldWidget.files)) return;
    _selectedIndexes.removeWhere((index) => !_isValidIndex(index));
    _activeTypeFilters.clear();
    _rebuildTreeItems();
    WidgetsBinding.instance.addPostFrameCallback((_) => _notifySelection());
  }

  @override
  Widget build(BuildContext context) {
    final selectedCount = _selectedIndexes.length;
    final allSelected = _allFileIndexes.isNotEmpty && selectedCount == _allFileIndexes.length;
    final partiallySelected = selectedCount > 0 && !allSelected;
    final selectedSize = _selectedIndexes.fold<int>(0, (sum, index) => sum + widget.files[index].size);

    return Column(
      children: [
        Expanded(
          child: FileTreeView<int>(
            items: _treeItems,
            keyPrefix: 'resolve-tree',
            headerLeading: ExcludeFocus(
              child: Checkbox(
                state: allSelected
                    ? CheckboxState.checked
                    : (partiallySelected ? CheckboxState.indeterminate : CheckboxState.unchecked),
                onChanged: _allFileIndexes.isEmpty ? null : (_) => _setFileSelection(_allFileIndexes, !allSelected),
              ),
            ),
            leadingBuilder: (context, node) {
              final indexes = node.leafItems.map((item) => item.data).toList(growable: false);
              final nodeSelectedCount = indexes.where(_selectedIndexes.contains).length;
              final selected = indexes.isNotEmpty && nodeSelectedCount == indexes.length;
              final partial = nodeSelectedCount > 0 && !selected;
              return ExcludeFocus(
                child: Checkbox(
                  state: selected
                      ? CheckboxState.checked
                      : (partial ? CheckboxState.indeterminate : CheckboxState.unchecked),
                  onChanged: (_) => _setFileSelection(indexes, !selected),
                ),
              );
            },
            onNodePressed: _toggleNodeSelection,
          ),
        ),
        const SizedBox(height: 12),
        _TreeFooter(
          selectedCount: selectedCount,
          totalCount: _allFileIndexes.length,
          selectedSize: selectedSize,
          unknownSize: selectedCount > 0 && selectedSize == 0,
          activeTypeFilters: _activeTypeFilters,
          onSelectVideo: () => _toggleTypeFilter(_FileTypeFilter.video),
          onSelectAudio: () => _toggleTypeFilter(_FileTypeFilter.audio),
          onSelectImage: () => _toggleTypeFilter(_FileTypeFilter.image),
        ),
      ],
    );
  }

  bool _isValidIndex(int index) => index >= 0 && index < widget.files.length;

  void _rebuildTreeItems() {
    _allFileIndexes = List<int>.generate(widget.files.length, (index) => index, growable: false);
    _treeItems = widget.files
        .asMap()
        .entries
        .map(
          (entry) => FileTreeItem<int>(
            key: entry.key.toString(),
            path: entry.value.path,
            name: entry.value.name,
            size: entry.value.size,
            data: entry.key,
          ),
        )
        .toList(growable: false);
  }

  void _setFileSelection(Iterable<int> indexes, bool selected) {
    setState(() {
      _activeTypeFilters.clear();
      if (selected) {
        _selectedIndexes.addAll(indexes);
      } else {
        _selectedIndexes.removeAll(indexes);
      }
    });
    _notifySelection();
  }

  void _toggleNodeSelection(FileTreeNode<int> node) {
    final indexes = node.leafItems.map((item) => item.data).toList(growable: false);
    final selected = indexes.isNotEmpty && indexes.every(_selectedIndexes.contains);
    _setFileSelection(indexes, !selected);
  }

  void _toggleTypeFilter(_FileTypeFilter filter) {
    setState(() {
      if (!_activeTypeFilters.remove(filter)) {
        _activeTypeFilters.add(filter);
      }
      final extensions = <String>{for (final activeFilter in _activeTypeFilters) ...activeFilter.extensions};
      _selectedIndexes
        ..clear()
        ..addAll(
          widget.files
              .asMap()
              .entries
              .where((entry) => extensions.contains(_extensionOf(entry.value.name)))
              .map((entry) => entry.key),
        );
    });
    _notifySelection();
  }

  void _notifySelection() {
    final values = _selectedIndexes.toList()..sort();
    widget.onSelectionChanged(values);
  }
}

class _TreeFooter extends StatelessWidget {
  const _TreeFooter({
    required this.selectedCount,
    required this.totalCount,
    required this.selectedSize,
    required this.unknownSize,
    required this.activeTypeFilters,
    required this.onSelectVideo,
    required this.onSelectAudio,
    required this.onSelectImage,
  });

  final int selectedCount;
  final int totalCount;
  final int selectedSize;
  final bool unknownSize;
  final Set<_FileTypeFilter> activeTypeFilters;
  final VoidCallback onSelectVideo;
  final VoidCallback onSelectAudio;
  final VoidCallback onSelectImage;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final filters = _FileTypeFilterGroup(
      videoActive: activeTypeFilters.contains(_FileTypeFilter.video),
      audioActive: activeTypeFilters.contains(_FileTypeFilter.audio),
      imageActive: activeTypeFilters.contains(_FileTypeFilter.image),
      onSelectVideo: onSelectVideo,
      onSelectAudio: onSelectAudio,
      onSelectImage: onSelectImage,
    );
    final stats = Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          context.l10n.selectedCount(selectedCount, totalCount),
          style: TextStyle(color: palette.textSecondary, fontSize: 12, fontWeight: FontWeight.w600),
        ),
        const SizedBox(width: 8),
        Text(
          unknownSize ? context.l10n.unknownSize : Util.fmtByte(selectedSize),
          style: TextStyle(color: palette.textSecondary, fontSize: 12, fontWeight: FontWeight.w600),
        ),
      ],
    );
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth < 360) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Align(alignment: Alignment.centerLeft, child: filters),
              const SizedBox(height: 4),
              Align(alignment: Alignment.centerRight, child: stats),
            ],
          );
        }
        return Row(crossAxisAlignment: CrossAxisAlignment.center, children: [filters, const Spacer(), stats]);
      },
    );
  }
}

class _FileTypeFilterGroup extends StatelessWidget {
  const _FileTypeFilterGroup({
    required this.videoActive,
    required this.audioActive,
    required this.imageActive,
    required this.onSelectVideo,
    required this.onSelectAudio,
    required this.onSelectImage,
  });

  final bool videoActive;
  final bool audioActive;
  final bool imageActive;
  final VoidCallback onSelectVideo;
  final VoidCallback onSelectAudio;
  final VoidCallback onSelectImage;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return DecoratedBox(
      decoration: BoxDecoration(
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          _FilterButton(
            key: const ValueKey('resolve-tree-filter-video'),
            icon: Icons.movie_outlined,
            active: videoActive,
            onPressed: onSelectVideo,
          ),
          _FilterDivider(color: palette.border),
          _FilterButton(
            key: const ValueKey('resolve-tree-filter-audio'),
            icon: Icons.audiotrack_outlined,
            active: audioActive,
            onPressed: onSelectAudio,
          ),
          _FilterDivider(color: palette.border),
          _FilterButton(
            key: const ValueKey('resolve-tree-filter-image'),
            icon: Icons.image_outlined,
            active: imageActive,
            onPressed: onSelectImage,
          ),
        ],
      ),
    );
  }
}

class _FilterDivider extends StatelessWidget {
  const _FilterDivider({required this.color});

  final Color color;

  @override
  Widget build(BuildContext context) {
    return SizedBox(width: 1, height: 18, child: ColoredBox(color: color));
  }
}

class _FilterButton extends StatelessWidget {
  const _FilterButton({super.key, required this.icon, required this.active, required this.onPressed});

  final IconData icon;
  final bool active;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return SizedBox(
      width: 40,
      height: 28,
      child: DecoratedBox(
        decoration: BoxDecoration(color: active ? palette.filterActiveBg : null),
        child: GhostButton(
          density: ButtonDensity.compact,
          onPressed: onPressed,
          child: Icon(icon, size: 14, color: active ? palette.brand : palette.textSecondary),
        ),
      ),
    );
  }
}

String _extensionOf(String name) {
  final index = name.lastIndexOf('.');
  if (index < 0 || index == name.length - 1) return '';
  return name.substring(index + 1).toLowerCase();
}

enum _FileTypeFilter { video, audio, image }

extension on _FileTypeFilter {
  Set<String> get extensions => switch (this) {
    _FileTypeFilter.video => _videoExts,
    _FileTypeFilter.audio => _audioExts,
    _FileTypeFilter.image => _imageExts,
  };
}

const _videoExts = <String>{'mp4', 'mkv', 'avi', 'mov', 'wmv', 'flv', 'webm', 'm4v'};
const _audioExts = <String>{'mp3', 'flac', 'wav', 'aac', 'm4a', 'ogg', 'ape'};
const _imageExts = <String>{'jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg'};
