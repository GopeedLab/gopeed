import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/features/extensions/application/extensions_controller.dart';
import 'package:gopeed/features/extensions/presentation/pages/extensions_page.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_empty_state.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  for (final width in [390.0, 768.0, 1024.0]) {
    testWidgets('empty extension market uses the shared state at width $width', (tester) async {
      await _pumpPage(tester, width, const ExtensionsState());
      expect(find.byType(AppEmptyState), findsOneWidget);
      expect(find.byKey(const ValueKey('extensions-empty-illustration')), findsOneWidget);
      expect(find.text('No extensions'), findsOneWidget);
      expect(find.byKey(const ValueKey('extensions-initial-skeleton')), findsNothing);
      expect(tester.takeException(), isNull);
    });

    testWidgets('empty installed list matches the market and can switch tabs at width $width', (tester) async {
      await _pumpPage(tester, width, const ExtensionsState(listFilter: ExtensionListFilter.installed));
      expect(find.text('No extensions'), findsOneWidget);
      expect(find.byKey(const ValueKey('extensions-empty-browse')), findsNothing);
      await tester.tap(find.text('Market'));
      await tester.pumpAndSettle();
      final container = ProviderScope.containerOf(tester.element(find.byType(ExtensionsPage)));
      expect(container.read(extensionsControllerProvider).value!.listFilter, ExtensionListFilter.market);
      expect(find.text('No extensions'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });

    testWidgets('empty search explains no matches at width $width', (tester) async {
      await _pumpPage(tester, width, const ExtensionsState(storeQuery: 'no-match'));
      expect(find.byKey(const ValueKey('extensions-empty-illustration')), findsOneWidget);
      expect(find.text('No matching extensions found'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });

    testWidgets('extension load failure can retry from the shared state at width $width', (tester) async {
      final controller = _FailedExtensionsController();
      await _pumpPage(tester, width, const ExtensionsState(), controller: controller);
      expect(find.byKey(const ValueKey('extensions-error-illustration')), findsOneWidget);
      expect(find.text('Unable to load extensions'), findsOneWidget);
      expect(find.textContaining('test-network-error'), findsNothing);
      await tester.tap(find.byKey(const ValueKey('extensions-error-retry')));
      await tester.pumpAndSettle();
      expect(controller.retryCalls, 1);
      expect(find.byKey(const ValueKey('extensions-error-state')), findsNothing);
      expect(find.text('No extensions'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
}

Future<void> _pumpPage(
  WidgetTester tester,
  double width,
  ExtensionsState initial, {
  ExtensionsController? controller,
}) async {
  tester.view.physicalSize = Size(width, 768);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [extensionsControllerProvider.overrideWith(() => controller ?? _EmptyExtensionsController(initial))],
      child: shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: const ExtensionsPage(),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

class _EmptyExtensionsController extends ExtensionsController {
  _EmptyExtensionsController(this.initial);

  final ExtensionsState initial;

  @override
  Future<ExtensionsState> build() async => initial;
}

class _FailedExtensionsController extends ExtensionsController {
  int retryCalls = 0;

  @override
  Future<ExtensionsState> build() async => throw StateError('test-network-error');

  @override
  Future<void> loadInitialData() async {
    retryCalls++;
    state = const AsyncValue.data(ExtensionsState());
  }
}
