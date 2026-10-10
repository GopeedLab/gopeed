import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/gopeed_site_api.dart';
import 'package:gopeed/api/model/extension.dart';
import 'package:gopeed/api/model/store_extension.dart';
import 'package:gopeed/api/model/update_check_extension_resp.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/features/extensions/application/extensions_controller.dart';
import 'package:gopeed/features/extensions/application/store_extension_details_provider.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('store loads while installed data and update checks are still pending', () async {
    final installed = Completer<List<Extension>>();
    final update = Completer<UpdateCheckExtensionResp>();
    var updateCalls = 0;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getExtensions, (_) => installed.future)
      ..bind(GopeedMethods.checkExtensionUpdate, (_) {
        updateCalls++;
        return update.future;
      });
    final site = _SiteApi();
    final container = _container(registry, site);
    final subscription = container.listen(extensionsControllerProvider, (_, _) {});
    addTearDown(subscription.close);

    await _settle();
    expect(site.listCalls, 1);
    expect(container.read(extensionsControllerProvider).value?.storeExtensions.single.id, 'market');
    expect(container.read(extensionsControllerProvider).value?.loadingStore, isFalse);
    expect(updateCalls, 0);

    installed.complete([_installed()]);
    await _settle();
    expect(updateCalls, 1);
    expect(container.read(extensionsControllerProvider).value?.loadingInstalled, isFalse);
    expect(container.read(extensionsControllerProvider).value?.storeExtensions.single.id, 'market');
    update.complete(UpdateCheckExtensionResp(newVersion: '2.0.0'));
    await _settle();
    expect(container.read(extensionsControllerProvider).value?.updateFlags, {'gopeed@test': '2.0.0'});
  });

  test('a stale update scan cannot restore flags after installed data changes', () async {
    var installed = [_installed()];
    final update = Completer<UpdateCheckExtensionResp>();
    var updateCalls = 0;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getExtensions, (_) => installed)
      ..bind(GopeedMethods.checkExtensionUpdate, (_) {
        updateCalls++;
        return update.future;
      });
    final container = _container(registry, _SiteApi());
    final controller = container.read(extensionsControllerProvider.notifier);
    await _settle();
    final running = controller.checkUpdate();
    expect(updateCalls, 1);
    installed = [];
    await controller.loadInstalled();
    update.complete(UpdateCheckExtensionResp(newVersion: '2.0.0'));
    await running;
    expect(container.read(extensionsControllerProvider).value?.installedExtensions, isEmpty);
    expect(container.read(extensionsControllerProvider).value?.updateFlags, isEmpty);
  });

  test('an update scan can finish after the controller is disposed', () async {
    final update = Completer<UpdateCheckExtensionResp>();
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getExtensions, (_) => [_installed()])
      ..bind(GopeedMethods.checkExtensionUpdate, (_) => update.future);
    final container = _container(registry, _SiteApi());
    final controller = container.read(extensionsControllerProvider.notifier);
    await _settle();
    final running = controller.checkUpdate();
    container.dispose();
    update.complete(UpdateCheckExtensionResp(newVersion: '2.0.0'));
    await running;
  });

  test('old search results cannot replace a newer search', () async {
    final old = Completer<StoreExtensionPage>();
    final latest = Completer<StoreExtensionPage>();
    final site = _SiteApi()
      ..onList = (query) => switch (query) {
        'old' => old.future,
        'new' => latest.future,
        _ => Future.value(_page('market')),
      };
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getExtensions, (_) => <Extension>[]);
    final container = _container(registry, site);
    final controller = container.read(extensionsControllerProvider.notifier);
    await _settle();
    final first = controller.searchStore('old');
    final second = controller.searchStore('new');
    latest.complete(_page('new-result'));
    await second;
    old.complete(_page('old-result'));
    await first;
    final state = container.read(extensionsControllerProvider).value!;
    expect(state.storeQuery, 'new');
    expect(state.storeExtensions.single.id, 'new-result');
    expect(state.loadingStore, isFalse);
  });

  test('overlapping cached pages do not duplicate an extension', () async {
    final site = _SiteApi();
    site.onList = (_) async => StoreExtensionPage.fromJson({
      'data': [
        {'id': 'market', 'version': '1.0.0'},
        if (site.listCalls > 1) {'id': 'second', 'version': '1.0.0'},
      ],
      'pagination': {'page': site.listCalls, 'limit': 20, 'hasNext': site.listCalls == 1},
    });
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getExtensions, (_) => <Extension>[]);
    final container = _container(registry, site);
    final controller = container.read(extensionsControllerProvider.notifier);
    await _settle();
    await controller.loadMoreStore();
    expect(container.read(extensionsControllerProvider).value?.storeExtensions.map((e) => e.id), ['market', 'second']);
  });

  test('detail requests are shared, cache empty READMEs, and distinguish versions', () async {
    final detail = Completer<StoreExtension>();
    final site = _SiteApi()..onDetail = (_, _) => detail.future;
    final container = ProviderContainer(overrides: [gopeedSiteApiProvider.overrideWithValue(site)]);
    addTearDown(container.dispose);
    const key = (id: 'market', version: '1.0.0');
    final provider = storeExtensionDetailsProvider(key);
    final subscription = container.listen(provider, (_, _) {});
    final first = container.read(provider.future);
    final second = container.read(provider.future);
    expect(site.detailCalls, 1);
    detail.complete(StoreExtension.fromJson({'id': 'market', 'version': '1.0.0', 'readme': null}));
    expect((await first).hasDetails, isTrue);
    await second;
    subscription.close();
    await _settle();
    await container.read(provider.future);
    expect(site.detailCalls, 1);
    await container.read(storeExtensionDetailsProvider((id: 'market', version: '2.0.0')).future);
    expect(site.detailCalls, 2);
  });

  test('failed details can be retried', () async {
    final site = _SiteApi()..onDetail = (_, _) => Future.error(StateError('offline'));
    final container = ProviderContainer(overrides: [gopeedSiteApiProvider.overrideWithValue(site)]);
    addTearDown(container.dispose);
    final provider = storeExtensionDetailsProvider((id: 'market', version: '1.0.0'));
    final subscription = container.listen(provider, (_, _) {});
    addTearDown(subscription.close);
    await expectLater(container.read(provider.future), throwsStateError);
    site.onDetail = (_, _) async => StoreExtension.fromJson({'id': 'market', 'readme': 'loaded'});
    container.invalidate(provider);
    expect((await container.read(provider.future)).readme, 'loaded');
    expect(site.detailCalls, 2);
  });
}

