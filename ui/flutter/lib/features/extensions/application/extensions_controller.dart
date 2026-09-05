import 'dart:async';
import 'dart:collection';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/gopeed_site_api.dart';
import '../../../api/model/extension.dart';
import '../../../api/model/install_extension.dart';
import '../../../api/model/store_extension.dart';
import '../../../api/model/switch_extension.dart';
import '../../../api/model/update_extension_settings.dart';
import '../../../core/capabilities/app_capabilities.dart';

final extensionsControllerProvider = AsyncNotifierProvider<ExtensionsController, ExtensionsState>(
  ExtensionsController.new,
);

enum ExtensionListFilter { market, installed }

class ExtensionListItem {
  const ExtensionListItem({this.installed, this.store});

  final Extension? installed;
  final StoreExtension? store;

  bool get isInstalled => installed != null;
  String get id => installed?.identity ?? store!.id;
  String get title => installed?.title ?? store?.title ?? '';
  String get author => installed?.author ?? store?.author ?? '';
  String get description => installed?.description ?? store?.description ?? '';
  String get version => installed?.version ?? store?.version ?? '0.0.0';
  String? get homepage => installed?.homepage.isNotEmpty == true ? installed!.homepage : store?.homepage;
  String? get repoUrl => installed?.repository?.url.isNotEmpty == true ? installed!.repository!.url : store?.repoUrl;
  int get stars => store?.stars ?? 0;
  int get installCount => store?.installCount ?? 0;
}

class ExtensionsState {
  const ExtensionsState({
    this.installedExtensions = const [],
    this.storeExtensions = const [],
    this.updateFlags = const {},
    this.storePagination,
    this.storeQuery = '',
    this.storeSort = StoreExtensionSort.stars,
    this.listFilter = ExtensionListFilter.market,
    this.loadingInstalled = false,
    this.loadingStore = false,
    this.loadingMoreStore = false,
    this.busyExtensionIds = const {},
    this.devMode = false,
  });

  final List<Extension> installedExtensions;
  final List<StoreExtension> storeExtensions;
  final Map<String, String> updateFlags;
  final StorePagination? storePagination;
  final String storeQuery;
  final StoreExtensionSort storeSort;
  final ExtensionListFilter listFilter;
  final bool loadingInstalled;
  final bool loadingStore;
  final bool loadingMoreStore;
  final Set<String> busyExtensionIds;
  final bool devMode;

  UnmodifiableMapView<String, Extension> get installedMap =>
      UnmodifiableMapView({for (final ext in installedExtensions) ext.identity: ext});

  UnmodifiableMapView<String, StoreExtension> get storeMap =>
      UnmodifiableMapView({for (final ext in storeExtensions) ext.id: ext});

  List<ExtensionListItem> get displayItems {
    if (listFilter == ExtensionListFilter.installed) {
      return installedExtensions
          .map((ext) => ExtensionListItem(installed: ext, store: storeMap[ext.identity]))
          .toList();
    }
    return storeExtensions.map((ext) => ExtensionListItem(installed: installedMap[ext.id], store: ext)).toList();
  }

  ExtensionListItem? findItem(String id) {
    final installed = installedMap[id];
    final store = storeMap[id];
    if (installed == null && store == null) return null;
    return ExtensionListItem(installed: installed, store: store);
  }

  ExtensionsState copyWith({
    List<Extension>? installedExtensions,
    List<StoreExtension>? storeExtensions,
    Map<String, String>? updateFlags,
    StorePagination? storePagination,
    bool clearStorePagination = false,
    String? storeQuery,
    StoreExtensionSort? storeSort,
    ExtensionListFilter? listFilter,
    bool? loadingInstalled,
    bool? loadingStore,
    bool? loadingMoreStore,
    Set<String>? busyExtensionIds,
    bool? devMode,
  }) {
    return ExtensionsState(
      installedExtensions: installedExtensions ?? this.installedExtensions,
      storeExtensions: storeExtensions ?? this.storeExtensions,
      updateFlags: updateFlags ?? this.updateFlags,
      storePagination: clearStorePagination ? null : storePagination ?? this.storePagination,
      storeQuery: storeQuery ?? this.storeQuery,
      storeSort: storeSort ?? this.storeSort,
      listFilter: listFilter ?? this.listFilter,
      loadingInstalled: loadingInstalled ?? this.loadingInstalled,
      loadingStore: loadingStore ?? this.loadingStore,
      loadingMoreStore: loadingMoreStore ?? this.loadingMoreStore,
      busyExtensionIds: busyExtensionIds ?? this.busyExtensionIds,
      devMode: devMode ?? this.devMode,
    );
  }
}

