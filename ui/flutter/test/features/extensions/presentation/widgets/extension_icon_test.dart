import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_svg/flutter_svg.dart';
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

  testWidgets('renders an installed SVG icon', (tester) async {
    final directory = Directory.systemTemp.createTempSync('gopeed-extension-icon-');
    addTearDown(() => directory.deleteSync(recursive: true));
    File(path.join(directory.path, 'icon.svg')).writeAsStringSync(
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16">'
      '<rect width="16" height="16" fill="red"/>'
      '</svg>',
    );
    final item = ExtensionListItem(
      installed: _installed(icon: 'icon.svg', devMode: true, devPath: directory.path),
    );
    await tester.runAsync(() async {
      await tester.pumpWidget(_icon(item));
      final svg = tester.widget<SvgPicture>(find.byType(SvgPicture));
      final picture = await vg.loadPicture(svg.bytesLoader, tester.element(find.byType(SvgPicture)));
      expect(picture.size, const Size(16, 16));
      picture.picture.dispose();
    });
    await tester.pumpAndSettle();

    expect(find.descendant(of: find.byType(SvgPicture), matching: find.byType(FittedBox)), findsOneWidget);
  });

  testWidgets('uses the SVG loader for a store icon URL with a query', (tester) async {
    const icon = 'https://cdn.example.com/icon.SVG?version=1';
    await tester.pumpWidget(_icon(ExtensionListItem(store: _store(icon: icon))));

    final svg = tester.widget<SvgPicture>(find.byType(SvgPicture));
    expect(svg.bytesLoader, isA<SvgNetworkLoader>().having((loader) => loader.url, 'url', icon));
  });

  testWidgets('uses the bundled fallback when the SVG file is missing', (tester) async {
    final directory = Directory.systemTemp.createTempSync('gopeed-extension-icon-');
    addTearDown(() => directory.deleteSync(recursive: true));
    final item = ExtensionListItem(
      installed: _installed(icon: 'missing.svg', devMode: true, devPath: directory.path),
    );
    await tester.pumpWidget(_icon(item));
    await tester.pumpAndSettle();

    final fallback = tester.widget<Image>(find.byType(Image));
    expect(
      fallback.image,
      isA<AssetImage>().having((asset) => asset.assetName, 'assetName', 'assets/extension/default_icon.png'),
    );
  });
}

Widget _icon(ExtensionListItem item) => Directionality(
  textDirection: TextDirection.ltr,
  child: ExtensionIcon(item: item, size: 40, borderRadius: BorderRadius.zero),
);

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

StoreExtension _store({String icon = 'https://cdn.example.com/icon.png'}) {
  return StoreExtension(
    id: 'gopeed@test',
    repoFullName: 'GopeedLab/test',
    repoUrl: 'https://github.com/GopeedLab/test',
    name: 'test',
    author: 'gopeed',
    title: 'Test',
    description: 'Test extension',
    icon: icon,
    version: '1.0.0',
    homepage: null,
    readme: null,
    installCount: 1,
    stars: 1,
    topics: const [],
  );
}
