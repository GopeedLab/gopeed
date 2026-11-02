import 'dart:async';
import 'dart:convert';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/storage_capability.dart';
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';
import 'package:gopeed/features/tasks/presentation/pages/create_task_window_page.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  final host = LocalAppCapabilities.instance;
  late _ConfigTransport transport;

  setUp(() {
    transport = _ConfigTransport();
    api.setTransportForTesting(transport);
  });

  test('local and child saves serialize the full history read-modify-write', () async {
    final first = host.capabilities.storage.saveCreateHistory(['A']);
    await transport.firstPutStarted.future;
    final second = host.registry.invoke(StorageMethods.saveCreateHistory.name, ['B']);
    await Future<void>.delayed(Duration.zero);
    final readsWhileSaving = transport.reads;
    transport.releaseFirstPut.complete();
    await Future.wait<Object?>([first, second]);
    expect(transport.config.extra.createHistory, ['B', 'A']);
    expect(readsWhileSaving, 1, reason: 'the next GET must wait for the previous PUT');
  });

  test('history and config reads wait for previously requested saves', () async {
    final saving = host.capabilities.storage.saveCreateHistory(['A']);
    await transport.firstPutStarted.future;
    final history = host.capabilities.storage.getCreateHistory();
    final config = host.capabilities.gopeed.getConfig();
    await Future<void>.delayed(Duration.zero);
    final readsWhileSaving = transport.reads;
    transport.releaseFirstPut.complete();
    await saving;
    expect(await history, ['A']);
    expect((await config).extra.createHistory, ['A']);
    expect(readsWhileSaving, 1);
  });

  for (final historyFirst in [true, false]) {
    test('overlapping full config and history saves preserve both changes (history first: $historyFirst)', () async {
      transport.config.extra.createHistory = ['existing'];
      final snapshot = await host.capabilities.gopeed.getConfig();
      snapshot.downloadDir = '/chosen';
      snapshot.maxRunning = 7;
      final first = historyFirst
          ? host.capabilities.storage.saveCreateHistory(['A'])
          : host.capabilities.gopeed.putConfig(snapshot);
      await transport.firstPutStarted.future;
      final second = historyFirst
          ? host.capabilities.gopeed.putConfig(snapshot)
          : host.capabilities.storage.saveCreateHistory(['A']);
      await Future<void>.delayed(Duration.zero);
      transport.releaseFirstPut.complete();
      await Future.wait([first, second]);
      expect(transport.config.extra.createHistory, ['A', 'existing']);
      expect(transport.config.downloadDir, '/chosen');
      expect(transport.config.maxRunning, 7);
    });
  }

  test('an old full config cannot restore removed or cleared history', () async {
    transport.blockFirstPut = false;
    transport.config.extra.createHistory = ['A', 'B'];
    final snapshot = await host.capabilities.gopeed.getConfig();
    await host.capabilities.storage.removeCreateHistory('A');
    await host.capabilities.gopeed.putConfig(snapshot);
    expect(transport.config.extra.createHistory, ['B']);
    await host.capabilities.storage.clearCreateHistory();
    await host.capabilities.gopeed.putConfig(snapshot);
    expect(transport.config.extra.createHistory, isEmpty);
    expect(snapshot.extra.createHistory, ['A', 'B'], reason: 'saving must not mutate the caller snapshot');
  });

  test('history save, remove, clear and save follow request order', () async {
    transport.config.extra.createHistory = ['existing'];
    final first = host.capabilities.storage.saveCreateHistory(['A']);
    await transport.firstPutStarted.future;
    final remove = host.capabilities.storage.removeCreateHistory('existing');
    final clear = host.capabilities.storage.clearCreateHistory();
    final second = host.capabilities.storage.saveCreateHistory(['B']);
    await Future<void>.delayed(Duration.zero);
    transport.releaseFirstPut.complete();
    await Future.wait([first, remove, clear, second]);
    expect(transport.config.extra.createHistory, ['B']);
  });

  test('a failed save reports its error without blocking subsequent saves or reads', () async {
    transport.failNextPut = true;
    final first = host.capabilities.storage.saveCreateHistory(['A']);
    final failed = expectLater(first, throwsA(isA<StateError>()));
    await transport.firstPutStarted.future;
    final second = host.capabilities.storage.saveCreateHistory(['B']);
    final history = host.capabilities.storage.getCreateHistory();
    transport.releaseFirstPut.complete();
    await failed;
    await second;
    expect(await history, ['B']);
  });

  testWidgets('confirm A, cancel, confirm B saves both while previews open immediately', (tester) async {
    tester.view.physicalSize = const Size(900, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(
      ProviderScope(
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: AppComponentThemes(child: const CreateTaskWindowPage()),
        ),
      ),
    );
    // The host queue is created outside the widget clock.
    await tester.runAsync(() => Future<void>.delayed(Duration.zero));
    await tester.pumpAndSettle();
    final input = find.byKey(const ValueKey('create-task-url-input'));
    await tester.enterText(input, 'https://example.com/A');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.runAsync(() => Future<void>.delayed(Duration.zero));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(transport.firstPutStarted.isCompleted, isTrue);
    expect(transport.releaseFirstPut.isCompleted, isFalse);
    expect(find.byKey(const ValueKey('resolve-cancel-button')), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
    await tester.pumpAndSettle();
    await tester.enterText(input, 'https://example.com/B');
    await tester.tap(find.byKey(const ValueKey('create-task-confirm-button')));
    await tester.runAsync(() => Future<void>.delayed(Duration.zero));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(find.byKey(const ValueKey('resolve-cancel-button')), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(shad.Icons.history));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    transport.releaseFirstPut.complete();
    await tester.runAsync(() => Future<void>.delayed(Duration.zero));
    await tester.pumpAndSettle();
    expect(transport.config.extra.createHistory, ['https://example.com/B', 'https://example.com/A']);
    for (final url in ['https://example.com/A', 'https://example.com/B']) {
      expect(find.byKey(ValueKey('create-history-item-$url')), findsOneWidget);
    }
    expect(tester.takeException(), isNull);
  });
}

class _ConfigTransport implements GopeedTransport {
  DownloaderConfig config = DownloaderConfig(downloadDir: '/downloads');
  final firstPutStarted = Completer<void>();
  final releaseFirstPut = Completer<void>();
  bool blockFirstPut = true;
  bool failNextPut = false;
  int reads = 0;
  int writes = 0;

  @override
  Future<dynamic> request(
    String path, {
    String method = 'GET',
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async {
    if (path == 'api/v1/resolve') {
      return {
        'code': 0,
        'data': {
          'id': 'rid',
          'res': {
            'files': [
              {'name': 'file.txt'},
            ],
          },
        },
      };
    }
    if (path != 'api/v1/config') throw StateError('Unexpected request: $method $path');
    if (method == 'GET') {
      reads++;
      return {'code': 0, 'data': jsonDecode(jsonEncode(config))};
    }
    final snapshot = DownloaderConfig.fromJson(jsonDecode(jsonEncode(data)) as Map<String, dynamic>);
    final shouldFail = failNextPut;
    failNextPut = false;
    if (++writes == 1 && blockFirstPut) {
      firstPutStarted.complete();
      await releaseFirstPut.future;
    }
    if (shouldFail) {
      throw StateError('config save failed');
    }
    config = snapshot;
    return {'code': 0};
  }

  @override
  String join(String path) => path;
}
