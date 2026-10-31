import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:gopeed/api/model/create_task_batch.dart';
import 'package:gopeed/api/model/create_task.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:gopeed/api/model/resolve_result.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/core/capabilities/storage_capability.dart';
import 'package:gopeed/features/tasks/presentation/pages/create_task_window_page.dart';
import 'package:gopeed/features/tasks/application/pending_create_task.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_http_headers_editor.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  testWidgets('cancel and confirm again resolves every unsubmitted URL afresh', (tester) async {
    tester.view.physicalSize = const Size(900, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final resolved = <String>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
      ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit())
      ..bind(GopeedMethods.resolve, (request) {
        resolved.add(request.req!.url);
        return ResolveResult(
          id: 'rid-${resolved.length}',
          res: Resource(files: [FileInfo(name: 'file')]),
        );
      });
    await tester.pumpWidget(
      ProviderScope(
        overrides: [appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry)))],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: AppComponentThemes(child: const CreateTaskWindowPage()),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final url = find.byKey(const ValueKey('create-task-url-input'));
    await tester.enterText(url, 'https://example.com/a\nhttps://example.com/b');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(resolved, ['https://example.com/a', 'https://example.com/b']);
    await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(resolved, [
      'https://example.com/a',
      'https://example.com/b',
      'https://example.com/a',
      'https://example.com/b',
    ]);
    await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });

  for (final failGroup in [true, false]) {
    testWidgets('reopening a preview retries only the failed batch (group=$failGroup)', (tester) async {
      final resolutions = <String>[];
      final batches = <CreateTaskBatch>[];
      var fail = true;
      final registry = CapabilityRegistry(createAppCapabilityCodecs())
        ..bind(GopeedMethods.resolve, (request) {
          resolutions.add(request.req!.url);
          return _resolved(request.req!.url);
        })
        ..bind(GopeedMethods.createTaskBatch, (batch) {
          batches.add(batch);
          if (batch.group == failGroup && fail) throw StateError('batch failed');
          return CreateTaskBatchResult(taskIds: ['created']);
        });
      await _openPage(tester, registry);
      await _openPreview(tester, urls: 'https://example.com/group\nhttps://example.com/other');
      await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      expect(batches, hasLength(2));
      await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
      await tester.pumpAndSettle();
      fail = false;
      await _openPreview(tester);
      final failedUrl = 'https://example.com/${failGroup ? 'group' : 'other'}';
      expect(resolutions, ['https://example.com/group', 'https://example.com/other', failedUrl]);
      await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      expect(batches, hasLength(3));
      expect(batches.where((batch) => batch.group != failGroup), hasLength(1));
      expect(find.byType(CreateTaskWindowPage), findsNothing);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('completion survives reordering and distinguishes duplicate source occurrences', (tester) async {
    final resolutions = <String>[];
    var groupCalls = 0;
    var fail = true;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.resolve, (request) {
        resolutions.add(request.req!.url);
        return _resolved(request.req!.url);
      })
      ..bind(GopeedMethods.createTaskBatch, (batch) {
        if (batch.group) {
          groupCalls++;
          if (groupCalls == 2 && fail) throw StateError('second occurrence failed');
        }
        return CreateTaskBatchResult(taskIds: ['created']);
      });
    await _openPage(tester, registry);
    await _openPreview(tester, urls: 'https://example.com/group\nhttps://example.com/other\nhttps://example.com/group');
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
    await tester.pumpAndSettle();
    fail = false;
    await _openPreview(tester, urls: 'https://example.com/other\nhttps://example.com/group\nhttps://example.com/group');
    expect(resolutions, [
      'https://example.com/group',
      'https://example.com/other',
      'https://example.com/group',
      'https://example.com/group',
    ]);
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(groupCalls, 3);
    expect(find.byType(CreateTaskWindowPage), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('unfinished sources resolve with edited options without recreating a completed group', (tester) async {
    final resolutions = <Request>[];
    final batches = <CreateTaskBatch>[];
    var fail = true;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.resolve, (request) {
        resolutions.add(request.req!);
        return _resolved(request.req!.url);
      })
      ..bind(GopeedMethods.createTaskBatch, (batch) {
        batches.add(batch);
        if (!batch.group && fail) throw StateError('batch failed');
        return CreateTaskBatchResult(taskIds: ['created']);
      });
    await _openPage(tester, registry);
    await _openPreview(tester, urls: 'https://example.com/group\nhttps://example.com/other');
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();
    tester.widget<AppHttpHeadersEditor>(find.byType(AppHttpHeadersEditor)).controller.replace({'Cookie': 'new-cookie'});
    await tester.enterText(find.byKey(const ValueKey('create-task-directory-input')), '/retry');
    fail = false;
    await _openPreview(tester);
    expect(resolutions.map((request) => request.url), [
      'https://example.com/group',
      'https://example.com/other',
      'https://example.com/other',
    ]);
    expect(ReqExtraHttp.fromJson(resolutions.last.extra! as Map<String, dynamic>).header['Cookie'], 'new-cookie');
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(batches.where((batch) => batch.group), hasLength(1));
    expect(batches.last.reqs!.single.opts!.path, '/retry');
    expect(tester.takeException(), isNull);
  });

  testWidgets('a new creation window can submit the same source again', (tester) async {
    var resolutions = 0;
    var submissions = 0;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.resolve, (request) {
        resolutions++;
        return _resolved(request.req!.url);
      })
      ..bind(GopeedMethods.createTaskBatch, (_) {
        submissions++;
        return CreateTaskBatchResult(taskIds: ['created']);
      });
    final router = await _openPage(tester, registry);
    for (var attempt = 0; attempt < 2; attempt++) {
      if (attempt > 0) {
        router.go('/create');
        await tester.pumpAndSettle();
      }
      await _openPreview(tester, urls: 'https://example.com/group');
      await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      expect(find.byType(CreateTaskWindowPage), findsNothing);
    }
    expect(resolutions, 2);
    expect(submissions, 2);
    expect(tester.takeException(), isNull);
  });

  for (final duringSubmission in [false, true]) {
    testWidgets('a new external task starts a fresh session (during submission: $duringSubmission)', (tester) async {
      final firstSubmission = Completer<CreateTaskBatchResult>();
      final resolutions = <Request>[];
      var groupSubmissions = 0;
      final registry = CapabilityRegistry(createAppCapabilityCodecs())
        ..bind(GopeedMethods.resolve, (request) {
          final req = request.req!;
          resolutions.add(req);
          return _resolved(req.url);
        })
        ..bind(GopeedMethods.createTaskBatch, (batch) {
          if (batch.group) {
            if (++groupSubmissions == 1 && duringSubmission) return firstSubmission.future;
          } else if (!duringSubmission) {
            throw StateError('ordinary batch failed');
          }
          return CreateTaskBatchResult(taskIds: ['created']);
        });
      await _openPage(tester, registry);
      final page = find.byType(CreateTaskWindowPage);
      final pageState = tester.state(page);
      final container = ProviderScope.containerOf(tester.element(page));
      await _openPreview(tester, urls: 'https://example.com/group\nhttps://example.com/other');
      await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      expect(groupSubmissions, 1);
      if (!duringSubmission) {
        await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
        await tester.pumpAndSettle();
      }

      container
          .read(pendingCreateTaskProvider.notifier)
          .set(
            CreateTask(
              req: Request(
                url: 'https://example.com/group',
                extra: ReqExtraHttp(method: 'GET', header: {'Cookie': 'new-intent'}, body: '').toJson(),
              ),
            ),
          );
      await tester.pump();
      if (duringSubmission) {
        firstSubmission.complete(CreateTaskBatchResult(taskIds: ['old-created']));
        await tester.pump();
      }
      await tester.pumpAndSettle();

      expect(tester.state(page), same(pageState));
      await _openPreview(tester);
      expect(resolutions.map((request) => request.url), [
        'https://example.com/group',
        'https://example.com/other',
        'https://example.com/group',
      ]);
      expect(ReqExtraHttp.fromJson(resolutions.last.extra! as Map<String, dynamic>).header['Cookie'], 'new-intent');
      await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      expect(groupSubmissions, 2);
      expect(page, findsNothing);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('a failed stale direct submission releases the new task form', (tester) async {
    final firstSubmission = Completer<CreateTaskBatchResult>();
    final requests = <CreateTaskBatch>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.createTaskBatch, (batch) {
        requests.add(batch);
        if (requests.length == 1) return firstSubmission.future;
        return CreateTaskBatchResult(taskIds: ['new-created']);
      });
    await _openPage(tester, registry);
    final container = ProviderScope.containerOf(tester.element(find.byType(CreateTaskWindowPage)));
    await tester.tap(find.byKey(const ValueKey('create-task-direct-download-checkbox')));
    await tester.enterText(find.byKey(const ValueKey('create-task-url-input')), 'https://example.com/old');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    expect(requests, hasLength(1));

    container.read(pendingCreateTaskProvider.notifier).set(CreateTask(req: Request(url: 'https://example.com/new')));
    await tester.pump();
    firstSubmission.completeError(StateError('old submission failed'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(find.byType(CreateTaskWindowPage), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(requests, hasLength(2));
    expect(requests.last.reqs!.single.req!.url, 'https://example.com/new');
    expect(find.byType(CreateTaskWindowPage), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('confirming only completed sources finishes without opening another preview', (tester) async {
    var resolutions = 0;
    var submissions = 0;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.resolve, (request) {
        resolutions++;
        return _resolved(request.req!.url);
      })
      ..bind(GopeedMethods.createTaskBatch, (batch) {
        submissions++;
        if (!batch.group) throw StateError('batch failed');
        return CreateTaskBatchResult(taskIds: ['created']);
      });
    await _openPage(tester, registry);
    await _openPreview(tester, urls: 'https://example.com/group\nhttps://example.com/other');
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('create-task-url-input')), 'https://example.com/group');
    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('create-task-checksum-expected')), 'invalid');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(resolutions, 2);
    expect(submissions, 2);
    expect(find.byType(CreateTaskWindowPage), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('switching to direct download skips completed preview batches', (tester) async {
    var resolutions = 0;
    final batches = <CreateTaskBatch>[];
    var fail = true;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.resolve, (request) {
        resolutions++;
        return _resolved(request.req!.url);
      })
      ..bind(GopeedMethods.createTaskBatch, (batch) {
        batches.add(batch);
        if (!batch.group && fail) throw StateError('batch failed');
        return CreateTaskBatchResult(taskIds: ['created']);
      });
    await _openPage(tester, registry);
    await _openPreview(tester, urls: 'https://example.com/group\nhttps://example.com/other');
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('create-task-direct-download-checkbox')));
    await tester.pump();
    fail = false;
    await _openPreview(tester);
    expect(resolutions, 2);
    expect(batches, hasLength(3));
    expect(batches.last.reqs!.single.req!.url, 'https://example.com/other');
    expect(find.byType(CreateTaskWindowPage), findsNothing);
    expect(tester.takeException(), isNull);
  });
}

ResolveResult _resolved(String url) => url.endsWith('/group')
    ? ResolveResult(
        res: Resource(
          name: 'album',
          files: [
            FileInfo(
              name: 'a.txt',
              req: Request(url: 'https://example.com/a.txt'),
            ),
            FileInfo(
              name: 'b.txt',
              req: Request(url: 'https://example.com/b.txt'),
            ),
          ],
        ),
      )
    : ResolveResult(
        id: 'rid-$url',
        res: Resource(files: [FileInfo(name: 'other.txt')]),
      );

Future<GoRouter> _openPage(WidgetTester tester, CapabilityRegistry registry) async {
  tester.view.physicalSize = const Size(900, 1000);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  registry
    ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig(downloadDir: '/downloads'))
    ..bind(StorageMethods.saveCreateHistory, (_) => const RpcUnit());
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
  return router;
}

Future<void> _openPreview(WidgetTester tester, {String? urls}) async {
  if (urls != null) await tester.enterText(find.byKey(const ValueKey('create-task-url-input')), urls);
  final confirm = find.byKey(const ValueKey('create-task-confirm-button'));
  await tester.ensureVisible(confirm);
  await tester.tap(confirm);
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 300));
}
