import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:gopeed/api/model/resolve_result.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/core/capabilities/storage_capability.dart';
import 'package:gopeed/features/tasks/presentation/pages/create_task_window_page.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  testWidgets('correcting a failed URL retains already-created sources', (tester) async {
    tester.view.physicalSize = const Size(900, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final resolved = <String>[];
    final created = <String>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit())
      ..bind(GopeedMethods.resolve, (request) {
        final name = Uri.parse(request.req!.url).pathSegments.last;
        resolved.add(name);
        if (name == 'bad') throw StateError('invalid URL');
        return ResolveResult(
          id: '$name-${resolved.length}',
          res: Resource(files: []),
        );
      })
      ..bind(GopeedMethods.createTask, (task) {
        created.add(task.rid!);
        return 'created-${created.length}';
      });
    final router = GoRouter(
      initialLocation: '/create',
      routes: [
        GoRoute(path: '/', builder: (_, _) => const SizedBox()),
        GoRoute(path: '/create', builder: (_, _) => const CreateTaskWindowPage()),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry)))],
        child: shad.ShadcnApp.router(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          routerConfig: router,
          builder: (_, child) => AppComponentThemes(child: child!),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final url = find.byKey(const ValueKey('create-task-url-input'));
    await tester.enterText(url, 'https://example.com/good\nhttps://example.com/bad');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(created, ['good-1']);
    await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
    await tester.pumpAndSettle();

    await tester.enterText(url, 'https://example.com/good\nhttps://example.com/fixed');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(resolved.where((item) => item == 'good'), hasLength(1));
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pumpAndSettle();
    expect(created.where((id) => id.startsWith('good-')), hasLength(1));
    expect(created, ['good-1', 'fixed-3']);
    expect(tester.takeException(), isNull);
  });
}
