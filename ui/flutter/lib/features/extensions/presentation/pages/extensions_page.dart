import 'dart:async';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart' show Icons, Tooltip;
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;
import 'package:url_launcher/url_launcher.dart';

import '../../../../api/model/extension.dart' as api_extension;
import '../../../../api/model/store_extension.dart';
import '../../../../core/utils/breakpoints.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_primary_button.dart';
import '../../../../shared/widgets/app_toast.dart';
import '../../../../l10n/l10n.dart';
import '../../../home/presentation/widgets/primary_rail.dart';
import '../../application/extensions_controller.dart';
import '../../application/pending_extension_install.dart';

const _extensionCardMinWidth = 280.0;
const _extensionGridSpacing = 10.0;

int _extensionGridColumnCount(double width) {
  return ((width + _extensionGridSpacing) / (_extensionCardMinWidth + _extensionGridSpacing))
      .floor()
      .clamp(1, 12)
      .toInt();
}

double _extensionGridCardWidth(double width) {
  final columns = _extensionGridColumnCount(width);
  return (width - _extensionGridSpacing * (columns - 1)) / columns;
}

class ExtensionsPage extends ConsumerStatefulWidget {
  const ExtensionsPage({super.key});

  @override
  ConsumerState<ExtensionsPage> createState() => _ExtensionsPageState();
}

class _ExtensionsPageState extends ConsumerState<ExtensionsPage> {
  final _searchController = TextEditingController();
  final _installController = TextEditingController();
  final _listScrollController = ScrollController();
  final Map<String, TextEditingController> _settingControllers = {};
  api_extension.Extension? _settingsExtension;
  shad.OverlayCompleter<dynamic>? _installPopover;
  bool _requestingNextPage = false;

  @override
  void initState() {
    super.initState();
    _listScrollController.addListener(_loadNextPageIfNeeded);
  }

