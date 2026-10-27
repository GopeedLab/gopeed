import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/features/tasks/presentation/widgets/resolve_file_tree.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
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