class ExtensionsController extends AsyncNotifier<ExtensionsState> {
  static const manualInstallBusyKey = '__manual_install__';
  int _devModeCount = 0;

  @override
  Future<ExtensionsState> build() async {
    var next = const ExtensionsState(loadingInstalled: true, loadingStore: true);
    state = AsyncValue.data(next);
    final installed = await _loadInstalled(refreshUpdates: true, current: next);
    next = state.value ?? installed;
    await _refreshStore(current: next);
    return state.value ?? next;
  }

  Future<void> loadInitialData() async {
    state = const AsyncValue.loading();
    state = AsyncValue.data(await build());
  }

  Future<void> loadInstalled({bool refreshUpdates = false}) async {
    await _loadInstalled(refreshUpdates: refreshUpdates, current: _current.copyWith(loadingInstalled: true));
  }

  Future<void> refreshStore() async {
    await _refreshStore(current: _current.copyWith(loadingStore: true));
  }

  Future<void> loadMoreStore() async {
    final current = _current;
    final pagination = current.storePagination;
    if (pagination == null || !pagination.hasNext || current.loadingMoreStore) return;
    state = AsyncValue.data(current.copyWith(loadingMoreStore: true));
    try {
      final page = await GopeedSiteApi.instance.getExtensions(
        page: pagination.page + 1,
        limit: pagination.limit,
        sort: current.storeSort,
        query: current.storeQuery,
      );
      state = AsyncValue.data(
        _current.copyWith(
          storeExtensions: [..._current.storeExtensions, ...page.data],
          storePagination: page.pagination,
          loadingMoreStore: false,
        ),
      );
    } catch (_) {
      state = AsyncValue.data(_current.copyWith(loadingMoreStore: false));
      rethrow;
    }
  }

  Future<void> searchStore(String query) async {
    state = AsyncValue.data(_current.copyWith(storeQuery: query.trim()));
    await refreshStore();
  }

  Future<void> changeSort(StoreExtensionSort sort) async {
    state = AsyncValue.data(_current.copyWith(storeSort: sort));
    await refreshStore();
  }

  void changeFilter(ExtensionListFilter filter) {
    state = AsyncValue.data(_current.copyWith(listFilter: filter));
  }

  bool canUpdateItem(ExtensionListItem item) {
    final current = _current;
    if (item.installed == null) return false;
    if (current.listFilter == ExtensionListFilter.market && item.store != null) {
      return _compareVersion(item.store!.version, item.installed!.version) > 0;
    }
    return current.updateFlags.containsKey(item.installed!.identity);
  }

  Future<void> installFromStore(StoreExtension extension) async {
    await _runBusy(extension.id, () async {
      final installUrl = (extension.directory ?? '').trim().isEmpty
          ? extension.repoUrl
          : '${extension.repoUrl}#${extension.directory!.trim()}';
      final installedId = await ref.read(gopeedServiceProvider).installExtension(InstallExtension(url: installUrl));
      await loadInstalled(refreshUpdates: false);
      unawaited(checkUpdate());
      _bumpStoreInstallCount(installedId.isNotEmpty ? installedId : extension.id);
      _reportInstallSafe(installedId.isNotEmpty ? installedId : extension.id);
    });
  }

  Future<void> installFromUrl(String url, {bool devInstall = false}) async {
    await _runBusy(manualInstallBusyKey, () async {
      final installedId = await ref
          .read(gopeedServiceProvider)
          .installExtension(InstallExtension(devMode: devInstall, url: url));
      await loadInstalled(refreshUpdates: false);
      unawaited(checkUpdate());
      if (installedId.isNotEmpty) {
        _bumpStoreInstallCount(installedId);
        _reportInstallSafe(installedId);
      }
    });
  }

  Future<void> toggleExtension(Extension extension, bool enabled) async {
    await _runBusy(extension.identity, () async {
      await ref.read(gopeedServiceProvider).switchExtension(extension.identity, SwitchExtension(status: enabled));
      await loadInstalled(refreshUpdates: false);
    });
  }

  Future<void> removeExtension(Extension extension) async {
    await _runBusy(extension.identity, () async {
      await ref.read(gopeedServiceProvider).deleteExtension(extension.identity);
      await loadInstalled(refreshUpdates: false);
      final flags = Map<String, String>.of(_current.updateFlags)..remove(extension.identity);
      state = AsyncValue.data(_current.copyWith(updateFlags: flags));
    });
  }

  Future<void> upgradeExtension(Extension extension) async {
    await _runBusy(extension.identity, () async {
      await ref.read(gopeedServiceProvider).updateExtension(extension.identity);
      await loadInstalled(refreshUpdates: false);
      unawaited(checkUpdate());
      _bumpStoreInstallCount(extension.identity);
      _reportInstallSafe(extension.identity);
    });
  }

