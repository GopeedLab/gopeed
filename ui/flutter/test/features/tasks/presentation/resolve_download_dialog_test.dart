import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/create_task_batch.dart';
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:gopeed/api/model/resolve_result.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/features/tasks/application/pending_download.dart';
import 'package:gopeed/features/tasks/presentation/widgets/resolve_download_dialog.dart';
import 'package:gopeed/features/tasks/presentation/widgets/resolve_file_tree.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_loading_button.dart';
import 'package:gopeed/shared/widgets/app_tooltip.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

PendingDownload pending(String id) => PendingDownload(
  id: id,
  request: Request(url: 'https://example.com/$id'),
  options: Options(),
);
ResolveResult resolved(String name) => ResolveResult(
  id: name,
  res: Resource(files: [FileInfo(name: name)]),
);

Future<void> open(
  WidgetTester tester,
  List<PendingDownload> sources,
  CapabilityRegistry registry, {
  ValueChanged<bool>? onClose,
  double width = 900,
}) async {
  tester.view.physicalSize = Size(width, 800);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  await tester.pumpWidget(
    shad.ShadcnApp(
      theme: AppTheme.light(),
      materialTheme: AppTheme.materialLight(),
      home: Builder(
        builder: (context) => shad.PrimaryButton(
          onPressed: () async {
            final result = await showResolveDownloadsDialog(
              context,
              sources: sources,
              service: GopeedService(LocalCapabilityInvoker(registry)),
            );
            onClose?.call(result);
          },
          child: const Text('Open'),
        ),
      ),
    ),
  );
  await tester.tap(find.text('Open'));
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 300));
}