  @override
  void dispose() {
    _installPopover?.remove();
    _searchController.dispose();
    _installController.dispose();
    _listScrollController
      ..removeListener(_loadNextPageIfNeeded)
      ..dispose();
    for (final controller in _settingControllers.values) {
      controller.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final isDesktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    ref.listen(pendingExtensionInstallProvider, (previous, next) {
      if (next == null) return;
      ref.read(pendingExtensionInstallProvider.notifier).clear();
      unawaited(
        _runAction(
          () => ref.read(extensionsControllerProvider.notifier).installFromUrl(next.url, devInstall: next.devMode),
        ),
      );
    });
    final stateAsync = ref.watch(extensionsControllerProvider);
    final body = Stack(
      children: [
        ColoredBox(
          color: palette.bg,
          child: Row(
            children: [
              if (MediaQuery.sizeOf(context).width >= Breakpoints.mobile)
                const PrimaryRail(activeSection: RailSection.extensions),
              Expanded(
                child: Padding(
                  padding: EdgeInsets.only(
                    top: MediaQuery.sizeOf(context).width >= Breakpoints.mobile
                        ? AppDesignTokens.windowHeaderHeight
                        : 0,
                  ),
                  child: stateAsync.when(
                    loading: () => const Center(child: shad.CircularProgressIndicator()),
                    error: (error, _) => _ErrorState(
                      error: error,
                      onRetry: () => ref.read(extensionsControllerProvider.notifier).loadInitialData(),
                    ),
                    data: (state) {
                      if (state.listFilter == ExtensionListFilter.market &&
                          state.storePagination?.hasNext == true &&
                          !state.loadingMoreStore) {
                        WidgetsBinding.instance.addPostFrameCallback((_) => _loadNextPageIfNeeded());
                      }
                      return _Content(
                        state: state,
                        scrollController: _listScrollController,
                        searchController: _searchController,
                        onSearch: (query) =>
                            _runAction(() => ref.read(extensionsControllerProvider.notifier).searchStore(query)),
                        onSort: (sort) =>
                            _runAction(() => ref.read(extensionsControllerProvider.notifier).changeSort(sort)),
                        onFilter: ref.read(extensionsControllerProvider.notifier).changeFilter,
                        onOpenInstall: _openInstallPopover,
                        onDevelopExtension: _openExtensionDevelopmentDocs,
                        onInstallFolder: _installFromFolder,
                        onRefresh: () =>
                            _runAction(() => ref.read(extensionsControllerProvider.notifier).loadInitialData()),
                        onItemAction: _runAction,
                        onOpenSettings: _openExtensionSettings,
                      );
                    },
                  ),
                ),
              ),
            ],
          ),
        ),
        if (_settingsExtension != null)
          _ExtensionSettingsPanel(
            extension: _settingsExtension!,
            controllers: _settingControllers,
            onClose: () => setState(() => _settingsExtension = null),
            onSave: _saveExtensionSettings,
          ),
      ],
    );

    return shad.Scaffold(
      backgroundColor: palette.bg,
      child: isDesktop
          ? body
          : Column(
              children: [
                Expanded(child: body),
                const PrimaryBottomNavigation(activeSection: RailSection.extensions),
              ],
            ),
    );
  }

  Future<void> _installFromUrl() async {
    final url = _installController.text.trim();
    if (url.isEmpty) return;
    await _runAction(() => ref.read(extensionsControllerProvider.notifier).installFromUrl(url));
  }

  Future<void> _loadNextPageIfNeeded() async {
    if (!mounted || _requestingNextPage || !_listScrollController.hasClients) return;
    if (_listScrollController.position.extentAfter > 240) return;
    final state = ref.read(extensionsControllerProvider).value;
    if (state == null ||
        state.listFilter != ExtensionListFilter.market ||
        state.storePagination?.hasNext != true ||
        state.loadingMoreStore) {
      return;
    }

    _requestingNextPage = true;
    try {
      await _runAction(() => ref.read(extensionsControllerProvider.notifier).loadMoreStore());
    } finally {
      _requestingNextPage = false;
    }
  }

  void _openInstallPopover(BuildContext anchorContext) {
    ref.read(extensionsControllerProvider.notifier).tryOpenDevMode();
    if (_installPopover != null) return;
    final overlay = const shad.PopoverOverlayHandler().show<void>(
      context: anchorContext,
      alignment: Alignment.topRight,
      anchorAlignment: Alignment.bottomRight,
      offset: const Offset(0, 8),
      modal: false,
      consumeOutsideTaps: false,
      builder: (context) => _InstallPopover(controller: _installController, onInstallUrl: _installFromUrl),
    );
    _installPopover = overlay;
    unawaited(
      overlay.animationFuture.whenComplete(() {
        if (identical(_installPopover, overlay)) _installPopover = null;
      }),
    );
  }

  Future<void> _installFromFolder() async {
    final folder = await FilePicker.platform.getDirectoryPath();
    if (folder == null || folder.isEmpty) return;
    await _runAction(() => ref.read(extensionsControllerProvider.notifier).installFromUrl(folder, devInstall: true));
  }

  Future<void> _openExtensionDevelopmentDocs() async {
    final opened = await launchUrl(Uri.parse('https://gopeed.com/docs/dev-extension'));
    if (!opened && mounted) {
      showAppToast(context, context.l10n.unableOpenExtensionDocs, type: AppToastType.error);
    }
  }

  void _openExtensionSettings(api_extension.Extension extension) {
    for (final controller in _settingControllers.values) {
      controller.dispose();
    }
    _settingControllers
      ..clear()
      ..addEntries(
        (extension.settings ?? const <api_extension.Setting>[]).map(
          (setting) => MapEntry(setting.name, TextEditingController(text: setting.value?.toString() ?? '')),
        ),
      );
    setState(() => _settingsExtension = extension);
  }

  Future<void> _saveExtensionSettings() async {
    final extension = _settingsExtension;
    if (extension == null) return;
    final values = <String, dynamic>{};
    for (final setting in extension.settings ?? const <api_extension.Setting>[]) {
      final text = _settingControllers[setting.name]?.text.trim() ?? '';
      values[setting.name] = switch (setting.type) {
        api_extension.SettingType.number => num.tryParse(text) ?? 0,
        api_extension.SettingType.boolean => text == 'true' || text == '1' || text.toLowerCase() == 'yes',
        api_extension.SettingType.string => text,
      };
    }
    await _runAction(() => ref.read(extensionsControllerProvider.notifier).saveExtensionSettings(extension, values));
    if (mounted) {
      setState(() => _settingsExtension = null);
    }
  }

  Future<void> _runAction(Future<void> Function() action) async {
    try {
      await action();
    } catch (error) {
      if (!mounted) return;
      showAppToast(context, error.toString(), type: AppToastType.error);
    }
  }
}

class _Content extends StatelessWidget {
  const _Content({
    required this.state,
    required this.scrollController,
    required this.searchController,
    required this.onSearch,
    required this.onSort,
    required this.onFilter,
    required this.onOpenInstall,
    required this.onDevelopExtension,
    required this.onInstallFolder,
    required this.onRefresh,
    required this.onItemAction,
    required this.onOpenSettings,
  });

