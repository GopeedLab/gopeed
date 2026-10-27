import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_delete_dialog.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  for (final width in [390.0, 700.0, 1024.0]) {
    testWidgets('group deletion confirms affected downloads and files at width $width', (tester) async {
      tester.view.physicalSize = Size(width, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      bool? keepFiles;
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: Builder(
            builder: (context) => shad.PrimaryButton(
              onPressed: () async {
                keepFiles = await showTaskDeleteDialog(
                  context,
                  taskCount: 1,
                  keepFiles: true,
                  memberCount: 65,
                  fileCount: 65,
                );
              },
              child: const Text('Open delete dialog'),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Open delete dialog'));
      await tester.pumpAndSettle();
      expect(find.text('Affects 65 downloads and 65 files.'), findsOneWidget);
      expect(keepFiles, isNull);
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();
      expect(keepFiles, isTrue);
      expect(tester.takeException(), isNull);
    });
  }
}
