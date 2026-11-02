import 'package:flutter/widgets.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/file_tree_view.dart';
import 'package:gopeed/shared/widgets/virtual_tree_view.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  testWidgets('clearing an explicit sort restores mixed file and directory input order', (tester) async {
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: FileTreeView<int>(
          items: const [
            FileTreeItem(key: 'z', path: '', name: 'z.txt', size: 3, data: 0),
            FileTreeItem(key: 'nested', path: 'folder', name: 'child.txt', size: 2, data: 1),
            FileTreeItem(key: 'a', path: '', name: 'a.txt', size: 1, data: 2),
          ],
        ),
      ),
    );
    await tester.pumpAndSettle();
    List<String> labels() => tester
        .widget<VirtualTreeView<FileTreeNode<int>>>(find.byType(VirtualTreeView<FileTreeNode<int>>))
        .nodes
        .cast<shad.TreeItemNode<FileTreeNode<int>>>()
        .map((node) => node.data.label)
        .toList();
    expect(labels(), ['z.txt', 'folder', 'a.txt']);
    final sort = find.byKey(const ValueKey('file-tree-sort-name-asc'));
    await tester.tap(sort);
    await tester.pump();
    expect(labels(), ['folder', 'a.txt', 'z.txt']);
    await tester.tap(sort);
    await tester.pump();
    expect(labels(), ['folder', 'z.txt', 'a.txt']);
    await tester.tap(sort);
    await tester.pump();
    expect(labels(), ['z.txt', 'folder', 'a.txt']);
  });

  for (final width in [390.0, 700.0, 1024.0]) {
    testWidgets('long directory display labels remain readable at $width', (tester) async {
      tester.view.physicalSize = Size(width, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final name = List.filled(12, 'long-resource-title').join('-');
      final nestedName = List.filled(14, 'nested-directory-title').join('-');
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: FileTreeView<int>(
            items: const [FileTreeItem(key: 'child', path: 'source-0/nested', name: 'a.txt', size: 10, data: 0)],
            directoryLabels: {'source-0': name, 'source-0/nested': nestedName},
          ),
        ),
      );
      await tester.pumpAndSettle();
      for (final label in [name, nestedName]) {
        final paragraph = tester.renderObject<RenderParagraph>(find.text(label));
        expect(paragraph.size.width, greaterThanOrEqualTo(paragraph.getMaxIntrinsicWidth(double.infinity)));
      }
      expect(find.byKey(const ValueKey('file-tree-horizontal-scrollbar')), findsOneWidget);
      final root = find.byKey(const ValueKey('file-tree-name-scroll-folder:source-0'));
      final before = tester.getTopLeft(find.text(name));
      final trailing = tester.getRect(find.byKey(const ValueKey('file-tree-trailing-folder:source-0')));
      await tester.drag(root, const Offset(-120, 0));
      await tester.pumpAndSettle();
      expect(tester.getTopLeft(find.text(name)).dx, lessThan(before.dx));
      expect(tester.getRect(find.byKey(const ValueKey('file-tree-trailing-folder:source-0'))), trailing);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('empty directory root keeps its identity and expansion when files arrive', (tester) async {
    late StateSetter update;
    var items = [
      const FileTreeItem(key: 'root', path: 'root', name: 'pending', size: 0, data: -1, directoryOnly: true),
    ];
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: StatefulBuilder(
          builder: (context, setState) {
            update = setState;
            return FileTreeView<int>(
              items: items,
              directoryLabels: const {'root': 'Source'},
              initialExpandedDepth: 1,
              iconWidgetBuilder: (_, node) => node.isFolder ? const Text('status') : const Text('file'),
            );
          },
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Source'), findsOneWidget);
    expect(find.text('pending'), findsNothing);
    expect(find.text('status'), findsOneWidget);
    update(() => items = [const FileTreeItem(key: 'child', path: 'root', name: 'payload.txt', size: 12, data: 0)]);
    await tester.pumpAndSettle();
    expect(find.text('Source'), findsOneWidget);
    expect(find.text('payload.txt'), findsOneWidget);
    expect(find.text('file'), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('file-tree-node-expand-folder:root')));
    await tester.pumpAndSettle();
    expect(find.text('payload.txt'), findsNothing);
    expect(find.text('Source'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('live tree updates preserve a collapse requested during its animation', (tester) async {
    late StateSetter update;
    var items = const [
      FileTreeItem(key: 'first', path: 'bundle', name: 'first.txt', size: 1, data: 0),
      FileTreeItem(key: 'second', path: 'bundle', name: 'second.txt', size: 1, data: 1),
      FileTreeItem(key: 'pending', path: 'source-b', name: 'source B', size: 0, data: 2, directoryOnly: true),
    ];
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: StatefulBuilder(
          builder: (context, setState) {
            update = setState;
            return FileTreeView<int>(
              items: items,
              directoryLabels: const {'bundle': 'bundle', 'source-b': 'source B'},
              initialExpandedDepth: 1,
            );
          },
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const ValueKey('file-tree-node-expand-folder:bundle')));
    await tester.pump(const Duration(milliseconds: 50));
    update(() {
      items = const [
        FileTreeItem(key: 'first', path: 'bundle', name: 'first.txt', size: 1, data: 0),
        FileTreeItem(key: 'second', path: 'bundle', name: 'second.txt', size: 1, data: 1),
        FileTreeItem(key: 'arrived', path: '', name: 'later.txt', size: 1, data: 3),
      ];
    });
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 250));

    final tree = tester.widget<VirtualTreeView<FileTreeNode<int>>>(find.byType(VirtualTreeView<FileTreeNode<int>>));
    final bundle = tree.nodes.cast<shad.TreeItemNode<FileTreeNode<int>>>().first;
    expect(bundle.data.label, 'bundle');
    expect(bundle.expanded, isFalse);
    expect(find.text('later.txt'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
