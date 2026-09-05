import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/extension.dart';
import 'package:gopeed/api/model/store_extension.dart';
import 'package:gopeed/features/extensions/application/extensions_controller.dart';
import 'package:gopeed/features/extensions/presentation/widgets/extension_icon.dart';
import 'package:path/path.dart' as path;

void main() {
  test('installed extension uses its local icon instead of the store icon', () {
    final item = ExtensionListItem(installed: _installed(), store: _store());

    final source = resolveExtensionIconSource(item, web: false, storageDirectory: '/app-data');

    expect(source.kind, ExtensionIconSourceKind.file);
    expect(source.location, path.join('/app-data', 'extensions', 'gopeed@test', 'images', 'icon.png'));
  });

  test('development extension uses its development directory', () {
    final item = ExtensionListItem(
      installed: _installed(devMode: true, devPath: '/extension-source'),
      store: _store(),
    );

    final source = resolveExtensionIconSource(item, web: false, storageDirectory: '/app-data');

    expect(source.kind, ExtensionIconSourceKind.file);
    expect(source.location, path.join('/extension-source', 'images', 'icon.png'));
  });

  test('web uses the local Gopeed extension file endpoint', () {
    final item = ExtensionListItem(installed: _installed(), store: _store());

    final source = resolveExtensionIconSource(item, web: true);

    expect(source.kind, ExtensionIconSourceKind.network);
    expect(source.location, '/fs/extensions/gopeed@test/images/icon.png');
  });

  test('uninstalled market extension keeps using its remote icon', () {
    final source = resolveExtensionIconSource(ExtensionListItem(store: _store()), web: false);

    expect(source.kind, ExtensionIconSourceKind.network);
    expect(source.location, 'https://cdn.example.com/icon.png');
  });

  test('installed extension without an icon uses the bundled fallback', () {
    final source = resolveExtensionIconSource(
      ExtensionListItem(
        installed: _installed(icon: ''),
        store: _store(),
      ),
      web: false,
      storageDirectory: '/app-data',
    );

    expect(source.kind, ExtensionIconSourceKind.asset);
    expect(source.location, 'assets/extension/default_icon.png');
  });
}

Extension _installed({String icon = 'images/icon.png', bool devMode = false, String devPath = ''}) {
  return Extension(
    identity: 'gopeed@test',
    name: 'test',
    author: 'gopeed',
    title: 'Test',
    description: 'Test extension',
    icon: icon,
    version: '1.0.0',
    homepage: '',
    repository: Repository(url: 'https://github.com/GopeedLab/test', directory: ''),
    disabled: false,
    devMode: devMode,
    devPath: devPath,
  );
}

StoreExtension _store() {
  return StoreExtension(
    id: 'gopeed@test',
    repoFullName: 'GopeedLab/test',
    repoUrl: 'https://github.com/GopeedLab/test',
    name: 'test',
    author: 'gopeed',
    title: 'Test',
    description: 'Test extension',
    icon: 'https://cdn.example.com/icon.png',
    version: '1.0.0',
    homepage: null,
    readme: null,
    installCount: 1,
    stars: 1,
    topics: const [],
  );
}
