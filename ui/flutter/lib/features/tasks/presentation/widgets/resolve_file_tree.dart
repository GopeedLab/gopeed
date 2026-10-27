import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' hide Column, Expanded, Row;

import '../../../../api/model/resource.dart';
import '../../application/pending_download.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/file_tree_view.dart';
import '../../../../util/util.dart';

typedef PendingFileKey = (String, int);

class ResolveFileTree extends StatefulWidget {
  const ResolveFileTree({super.key, required this.sources, required this.onSelectionChanged});

  final List<PendingDownload> sources;
  final ValueChanged<Set<PendingFileKey>> onSelectionChanged;

  @override
  State<ResolveFileTree> createState() => _ResolveFileTreeState();
}

class _ResolveFileTreeState extends State<ResolveFileTree> {
  late List<FileTreeItem<PendingFileKey>> _treeItems;
  late Map<PendingFileKey, FileInfo> _files;
  late Map<String, String> _labels;
  final Set<PendingFileKey> _selected = {};
  final Set<_FileTypeFilter> _activeTypeFilters = {};

  @override
  void initState() {
    super.initState();
    _rebuildTreeItems();
  }

  @override
  void didUpdateWidget(covariant ResolveFileTree oldWidget) {
    super.didUpdateWidget(oldWidget);
    _rebuildTreeItems();
  }

  void _rebuildTreeItems() {
    _files = {};
    _labels = {};
    _treeItems = [];
    _selected.clear();
    for (final source in widget.sources) {
      if (!source.included || source.result == null || source.created) continue;
      _labels[source.id] = source.name;
      final files = source.files.isEmpty ? [FileInfo(name: source.name)] : source.files;
      for (final entry in files.indexed) {
        if (source.createdFiles.containsKey(entry.$1)) continue;
        final key = (source.id, entry.$1);
        final file = entry.$2;
        _files[key] = file;
        if (source.selected.contains(entry.$1)) _selected.add(key);
        _treeItems.add(
          FileTreeItem<PendingFileKey>(
            key: '${source.id}:${entry.$1}',
            path: '${source.id}/${file.path}',
            name: file.name,
            size: file.size,
            data: key,
          ),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final allSelected = _files.isNotEmpty && _selected.length == _files.length;
    final partial = _selected.isNotEmpty && !allSelected;
    return Column(
      children: [
        Expanded(
          child: FileTreeView<PendingFileKey>(
            items: _treeItems,
            directoryLabels: _labels,
            initialExpandedDepth: 2,
            keyPrefix: 'resolve-tree',
            rowHeight: _resolveTreeRowHeight,
            contentTextStyle: _resolveFileNameStyle,
            headerLeading: ExcludeFocus(
              child: Checkbox(
                state: allSelected
                    ? CheckboxState.checked
                    : partial
                    ? CheckboxState.indeterminate
                    : CheckboxState.unchecked,
                onChanged: _files.isEmpty ? null : (_) => _setSelection(_files.keys, !allSelected),
              ),
            ),
            leadingBuilder: (context, node) {
              final keys = node.leafItems.map((item) => item.data).toList();
              final count = keys.where(_selected.contains).length;
              return ExcludeFocus(
                child: Checkbox(
                  state: count == keys.length
                      ? CheckboxState.checked
                      : count > 0
                      ? CheckboxState.indeterminate
                      : CheckboxState.unchecked,
                  onChanged: (_) => _setSelection(keys, count != keys.length),
                ),
              );
            },
            onNodePressed: (node) {
              final keys = node.leafItems.map((item) => item.data).toList();
              _setSelection(keys, !keys.every(_selected.contains));
            },
          ),
        ),
        const SizedBox(height: 12),
        _TreeFooter(
          selectedCount: _selected.length,
          totalCount: _files.length,
          selectedSize: _selected.fold(0, (sum, key) => sum + _files[key]!.size),
          unknownSize: _selected.any((key) => _files[key]!.size <= 0),
          activeTypeFilters: _activeTypeFilters,
          onSelectVideo: () => _toggleTypeFilter(_FileTypeFilter.video),
          onSelectAudio: () => _toggleTypeFilter(_FileTypeFilter.audio),
          onSelectImage: () => _toggleTypeFilter(_FileTypeFilter.image),
        ),
      ],
    );
  }

  void _setSelection(Iterable<PendingFileKey> keys, bool selected) {
    setState(() {
      _activeTypeFilters.clear();
      if (selected) {
        _selected.addAll(keys);
      } else {
        _selected.removeAll(keys);
      }
    });
    widget.onSelectionChanged(Set.of(_selected));
  }

  void _toggleTypeFilter(_FileTypeFilter filter) {
    setState(() {
      if (!_activeTypeFilters.remove(filter)) _activeTypeFilters.add(filter);
      final extensions = {for (final active in _activeTypeFilters) ...active.extensions};
      _selected
        ..clear()
        ..addAll(
          _files.entries
              .where((entry) => extensions.contains(_extensionOf(entry.value.name)))
              .map((entry) => entry.key),
        );
    });
    widget.onSelectionChanged(Set.of(_selected));
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
    final filters = KeyedSubtree(
      key: const ValueKey('resolve-tree-filters'),
      child: _FileTypeFilterGroup(
        videoActive: activeTypeFilters.contains(_FileTypeFilter.video),
        audioActive: activeTypeFilters.contains(_FileTypeFilter.audio),
        imageActive: activeTypeFilters.contains(_FileTypeFilter.image),
        onSelectVideo: onSelectVideo,
        onSelectAudio: onSelectAudio,
        onSelectImage: onSelectImage,
      ),
    );
    final statsStyle = TextStyle(color: palette.textSecondary, fontSize: 12, fontWeight: FontWeight.w600);
    final selectedLabel = context.l10n.selectedCount(selectedCount, totalCount);
    final sizeLabel = unknownSize ? context.l10n.unknownSize : Util.fmtByte(selectedSize);
    final baseline = MediaQuery.textScalerOf(context).scale(statsStyle.fontSize!);
    final stats = Wrap(
      key: const ValueKey('resolve-tree-selection-stats'),
      alignment: WrapAlignment.end,
      spacing: AppDesignTokens.space8,
      runSpacing: AppDesignTokens.space4,
      crossAxisAlignment: WrapCrossAlignment.start,
      children: [
        Baseline(
          baseline: baseline,
          baselineType: TextBaseline.alphabetic,
          child: Text(key: const ValueKey('resolve-tree-selected-count'), selectedLabel, style: statsStyle),
        ),
        Baseline(
          baseline: baseline,
          baselineType: TextBaseline.alphabetic,
          child: Text(key: const ValueKey('resolve-tree-selected-size'), sizeLabel, style: statsStyle),
        ),
      ],
    );

    return OverflowBar(
      key: const ValueKey('resolve-tree-footer'),
      spacing: AppDesignTokens.space8,
      alignment: MainAxisAlignment.spaceBetween,
      overflowAlignment: OverflowBarAlignment.end,
      overflowSpacing: AppDesignTokens.space4,
      children: [filters, stats],
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
    return SizedBox(
      width: _fileTypeFilterDividerWidth,
      height: 18,
      child: ColoredBox(color: color),
    );
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
      width: _fileTypeFilterButtonWidth,
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
const _resolveTreeRowHeight = 36.0;
const _resolveFileNameStyle = TextStyle(fontSize: 14, height: 1.25);
const _fileTypeFilterButtonWidth = 40.0;
const _fileTypeFilterDividerWidth = 1.0;