  Future<void> saveExtensionSettings(Extension extension, Map<String, dynamic> settings) async {
    await _runBusy(extension.identity, () async {
      await ref
          .read(gopeedServiceProvider)
          .updateExtensionSettings(extension.identity, UpdateExtensionSettings(settings: settings));
      await loadInstalled(refreshUpdates: false);
    });
  }

  Future<void> checkUpdate() async {
    final flags = <String, String>{};
    for (final ext in _current.installedExtensions) {
      try {
        final resp = await ref.read(gopeedServiceProvider).upgradeCheckExtension(ext.identity);
        if (resp.newVersion.isNotEmpty) {
          flags[ext.identity] = resp.newVersion;
        }
      } catch (_) {}
    }
    state = AsyncValue.data(_current.copyWith(updateFlags: flags));
  }

  void tryOpenDevMode() {
    if (_devModeCount == 0) {
      Future.delayed(const Duration(seconds: 2), () {
        if (_current.devMode) return;
        _devModeCount = 0;
      });
    }
    _devModeCount++;
    if (_devModeCount >= 5) {
      state = AsyncValue.data(_current.copyWith(devMode: true));
    }
  }

  ExtensionsState get _current => state.value ?? const ExtensionsState();

  Future<ExtensionsState> _loadInstalled({required bool refreshUpdates, required ExtensionsState current}) async {
    state = AsyncValue.data(current.copyWith(loadingInstalled: true));
    final installed = await ref.read(gopeedServiceProvider).getExtensions();
    state = AsyncValue.data(_current.copyWith(installedExtensions: installed, loadingInstalled: false));
    if (refreshUpdates) {
      await checkUpdate();
    }
    return _current;
  }

  Future<void> _refreshStore({required ExtensionsState current}) async {
    state = AsyncValue.data(current.copyWith(loadingStore: true));
    final page = await GopeedSiteApi.instance.getExtensions(
      page: 1,
      limit: 20,
      sort: _current.storeSort,
      query: _current.storeQuery,
    );
    state = AsyncValue.data(
      _current.copyWith(storeExtensions: page.data, storePagination: page.pagination, loadingStore: false),
    );
  }

  Future<void> _runBusy(String id, Future<void> Function() action) async {
    if (_current.busyExtensionIds.contains(id)) return;
    state = AsyncValue.data(_current.copyWith(busyExtensionIds: {..._current.busyExtensionIds, id}));
    try {
      await action();
    } finally {
      final busy = Set<String>.of(_current.busyExtensionIds)..remove(id);
      state = AsyncValue.data(_current.copyWith(busyExtensionIds: busy));
    }
  }

  void _bumpStoreInstallCount(String id) {
    final current = _current;
    final index = current.storeExtensions.indexWhere((e) => e.id == id);
    if (index < 0) return;
    final ext = current.storeExtensions[index];
    final updated = [...current.storeExtensions];
    updated[index] = StoreExtension(
      id: ext.id,
      repoFullName: ext.repoFullName,
      repoUrl: ext.repoUrl,
      directory: ext.directory,
      commitSha: ext.commitSha,
      name: ext.name,
      author: ext.author,
      title: ext.title,
      description: ext.description,
      icon: ext.icon,
      version: ext.version,
      homepage: ext.homepage,
      readme: ext.readme,
      installCount: ext.installCount + 1,
      stars: ext.stars,
      topics: ext.topics,
      createdAt: ext.createdAt,
      updatedAt: ext.updatedAt,
    );
    state = AsyncValue.data(current.copyWith(storeExtensions: updated));
  }

  void _reportInstallSafe(String id) {
    unawaited(() async {
      try {
        await GopeedSiteApi.instance.reportExtensionInstall(id);
      } catch (_) {}
    }());
  }

  static int _compareVersion(String a, String b) {
    final aNums = _toVersionNumbers(a);
    final bNums = _toVersionNumbers(b);
    final maxLen = aNums.length > bNums.length ? aNums.length : bNums.length;
    for (var i = 0; i < maxLen; i++) {
      final left = i < aNums.length ? aNums[i] : 0;
      final right = i < bNums.length ? bNums[i] : 0;
      if (left > right) return 1;
      if (left < right) return -1;
    }
    return 0;
  }

  static List<int> _toVersionNumbers(String version) {
    return version
        .split(RegExp(r'[^0-9]+'))
        .where((part) => part.isNotEmpty)
        .map((part) => int.tryParse(part) ?? 0)
        .toList();
  }
}