  final ExtensionsState state;
  final ScrollController scrollController;
  final TextEditingController searchController;
  final ValueChanged<String> onSearch;
  final ValueChanged<StoreExtensionSort> onSort;
  final ValueChanged<ExtensionListFilter> onFilter;
  final ValueChanged<BuildContext> onOpenInstall;
  final VoidCallback onDevelopExtension;
  final VoidCallback onInstallFolder;
  final VoidCallback onRefresh;
  final Future<void> Function(Future<void> Function() action) onItemAction;
  final ValueChanged<api_extension.Extension> onOpenSettings;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final horizontalPadding = MediaQuery.sizeOf(context).width < Breakpoints.mobile ? 16.0 : 32.0;
    return Column(
      children: [
        Padding(
          padding: EdgeInsets.fromLTRB(horizontalPadding, 18, horizontalPadding, 12),
          child: Column(
            children: [
              _Toolbar(
                state: state,
                searchController: searchController,
                onSearch: onSearch,
                onSort: onSort,
                onRefresh: onRefresh,
                onOpenInstall: onOpenInstall,
                onDevelopExtension: onDevelopExtension,
                onInstallFolder: onInstallFolder,
              ),
              const SizedBox(height: 12),
              _FilterBar(state: state, onFilter: onFilter),
            ],
          ),
        ),
        Expanded(
          child: CustomScrollView(
            key: const ValueKey('extensions-list-scroll-view'),
            controller: scrollController,
            slivers: [
              if ((state.loadingInstalled || state.loadingStore) && state.displayItems.isEmpty)
                const SliverFillRemaining(child: Center(child: shad.CircularProgressIndicator()))
              else if (state.displayItems.isEmpty)
                SliverFillRemaining(
                  child: Center(
                    child: Text(
                      context.l10n.noExtensions,
                      style: TextStyle(color: palette.textSecondary, fontSize: 14),
                    ),
                  ),
                )
              else
                SliverPadding(
                  padding: EdgeInsets.fromLTRB(horizontalPadding, 0, horizontalPadding, 20),
                  sliver: SliverLayoutBuilder(
                    builder: (context, constraints) {
                      final width = constraints.crossAxisExtent;
                      final crossAxisCount = _extensionGridColumnCount(width);
                      return SliverGrid(
                        delegate: SliverChildBuilderDelegate(
                          (context, index) => _ExtensionCard(
                            item: state.displayItems[index],
                            busy: state.busyExtensionIds.contains(state.displayItems[index].id),
                            canUpdate: _canUpdate(state, state.displayItems[index]),
                            onAction: onItemAction,
                            onOpenSettings: onOpenSettings,
                          ),
                          childCount: state.displayItems.length,
                        ),
                        gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                          crossAxisCount: crossAxisCount,
                          mainAxisSpacing: _extensionGridSpacing,
                          crossAxisSpacing: _extensionGridSpacing,
                          mainAxisExtent: 178,
                        ),
                      );
                    },
                  ),
                ),
              if (state.listFilter == ExtensionListFilter.market && state.loadingMoreStore)
                SliverToBoxAdapter(
                  child: Padding(
                    padding: const EdgeInsets.only(bottom: 24, top: 4),
                    child: Center(child: const SizedBox.square(dimension: 18, child: shad.CircularProgressIndicator())),
                  ),
                ),
            ],
          ),
        ),
      ],
    );
  }

  bool _canUpdate(ExtensionsState state, ExtensionListItem item) {
    if (item.installed == null) return false;
    if (state.listFilter == ExtensionListFilter.market && item.store != null) {
      return _compareVersion(item.store!.version, item.installed!.version) > 0;
    }
    return state.updateFlags.containsKey(item.installed!.identity);
  }

  int _compareVersion(String a, String b) {
    final aNums = a.split(RegExp(r'[^0-9]+')).where((part) => part.isNotEmpty).map((part) => int.tryParse(part) ?? 0);
    final bNums = b.split(RegExp(r'[^0-9]+')).where((part) => part.isNotEmpty).map((part) => int.tryParse(part) ?? 0);
    final left = aNums.toList();
    final right = bNums.toList();
    final maxLen = left.length > right.length ? left.length : right.length;
    for (var i = 0; i < maxLen; i++) {
      final l = i < left.length ? left[i] : 0;
      final r = i < right.length ? right[i] : 0;
      if (l != r) return l.compareTo(r);
    }
    return 0;
  }
}

