import 'package:flutter/widgets.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/utils/breakpoints.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_tooltip.dart';
import 'package:gopeed/shared/widgets/file_tree_horizontal_viewport.dart';
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

  for (final width in [320.0, 390.0, 700.0, 1024.0]) {
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
        expect(paragraph.text.toPlainText(), label);
        expect(paragraph.overflow, TextOverflow.ellipsis);
        expect(find.byWidgetPredicate((widget) => widget is AppTooltip && widget.message == label), findsOneWidget);
      }
      expect(find.byKey(const ValueKey('file-tree-horizontal-scrollbar')), findsOneWidget);
      final root = find.byKey(const ValueKey('file-tree-name-scroll-folder:source-0'));
      final before = tester.getTopLeft(find.text(name));
      final trailing = tester.getRect(find.byKey(const ValueKey('file-tree-trailing-folder:source-0')));
      final expand = find.byKey(const ValueKey('file-tree-node-expand-folder:source-0'));
      final expandBefore = tester.getTopLeft(expand);
      final header = find.byKey(const ValueKey('file-tree-sort-size-asc'));
      final headerBefore = tester.getTopLeft(header);
      await tester.dragFrom(tester.getTopLeft(root) + const Offset(40, 10), const Offset(-120, 0));
      await tester.pumpAndSettle();
      final delta = tester.getTopLeft(find.text(name)).dx - before.dx;
      expect(delta, lessThan(0));
      final trailingAfter = tester.getRect(find.byKey(const ValueKey('file-tree-trailing-folder:source-0')));
      if (width < Breakpoints.mobile) {
        expect(trailingAfter.left - trailing.left, closeTo(delta, 0.01));
        expect(tester.getTopLeft(expand).dx - expandBefore.dx, closeTo(delta, 0.01));
        expect(tester.getTopLeft(header).dx - headerBefore.dx, closeTo(delta, 0.01));
      } else {
        expect(trailingAfter, trailing);
        expect(tester.getTopLeft(expand), expandBefore);
        expect(tester.getTopLeft(header), headerBefore);
      }
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('deep trees scroll all controls and keep their canvas stable when folders toggle', (tester) async {
    tester.view.physicalSize = const Size(1024, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    const name = '课程视频与配套练习_最终修订版.mp4';
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Center(
          child: SizedBox(
            width: 320,
            height: 500,
            child: FileTreeView<int>(
              items: const [FileTreeItem(key: 'deep', path: 'a/b/c/d/e/f', name: name, size: 1024, data: 0)],
              leadingBuilder: (_, _) => const SizedBox(width: 16, height: 16),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    // The list's width, rather than the desktop window's width, chooses the mode.
    expect(find.byType(FileTreeHorizontalViewport), findsOneWidget);
    final horizontal = tester.widget<SingleChildScrollView>(find.byKey(const ValueKey('file-tree-horizontal-scroll')));
    final controller = horizontal.controller!;
    final extent = controller.position.maxScrollExtent;
    expect(extent, greaterThan(0));
    final paragraph = tester.renderObject<RenderParagraph>(find.text(name));
    expect(paragraph.overflow, TextOverflow.ellipsis);
    expect(paragraph.text.toPlainText(), name);
    final root = find.byKey(const ValueKey('file-tree-node-expand-folder:a'));
    final trailing = find.byKey(const ValueKey('file-tree-trailing-file:deep'));
    final before = tester.getTopLeft(trailing);
    final label = find.byKey(const ValueKey('file-tree-name-scroll-folder:a'));
    await tester.dragFrom(tester.getTopLeft(label) + const Offset(20, 8), const Offset(-100, 0));
    await tester.pumpAndSettle();
    expect(tester.getTopLeft(trailing).dx, lessThan(before.dx));
    controller.jumpTo(controller.position.maxScrollExtent);
    await tester.pump();
    final viewport = tester.getRect(find.byType(FileTreeHorizontalViewport));
    expect(tester.getRect(trailing).right, lessThanOrEqualTo(viewport.right));
    expectNameTailVisible(tester, name, viewport);
    controller.jumpTo(0);
    await tester.pump();
    await tester.tap(root);
    await tester.pumpAndSettle();
    expect(find.text(name), findsNothing);
    expect(find.byType(FileTreeHorizontalViewport), findsOneWidget);
    expect(controller.position.maxScrollExtent, extent);
    await tester.tap(root);
    await tester.pumpAndSettle();
    expect(find.text(name), findsOneWidget);
    expect(controller.position.maxScrollExtent, extent);
    expect(tester.takeException(), isNull);
  });

  for (final width in [320.0, 1024.0]) {
    testWidgets('long names show ellipsis, reveal their tail on scroll and expose full text at $width', (tester) async {
      tester.view.physicalSize = Size(width, 600);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final name = '${List.filled(8, '课程视频与完整配套资料').join('_')}.mp4';
      var presses = 0;
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: FileTreeView<int>(
            items: [FileTreeItem(key: 'long', path: '', name: name, size: 1024, data: 0)],
            onNodePressed: (_) => presses++,
          ),
        ),
      );
      await tester.pumpAndSettle();
      final label = find.text(name);
      final paragraph = tester.renderObject<RenderParagraph>(label);
      expect(paragraph.didExceedMaxLines, isTrue);
      expect(paragraph.overflow, TextOverflow.ellipsis);
      expect(paragraph.text.toPlainText(), name);
      expect(
        paragraph.getBoxesForSelection(TextSelection(baseOffset: name.length - 4, extentOffset: name.length)),
        isEmpty,
      );

      final horizontal = tester.widget<SingleChildScrollView>(
        find.byKey(const ValueKey('file-tree-horizontal-scroll')),
      );
      horizontal.controller!.jumpTo(horizontal.controller!.position.maxScrollExtent);
      await tester.pumpAndSettle();
      expectNameTailVisible(tester, name, Rect.fromLTWH(0, 0, width, 600));
      horizontal.controller!.jumpTo(0);
      await tester.pump();
      await tester.longPressAt(tester.getTopLeft(label) + const Offset(20, 8));
      await tester.pumpAndSettle();
      expect(find.text(name), findsNWidgets(2));
      expect(presses, 0);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('resizing changes scrolling mode without losing folder state or tap handling', (tester) async {
    tester.view.physicalSize = const Size(1024, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    late StateSetter update;
    var width = 390.0;
    final pressed = <String>[];
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: StatefulBuilder(
          builder: (context, setState) {
            update = setState;
            return Center(
              child: SizedBox(
                width: width,
                height: 500,
                child: FileTreeView<int>(
                  items: const [FileTreeItem(key: 'file', path: 'bundle', name: 'file.txt', size: 10, data: 0)],
                  directoryLabels: const {
                    'bundle': 'Long resource title that needs horizontal scrolling in both desktop and mobile views',
                  },
                  onNodePressed: (node) => pressed.add(node.key),
                ),
              ),
            );
          },
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byType(FileTreeHorizontalViewport), findsOneWidget);
    await tester.tapAt(tester.getTopLeft(find.text('file.txt')) + const Offset(20, 8));
    await tester.pump();
    expect(pressed, ['file:file']);
    final expand = find.byKey(const ValueKey('file-tree-node-expand-folder:bundle'));
    await tester.tap(expand);
    await tester.pumpAndSettle();
    update(() => width = 900);
    await tester.pumpAndSettle();
    expect(find.byType(FileTreeHorizontalViewport), findsNothing);
    expect(find.text('file.txt'), findsNothing);
    update(() => width = 390);
    await tester.pumpAndSettle();
    expect(find.byType(FileTreeHorizontalViewport), findsOneWidget);
    expect(find.text('file.txt'), findsNothing);
    await tester.tap(expand);
    await tester.pumpAndSettle();
    await tester.tapAt(tester.getTopLeft(find.text('file.txt')) + const Offset(20, 8));
    await tester.pump();
    expect(pressed, ['file:file', 'file:file']);
    expect(tester.takeException(), isNull);
  });

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

void expectNameTailVisible(WidgetTester tester, String name, Rect viewport) {
  final paragraph = tester.renderObject<RenderParagraph>(find.text(name));
  final boxes = paragraph.getBoxesForSelection(TextSelection(baseOffset: name.length - 4, extentOffset: name.length));
  expect(boxes, isNotEmpty);
  for (final box in boxes) {
    expect(paragraph.localToGlobal(Offset(box.left, 0)).dx, greaterThanOrEqualTo(viewport.left));
    expect(paragraph.localToGlobal(Offset(box.right, 0)).dx, lessThanOrEqualTo(viewport.right));
  }
}
