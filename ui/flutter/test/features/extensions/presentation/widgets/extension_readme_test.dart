import 'dart:async';
import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/extension.dart';
import 'package:gopeed/api/model/store_extension.dart';
import 'package:gopeed/features/extensions/application/extensions_controller.dart';
import 'package:gopeed/features/extensions/presentation/pages/extension_details_page.dart';
import 'package:gopeed/features/extensions/presentation/widgets/extension_detail_view.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:path/path.dart' as path;
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  for (final mobile in [false, true]) {
    testWidgets('local README renders while the market request is pending (mobile: $mobile)', (tester) async {
      final directory = Directory.systemTemp.createTempSync('gopeed-readme-');
      addTearDown(() => directory.deleteSync(recursive: true));
      File(path.join(directory.path, 'README.md')).writeAsStringSync('# Local README');
      final installed = _installed(directory.path);
      final controller = _ReadmeController(
        ExtensionsState(installedExtensions: [installed], loadingStore: true),
        pendingMarket: true,
      );
      await _pumpDetails(tester, controller, ExtensionListItem(installed: installed), mobile: mobile);

      expect(controller.marketRequest.isCompleted, isFalse);
      expect(find.text('Local README'), findsOneWidget);
      expect(tester.takeException(), isNull);

      // A stale market response must not replace the installed documentation.
      controller.finishMarket();
      await tester.pumpAndSettle();
      expect(find.text('Local README'), findsOneWidget);
      expect(find.text('Old market README'), findsNothing);
    });

    for (final mutate in [false, true]) {
      testWidgets('open README reloads after an upgrade (mobile: $mobile, mutate: $mutate)', (tester) async {
        final directory = Directory.systemTemp.createTempSync('gopeed-readme-');
        addTearDown(() => directory.deleteSync(recursive: true));
        final readme = File(path.join(directory.path, 'README.md'))..writeAsStringSync('# Before upgrade');
        final installed = _installed(directory.path);
        final controller = _ReadmeController(
          ExtensionsState(installedExtensions: [installed], storeExtensions: [_store()]),
        );
        await _pumpDetails(
          tester,
          controller,
          ExtensionListItem(installed: installed, store: _store()),
          mobile: mobile,
        );
        expect(find.text('Before upgrade'), findsOneWidget);

        readme.writeAsStringSync('# After upgrade');
        controller.publishVersion('2.0.0', mutate: mutate);
        await tester.pumpAndSettle();

        expect(find.text('After upgrade'), findsOneWidget);
        expect(find.text('Before upgrade'), findsNothing);
        expect(find.text('Old market README'), findsNothing);
        expect(tester.takeException(), isNull);
      });
    }
  }

  testWidgets('missing local README falls back to the store copy', (tester) async {
    final directory = Directory.systemTemp.createTempSync('gopeed-readme-');
    addTearDown(() => directory.deleteSync(recursive: true));
    final installed = _installed(directory.path);
    final controller = _ReadmeController(
      ExtensionsState(installedExtensions: [installed], storeExtensions: [_store()]),
    );
    await _pumpDetails(tester, controller, ExtensionListItem(installed: installed, store: _store()), mobile: false);
    expect(find.text('Old market README'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}

Future<void> _pumpDetails(
  WidgetTester tester,
  _ReadmeController controller,
  ExtensionListItem item, {
  required bool mobile,
}) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = Size(mobile ? 390 : 1100, 900);
  addTearDown(tester.view.resetDevicePixelRatio);
  addTearDown(tester.view.resetPhysicalSize);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [extensionsControllerProvider.overrideWith(() => controller)],
      child: shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: mobile
            ? ExtensionDetailsPage(extensionId: item.id, initialItem: item)
            : ExtensionDetailDrawer(item: item, onClose: () {}),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

class _ReadmeController extends ExtensionsController {
  _ReadmeController(this.initial, {this.pendingMarket = false});

  final ExtensionsState initial;
  final bool pendingMarket;
  final marketRequest = Completer<ExtensionsState>();

  @override
  Future<ExtensionsState> build() async {
    if (!pendingMarket) return initial;
    state = AsyncValue.data(initial);
    return marketRequest.future;
  }

  void finishMarket() {
    marketRequest.complete(state.requireValue.copyWith(storeExtensions: [_store()], loadingStore: false));
  }

  void publishVersion(String version, {required bool mutate}) {
    final current = state.requireValue;
    final installed = current.installedExtensions.single;
    final updated = mutate ? installed : Extension.fromJson(installed.toJson());
    updated.version = version;
    state = AsyncValue.data(current.copyWith(installedExtensions: [updated]));
  }
}

Extension _installed(String directory) => Extension(
  identity: 'gopeed@test',
  name: 'test',
  author: 'Gopeed',
  title: 'Test extension',
  description: 'Installed extension',
  icon: '',
  version: '1.0.0',
  homepage: '',
  repository: null,
  disabled: false,
  devMode: true,
  devPath: directory,
);

StoreExtension _store() => StoreExtension(
  id: 'gopeed@test',
  repoFullName: 'GopeedLab/test',
  repoUrl: 'https://github.com/GopeedLab/test',
  name: 'test',
  author: 'Gopeed',
  title: 'Test extension',
  description: 'Store extension',
  version: '1.0.0',
  readme: '# Old market README',
  installCount: 1,
  stars: 1,
  topics: const [],
);