void main() {
  for (final width in [390.0, 700.0, 1024.0]) {
    testWidgets('file list fits and exposes ready selections at width $width', (tester) async {
      final registry = CapabilityRegistry(createAppCapabilityCodecs())
        ..bind(GopeedMethods.resolve, (_) => resolved('file.txt'));
      await open(tester, [pending('source')], registry, width: width);
      await tester.pumpAndSettle();
      expect(find.text('File List'), findsOneWidget);
      expect(tester.widget<AppLoadingButton>(find.byKey(const ValueKey('resolve-create-button'))).onPressed, isNotNull);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('shows all roots immediately and resolves sequentially while preserving choices', (tester) async {
    final calls = <String>[];
    final responses = [Completer<ResolveResult>(), Completer<ResolveResult>()];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.resolve, (request) {
        calls.add(request.req!.url);
        return responses[calls.length - 1].future;
      });
    final sources = [pending('a'), pending('b')];
    await open(tester, sources, registry);
    expect(find.text('File List'), findsOneWidget);
    expect(find.text('https://example.com/a'), findsOneWidget);
    expect(find.text('https://example.com/b'), findsOneWidget);
    expect(calls, ['https://example.com/a']);
    expect(tester.widget<AppLoadingButton>(find.byKey(const ValueKey('resolve-create-button'))).onPressed, isNull);
    expect(find.byWidgetPredicate((w) => w is AppTooltip && w.message == 'Waiting to resolve'), findsOneWidget);
    responses.first.complete(resolved('first.txt'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(calls, ['https://example.com/a', 'https://example.com/b']);
    expect(find.text('first.txt'), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('resolve-tree-name-scroll-file:a:0')));
    await tester.pump();
    expect(sources.first.selected, isEmpty);
    responses.last.complete(resolved('second.txt'));
    await tester.pumpAndSettle();
    expect(sources.first.selected, isEmpty);
    expect(sources.last.selected, {0});
    expect(tester.widget<AppLoadingButton>(find.byKey(const ValueKey('resolve-create-button'))).onPressed, isNotNull);
    expect(tester.takeException(), isNull);
  });

  for (final filters in [
    <String>['video'],
    <String>['video', 'audio'],
  ]) {
    testWidgets('new files clear type highlights without changing previous selections ($filters)', (tester) async {
      final response = Completer<ResolveResult>();
      final first = pending('first')
        ..result = ResolveResult(
          res: Resource(
            name: 'bundle',
            files: [
              FileInfo(name: 'video.mp4', size: 10),
              FileInfo(name: 'song.mp3', size: 20),
              FileInfo(name: 'picture.png', size: 30),
            ],
          ),
        )
        ..selected.addAll([0, 1, 2]);
      final second = pending('second');
      final batches = <CreateTaskBatch>[];
      final registry = CapabilityRegistry(createAppCapabilityCodecs())
        ..bind(GopeedMethods.resolve, (_) => response.future)
        ..bind(GopeedMethods.createTaskBatch, (batch) {
          batches.add(batch);
          return CreateTaskBatchResult(taskIds: ['created']);
        });
      await open(tester, [first, second], registry);
      for (final filter in filters) {
        await tester.tap(find.byKey(ValueKey('resolve-tree-filter-$filter')));
        await tester.pump();
        // Each selection is written back by the parent and rebuilds the tree.
        expect(typeFilterActive(tester, filter), isTrue);
      }
      final expectedFirst = filters.length == 1 ? {0} : {0, 1};
      expect(first.selected, expectedFirst);
      expect(tester.widget<AppLoadingButton>(find.byKey(const ValueKey('resolve-create-button'))).onPressed, isNull);
      final incoming = filters.length == 1 ? 'new-audio.mp3' : 'new-image.png';
      response.complete(
        ResolveResult(
          id: 'incoming-rid',
          res: Resource(files: [FileInfo(name: incoming, size: 40)]),
        ),
      );
      await tester.pumpAndSettle();
      expect(first.selected, expectedFirst);
      expect(second.selected, {0});
      for (final filter in filters) {
        expect(typeFilterActive(tester, filter), isFalse);
      }
      expect(find.text('${expectedFirst.length + 1} / 4 selected'), findsOneWidget);
      final size = tester.widget<Text>(find.byKey(const ValueKey('resolve-tree-selected-size'))).data;
      expect(size, '${filters.length == 1 ? 50 : 70} B');
      await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
      await tester.pumpAndSettle();
      expect(batches, hasLength(2));
      expect(batches.first.group, isTrue);
      expect(
        batches.first.reqs!.map((item) => item.opts!.name),
        filters.length == 1 ? ['video.mp4'] : ['video.mp4', 'song.mp3'],
      );
      expect(batches.last.group, isFalse);
      expect(batches.last.reqs!.single.rid, 'incoming-rid');
      expect(batches.last.reqs!.single.opts!.selectFiles, [0]);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('a failed source preserves type highlights and the successful source selection', (tester) async {
    final response = Completer<ResolveResult>();
    final first = pending('first')
      ..result = ResolveResult(
        res: Resource(
          files: [
            FileInfo(name: 'video.mp4'),
            FileInfo(name: 'song.mp3'),
          ],
        ),
      )
      ..selected.addAll([0, 1]);
    final second = pending('failed');
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.resolve, (_) => response.future);
    await open(tester, [first, second], registry);
    await tester.tap(find.byKey(const ValueKey('resolve-tree-filter-video')));
    await tester.pump();
    expect(typeFilterActive(tester, 'video'), isTrue);
    response.completeError(StateError('unavailable'));
    await tester.pumpAndSettle();
    expect(first.selected, {0});
    expect(second.selected, isEmpty);
    expect(typeFilterActive(tester, 'video'), isTrue);
    expect(find.text('1 / 2 selected'), findsOneWidget);
    expect(tester.widget<AppLoadingButton>(find.byKey(const ValueKey('resolve-create-button'))).onPressed, isNotNull);
    await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });

  testWidgets('same-name sources synchronize individual, root and all-file selections', (tester) async {
    final sources = [
      for (final id in ['first', 'second'])
        pending(id)
          ..result = ResolveResult(
            res: Resource(
              name: 'bundle',
              files: [
                FileInfo(name: 'a.txt'),
                FileInfo(name: 'b.txt'),
              ],
            ),
          )
          ..selected.addAll([0, 1]),
    ];
    await open(tester, sources, CapabilityRegistry(createAppCapabilityCodecs()));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('resolve-tree-name-scroll-file:first:0')));
    await tester.pumpAndSettle();
    expect(sources.first.selected, {1});
    expect(sources.last.selected, {0, 1});

    await tester.tap(find.byKey(const ValueKey('resolve-tree-name-scroll-folder:second')));
    await tester.pumpAndSettle();
    expect(sources.first.selected, {1});
    expect(sources.last.selected, isEmpty);
    await tester.tap(find.byKey(const ValueKey('resolve-tree-name-scroll-file:first:1')));
    await tester.pumpAndSettle();
    expect(sources.first.selected, isEmpty);
    expect(tester.widget<AppLoadingButton>(find.byKey(const ValueKey('resolve-create-button'))).onPressed, isNull);

    final selectAll = find.descendant(of: find.byType(ResolveFileTree), matching: find.byType(shad.Checkbox)).first;
    await tester.tap(selectAll);
    await tester.pumpAndSettle();
    expect(sources.every((source) => source.selected.length == 2 && source.selected.containsAll([0, 1])), isTrue);
    await tester.tap(selectAll);
    await tester.pumpAndSettle();
    expect(sources.every((source) => source.selected.isEmpty), isTrue);
    await tester.tap(find.byKey(const ValueKey('resolve-cancel-button')));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });

  testWidgets('cancel stops the queue and ignores the late response', (tester) async {
    final response = Completer<ResolveResult>();
    var calls = 0;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.resolve, (_) {
        calls++;
        return response.future;
      });
    bool? closed;
    await open(tester, [pending('a'), pending('b')], registry, onClose: (value) => closed = value);
    await tester.tap(find.byKey(const ValueKey('resolve-close-button')));
    await tester.pumpAndSettle();
    response.complete(resolved('late.txt'));
    await tester.pumpAndSettle();
    expect(calls, 1);
    expect(closed, isFalse);
    expect(find.text('late.txt'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('failed root is disabled and only ready selections enter a single batch', (tester) async {
    final batches = <CreateTaskBatch>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.resolve, (request) {
        if (request.req!.url.endsWith('/bad')) throw StateError('unavailable');
        return resolved('ready.txt');
      })
      ..bind(GopeedMethods.createTaskBatch, (batch) {
        batches.add(batch);
        return CreateTaskBatchResult(taskIds: ['created']);
      });
    bool? closed;
    await open(tester, [pending('bad'), pending('good')], registry, onClose: (value) => closed = value);
    await tester.pumpAndSettle();
    expect(find.textContaining('unavailable'), findsNothing);
    expect(find.byWidgetPredicate((w) => w is AppTooltip && w.message.contains('unavailable')), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pumpAndSettle();
    expect(batches.single.reqs!.single.rid, 'ready.txt');
    expect(closed, isTrue);
    expect(tester.takeException(), isNull);
  });
}

bool typeFilterActive(WidgetTester tester, String filter) {
  final decoration =
      tester
              .widget<DecoratedBox>(
                find
                    .descendant(
                      of: find.byKey(ValueKey('resolve-tree-filter-$filter')),
                      matching: find.byType(DecoratedBox),
                    )
                    .first,
              )
              .decoration
          as BoxDecoration;
  return decoration.color != null;
}
