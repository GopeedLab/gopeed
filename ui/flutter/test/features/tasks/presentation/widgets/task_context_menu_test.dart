import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/task.dart' as api_task;
import 'package:gopeed/features/tasks/domain/task_record.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_context_menu.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/widgets/app_text_field.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../support/context_menu_event.dart';

void main() {
  testWidgets('task actions remain enabled by status and run once after reopening', (tester) async {
    final calls = <String, int>{};
    await tester.pumpWidget(_host(calls));
    await tester.pumpAndSettle();

    await _open(tester, 'paused');
    expect(_button(tester, 'resume').enabled, isTrue);
    expect(_button(tester, 'pause').enabled, isFalse);
    await tester.tap(find.byKey(const ValueKey('task-context-resume')));
    await tester.pumpAndSettle();
    expect(calls, {'paused.resume': 1});

    await _open(tester, 'paused');
    await tester.tap(find.byKey(const ValueKey('task-context-details')));
    await tester.pumpAndSettle();
    expect(calls['paused.details'], 1);

    await _open(tester, 'downloading');
    expect(_button(tester, 'resume').enabled, isFalse);
    expect(_button(tester, 'pause').enabled, isTrue);
    await tester.tap(find.byKey(const ValueKey('task-context-pause')));
    await tester.pumpAndSettle();
    expect(calls['downloading.pause'], 1);

    await _open(tester, 'paused');
    await tester.tap(find.byKey(const ValueKey('task-context-select')));
    await tester.pumpAndSettle();
    expect(calls['paused.select'], 1);
    expect(calls.values, everyElement(1));
  });

  testWidgets('task menus close on outside click and Escape and reopen after removal', (tester) async {
    final calls = <String, int>{};
    await tester.pumpWidget(_host(calls));
    await tester.pumpAndSettle();
    await _open(tester, 'paused');
    await tester.tapAt(_blank);
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('task-context-details')), findsNothing);
    expect(calls, isEmpty, reason: 'outside click only closes the menu');

    await _open(tester, 'downloading');
    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('task-context-details')), findsNothing);
    expect(calls, isEmpty, reason: 'Escape only closes the menu');

    await tester.pumpWidget(_host(calls, showTasks: false));
    await tester.pumpAndSettle();
    await tester.pumpWidget(_host(calls));
    await tester.pumpAndSettle();
    await _open(tester, 'paused');
    await tester.tap(find.byKey(const ValueKey('task-context-details')));
    await tester.pumpAndSettle();
    expect(calls, {'paused.details': 1});
  });

  testWidgets('Web cancels only task and popup contextmenu events and cleans up after removal', (tester) async {
    final calls = <String, int>{};
    await tester.pumpWidget(_host(calls));
    await tester.pumpAndSettle();
    expect(BrowserContextMenu.enabled, isTrue);
    final firstPosition = tester.getCenter(find.byKey(const ValueKey('paused')));
    final secondPosition = tester.getCenter(find.byKey(const ValueKey('downloading')));
    final inputPosition = tester.getCenter(find.byType(AppTextField));

    // Unlike a Flutter gesture, this observes cancellation of a DOM event.
    expect(dispatchContextMenu(firstPosition, tester.view.viewId), isTrue);
    expect(dispatchContextMenu(secondPosition, tester.view.viewId), isTrue);
    expect(dispatchContextMenu(inputPosition, tester.view.viewId), isFalse);
    expect(dispatchContextMenu(_blank, tester.view.viewId), isFalse);

    await _open(tester, 'paused');
    final popupPosition = tester.getCenter(find.byKey(const ValueKey('task-context-details')));
    expect(dispatchContextMenu(popupPosition, tester.view.viewId), isTrue);
    expect(dispatchContextMenu(_blank, tester.view.viewId), isFalse);

    await tester.tap(find.byKey(const ValueKey('task-context-update-url')));
    // Submenus follow their anchor with a persistent ticker while open.
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 200));
    final manualUpdate = find.text('Update manually');
    expect(manualUpdate, findsOneWidget);
    expect(dispatchContextMenu(tester.getCenter(manualUpdate), tester.view.viewId), isTrue);

    // shadcn keeps an already-open overlay alive when its anchor is removed.
    // Its own captured region must survive independently of the task cards.
    await tester.pumpWidget(_host(calls, showTasks: false));
    await tester.pump(const Duration(milliseconds: 200));
    expect(find.byType(TaskContextMenu), findsNothing);
    expect(find.byKey(const ValueKey('task-context-details')), findsOneWidget);
    expect(dispatchContextMenu(popupPosition, tester.view.viewId), isTrue);
    expect(dispatchContextMenu(firstPosition, tester.view.viewId), isFalse);

    await tester.tapAt(_blank);
    await tester.pumpAndSettle();
    expect(dispatchContextMenu(popupPosition, tester.view.viewId), isFalse);
    expect(dispatchContextMenu(inputPosition, tester.view.viewId), isFalse);
    expect(BrowserContextMenu.enabled, isTrue);

    await tester.pumpWidget(_host(calls, showTasks: false));
    await tester.pumpAndSettle();
    expect(dispatchContextMenu(firstPosition, tester.view.viewId), isFalse);
    expect(dispatchContextMenu(secondPosition, tester.view.viewId), isFalse);
    expect(dispatchContextMenu(inputPosition, tester.view.viewId), isFalse);

    await tester.pumpWidget(_host(calls));
    await tester.pumpAndSettle();
    expect(dispatchContextMenu(firstPosition, tester.view.viewId), isTrue);
    await _open(tester, 'downloading');
    await tester.tap(find.byKey(const ValueKey('task-context-select')));
    await tester.pumpAndSettle();
    expect(calls, {'downloading.select': 1});
    expect(BrowserContextMenu.enabled, isTrue);

    await _open(tester, 'paused');
    await tester.pumpWidget(const SizedBox.expand());
    await tester.pumpAndSettle();
    expect(dispatchContextMenu(firstPosition, tester.view.viewId), isFalse);
    expect(dispatchContextMenu(popupPosition, tester.view.viewId), isFalse);
    expect(dispatchContextMenu(inputPosition, tester.view.viewId), isFalse);
  }, skip: !kIsWeb);
}

