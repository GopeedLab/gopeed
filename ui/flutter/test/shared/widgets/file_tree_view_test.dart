import 'package:flutter/widgets.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/file_tree_view.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
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
}
