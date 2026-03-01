import 'dart:async';

import 'package:get/get.dart';
import 'package:gopeed/api/api.dart';
import 'package:gopeed/util/gopeed_api.dart';

import '../../../../api/model/extension.dart';
import '../../../../api/model/install_extension.dart';
import '../../../../util/message.dart';

enum ExtensionFilter { all, installed, notInstalled }

class ExtensionController extends GetxController {
  // ---------------------------------------------------------------------------
  // Installed extensions state
  // ---------------------------------------------------------------------------
  final extensions = <Extension>[].obs;
  final updateFlags = <String, String>{}.obs;
  final devMode = false.obs;
  var _devModeCount = 0;

  /// Flag to prevent handling pending install multiple times
  bool pendingInstallHandled = false;

  // ---------------------------------------------------------------------------
  // Store state
  // ---------------------------------------------------------------------------
  final storeExtensions = <GopeedExtension>[].obs;
  final storeLoading = false.obs;
  final storeHasNext = false.obs;
  final storeSortField = GopeedExtensionSortField.stars.obs;
  final storeSortOrder = GopeedExtensionSortOrder.desc.obs;
  final storeQuery = ''.obs;

  /// Per-extension installation loading state (key = store extension id).
  final storeInstalling = <String, bool>{}.obs;

  /// Current list filter.
  final extensionFilter = ExtensionFilter.all.obs;

  int _storePage = 1;
  Timer? _searchDebounce;

  // ---------------------------------------------------------------------------
  // Lifecycle
  // ---------------------------------------------------------------------------

  @override
  void onInit() async {
    super.onInit();
    await load();
    checkUpdate();
    loadStore();
  }

  @override
  void onClose() {
    _searchDebounce?.cancel();
    super.onClose();
  }

  // ---------------------------------------------------------------------------
  // Installed extension methods
  // ---------------------------------------------------------------------------

  Future<void> load() async {
    extensions.value = await getExtensions();
  }

  Future<void> checkUpdate() async {
    for (final ext in extensions) {
      final resp = await upgradeCheckExtension(ext.identity);
      if (resp.newVersion.isNotEmpty) {
        updateFlags[ext.identity] = resp.newVersion;
      }
    }
  }

  /// Try to open dev mode when install button is clicked 5 times in 2 seconds.
  void tryOpenDevMode() {
    if (_devModeCount == 0) {
      Future.delayed(const Duration(seconds: 2), () {
        if (devMode.value) return;
        devMode.value = false;
        _devModeCount = 0;
      });
    }
    _devModeCount++;
    if (_devModeCount >= 5) {
      if (!devMode.value) {
        devMode.value = true;
        showMessage('tip'.tr, 'Developer mode activated');
      }
    }
  }

  // ---------------------------------------------------------------------------
  // Store methods
  // ---------------------------------------------------------------------------

  /// Load one page of store extensions. Pass [reset] = true to start fresh.
  Future<void> loadStore({bool reset = false}) async {
    if (storeLoading.value) return;
    if (reset) {
      _storePage = 1;
      storeExtensions.clear();
    }
    storeLoading.value = true;
    try {
      final result = await gopeedSearchExtensions(
        page: _storePage,
        sort: storeSortField.value,
        order: storeSortOrder.value,
        q: storeQuery.value.isEmpty ? null : storeQuery.value,
      );
      storeExtensions.addAll(result.data);
      storeHasNext.value = result.pagination.hasNext;
      _storePage++;
    } catch (_) {
      // Best-effort — silently ignore network failures.
    } finally {
      storeLoading.value = false;
    }
  }

  /// Called when the search query changes (with debounce).
  void onStoreQueryChanged(String q) {
    _searchDebounce?.cancel();
    storeQuery.value = q;
    _searchDebounce = Timer(const Duration(milliseconds: 400), () {
      loadStore(reset: true);
    });
  }

  /// Called when a sort field chip is tapped.
  /// Tapping the active field toggles the direction; tapping another resets to desc.
  void onStoreSortChanged(GopeedExtensionSortField field) {
    if (storeSortField.value == field) {
      storeSortOrder.value =
          storeSortOrder.value == GopeedExtensionSortOrder.desc
              ? GopeedExtensionSortOrder.asc
              : GopeedExtensionSortOrder.desc;
    } else {
      storeSortField.value = field;
      storeSortOrder.value = GopeedExtensionSortOrder.desc;
    }
    loadStore(reset: true);
  }

  Future<void> loadMoreStore() => loadStore();

  /// Install a store extension, track loading state reactively.
  Future<void> installFromStore(GopeedExtension ext) async {
    if (ext.repoUrl.isEmpty) return;
    storeInstalling[ext.id] = true;
    try {
      final identity =
          await installExtension(InstallExtension(url: ext.repoUrl));
      gopeedReportExtensionInstall(identity);
      await load();
      showMessage('tip'.tr, 'extensionInstallSuccess'.tr);
    } catch (e) {
      showErrorMessage(e);
    } finally {
      storeInstalling.remove(ext.id);
    }
  }
}