const _blank = Offset(700, 500);

Future<void> _open(WidgetTester tester, String id) async {
  await tester.tapAt(tester.getCenter(find.byKey(ValueKey(id))), buttons: kSecondaryMouseButton);
  await tester.pumpAndSettle();
  expect(find.byKey(const ValueKey('task-context-details')), findsOneWidget);
}

shad.MenuButton _button(WidgetTester tester, String name) {
  return tester.widget<shad.MenuButton>(find.byKey(ValueKey('task-context-$name')));
}

Widget _host(Map<String, int> calls, {bool showTasks = true}) {
  return shad.ShadcnApp(
    disableBrowserContextMenu: false,
    theme: AppTheme.light().copyWith(platform: () => TargetPlatform.windows),
    materialTheme: AppTheme.materialLight(),
    home: AppComponentThemes(
      child: shad.Scaffold(
        child: Stack(
          children: [
            if (showTasks)
              for (final (id, status, top) in [
                ('paused', TaskStatus.paused, 20.0),
                ('downloading', TaskStatus.downloading, 100.0),
              ])
                Positioned(
                  left: 20,
                  top: top,
                  child: TaskContextMenu(
                    key: ValueKey('menu-$id'),
                    task: TaskRecord(
                      id: id,
                      name: '$id.zip',
                      status: status,
                      downloaded: '0 B',
                      url: 'https://example.com/$id.zip',
                      storagePath: '/downloads/$id.zip',
                      files: const [],
                      uploading: false,
                      protocol: api_task.Protocol.http,
                    ),
                    actions: _actions(id, calls),
                    child: SizedBox(
                      key: ValueKey(id),
                      width: 240,
                      height: 64,
                      child: const ColoredBox(color: Color(0xFFFFFFFF)),
                    ),
                  ),
                ),
            const Positioned(
              left: 20,
              top: 220,
              width: 240,
              child: AppTextField(initialValue: 'https://example.com/download'),
            ),
          ],
        ),
      ),
    ),
  );
}

TaskContextActions _actions(String id, Map<String, int> calls) {
  void record(String name) => calls.update('$id.$name', (count) => count + 1, ifAbsent: () => 1);
  return TaskContextActions(
    selected: false,
    allSelected: false,
    listeningForUpdate: false,
    onShowDetails: () => record('details'),
    onOpenFile: () => record('openFile'),
    onOpenDirectory: () => record('openDirectory'),
    onBrowseFiles: () => record('browseFiles'),
    onToggleSelectAll: () => record('selectAll'),
    onToggleSelected: () => record('select'),
    onPause: () => record('pause'),
    onResume: () => record('resume'),
    onDelete: () => record('delete'),
    onUpdateUrl: () => record('updateUrl'),
    onToggleUpdateListener: () => record('listen'),
  );
}
