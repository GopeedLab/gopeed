import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/gopeed_site_api.dart';
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
      final pending = Completer<StoreExtension>();
      final site = _DetailApi()..request = pending.future;
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
      pending.completeError(StateError('offline'));
      await tester.pump();
      await tester.pump();
      expect(find.byKey(const ValueKey('extension-details-retry')), findsOneWidget);
      site.request = Future.value(
        StoreExtension.fromJson({
          'id': 'market',
          'title': 'Market',
          'version': '1.0.0',
          'readme': '# Loaded details',
          'homepage': 'https://example.com',
        }),
      );
      await tester.tap(find.byKey(const ValueKey('extension-details-retry')));
      await tester.pumpAndSettle();
      expect(find.text('Loaded details'), findsOneWidget);
      expect(find.byKey(const ValueKey('extension-details-homepage')), findsOneWidget);
      expect(find.byKey(const ValueKey('extension-details-retry')), findsNothing);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
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

  @override
  Future<StoreExtension> getExtension(String id, {String? version}) => request;

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