class _Toolbar extends StatelessWidget {
  const _Toolbar({
    required this.state,
    required this.searchController,
    required this.onSearch,
    required this.onSort,
    required this.onRefresh,
    required this.onOpenInstall,
    required this.onDevelopExtension,
    required this.onInstallFolder,
  });

  final ExtensionsState state;
  final TextEditingController searchController;
  final ValueChanged<String> onSearch;
  final ValueChanged<StoreExtensionSort> onSort;
  final VoidCallback onRefresh;
  final ValueChanged<BuildContext> onOpenInstall;
  final VoidCallback onDevelopExtension;
  final VoidCallback onInstallFolder;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final search = shad.TextField(
      key: const ValueKey('extension-search-input'),
      controller: searchController,
      placeholder: Text(context.l10n.searchExtensions, style: TextStyle(color: palette.searchHint)),
      features: [shad.InputFeature.leading(Icon(Icons.search_rounded, size: 16, color: palette.textMuted))],
      onSubmitted: onSearch,
    );
    final sorting = Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _Segmented<StoreExtensionSort>(
          key: const ValueKey('extension-sort-control'),
          value: state.storeSort,
          values: [
            (StoreExtensionSort.stars, context.l10n.extensionSortStars),
            (StoreExtensionSort.installs, context.l10n.extensionSortInstalls),
            (StoreExtensionSort.updated, context.l10n.extensionSortUpdated),
          ],
          onChanged: onSort,
        ),
        const SizedBox(width: 8),
        _OutlineToolbarIconButton(
          key: const ValueKey('refresh-extensions-button'),
          tooltip: context.l10n.refresh,
          icon: Icons.refresh,
          onPressed: onRefresh,
        ),
      ],
    );
    final installActions = Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        _OutlineToolbarIconButton(
          key: const ValueKey('develop-extension-button'),
          tooltip: context.l10n.extensionDeveloperGuide,
          icon: Icons.menu_book_outlined,
          onPressed: onDevelopExtension,
        ),
        const SizedBox(width: 8),
        if (state.devMode) ...[
          _OutlineToolbarIconButton(
            key: const ValueKey('load-local-extension-button'),
            tooltip: context.l10n.extensionLoadLocal,
            icon: Icons.folder_open_outlined,
            onPressed: state.busyExtensionIds.contains(ExtensionsController.manualInstallBusyKey)
                ? null
                : onInstallFolder,
          ),
          const SizedBox(width: 8),
        ],
        Builder(
          builder: (buttonContext) => _OutlineToolbarIconButton(
            key: const ValueKey('install-extension-button'),
            tooltip: context.l10n.extensionInstallFromUrl,
            icon: Icons.add_link,
            onPressed: state.busyExtensionIds.contains(ExtensionsController.manualInstallBusyKey)
                ? null
                : () => onOpenInstall(buttonContext),
          ),
        ),
      ],
    );
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth >= 720) {
          return Row(
            children: [
              SizedBox(width: _extensionGridCardWidth(constraints.maxWidth), child: search),
              const SizedBox(width: 10),
              Flexible(
                fit: FlexFit.loose,
                child: SingleChildScrollView(scrollDirection: Axis.horizontal, child: sorting),
              ),
              const Spacer(),
              installActions,
            ],
          );
        }
        return Column(
          children: [
            SizedBox(width: double.infinity, child: search),
            const SizedBox(height: 10),
            Row(
              children: [
                Expanded(
                  child: Align(
                    alignment: Alignment.centerLeft,
                    child: SingleChildScrollView(scrollDirection: Axis.horizontal, child: sorting),
                  ),
                ),
                const SizedBox(width: 10),
                installActions,
              ],
            ),
          ],
        );
      },
    );
  }
}

