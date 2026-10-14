import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/gopeed_site_api.dart';
import 'package:gopeed/api/model/extension.dart';
import 'package:gopeed/api/model/store_extension.dart';
import 'package:gopeed/features/extensions/application/extensions_controller.dart';
import 'package:gopeed/features/extensions/application/store_extension_details_provider.dart';
import 'package:gopeed/features/extensions/presentation/widgets/extension_detail_view.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  for (final width in [390.0, 1024.0]) {
    testWidgets('summary details load and can retry at width $width', (tester) async {
      tester.view.physicalSize = Size(width, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final attempts = List.generate(3, (_) => Completer<StoreExtension>());
      final site = _DetailApi();
      site.onRequest = () => attempts[site.detailCalls - 1].future;
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            extensionsControllerProvider.overrideWith(_SummaryController.new),
            gopeedSiteApiProvider.overrideWithValue(site),
          ],
          child: shad.ShadcnApp(
            theme: AppTheme.light(),
            materialTheme: AppTheme.materialLight(),
            home: shad.Scaffold(
              child: ExtensionDetailView(
                item: ExtensionListItem(store: _summary()),
                mobile: width < 600,
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      expect(find.byKey(const ValueKey('extension-details-loading')), findsOneWidget);
      expect(find.byKey(const ValueKey('extension-details-install')), findsOneWidget);
      expect(site.detailCalls, 1);
      for (var attempt = 0; attempt < 2; attempt++) {
        attempts[attempt].completeError(StateError('offline'));
        await tester.pump();
        expect(find.byKey(const ValueKey('extension-details-loading')), findsOneWidget);
        expect(find.byKey(const ValueKey('extension-details-retry')), findsNothing);
        await tester.pump(Duration(milliseconds: 300 * (attempt + 1)));
        expect(site.detailCalls, attempt + 2);
      }
      attempts.last.completeError(StateError('offline'));
      await tester.pump();
      await tester.pump();
      expect(find.byKey(const ValueKey('extension-details-retry')), findsOneWidget);
      expect(find.byKey(const ValueKey('extension-details-loading')), findsNothing);
      expect(find.byIcon(Icons.refresh_rounded), findsOneWidget);
      expect(find.byKey(const ValueKey('extension-details-error-illustration')), findsOneWidget);
      expect(find.text('Unable to load extension details'), findsOneWidget);
      expect(find.byKey(const ValueKey('extension-details-install')), findsOneWidget);
      await tester.pump(const Duration(seconds: 5));
      expect(site.detailCalls, 3);
      final reload = Completer<StoreExtension>();
      site.onRequest = () => reload.future;
      await tester.tap(find.byKey(const ValueKey('extension-details-retry')));
      await tester.pump();
      expect(site.detailCalls, 4);
      expect(find.byKey(const ValueKey('extension-details-loading')), findsOneWidget);
      expect(find.byKey(const ValueKey('extension-details-retry')), findsNothing);
      reload.complete(
        StoreExtension.fromJson({
          'id': 'market',
          'title': 'Market',
          'version': '1.0.0',
          'readme': '# Loaded details',
          'homepage': 'https://example.com',
        }),
      );
      await tester.pumpAndSettle();
      expect(find.text('Loaded details'), findsOneWidget);
      expect(find.byKey(const ValueKey('extension-details-homepage')), findsOneWidget);
      expect(find.byKey(const ValueKey('extension-details-retry')), findsNothing);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
    });

    testWidgets('installed summary uses local details without a remote request at width $width', (tester) async {
      tester.view.physicalSize = Size(width, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final directory = Directory.systemTemp.createTempSync('gopeed-installed-detail-');
      addTearDown(() => directory.deleteSync(recursive: true));
      File('${directory.path}/README.md').writeAsStringSync('# Installed documentation');
      final installed = Extension(
        identity: 'market',
        name: 'market',
        author: 'Local author',
        title: 'Local extension',
        description: 'Installed manifest',
        icon: '',
        version: '1.0.0',
        homepage: 'https://example.com/local',
        repository: Repository(url: 'https://github.com/test/local', directory: ''),
        disabled: false,
        devMode: true,
        devPath: directory.path,
      );
      final site = _DetailApi()..request = Completer<StoreExtension>().future;
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            extensionsControllerProvider.overrideWith(() => _InstalledSummaryController(installed)),
            gopeedSiteApiProvider.overrideWithValue(site),
          ],
          child: shad.ShadcnApp(
            theme: AppTheme.light(),
            materialTheme: AppTheme.materialLight(),
            home: shad.Scaffold(
              child: ExtensionDetailView(
                item: ExtensionListItem(installed: installed, store: _summary()),
                mobile: width < 600,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(site.detailCalls, 0);
      expect(find.byKey(const ValueKey('extension-details-loading')), findsNothing);
      expect(find.byKey(const ValueKey('extension-details-retry')), findsNothing);
      expect(find.text('Installed documentation'), findsOneWidget);
      expect(find.byKey(const ValueKey('extension-details-homepage')), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    });
  }
}

StoreExtension _summary() => StoreExtension.fromJson({
  'id': 'market',
  'title': 'Market',
  'version': '1.0.0',
  'repoUrl': 'https://github.com/test/repo',
});

class _SummaryController extends ExtensionsController {
  @override
  Future<ExtensionsState> build() async => ExtensionsState(storeExtensions: [_summary()]);
}

class _DetailApi implements GopeedSiteApi {
  late Future<StoreExtension> request;
  Future<StoreExtension> Function()? onRequest;
  int detailCalls = 0;

  @override
  Future<StoreExtension> getExtension(String id, {String? version}) {
    detailCalls++;
    return onRequest?.call() ?? request;
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class _InstalledSummaryController extends ExtensionsController {
  _InstalledSummaryController(this.installed);

  final Extension installed;

  @override
  Future<ExtensionsState> build() async =>
      ExtensionsState(installedExtensions: [installed], storeExtensions: [_summary()]);
}
