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
        name: 'file.txt',
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