class _FilterBar extends StatelessWidget {
  const _FilterBar({required this.state, required this.onFilter});

  final ExtensionsState state;
  final ValueChanged<ExtensionListFilter> onFilter;

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.centerLeft,
      child: _Segmented<ExtensionListFilter>(
        value: state.listFilter,
        values: [
          (ExtensionListFilter.market, context.l10n.extensionFilterMarket),
          (ExtensionListFilter.installed, context.l10n.extensionFilterInstalled),
        ],
        onChanged: onFilter,
      ),
    );
  }
}

class _OutlineToolbarIconButton extends StatelessWidget {
  const _OutlineToolbarIconButton({super.key, required this.tooltip, required this.icon, required this.onPressed});

  final String tooltip;
  final IconData icon;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: tooltip,
      child: SizedBox.square(
        dimension: 32,
        child: shad.IconButton.outline(size: shad.ButtonSize.xSmall, onPressed: onPressed, icon: Icon(icon, size: 17)),
      ),
    );
  }
}

class _InstallPopover extends StatefulWidget {
  const _InstallPopover({required this.controller, required this.onInstallUrl});

  final TextEditingController controller;
  final Future<void> Function() onInstallUrl;

  @override
  State<_InstallPopover> createState() => _InstallPopoverState();
}

class _InstallPopoverState extends State<_InstallPopover> {
  bool _installing = false;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(
      key: const ValueKey('extension-install-popover'),
      width: 328,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: palette.cardBg,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: palette.border),
        boxShadow: [BoxShadow(color: const Color(0x26000000), blurRadius: 18, offset: const Offset(0, 6))],
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            context.l10n.extensionInstallFromUrl,
            style: TextStyle(color: palette.textPrimary, fontSize: 13, fontWeight: FontWeight.w700),
          ),
          const SizedBox(height: 10),
          shad.TextField(
            key: const ValueKey('extension-install-url-input'),
            controller: widget.controller,
            autofocus: true,
            placeholder: const Text('https://github.com/author/repo'),
            onSubmitted: (_) => _install(),
            features: [
              shad.InputFeature.trailing(
                SizedBox.square(
                  dimension: 28,
                  child: shad.IconButton.outline(
                    size: shad.ButtonSize.xSmall,
                    onPressed: _installing ? null : _install,
                    icon: _installing
                        ? const SizedBox.square(dimension: 13, child: shad.CircularProgressIndicator())
                        : const Icon(Icons.download_outlined, size: 16),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Future<void> _install() async {
    if (_installing || widget.controller.text.trim().isEmpty) return;
    setState(() => _installing = true);
    await widget.onInstallUrl();
    if (!mounted) return;
    setState(() => _installing = false);
    await shad.closeOverlay(context);
  }
}

class _Segmented<T> extends StatelessWidget {
  const _Segmented({super.key, required this.value, required this.values, required this.onChanged});

  final T value;
  final List<(T, String)> values;
  final ValueChanged<T> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: palette.surfaceSoft,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: palette.border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: values.map((entry) {
          final selected = entry.$1 == value;
          return GestureDetector(
            onTap: () => onChanged(entry.$1),
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
              decoration: BoxDecoration(
                color: selected ? palette.cardBg : null,
                borderRadius: BorderRadius.circular(4),
              ),
              child: Text(
                entry.$2,
                style: TextStyle(
                  color: selected ? palette.textPrimary : palette.textSecondary,
                  fontSize: 12,
                  fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
                ),
              ),
            ),
          );
        }).toList(),
      ),
    );
  }
}

class _ExtensionCard extends ConsumerWidget {
  const _ExtensionCard({
    required this.item,
    required this.busy,
    required this.canUpdate,
    required this.onAction,
    required this.onOpenSettings,
  });

