import 'package:flutter/widgets.dart';
import 'package:gopeed/core/icons/gopeed_icons.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/features/tasks/presentation/widgets/resolve_file_tree.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/file_tree_view.dart';
import 'package:gopeed/shared/widgets/virtual_tree_view.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:gopeed/api/model/resolve_result.dart';
import 'package:gopeed/features/tasks/application/pending_download.dart';

PendingDownload source(String id) =>
    PendingDownload(
        id: id,
        request: Request(url: 'https://example.com/$id'),
        options: Options(),
      )
      ..result = ResolveResult(
        res: Resource(
          name: 'bundle',
          files: [
            FileInfo(name: '0.txt'),
            FileInfo(name: '1.txt'),
          ],
        ),
      )
      ..selected.addAll([0, 1]);

void main() {
  for (final width in [390.0, 700.0, 1024.0]) {
    testWidgets('live mixed sources keep input order, selection and expansion at $width', (tester) async {
      tester.view.physicalSize = Size(width, 1000);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final sources = [
        for (final id in ['http', 'group', 'ungrouped', 'bt', 'failed'])
          PendingDownload(
            id: id,
            request: Request(url: 'https://example.com/$id'),
            options: Options(),
          ),
      ];
      final http = sources[0];
      final group = sources[1];
      final ungrouped = sources[2];
      final bt = sources[3];
      bt.request.url = 'magnet:?xt=urn:btih:infohash';
      Future<void> show() async {
        await tester.pumpWidget(
          shad.ShadcnApp(
            theme: AppTheme.light(),
            materialTheme: AppTheme.materialLight(),
            home: ResolveFileTree(
              sources: sources,
              onSelectionChanged: (keys) {
                for (final pending in sources) {
                  pending.selected
                    ..clear()
                    ..addAll(keys.where((key) => key.$1 == pending.id).map((key) => key.$2));
                }
              },
            ),
          ),
        );
        await tester.pump();
      }

      List<shad.TreeItemNode<FileTreeNode<PendingFileKey>>> roots() => tester
          .widget<VirtualTreeView<FileTreeNode<PendingFileKey>>>(
            find.byType(VirtualTreeView<FileTreeNode<PendingFileKey>>),
          )
          .nodes
          .cast<shad.TreeItemNode<FileTreeNode<PendingFileKey>>>();
      void expectOrder(List<String> labels) => expect(roots().map((node) => node.data.label), labels);

      await show();
      expectOrder(sources.map((pending) => pending.request.url).toList());
      http.result = ResolveResult(
        id: 'http-rid',
        res: Resource(files: [FileInfo(name: 'z-http.txt')]),
      );
      http.selected.add(0);
      await show();
      expectOrder(['z-http.txt', ...sources.skip(1).map((pending) => pending.request.url)]);
      await tester.tap(find.byKey(const ValueKey('resolve-tree-name-scroll-file:http:0')));
      await tester.pump();
      expect(http.selected, isEmpty);

      group.result = ResolveResult(
        res: Resource(
          name: 'bundle',
          files: [
            FileInfo(name: 'one.txt'),
            FileInfo(name: 'two.txt'),
          ],
        ),
      );
      group.selected.addAll([0, 1]);
      await show();
      expectOrder(['z-http.txt', 'bundle', ...sources.skip(2).map((pending) => pending.request.url)]);

      ungrouped.result = ResolveResult(
        res: Resource(
          files: [
            FileInfo(name: 'z-first.txt'),
            FileInfo(name: 'nested.txt', path: 'directory'),
            FileInfo(name: 'a-last.txt'),
          ],
        ),
      );
      ungrouped.selected.addAll([0, 1, 2]);
      await tester.tap(find.byKey(const ValueKey('resolve-tree-node-expand-folder:group')));
      await tester.pump(const Duration(milliseconds: 50));
      await show();
      await tester.pump(const Duration(milliseconds: 250));
      expect(roots()[1].expanded, isFalse);
      expectOrder([
        'z-http.txt',
        'bundle',
        'z-first.txt',
        'directory',
        'a-last.txt',
        bt.request.url,
        sources.last.request.url,
      ]);

      bt.result = ResolveResult(
        id: 'bt-rid',
        res: Resource(
          name: 'torrent',
          files: [
            FileInfo(name: 'bt-one'),
            FileInfo(name: 'bt-two'),
          ],
        ),
      );
      bt.selected.addAll([0, 1]);
      await show();
      sources.last.resolveError = 'cannot resolve';
      await show();
      expectOrder([
        'z-http.txt',
        'bundle',
        'z-first.txt',
        'directory',
        'a-last.txt',
        'torrent',
        sources.last.request.url,
      ]);
      expect(roots()[1].expanded, isFalse);
      expect(http.selected, isEmpty);
      expect(group.selected, {0, 1});
      expect(ungrouped.selected, {0, 1, 2});
      expect(bt.selected, {0, 1});
      expect(sources.last.selected, isEmpty);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('ED2K uses a file icon and filename while BT uses its resource name and folder icon', (tester) async {
    final ed2k =
        PendingDownload(
            id: 'ed2k',
            request: Request(url: 'ed2k://|file|payload.bin|4|0123456789ABCDEF0123456789ABCDEF|/'),
            options: Options(),
          )
          ..result = ResolveResult(
            id: 'ed2k-rid',
            res: Resource(
              hash: '0123456789ABCDEF0123456789ABCDEF',
              files: [FileInfo(name: 'payload.bin')],
            ),
          )
          ..selected.add(0);
    final bt =
        PendingDownload(
            id: 'bt',
            request: Request(url: 'magnet:?xt=urn:btih:infohash'),
            options: Options(),
          )
          ..result = ResolveResult(
            id: 'bt-rid',
            res: Resource(
              name: 'torrent resource',
              hash: 'infohash',
              files: [FileInfo(name: 'torrent-file.bin')],
            ),
          )
          ..selected.add(0);
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: ResolveFileTree(sources: [ed2k, bt], onSelectionChanged: (_) {}),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('payload.bin'), findsOneWidget);
    expect(find.text('torrent resource'), findsOneWidget);
    expect(find.byIcon(GopeedIcons.folderBt), findsOneWidget);
    expect(find.text(ed2k.request.url), findsNothing);
    expect(tester.takeException(), isNull);
  });

  for (final directory in [false, true]) {
    testWidgets('FTP resource ${directory ? 'directory' : 'file'} keeps its actual structure', (tester) async {
      final pending = source('ftp');
      pending.request.url = directory ? 'ftp://files.example/folder' : 'ftpes://files.example/file.torrent';
      pending.result = ResolveResult(
        id: 'ftp-rid',
        res: Resource(
          name: directory ? 'folder' : '',
          files: [FileInfo(name: 'file.torrent')],
        ),
      );
      pending.selected
        ..clear()
        ..add(0);
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: ResolveFileTree(sources: [pending], onSelectionChanged: (_) {}),
        ),
      );
      await tester.pumpAndSettle();
      expect(pending.isBt, isFalse);
      expect(pending.isGroup, isFalse);
      expect(pending.name, directory ? 'folder' : 'file.torrent');
      expect(find.text('folder'), directory ? findsOneWidget : findsNothing);
      expect(find.byIcon(shad.BootstrapIcons.folder2), directory ? findsOneWidget : findsNothing);
      expect(find.byIcon(GopeedIcons.folderBt), findsNothing);
      expect(find.text('file.torrent'), findsOneWidget);
      expect(planDownloads([pending]).single.request.reqs!.single.rid, 'ftp-rid');
      expect(tester.takeException(), isNull);
    });
  }

  for (final extension in [true, false]) {
    testWidgets('magnet root icon follows the resolved resource (extension=$extension)', (tester) async {
      final pending = source('magnet');
      pending.request.url = 'magnet:?xt=urn:btih:infohash';
      pending.result!.id = extension ? '' : 'bt-rid';
      for (final file in pending.files) {
        if (extension) file.req = Request(url: 'https://example.com/${file.name}');
      }
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: ResolveFileTree(sources: [pending], onSelectionChanged: (_) {}),
        ),
      );
      await tester.pumpAndSettle();
      expect(pending.isGroup, extension);
      expect(find.byIcon(GopeedIcons.folderBt), extension ? findsNothing : findsOneWidget);
      expect(find.byIcon(GopeedIcons.file), findsNWidgets(2));
      expect(find.byIcon(shad.BootstrapIcons.folder2), extension ? findsOneWidget : findsNothing);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('ordinary single files replace URL placeholders without a source directory', (tester) async {
    final pending = PendingDownload(
      id: 'http',
      request: Request(url: 'https://example.com/file'),
      options: Options(),
    );
    Future<void> show() async {
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: ResolveFileTree(sources: [pending], onSelectionChanged: (_) {}),
        ),
      );
      await tester.pump();
    }

    await show();
    expect(find.text(pending.request.url), findsOneWidget);
    pending.result = ResolveResult(
      id: 'http-rid',
      res: Resource(
        name: '',
        files: [FileInfo(name: 'file.txt')],
      ),
    );
    pending.selected.add(0);
    await show();
    expect(find.text('file.txt'), findsOneWidget);
    expect(find.text(pending.request.url), findsNothing);
    final tree = tester.widget<VirtualTreeView<FileTreeNode<PendingFileKey>>>(
      find.byType(VirtualTreeView<FileTreeNode<PendingFileKey>>),
    );
    expect(tree.nodes, hasLength(1));
    expect(tree.nodes.single.children, isEmpty);
    expect((tree.nodes.single as shad.TreeItemNode<FileTreeNode<PendingFileKey>>).data.item!.data, ('http', 0));
    expect(tester.takeException(), isNull);
  });

  for (final filePath in ['', 'disc1']) {
    testWidgets('named single-file extension keeps its resource directory (path=$filePath)', (tester) async {
      final pending =
          PendingDownload(
              id: 'extension',
              request: Request(url: 'https://example.com/track.mp3'),
              options: Options(path: '/downloads'),
            )
            ..result = ResolveResult(
              res: Resource(
                name: 'album',
                files: [
                  FileInfo(
                    name: 'track.mp3',
                    path: filePath,
                    req: Request(url: 'https://cdn.example.com/track.mp3'),
                  ),
                ],
              ),
            )
            ..selected.add(0);
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: ResolveFileTree(sources: [pending], onSelectionChanged: (_) {}),
        ),
      );
      await tester.pumpAndSettle();

      final tree = tester.widget<VirtualTreeView<FileTreeNode<PendingFileKey>>>(
        find.byType(VirtualTreeView<FileTreeNode<PendingFileKey>>),
      );
      final roots = tree.nodes.cast<shad.TreeItemNode<FileTreeNode<PendingFileKey>>>();
      expect(roots.map((node) => node.data.label), ['album']);
      expect(roots.single.expanded, isTrue);
      final child = roots.single.children.single as shad.TreeItemNode<FileTreeNode<PendingFileKey>>;
      expect(child.data.label, filePath.isEmpty ? 'track.mp3' : 'disc1');
      final plan = planDownloads([pending]).single.request;
      expect(plan.group, isFalse);
      expect(plan.reqs!.single.opts!.path, filePath.isEmpty ? '/downloads/album' : '/downloads/album/disc1');
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('a live group replaces its URL placeholder with an expanded group directory', (tester) async {
    final pending = PendingDownload(
      id: 'group',
      request: Request(url: 'https://example.com/group'),
      options: Options(),
    );
    Future<void> show() async {
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: ResolveFileTree(sources: [pending], onSelectionChanged: (_) {}),
        ),
      );
      await tester.pump();
    }

    await show();
    pending.result = ResolveResult(
      res: Resource(
        name: 'bundle',
        files: [
          FileInfo(name: 'one.txt'),
          FileInfo(name: 'two.txt'),
        ],
      ),
    );
    pending.selected.addAll([0, 1]);
    await show();
    expect(find.text('bundle'), findsOneWidget);
    expect(find.text('one.txt'), findsOneWidget);
    expect(find.text('two.txt'), findsOneWidget);
    expect(find.byIcon(shad.BootstrapIcons.folder2), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  for (final width in [390.0, 700.0, 1024.0]) {
    testWidgets('ungrouped roots preserve relative directories and source selection at $width', (tester) async {
      tester.view.physicalSize = Size(width, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final sources = [source('first'), source('second')];
      for (final pending in sources) {
        pending.result!.res.name = '';
        pending.files[0].name = 'same.txt';
        pending.files[1].name = 'video.mp4';
        pending.files[1].path = 'nested';
      }
      Set<PendingFileKey> selected = {};
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: ResolveFileTree(sources: sources, onSelectionChanged: (keys) => selected = keys),
        ),
      );
      await tester.pumpAndSettle();
      final tree = tester.widget<VirtualTreeView<FileTreeNode<PendingFileKey>>>(
        find.byType(VirtualTreeView<FileTreeNode<PendingFileKey>>),
      );
      expect(tree.nodes, hasLength(4));
      expect(tree.nodes.cast<shad.TreeItemNode<FileTreeNode<PendingFileKey>>>().map((node) => node.data.label), [
        'same.txt',
        'nested',
        'same.txt',
        'nested',
      ]);
      expect(
        tree.nodes.cast<shad.TreeItemNode<FileTreeNode<PendingFileKey>>>().map((node) => node.data.key).toSet(),
        hasLength(4),
      );
      expect(find.text('nested'), findsNWidgets(2));
      expect(find.text('same.txt'), findsNWidgets(2));
      expect(find.text(sources.first.request.url), findsNothing);
      await tester.tap(find.byKey(const ValueKey('resolve-tree-name-scroll-file:first:0')));
      await tester.pump();
      expect(selected, {('first', 1), ('second', 0), ('second', 1)});
      final secondDirectoryKey = (tree.nodes.last as shad.TreeItemNode<FileTreeNode<PendingFileKey>>).data.key;
      await tester.tap(find.byKey(ValueKey('resolve-tree-name-scroll-$secondDirectoryKey')));
      await tester.pump();
      expect(selected, {('first', 1), ('second', 0)});
      await tester.tap(find.byKey(const ValueKey('resolve-tree-filter-video')));
      await tester.pump();
      expect(selected, {('first', 1), ('second', 1)});
      expect(tester.takeException(), isNull);
    });
  }

  for (final width in [390.0, 700.0, 1024.0]) {
    testWidgets('same-name roots isolate file indexes and root selection at $width', (tester) async {
      tester.view.physicalSize = Size(width, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final first = source('first');
      final second = source('second');
      Set<PendingFileKey> selected = {('first', 0), ('first', 1), ('second', 0), ('second', 1)};
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: ResolveFileTree(sources: [first, second], onSelectionChanged: (keys) => selected = keys),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('bundle'), findsNWidgets(2));
      await tester.tap(find.byKey(const ValueKey('resolve-tree-name-scroll-file:first:0')));
      await tester.pump();
      expect(selected, {('first', 1), ('second', 0), ('second', 1)});
      await tester.tap(find.byKey(const ValueKey('resolve-tree-name-scroll-folder:second')));
      await tester.pump();
      expect(selected, {('first', 1)});
      expect(tester.takeException(), isNull);
    });
  }
}
