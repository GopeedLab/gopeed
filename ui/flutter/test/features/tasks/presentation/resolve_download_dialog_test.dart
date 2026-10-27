import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
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
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

void main() {
  testWidgets('skipped failed source can resolve again and be included without repeating successes', (tester) async {
    tester.view.physicalSize = const Size(900, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final ready =
        PendingDownload(
            id: 'ready',
            request: Request(url: 'https://example.com/ready'),
            options: Options(),
          )
          ..result = ResolveResult(
            id: 'ready-rid',
            res: Resource(files: [FileInfo(name: 'ready.txt')]),
          )
          ..selected.add(0);
    final failed = PendingDownload(
      id: 'failed',
      request: Request(url: 'https://example.com/failed'),
      options: Options(),
    )..resolveError = 'resolve failed';
    final created = <String>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(
        GopeedMethods.resolve,
        (_) => ResolveResult(
          id: 'retry-rid',
          res: Resource(files: [FileInfo(name: 'retry.txt')]),
        ),
      )
      ..bind(GopeedMethods.createTask, (request) {
        created.add(request.rid!);
        return 'created';
      });
    bool? result;
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Builder(
          builder: (context) => shad.PrimaryButton(
            onPressed: () async {
              result = await showResolveDownloadsDialog(
                context,
                sources: [ready, failed],
                service: GopeedService(LocalCapabilityInvoker(registry)),
                directoryManuallySet: false,
              );
            },
            child: const Text('Open'),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(shad.Checkbox).last);
    await tester.pump();
    expect(failed.included, isFalse);
    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();
    expect(failed.result, isNotNull);
    expect(failed.included, isFalse);
    expect(find.text('retry.txt'), findsOneWidget); // Restorable source label remains visible.
    final treeCheckboxes = find.descendant(of: find.byType(ResolveFileTree), matching: find.byType(shad.Checkbox));
    await tester.tap(treeCheckboxes.last);
    await tester.pump();
    expect(failed.selected, {0});
    await tester.tap(treeCheckboxes.last);
    await tester.pump();
    expect(ready.selected, {0});
    await tester.tap(find.byType(shad.Checkbox).last);
    await tester.pumpAndSettle();
    expect(failed.included, isTrue);
    await tester.tap(find.byKey(const ValueKey('resolve-create-button')));
    await tester.pumpAndSettle();
    expect(created, ['ready-rid', 'retry-rid']);
    expect(result, isTrue);
    expect(tester.takeException(), isNull);
  });
}