  final ExtensionListItem item;
  final bool busy;
  final bool canUpdate;
  final Future<void> Function(Future<void> Function() action) onAction;
  final ValueChanged<api_extension.Extension> onOpenSettings;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final palette = AppPalette.of(context);
    final installed = item.installed;
    return Container(
      key: ValueKey('extension-card-${item.id}'),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: palette.cardBg,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: palette.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              _ExtensionIcon(item: item),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      item.title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(color: palette.textPrimary, fontSize: 14, fontWeight: FontWeight.w700),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      '${item.author} · v${item.version}',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(color: palette.textMuted, fontSize: 11),
                    ),
                  ],
                ),
              ),
              if (installed != null)
                shad.Switch(
                  value: !installed.disabled,
                  onChanged: busy
                      ? null
                      : (enabled) => onAction(
                          () => ref.read(extensionsControllerProvider.notifier).toggleExtension(installed, enabled),
                        ),
                ),
            ],
          ),
          const SizedBox(height: 10),
          Text(
            item.description,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(color: palette.textSecondary, fontSize: 12, height: 1.25),
          ),
          const Spacer(),
          Row(
            children: [
              if (item.store != null) ...[
                Icon(Icons.star_rounded, size: 15, color: palette.textMuted),
                const SizedBox(width: 3),
                Text('${item.stars}', style: TextStyle(color: palette.textSecondary, fontSize: 12)),
                const SizedBox(width: 10),
                Icon(Icons.download_outlined, size: 15, color: palette.textMuted),
                const SizedBox(width: 3),
                Text('${item.installCount}', style: TextStyle(color: palette.textSecondary, fontSize: 12)),
              ],
              const Spacer(),
              if (canUpdate && installed != null)
                shad.GhostButton(
                  density: shad.ButtonDensity.icon,
                  onPressed: busy
                      ? null
                      : () =>
                            onAction(() => ref.read(extensionsControllerProvider.notifier).upgradeExtension(installed)),
                  child: const Icon(Icons.refresh),
                ),
              if (installed == null && item.store != null)
                shad.GhostButton(
                  density: shad.ButtonDensity.icon,
                  onPressed: busy
                      ? null
                      : () => onAction(
                          () => ref.read(extensionsControllerProvider.notifier).installFromStore(item.store!),
                        ),
                  child: const Icon(Icons.download),
                ),
              if ((item.homepage ?? '').isNotEmpty)
                shad.GhostButton(
                  density: shad.ButtonDensity.icon,
                  onPressed: () => unawaited(launchUrl(Uri.parse(item.homepage!))),
                  child: const Icon(Icons.home_outlined),
                ),
              if ((item.repoUrl ?? '').isNotEmpty)
                shad.GhostButton(
                  density: shad.ButtonDensity.icon,
                  onPressed: () => unawaited(launchUrl(Uri.parse(item.repoUrl!))),
                  child: const Icon(Icons.code),
                ),
              if (installed?.settings?.isNotEmpty == true)
                shad.GhostButton(
                  density: shad.ButtonDensity.icon,
                  onPressed: busy ? null : () => onOpenSettings(installed!),
                  child: const Icon(Icons.settings_outlined),
                ),
              if (installed != null)
                shad.GhostButton(
                  key: ValueKey('remove-extension-${installed.identity}'),
                  density: shad.ButtonDensity.icon,
                  onPressed: busy
                      ? null
                      : () async {
                          final confirmed = await _showRemoveExtensionDialog(context, installed.title);
                          if (!confirmed || !context.mounted) return;
                          await onAction(
                            () => ref.read(extensionsControllerProvider.notifier).removeExtension(installed),
                          );
                        },
                  child: const Icon(Icons.delete_outline),
                ),
            ],
          ),
        ],
      ),
    );
  }
}

Future<bool> _showRemoveExtensionDialog(BuildContext context, String extensionName) async {
  final overlay = const shad.DialogOverlayHandler().show<bool>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: false,
    builder: (dialogContext) => shad.AlertDialog(
      key: const ValueKey('remove-extension-dialog'),
      title: Text(dialogContext.l10n.removeExtensionTitle),
      content: Text(dialogContext.l10n.removeExtensionConfirm(extensionName)),
      actions: [
        shad.SecondaryButton(
          key: const ValueKey('cancel-remove-extension-button'),
          onPressed: () => shad.closeOverlay(dialogContext, false),
          child: Text(dialogContext.l10n.cancel),
        ),
        shad.DestructiveButton(
          key: const ValueKey('confirm-remove-extension-button'),
          onPressed: () => shad.closeOverlay(dialogContext, true),
          child: Text(dialogContext.l10n.uninstall),
        ),
      ],
    ),
  );
  return await overlay.future ?? false;
}

class _ExtensionIcon extends StatelessWidget {
  const _ExtensionIcon({required this.item});

  final ExtensionListItem item;