ProviderContainer _container(CapabilityRegistry registry, _SiteApi site) {
  final container = ProviderContainer(
    overrides: [
      appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry))),
      gopeedSiteApiProvider.overrideWithValue(site),
    ],
  );
  addTearDown(container.dispose);
  return container;
}

Future<void> _settle() async {
  for (var i = 0; i < 5; i++) {
    await Future<void>.delayed(Duration.zero);
  }
}

StoreExtensionPage _page(String id) => StoreExtensionPage.fromJson({
  'data': [
    {'id': id, 'version': '1.0.0'},
  ],
  'pagination': {'page': 1, 'limit': 20, 'total': 1, 'totalPages': 1},
});

Extension _installed() => Extension(
  identity: 'gopeed@test',
  name: 'test',
  author: 'gopeed',
  title: 'Test',
  description: '',
  icon: '',
  version: '1.0.0',
  homepage: '',
  repository: null,
  disabled: false,
  devMode: false,
  devPath: '',
);

class _SiteApi implements GopeedSiteApi {
  var listCalls = 0;
  var detailCalls = 0;
  Future<StoreExtensionPage> Function(String?) onList = (_) async => _page('market');
  Future<StoreExtension> Function(String, String?) onDetail = (id, _) async =>
      StoreExtension.fromJson({'id': id, 'readme': null});

  @override
  Future<StoreExtensionPage> getExtensions({
    int page = 1,
    int limit = 20,
    StoreExtensionSort sort = StoreExtensionSort.stars,
    StoreSortOrder order = StoreSortOrder.desc,
    String? query,
    bool forceRefresh = false,
  }) {
    listCalls++;
    return onList(query);
  }

  @override
  Future<StoreExtension> getExtension(String id, {String? version}) {
    detailCalls++;
    return onDetail(id, version);
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