  @override
  Widget build(BuildContext context) {
    final icon = item.icon;
    Widget image;
    if (icon != null && icon.isNotEmpty) {
      image = Image.network(icon, width: 40, height: 40, fit: BoxFit.cover, errorBuilder: (_, _, _) => _fallback());
    } else {
      image = _fallback();
    }
    return ClipRRect(borderRadius: BorderRadius.circular(6), child: image);
  }

  Widget _fallback() => Image.asset('assets/extension/default_icon.png', width: 40, height: 40, fit: BoxFit.cover);
}

class _ExtensionSettingsPanel extends StatelessWidget {
  const _ExtensionSettingsPanel({
    required this.extension,
    required this.controllers,
    required this.onClose,
    required this.onSave,
  });

  final api_extension.Extension extension;
  final Map<String, TextEditingController> controllers;
  final VoidCallback onClose;
  final VoidCallback onSave;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final settings = extension.settings ?? const <api_extension.Setting>[];
    return Positioned.fill(
      child: Row(
        children: [
          Expanded(
            child: GestureDetector(
              onTap: onClose,
              child: ColoredBox(color: const Color(0x66000000), child: const SizedBox.expand()),
            ),
          ),
          Container(
            width: MediaQuery.sizeOf(context).width < Breakpoints.mobile ? MediaQuery.sizeOf(context).width : 420,
            decoration: BoxDecoration(
              color: palette.bg,
              border: Border(left: BorderSide(color: palette.border)),
            ),
            child: SafeArea(
              child: Column(
                children: [
                  SizedBox(
                    height: 56,
                    child: Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 18),
                      child: Row(
                        children: [
                          Expanded(
                            child: Text(
                              extension.title,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(color: palette.textPrimary, fontSize: 16, fontWeight: FontWeight.w800),
                            ),
                          ),
                          shad.GhostButton(
                            density: shad.ButtonDensity.icon,
                            onPressed: onClose,
                            child: const Icon(Icons.close),
                          ),
                        ],
                      ),
                    ),
                  ),
                  Expanded(
                    child: ListView.separated(
                      padding: const EdgeInsets.all(18),
                      itemCount: settings.length,
                      separatorBuilder: (_, _) => const SizedBox(height: 14),
                      itemBuilder: (context, index) {
                        final setting = settings[index];
                        return _ExtensionSettingField(setting: setting, controller: controllers[setting.name]!);
                      },
                    ),
                  ),
                  Container(
                    padding: const EdgeInsets.all(18),
                    decoration: BoxDecoration(
                      border: Border(top: BorderSide(color: palette.border)),
                    ),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.end,
                      children: [
                        shad.SecondaryButton(onPressed: onClose, child: Text(context.l10n.cancel)),
                        const SizedBox(width: 10),
                        AppPrimaryButton(onPressed: onSave, child: Text(context.l10n.save)),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _ExtensionSettingField extends StatelessWidget {
  const _ExtensionSettingField({required this.setting, required this.controller});

  final api_extension.Setting setting;
  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          setting.title,
          style: TextStyle(color: palette.textPrimary, fontSize: 13, fontWeight: FontWeight.w700),
        ),
        if (setting.description.isNotEmpty) ...[
          const SizedBox(height: 4),
          Text(setting.description, style: TextStyle(color: palette.textSecondary, fontSize: 12, height: 1.35)),
        ],
        const SizedBox(height: 8),
        if (setting.type == api_extension.SettingType.boolean)
          shad.TextField(controller: controller, placeholder: Text(context.l10n.booleanValueHint))
        else
          shad.TextField(
            controller: controller,
            keyboardType: setting.type == api_extension.SettingType.number ? TextInputType.number : TextInputType.text,
          ),
      ],
    );
  }
}

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.error, required this.onRetry});

  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.extension_off_outlined, size: 32, color: palette.error),
            const SizedBox(height: 12),
            Text(
              context.l10n.unableLoadExtensions,
              style: TextStyle(color: palette.textPrimary, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 8),
            Text(
              error.toString(),
              textAlign: TextAlign.center,
              style: TextStyle(color: palette.textSecondary),
            ),
            const SizedBox(height: 16),
            shad.SecondaryButton(onPressed: onRetry, child: Text(context.l10n.retry)),
          ],
        ),
      ),
    );
  }
}
