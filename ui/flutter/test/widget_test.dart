import 'package:flutter/material.dart' show Icons, Scrollbar;
import 'package:flutter/gestures.dart' show PointerDeviceKind, kDoubleTapMinTime, kSecondaryMouseButton;
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;
import 'package:gopeed/app/app.dart';
import 'package:gopeed/app/application/app_appearance_controller.dart';
import 'package:gopeed/app/application/app_deep_link_controller.dart';
import 'package:gopeed/app/application/app_notification_controller.dart';
import 'package:gopeed/app/application/app_platform_controller.dart';
import 'package:gopeed/app/application/app_runtime_controller.dart';
import 'package:gopeed/api/model/create_task.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/api/model/extension.dart' as api_extension;
import 'package:gopeed/api/model/meta.dart';
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:gopeed/api/model/store_extension.dart';
import 'package:gopeed/api/model/task.dart' as api_task;
import 'package:gopeed/core/common/start_config.dart';
import 'package:gopeed/core/icons/gopeed_icons.dart';
import 'package:gopeed/features/home/presentation/widgets/tasks_top_bar.dart';
import 'package:gopeed/features/extensions/application/extensions_controller.dart';
import 'package:gopeed/features/extensions/presentation/pages/extensions_page.dart';
import 'package:gopeed/features/home/presentation/widgets/primary_rail.dart';
import 'package:gopeed/features/tasks/application/pending_update_task.dart';
import 'package:gopeed/features/tasks/application/task_batch_selection_controller.dart';
import 'package:gopeed/features/tasks/application/task_runtime_status_provider.dart';
import 'package:gopeed/features/tasks/application/tasks_controller.dart';
import 'package:gopeed/features/settings/application/settings_controller.dart';
import 'package:gopeed/features/settings/presentation/pages/settings_page.dart';
import 'package:gopeed/features/settings/presentation/widgets/download_categories_setting.dart';
import 'package:gopeed/features/settings/presentation/widgets/settings_item.dart';
import 'package:gopeed/features/settings/presentation/widgets/settings_language_select.dart';
import 'package:gopeed/features/settings/presentation/widgets/settings_list_editor.dart';
import 'package:gopeed/features/tasks/domain/task_record.dart';
import 'package:gopeed/features/tasks/presentation/pages/create_task_window_page.dart';
import 'package:gopeed/features/tasks/presentation/widgets/resolve_file_tree.dart';
import 'package:gopeed/features/tasks/presentation/widgets/speed_monitor_card.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_batch_selection_builder.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_file_tree.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_file_manager.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_progress_bar.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_card/task_card.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_context_menu.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_delete_dialog.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_drawer.dart';
import 'package:gopeed/features/tasks/presentation/widgets/pending_update_dialog.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_statistics/piece_map.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_statistics/peer_table.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_statistics/task_statistics_tab.dart';
import 'package:gopeed/api/model/task_stats.dart';
import 'package:gopeed/l10n/l10n.dart';
import 'package:gopeed/shared/theme/app_palette.dart';
import 'package:gopeed/shared/theme/app_component_themes.dart';
import 'package:gopeed/shared/theme/app_design_tokens.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:gopeed/shared/theme/app_theme_color.dart';
import 'package:gopeed/shared/widgets/app_primary_button.dart';
import 'package:gopeed/shared/widgets/app_toast.dart';
import 'package:gopeed/shared/widgets/gopeed_app_mark.dart';
import 'package:gopeed/shared/widgets/responsive_menu_layout.dart';
import 'package:gopeed/shared/widgets/virtual_tree_view.dart';
import 'package:gopeed/util/updater.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized().platformDispatcher.localeTestValue = const Locale('en');

  testWidgets('tasks page renders key content', (WidgetTester tester) async {
    tester.view.physicalSize = const Size(1024, 768);
    tester.view.devicePixelRatio = 1;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          tasksControllerProvider.overrideWith(FakeTasksController.new),
        ],
        child: const GopeedApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Downloading'), findsOneWidget);
    expect(find.text('Completed'), findsOneWidget);
    expect(find.text('Failed'), findsOneWidget);
    expect(find.text('Add Task'), findsOneWidget);
    expect(find.text('No tasks in this list'), findsOneWidget);
    expect(find.text('Create Task'), findsNothing);
    expect(find.byKey(const ValueKey('task-empty-illustration')), findsOneWidget);
    expect(find.text('Create a task or refresh the list.'), findsNothing);
    expect(find.text('Refresh'), findsNothing);
    expect(find.textContaining('ACTIVE CONNECTIONS'), findsNothing);
    expect(find.ancestor(of: find.text('Add Task'), matching: find.byType(AppPrimaryButton)), findsOneWidget);
    final taskActions = find.byKey(const ValueKey('tasks-top-action-buttons'));
    expect(find.descendant(of: taskActions, matching: find.byType(shad.IconButton)), findsNWidgets(4));
    expect(
      tester.widget<Container>(find.byKey(const ValueKey('primary-rail-active-indicator'))).color,
      AppPalette.light.textPrimary,
    );
    expect(find.byType(GopeedAppMark), findsOneWidget);
    expect(find.byKey(const ValueKey('primary-rail-app-mark')), findsOneWidget);
    expect(tester.getSize(find.byKey(const ValueKey('primary-rail-app-mark'))), const Size.square(24));
    expect(find.descendant(of: find.byType(GopeedAppMark), matching: find.byType(CustomPaint)), findsOneWidget);
    expect(find.descendant(of: find.byType(GopeedAppMark), matching: find.byType(Image)), findsNothing);
    expect(find.text('M'), findsNothing);
    expect(find.text('V1.0'), findsNothing);
    expect(
      tester.getSize(find.byKey(const ValueKey('secondary-navigation-pane'))).width,
      AppDesignTokens.filterSidebarWidth,
    );
    expect(tester.getSize(find.byKey(const ValueKey('speed-monitor-gauge'))), const Size(60, 32));
    final gaugeRect = tester.getRect(find.byKey(const ValueKey('speed-monitor-gauge')));
    final downloadLineRect = tester.getRect(find.byKey(const ValueKey('speed-monitor-download-line')));
    final valuesRect = tester.getRect(find.byKey(const ValueKey('speed-monitor-values')));
    expect(gaugeRect.right, lessThan(downloadLineRect.left));
    expect(gaugeRect.center.dy, closeTo(valuesRect.center.dy, 0.01));
    expect(find.text('B/s'), findsNWidgets(2));
    expect(
      find.descendant(of: find.byKey(const ValueKey('speed-monitor-values')), matching: find.byType(Icon)),
      findsNWidgets(2),
    );
  });

  testWidgets('task loading failures use the empty state without retry controls', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 768));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          tasksControllerProvider.overrideWith(FailingTasksController.new),
        ],
        child: const GopeedApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('No tasks in this list'), findsOneWidget);
    expect(find.byKey(const ValueKey('task-empty-illustration')), findsOneWidget);
    expect(find.text('Retry'), findsNothing);
    expect(find.text('Unable to load tasks'), findsNothing);
  });

  testWidgets('task search uses a specific empty result message', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 768));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          tasksControllerProvider.overrideWith(RecordingTasksController.new),
        ],
        child: const GopeedApp(),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));

    await tester.enterText(find.byType(shad.TextField), 'not-a-task');
    await tester.pump();

    expect(find.text('No matching tasks found'), findsOneWidget);
    expect(find.text('No tasks in this list'), findsNothing);
  });

  testWidgets('tasks page mobile header places actions above text tabs', (WidgetTester tester) async {
    tester.view.physicalSize = const Size(390, 760);
    tester.view.devicePixelRatio = 1;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          tasksControllerProvider.overrideWith(FakeTasksController.new),
        ],
        child: const GopeedApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Tasks'), findsNWidgets(2));
    expect(find.text('Downloading 0'), findsOneWidget);
    expect(find.text('Completed 0'), findsOneWidget);
    expect(find.text('Failed 0'), findsOneWidget);
  });

  testWidgets('extensions grid adds columns from a minimum card width and keeps desktop toolbar aligned', (
    WidgetTester tester,
  ) async {
    await _setTestSize(tester, const Size(1100, 900));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [extensionsControllerProvider.overrideWith(FakeExtensionsController.new)],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const ExtensionsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    int gridColumns() {
      final grid = tester.widget<SliverGrid>(find.byType(SliverGrid));
      return (grid.gridDelegate as SliverGridDelegateWithFixedCrossAxisCount).crossAxisCount;
    }

    expect(gridColumns(), 3);
    final searchRect = tester.getRect(find.byKey(const ValueKey('extension-search-input')));
    final sortRect = tester.getRect(find.byKey(const ValueKey('extension-sort-control')));
    final developRect = tester.getRect(find.byKey(const ValueKey('develop-extension-button')));
    final installRect = tester.getRect(find.byKey(const ValueKey('install-extension-button')));
    final cardRect = tester.getRect(find.byKey(const ValueKey('extension-card-extension-0')));
    expect(searchRect.width, closeTo(cardRect.width, 0.01));
    expect(sortRect.left, closeTo(searchRect.right + 10, 0.01));
    expect(sortRect.center.dy, closeTo(searchRect.center.dy, 0.01));
    expect(installRect.center.dy, closeTo(searchRect.center.dy, 0.01));
    expect(developRect.right, lessThan(installRect.left));
    expect(installRect.left, greaterThan(sortRect.right));

    tester.view.physicalSize = const Size(1400, 900);
    await tester.pumpAndSettle();
    expect(gridColumns(), 4);
    expect(tester.takeException(), isNull);
  });

  testWidgets('extensions mobile toolbar gives search its own row and keeps actions split below', (
    WidgetTester tester,
  ) async {
    await _setTestSize(tester, const Size(390, 760));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [extensionsControllerProvider.overrideWith(FakeExtensionsController.new)],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const ExtensionsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final searchRect = tester.getRect(find.byKey(const ValueKey('extension-search-input')));
    final sortRect = tester.getRect(find.byKey(const ValueKey('extension-sort-control')));
    final developRect = tester.getRect(find.byKey(const ValueKey('develop-extension-button')));
    final installRect = tester.getRect(find.byKey(const ValueKey('install-extension-button')));
    final cardRect = tester.getRect(find.byKey(const ValueKey('extension-card-extension-0')));

    expect(searchRect.width, closeTo(358, 0.01));
    expect(searchRect.width, closeTo(cardRect.width, 0.01));
    expect(sortRect.top, greaterThan(searchRect.bottom));
    expect(sortRect.left, closeTo(searchRect.left, 0.01));
    expect(installRect.right, closeTo(searchRect.right, 0.01));
    expect(installRect.center.dy, closeTo(sortRect.center.dy, 0.01));
    expect(developRect.right, lessThan(installRect.left));

    final list = find.byKey(const ValueKey('extensions-list-scroll-view'));
    final scrollable = find.descendant(of: list, matching: find.byType(Scrollable));
    await tester.drag(list, const Offset(0, -300));
    await tester.pumpAndSettle();
    expect(tester.getRect(find.byKey(const ValueKey('extension-search-input'))).top, closeTo(searchRect.top, 0.01));
    expect(tester.state<ScrollableState>(scrollable).position.pixels, greaterThan(0));
    expect(tester.takeException(), isNull);
  });

  testWidgets('removing an extension requires explicit confirmation', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1100, 900));
    late FakeExtensionsController controller;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [extensionsControllerProvider.overrideWith(() => controller = FakeExtensionsController())],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const ExtensionsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final removeButton = find.byKey(const ValueKey('remove-extension-extension-0'));
    await tester.tap(removeButton);
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('remove-extension-dialog')), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('cancel-remove-extension-button')));
    await tester.pumpAndSettle();
    expect(controller.removeCalls, 0);
    expect(find.byKey(const ValueKey('remove-extension-dialog')), findsNothing);

    await tester.tap(removeButton);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('confirm-remove-extension-button')));
    await tester.pumpAndSettle();
    expect(controller.removeCalls, 1);
    expect(find.byKey(const ValueKey('remove-extension-dialog')), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('extension market automatically loads the next page near the list bottom', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(390, 760));
    late PaginatedExtensionsController controller;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [extensionsControllerProvider.overrideWith(() => controller = PaginatedExtensionsController())],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const ExtensionsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Load More'), findsNothing);
    expect(controller.loadMoreCalls, 0);

    await tester.fling(find.byKey(const ValueKey('extensions-list-scroll-view')), const Offset(0, -1400), 3000);
    await tester.pumpAndSettle();

    expect(controller.loadMoreCalls, 1);
    expect(find.text('Load More'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('extension install popover unlocks local loading and installed cards use switches', (
    WidgetTester tester,
  ) async {
    await _setTestSize(tester, const Size(1100, 900));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [extensionsControllerProvider.overrideWith(FakeExtensionsController.new)],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const ExtensionsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(shad.Switch), findsOneWidget);
    expect(find.byKey(const ValueKey('load-local-extension-button')), findsNothing);
    final installButton = find.byKey(const ValueKey('install-extension-button'));
    expect(find.descendant(of: installButton, matching: find.byType(shad.IconButton)), findsOneWidget);

    for (var index = 0; index < 5; index++) {
      await tester.tap(installButton);
      await tester.pump(const Duration(milliseconds: 100));
    }

    expect(find.byKey(const ValueKey('extension-install-popover')), findsOneWidget);
    expect(find.byKey(const ValueKey('extension-install-url-input')), findsOneWidget);
    final localButton = find.byKey(const ValueKey('load-local-extension-button'));
    expect(localButton, findsOneWidget);
    expect(find.descendant(of: localButton, matching: find.byType(shad.IconButton)), findsOneWidget);

    await tester.pump(const Duration(seconds: 3));
    expect(tester.takeException(), isNull);
  });

  testWidgets('responsive menu uses sidebar on desktop and two-level navigation on mobile', (
    WidgetTester tester,
  ) async {
    await _setTestSize(tester, const Size(1024, 768));
    await tester.pumpWidget(const _ResponsiveMenuHarness());
    await tester.pumpAndSettle();

    expect(find.text('SETTINGS'), findsOneWidget);
    expect(find.text('General content'), findsOneWidget);

    await _setTestSize(tester, const Size(390, 760));
    await tester.pumpWidget(const _ResponsiveMenuHarness());
    await tester.pumpAndSettle();

    expect(find.text('SETTINGS'), findsOneWidget);
    expect(find.text('General content'), findsNothing);

    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();

    expect(find.text('Advanced content'), findsOneWidget);
    expect(find.text('SETTINGS'), findsNothing);
  });

  testWidgets('settings use three top-level entries and ordered groups', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 768));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(FakePlatformController.new),
          settingsControllerProvider.overrideWith(FakeSettingsController.new),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Basic'), findsWidgets);
    expect(find.text('Downloads'), findsOneWidget);
    expect(find.text('Advanced'), findsOneWidget);
    expect(find.text('网络设置'), findsNothing);
    expect(
      tester.getSize(find.byKey(const ValueKey('secondary-navigation-pane'))).width,
      AppDesignTokens.filterSidebarWidth,
    );
    expect(find.byKey(const ValueKey('theme-mode-system')), findsOneWidget);
    expect(find.text('Light'), findsOneWidget);
    expect(find.text('Dark'), findsOneWidget);
    expect(find.text('Accent Color'), findsOneWidget);
    expect(find.text('强调色'), findsNothing);
    expect(find.byKey(const ValueKey('theme-color-green')), findsOneWidget);
    expect(find.byKey(const ValueKey('theme-color-purple')), findsOneWidget);
    expect(find.byKey(const ValueKey('settings-language-select')), findsOneWidget);
    expect(find.text('Browser Extension'), findsOneWidget);
    expect(find.byKey(const ValueKey('browser-extension-chrome')), findsOneWidget);
    expect(find.byKey(const ValueKey('browser-extension-edge')), findsOneWidget);
    expect(find.byKey(const ValueKey('browser-extension-firefox')), findsOneWidget);
    expect(find.text('Version'), findsOneWidget);
    expect(find.text('Current version v-'), findsOneWidget);
    expect(find.byKey(const ValueKey('check-app-update-button')), findsOneWidget);
    expect(find.text('Check for Updates'), findsOneWidget);
    expect(find.text('Usage Analytics'), findsOneWidget);
    expect(find.text('Contributors'), findsOneWidget);
    expect(find.byKey(const ValueKey('gopeed-contributors')), findsOneWidget);
    expect(tester.getTopLeft(find.text('Notify for updates')).dy, lessThan(tester.getTopLeft(find.text('Version')).dy));
    final swatchCenters = AppThemeColor.values
        .map((color) => tester.getCenter(find.byKey(ValueKey('theme-color-${color.key}'))).dy)
        .toList();
    expect(swatchCenters.reduce((a, b) => a < b ? a : b), closeTo(swatchCenters.first, 0.01));
    expect(swatchCenters.reduce((a, b) => a > b ? a : b), closeTo(swatchCenters.first, 0.01));

    await tester.tap(find.text('Downloads'));
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('download-categories-editor')), findsOneWidget);
    expect(find.text('Archives'), findsOneWidget);
    expect(find.text('Extract Archives Automatically'), findsOneWidget);
    expect(find.text('Delete Archives After Extraction'), findsOneWidget);
    expect(find.text('Delete .torrent file after BT task creation'), findsNothing);
    expect(find.text('HTTP'), findsOneWidget);
    expect(find.text('BitTorrent'), findsOneWidget);
    expect(find.text('ED2K'), findsOneWidget);

    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();
    expect(find.text('Network'), findsOneWidget);
    expect(find.text('API'), findsOneWidget);
    expect(find.text('Developer'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('dependent settings follow the legacy visibility rules', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 900));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(FakePlatformController.new),
          settingsControllerProvider.overrideWith(FakeSettingsController.new),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Downloads'));
    await tester.pumpAndSettle();

    Finder settingSwitch(String label) => find.descendant(
      of: find.ancestor(of: find.text(label), matching: find.byType(SettingsItem)),
      matching: find.byType(shad.Switch),
    );

    await tester.tap(settingSwitch('Extract Archives Automatically'));
    await tester.pumpAndSettle();
    expect(find.text('Delete Archives After Extraction'), findsNothing);

    await tester.tap(settingSwitch('Auto create BT tasks from .torrent files'));
    await tester.pumpAndSettle();
    expect(find.text('Delete .torrent file after BT task creation'), findsOneWidget);

    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();
    expect(find.text('Protocol'), findsOneWidget);
    expect(find.text('Username'), findsNothing);
    await tester.tap(find.text('Custom Proxy'));
    await tester.pumpAndSettle();
    expect(find.text('Protocol'), findsNWidgets(2));
    expect(find.text('Username'), findsOneWidget);
    expect(find.text('Password'), findsOneWidget);
  });

  testWidgets('numeric settings use compact bounded spinner inputs', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 900));
    final settingsController = FakeSettingsController();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(FakePlatformController.new),
          settingsControllerProvider.overrideWith(() => settingsController),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Downloads'));
    await tester.pumpAndSettle();

    final maxRunning = find.byKey(const ValueKey('max-running-input'));
    expect(maxRunning, findsOneWidget);
    expect(find.text('Maximum number of tasks that may download at the same time'), findsOneWidget);
    expect(tester.getSize(maxRunning).width, AppDesignTokens.settingsNumberControlWidth);
    expect(tester.widget<shad.TextField>(maxRunning).features.single, isA<shad.InputSpinnerFeature>());

    await tester.enterText(maxRunning, '999');
    await tester.pump(const Duration(milliseconds: 700));

    expect(tester.widget<shad.TextField>(maxRunning).controller!.text, '256');
    expect(settingsController.state.requireValue.config.maxRunning, 256);

    final seedRatio = find.byKey(const ValueKey('bt-seed-ratio-input'));
    await tester.ensureVisible(seedRatio);
    await tester.pumpAndSettle();
    await tester.enterText(seedRatio, '1.1');
    final seedRatioStepperButtons = find.descendant(of: seedRatio, matching: find.byType(shad.IconButton));
    expect(seedRatioStepperButtons, findsNWidgets(2));
    await tester.tap(seedRatioStepperButtons.first);
    await tester.pump();
    expect(tester.widget<shad.TextField>(seedRatio).controller!.text, '1.2');
    expect(tester.takeException(), isNull);
  });

  testWidgets('ED2K list settings use multiline text and persist comma-separated values', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 900));
    final settingsController = Ed2kSettingsController();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(FakePlatformController.new),
          settingsControllerProvider.overrideWith(() => settingsController),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Downloads'));
    await tester.pumpAndSettle();

    final servers = find.byKey(const ValueKey('ed2k-server-address-input'));
    final serverMet = find.byKey(const ValueKey('ed2k-server-met-input'));
    final nodesDat = find.byKey(const ValueKey('ed2k-nodes-dat-input'));
    expect(tester.widget<shad.TextField>(servers).controller!.text, 'server-a:4661\nserver-b:4661');
    expect(tester.widget<shad.TextField>(servers).maxLines, 5);
    expect(tester.widget<shad.TextField>(serverMet).maxLines, 4);
    expect(tester.widget<shad.TextField>(nodesDat).maxLines, 4);

    await tester.ensureVisible(servers);
    await tester.pumpAndSettle();
    await tester.enterText(servers, 'one.example:4661\ntwo.example:4661');
    await tester.pump(const Duration(milliseconds: 700));
    expect(
      settingsController.state.requireValue.config.protocolConfig.ed2k.serverAddr,
      'one.example:4661,two.example:4661',
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('proxy and API TCP addresses use separate host and port inputs', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 900));
    final settingsController = ProxySettingsController();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(FakePlatformController.new),
          settingsControllerProvider.overrideWith(() => settingsController),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();

    String inputText(String key) => tester.widget<shad.TextField>(find.byKey(ValueKey(key))).controller!.text;

    expect(inputText('proxy-host-input'), 'proxy.example.com');
    expect(inputText('proxy-port-input'), '1080');
    expect(inputText('api-host-input'), '127.0.0.1');
    expect(inputText('api-port-input'), '9999');
    expect(
      tester.widget<shad.TextField>(find.byKey(const ValueKey('proxy-port-input'))).features.single,
      isA<shad.InputSpinnerFeature>(),
    );

    await tester.enterText(find.byKey(const ValueKey('proxy-port-input')), '3128');
    await tester.pump(const Duration(milliseconds: 700));
    expect(settingsController.state.requireValue.config.proxy.host, 'proxy.example.com:3128');
    expect(tester.takeException(), isNull);
  });

  testWidgets('API settings keep a validated draft until the save button is pressed', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 900));
    final runtimeController = RecordingStartConfigRuntimeController();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(() => runtimeController),
          appPlatformControllerProvider.overrideWith(FakePlatformController.new),
          settingsControllerProvider.overrideWith(FakeSettingsController.new),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();

    final host = find.byKey(const ValueKey('api-host-input'));
    final port = find.byKey(const ValueKey('api-port-input'));
    final saveButton = find.byKey(const ValueKey('save-api-config-button'));
    expect(tester.widget<shad.TextField>(host).controller!.text, '192.168.1.20');
    expect(tester.widget<shad.TextField>(port).controller!.text, '4321');
    expect(tester.widget<shad.SecondaryButton>(saveButton).onPressed, isNull);

    await tester.enterText(host, '');
    await tester.enterText(port, '0');
    await tester.pump(const Duration(seconds: 1));
    expect(runtimeController.savedStartConfig, isNull);
    expect(find.text('Unsaved API changes'), findsNothing);
    expect(tester.widget<shad.SecondaryButton>(saveButton).onPressed, isNotNull);

    await tester.ensureVisible(saveButton);
    await tester.pumpAndSettle();
    await tester.tap(saveButton);
    await tester.pumpAndSettle();
    expect(runtimeController.savedStartConfig, isNull);

    await tester.enterText(host, '127.0.0.1');
    await tester.tap(saveButton);
    await tester.pumpAndSettle();
    expect(runtimeController.savedStartConfig?.network, 'tcp');
    expect(runtimeController.savedStartConfig?.address, '127.0.0.1:0');
    expect(tester.widget<shad.SecondaryButton>(saveButton).onPressed, isNull);
    expect(find.text('Restart required'), findsNothing);
    expect(find.text('Auto saved'), findsNothing);
    await tester.pump(const Duration(seconds: 6));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });

  testWidgets('shared toast uses content-driven width', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 700));
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: AppComponentThemes(
          child: Builder(
            builder: (context) => Center(
              child: shad.PrimaryButton(
                onPressed: () => showAppToast(context, '保存成功', type: AppToastType.success),
                child: const Text('Show toast'),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Show toast'));
    await tester.pump(const Duration(milliseconds: 600));

    final toast = find.byKey(const ValueKey('app-toast-content'));
    expect(toast, findsOneWidget);
    expect(tester.getSize(toast).width, lessThan(320));

    await tester.pump(const Duration(seconds: 6));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });

  testWidgets('tracker subscriptions support selecting and clearing all entries', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 900));
    final settingsController = FakeSettingsController();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(FakePlatformController.new),
          settingsControllerProvider.overrideWith(() => settingsController),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Downloads'));
    await tester.pumpAndSettle();

    final selectAll = find.byKey(const ValueKey('tracker-select-all'));
    expect(selectAll, findsOneWidget);
    expect(find.descendant(of: selectAll, matching: find.text('Select All')), findsOneWidget);

    await tester.ensureVisible(selectAll);
    await tester.pumpAndSettle();
    await tester.tap(selectAll);
    await tester.pumpAndSettle();
    expect(settingsController.state.requireValue.config.extra.bt.trackerSubscribeUrls, allTrackerSubscribeUrls);

    await tester.tap(selectAll);
    await tester.pumpAndSettle();
    expect(settingsController.state.requireValue.config.extra.bt.trackerSubscribeUrls, isEmpty);
    expect(tester.takeException(), isNull);
  });

  testWidgets('settings list add buttons align to the right edge', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 900));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(FakePlatformController.new),
          settingsControllerProvider.overrideWith(FakeSettingsController.new),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Advanced'));
    await tester.pumpAndSettle();

    for (final key in const [ValueKey('add-webhook-button'), ValueKey('add-script-button')]) {
      final button = find.byKey(key);
      expect(button, findsOneWidget);
      final editor = find.ancestor(of: button, matching: find.byType(SettingsListEditor));
      expect(tester.getRect(button).right, closeTo(tester.getRect(editor).right, 0.01));
    }
    expect(tester.takeException(), isNull);
  });

  testWidgets('download categories use structured add and list controls', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 900));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(FakePlatformController.new),
          settingsControllerProvider.overrideWith(CategorySettingsController.new),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Downloads'));
    await tester.pumpAndSettle();

    expect(find.text('资料'), findsOneWidget);
    expect(find.text('/tmp/data'), findsOneWidget);
    expect(find.byKey(const ValueKey('add-download-category')), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('add-download-category')));
    await tester.pumpAndSettle();
    expect(find.byType(DownloadCategoriesControl), findsOneWidget);
    await tester.enterText(find.byKey(const ValueKey('download-category-name')), '归档');
    await tester.enterText(find.byKey(const ValueKey('download-category-path')), '/tmp/archive');
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();

    expect(find.text('归档'), findsOneWidget);
    expect(find.text('/tmp/archive'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('edit-download-category-资料')));
    await tester.pumpAndSettle();
    expect(find.text('Edit Category'), findsOneWidget);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(find.text('Edit Category'), findsNothing);
    expect(find.text('资料'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('settings content stretches responsively and stays centered', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1800, 900));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(FakePlatformController.new),
          settingsControllerProvider.overrideWith(FakeSettingsController.new),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    Rect contentRect() => tester.getRect(find.byKey(const ValueKey('settings-body-content')));
    Rect sidebarRect() => tester.getRect(find.byKey(const ValueKey('secondary-navigation-pane')));

    final wideContent = contentRect();
    final wideSidebar = sidebarRect();
    expect(wideContent.width, AppDesignTokens.settingsContentMaxWidth);
    expect(wideContent.left - wideSidebar.right, closeTo(1800 - wideContent.right, 0.01));
    expect(wideContent.left - wideSidebar.right, greaterThan(AppDesignTokens.space24));

    tester.view.physicalSize = const Size(1280, 900);
    await tester.pumpAndSettle();

    final standardDesktopContent = contentRect();
    final standardDesktopSidebar = sidebarRect();
    expect(standardDesktopContent.width, AppDesignTokens.settingsContentMaxWidth);
    expect(
      standardDesktopContent.left - standardDesktopSidebar.right,
      closeTo(1280 - standardDesktopContent.right, 0.01),
    );
    expect(standardDesktopContent.left - standardDesktopSidebar.right, greaterThan(AppDesignTokens.space24));

    tester.view.physicalSize = const Size(910, 900);
    await tester.pumpAndSettle();

    final compactContent = contentRect();
    final compactSidebar = sidebarRect();
    expect(compactContent.left - compactSidebar.right, AppDesignTokens.space24);
    expect(910 - compactContent.right, AppDesignTokens.space24);
    expect(tester.takeException(), isNull);
  });

  testWidgets('settings items keep controls wider while labels grow responsively', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1000, 700));
    const subtitle = '这是一段足够长的设置说明，用于验证窗口变宽以后左侧说明区域会随可用空间一起增长';

    Future<({Size label, Size control})> layoutAt(double width) async {
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: Center(
            child: SizedBox(
              width: width,
              child: const SettingsItem(
                title: '设置项',
                subtitle: subtitle,
                child: SizedBox(key: ValueKey('responsive-setting-control'), width: 220, height: 32),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      return (
        label: tester.getSize(find.text(subtitle)),
        control: tester.getSize(find.byKey(const ValueKey('responsive-setting-control'))),
      );
    }

    final compact = await layoutAt(600);
    final wide = await layoutAt(800);

    expect(wide.label.width, greaterThan(compact.label.width));
    expect(wide.label.height, lessThan(compact.label.height));
    expect(compact.control.width, 220);
    expect(wide.control.width, 220);
  });

  testWidgets('available app update uses a theme-colored icon and text button', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 768));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(AvailableUpdatePlatformController.new),
          settingsControllerProvider.overrideWith(FakeSettingsController.new),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Current version v-'), findsOneWidget);
    expect(find.textContaining('9.9.9'), findsNothing);
    expect(find.byKey(const ValueKey('app-update-available-indicator')), findsNothing);
    expect(find.text('Update'), findsOneWidget);
    final button = find.byKey(const ValueKey('check-app-update-button'));
    expect(find.descendant(of: button, matching: find.byIcon(Icons.system_update_alt_outlined)), findsOneWidget);
    expect(find.ancestor(of: find.text('Update'), matching: find.byType(shad.SecondaryButton)), findsOneWidget);
  });

  testWidgets('checking for updates replaces button content with a centered loader', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 768));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(CheckingUpdatePlatformController.new),
          settingsControllerProvider.overrideWith(FakeSettingsController.new),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    final button = find.byKey(const ValueKey('check-app-update-button'));
    expect(find.descendant(of: button, matching: find.byType(shad.CircularProgressIndicator)), findsOneWidget);
    expect(find.descendant(of: button, matching: find.text('Check for Updates')), findsNothing);
    expect(find.text('正在检查'), findsNothing);
  });

  testWidgets('settings language uses a dropdown and saves the selected locale', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 768));
    final settingsController = FakeSettingsController();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(FakePlatformController.new),
          settingsControllerProvider.overrideWith(() => settingsController),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const ValueKey('settings-language-select')));
    await tester.pumpAndSettle();
    expect(find.text('English'), findsOneWidget);
    expect(SettingsLanguageSelect.supportedValues, hasLength(21));
    expect(SettingsLanguageSelect.supportedValues, containsAll(<String>['zh', 'zh_TW', 'pt']));
    expect(SettingsLanguageSelect.supportedValues, isNot(contains('zh_CN')));
    expect(SettingsLanguageSelect.supportedValues, isNot(contains('pt_BR')));
    expect(supportedLocaleFromConfig('zh_CN'), isNull);
    expect(supportedLocaleFromConfig('pt_BR'), isNull);
    expect(supportedLocaleFromConfig('zh'), const Locale('zh'));
    expect(supportedLocaleFromConfig('zh_TW'), const Locale('zh', 'TW'));
    expect(lookupAppLocalizations(const Locale('zh')).label, '简体中文');
    expect(lookupAppLocalizations(const Locale('zh', 'TW')).label, '繁體中文');

    await tester.tap(find.text('English'));
    await tester.pumpAndSettle();

    expect(settingsController.state.requireValue.config.extra.locale, 'en');
    expect(find.text('English'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('theme settings adapt to mobile content layout', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(390, 760));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          appPlatformControllerProvider.overrideWith(FakePlatformController.new),
          settingsControllerProvider.overrideWith(FakeSettingsController.new),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Basic'));
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('theme-mode-system')), findsOneWidget);
    expect(find.byKey(const ValueKey('theme-color-green')), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('light task card hover uses a distinct semantic surface', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(760, 220));
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(24),
          child: TaskCard(
            task: _taskRecord(id: 'hover', name: 'hover.zip'),
            selected: false,
            batchMode: false,
            selectedInBatch: false,
            onPressed: () {},
            onToggleBatch: () {},
          ),
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 100));

    final gesture = await tester.createGesture(kind: PointerDeviceKind.mouse);
    addTearDown(gesture.removePointer);
    await gesture.addPointer(location: Offset.zero);
    await gesture.moveTo(tester.getCenter(find.byType(TaskCard)));
    await tester.pump();

    final hoverSurface = tester.widget<ColoredBox>(find.byKey(const ValueKey('task-card-hover-surface')));
    expect(hoverSurface.color, AppPalette.light.taskCardHoverBg);
    expect(hoverSurface.color, isNot(AppPalette.light.bg));
    expect(hoverSurface.color, isNot(AppPalette.light.cardBg));
  });

  test('appearance controller initializes legacy configuration safely', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    container
        .read(appAppearanceControllerProvider.notifier)
        .initialize(ExtraConfig(themeMode: 'dark', themeColor: 'purple'));

    final appearance = container.read(appAppearanceControllerProvider);
    expect(appearance.themeMode, AppThemeMode.dark);
    expect(appearance.themeColor, AppThemeColor.purple);
    expect(AppThemeColor.fromKey('missing'), AppThemeColor.green);
  });

  test('theme colors map to shad primary and ring only', () {
    final green = AppPalette.lightFor(AppThemeColor.green);
    final purple = AppPalette.lightFor(AppThemeColor.purple);
    final shadTheme = AppTheme.light(AppThemeColor.purple);

    expect(purple.brand, AppThemeColor.purple.light);
    expect(purple.brand, isNot(green.brand));
    expect(purple.brandProgress, isNot(green.brandProgress));
    expect(purple.primaryActionBg, green.primaryActionBg);
    expect(purple.primaryActionForeground, green.primaryActionForeground);
    expect(shadTheme.colorScheme.primary, AppThemeColor.purple.light);
    expect(shadTheme.colorScheme.ring, AppThemeColor.purple.light);
    expect(shadTheme.colorScheme.accent, AppPalette.light.inputBg);
  });

  testWidgets('create task advanced options scroll smoothly into view', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(700, 500));
    await tester.pumpWidget(const ProviderScope(child: _CreateTaskPageHarness()));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Advanced'));
    await tester.pumpAndSettle();

    final scrollView = tester.widget<SingleChildScrollView>(find.byKey(const ValueKey('create-task-form-scroll')));
    final controller = scrollView.controller!;
    final initialOffset = controller.offset;

    await tester.tap(find.text('Advanced'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pump(const Duration(milliseconds: 80));
    final movingOffset = controller.offset;
    await tester.pump(const Duration(milliseconds: 300));
    final settledOffset = controller.offset;

    expect(movingOffset, greaterThan(initialOffset));
    expect(settledOffset, greaterThanOrEqualTo(movingOffset));
    expect(settledOffset, lessThanOrEqualTo(controller.position.maxScrollExtent));
    expect(find.text('User-Agent'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('create task labels direct download without a mode field', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(700, 500));
    await tester.pumpWidget(const ProviderScope(child: _CreateTaskPageHarness()));
    await tester.pump();

    expect(find.text('Mode'), findsNothing);
    expect(find.text('Direct Download'), findsOneWidget);
    expect(find.text('Skip resolving and create tasks immediately'), findsOneWidget);
    expect(
      tester.getRect(find.byType(shad.Checkbox)).left,
      greaterThan(tester.getRect(find.text('Direct Download')).right),
    );
    await tester.pump(const Duration(seconds: 1));
  });

  testWidgets('app focus outlines use the compact global border', (WidgetTester tester) async {
    late shad.FocusOutlineTheme focusTheme;
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(AppThemeColor.purple),
        materialTheme: AppTheme.materialLight(AppThemeColor.purple),
        home: AppComponentThemes(
          child: Builder(
            builder: (context) {
              focusTheme = shad.ComponentTheme.of<shad.FocusOutlineTheme>(context);
              return const SizedBox();
            },
          ),
        ),
      ),
    );

    expect(focusTheme.align, 1);
    expect(focusTheme.border?.top.width, 1);
  });

  testWidgets('task toolbar actions have only neutral enabled and disabled colors', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 120));
    final searchController = TextEditingController();
    addTearDown(searchController.dispose);
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(AppThemeColor.purple),
        materialTheme: AppTheme.materialLight(AppThemeColor.purple),
        home: TasksTopBar(
          searchController: searchController,
          onAddTask: () {},
          batchMode: true,
          selectedBatchCount: 2,
          canPauseSelected: true,
          canResumeSelected: true,
          onToggleBatchMode: () {},
          onPauseSelected: () {},
          onResumeSelected: () {},
          onDeleteSelected: () {},
        ),
      ),
    );
    await tester.pump();

    final actionStyles = find.descendant(
      of: find.byKey(const ValueKey('tasks-top-action-buttons')),
      matching: find.byType(shad.ButtonStyleOverride),
    );
    expect(actionStyles, findsNWidgets(4));
    for (final element in actionStyles.evaluate()) {
      final style = element.widget as shad.ButtonStyleOverride;
      final enabledDecoration =
          style.decoration!(element, const <WidgetState>{}, const BoxDecoration()) as BoxDecoration;
      final disabledDecoration =
          style.decoration!(element, <WidgetState>{WidgetState.disabled}, const BoxDecoration()) as BoxDecoration;
      final enabledIcon = style.iconTheme!(element, const <WidgetState>{}, const IconThemeData());
      final disabledIcon = style.iconTheme!(element, <WidgetState>{WidgetState.disabled}, const IconThemeData());

      expect((enabledDecoration.border! as Border).top.color, AppPalette.light.border);
      expect(enabledIcon.color, AppPalette.light.textSecondary);
      expect((disabledDecoration.border! as Border).top.color, AppPalette.light.border.withValues(alpha: 0.5));
      expect(disabledIcon.color, AppPalette.light.textMuted.withValues(alpha: 0.6));
    }
  });

  testWidgets('resolved file selector uses aligned tree header and sortable columns', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(760, 620));
    var selected = <int>[];
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(24),
          child: ResolveFileTree(
            files: [
              FileInfo(path: 'videos', name: 'clip.mp4', size: 1024),
              FileInfo(path: 'docs', name: 'notes.txt', size: 128),
              FileInfo(path: 'docs', name: 'manual.pdf', size: 256),
            ],
            initialSelection: const [0, 1, 2],
            onSelectionChanged: (values) => selected = values,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final treeFinder = find.byWidgetPredicate((widget) => widget is VirtualTreeView);
    expect(treeFinder, findsOneWidget);
    final tree = tester.widget<VirtualTreeView<dynamic>>(treeFinder);
    expect(tree.branchLine, same(shad.BranchLine.path));
    expect(
      tester.widgetList<shad.Checkbox>(find.byType(shad.Checkbox)).every((checkbox) => checkbox.size == null),
      isTrue,
    );
    final checkboxes = find.byType(shad.Checkbox);
    expect(tester.getTopLeft(checkboxes.at(0)).dx, closeTo(tester.getTopLeft(checkboxes.at(1)).dx, 0.5));
    final headerCenterY = tester.getRect(checkboxes.at(0)).center.dy;
    expect(
      tester.getRect(find.byKey(const ValueKey('resolve-tree-expand-toggle'))).center.dy,
      closeTo(headerCenterY, 0.01),
    );
    expect(tester.getRect(find.text('Name')).center.dy, closeTo(headerCenterY, 0.01));
    expect(tester.getRect(find.text('Size')).center.dy, closeTo(headerCenterY, 0.01));
    final expandTop = tester.getRect(find.byKey(const ValueKey('resolve-tree-expand-top')));
    final expandBottom = tester.getRect(find.byKey(const ValueKey('resolve-tree-expand-bottom')));
    expect((expandTop.top + expandBottom.bottom) / 2, closeTo(headerCenterY, 0.01));
    expect(find.text('Invert visible'), findsNothing);
    expect(find.text('Clear'), findsNothing);
    expect(find.byType(shad.TextField), findsNothing);
    expect(find.byKey(const ValueKey('resolve-tree-expand-toggle')), findsOneWidget);
    expect(find.byType(shad.OutlinedContainer), findsOneWidget);
    expect(find.text('clip.mp4'), findsOneWidget);
    expect(tester.widget<Text>(find.text('clip.mp4')).style?.color, AppPalette.light.textPrimary);

    await tester.tap(find.byKey(const ValueKey('resolve-tree-expand-toggle')));
    await tester.pumpAndSettle();
    expect(find.text('clip.mp4').hitTestable(), findsNothing);

    await tester.tap(find.byKey(const ValueKey('resolve-tree-expand-toggle')));
    await tester.pumpAndSettle();
    expect(find.text('clip.mp4'), findsOneWidget);

    final videosExpand = find.byKey(const ValueKey('resolve-tree-node-expand-folder:videos'));
    expect(videosExpand, findsOneWidget);
    expect(find.byKey(const ValueKey('resolve-tree-node-expand-file:0')), findsOneWidget);

    await tester.tap(videosExpand);
    await tester.pumpAndSettle();
    expect(tester.widget<VirtualTreeView<dynamic>>(treeFinder).nodes.first.expanded, isFalse);
    expect(find.text('clip.mp4').hitTestable(), findsNothing);
    expect(find.text('videos').hitTestable(), findsOneWidget);
    expect(selected, [0, 1, 2]);

    await tester.tap(find.text('videos'));
    await tester.pump();
    expect(tester.widget<VirtualTreeView<dynamic>>(treeFinder).nodes.first.expanded, isFalse);
    expect(selected, [1, 2]);

    await tester.tap(find.text('videos'));
    await tester.pump();
    expect(tester.widget<VirtualTreeView<dynamic>>(treeFinder).nodes.first.expanded, isFalse);
    expect(selected, [0, 1, 2]);

    await tester.tap(videosExpand);
    await tester.pumpAndSettle();
    expect(tester.widget<VirtualTreeView<dynamic>>(treeFinder).nodes.first.expanded, isTrue);
    expect(find.text('clip.mp4'), findsOneWidget);

    BoxDecoration sortDecoration(String key) {
      final decoration = find.descendant(of: find.byKey(ValueKey(key)), matching: find.byType(DecoratedBox));
      return tester.widget<DecoratedBox>(decoration).decoration as BoxDecoration;
    }

    expect(sortDecoration('resolve-tree-sort-name-asc').color, isNull);
    expect(sortDecoration('resolve-tree-sort-name-desc').color, isNull);
    final nameCenterY = tester.getRect(find.text('Name')).center.dy;
    final ascendingRect = tester.getRect(find.byKey(const ValueKey('resolve-tree-sort-name-asc')));
    final descendingRect = tester.getRect(find.byKey(const ValueKey('resolve-tree-sort-name-desc')));
    expect((ascendingRect.top + descendingRect.bottom) / 2, closeTo(nameCenterY, 0.01));

    await tester.tap(find.text('Name'));
    await tester.pumpAndSettle();
    expect(sortDecoration('resolve-tree-sort-name-asc').color, isNotNull);
    expect(sortDecoration('resolve-tree-sort-name-desc').color, isNull);

    await tester.tap(find.text('Name'));
    await tester.pumpAndSettle();
    expect(sortDecoration('resolve-tree-sort-name-asc').color, isNull);
    expect(sortDecoration('resolve-tree-sort-name-desc').color, isNotNull);

    expect(find.text('notes.txt'), findsOneWidget);
    expect(find.text('clip.mp4'), findsOneWidget);
    expect(selected, [0, 1, 2]);

    tester.view.physicalSize = const Size(360, 620);
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });

  testWidgets('file type filters support multiple selection and toggling off', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(760, 620));
    var selected = <int>[];
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(24),
          child: ResolveFileTree(
            files: [
              FileInfo(path: '', name: 'clip.mp4', size: 100),
              FileInfo(path: '', name: 'song.mp3', size: 200),
              FileInfo(path: '', name: 'cover.png', size: 300),
            ],
            initialSelection: const [],
            onSelectionChanged: (values) => selected = values,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    Color? filterBackground(String key) {
      final decoration = find.descendant(of: find.byKey(ValueKey(key)), matching: find.byType(DecoratedBox));
      return (tester.widget<DecoratedBox>(decoration.first).decoration as BoxDecoration).color;
    }

    await tester.tap(find.byKey(const ValueKey('resolve-tree-filter-video')));
    await tester.pump();
    expect(selected, [0]);
    expect(filterBackground('resolve-tree-filter-video'), AppPalette.light.filterActiveBg);

    await tester.tap(find.byKey(const ValueKey('resolve-tree-filter-audio')));
    await tester.pump();
    expect(selected, [0, 1]);
    expect(filterBackground('resolve-tree-filter-audio'), AppPalette.light.filterActiveBg);

    await tester.tap(find.byKey(const ValueKey('resolve-tree-filter-video')));
    await tester.pump();
    expect(selected, [1]);
    expect(filterBackground('resolve-tree-filter-video'), isNull);

    await tester.tap(find.byKey(const ValueKey('resolve-tree-filter-audio')));
    await tester.pump();
    expect(selected, isEmpty);
  });

  testWidgets('file name area scrolls horizontally while the trailing column stays fixed', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(420, 420));
    const longName =
        'this-is-a-very-long-file-name-that-needs-horizontal-scrolling-to-be-read-completely.archive.tar.gz';
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(24),
          child: ResolveFileTree(
            files: [FileInfo(name: longName, size: 1024)],
            initialSelection: const [0],
            onSelectionChanged: (_) {},
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final horizontalScroll = find.byKey(const ValueKey('resolve-tree-horizontal-scroll'));
    final trailing = find.byKey(const ValueKey('resolve-tree-trailing-file:0'));
    expect(find.byKey(const ValueKey('resolve-tree-horizontal-scrollbar')), findsOneWidget);
    expect(horizontalScroll, findsOneWidget);
    expect(trailing, findsOneWidget);

    final scrollable = find.descendant(of: horizontalScroll, matching: find.byType(Scrollable));
    final position = tester.state<ScrollableState>(scrollable).position;
    expect(position.maxScrollExtent, greaterThan(0));
    final nameLeftBefore = tester.getRect(find.text(longName)).left;
    final trailingLeftBefore = tester.getRect(trailing).left;

    await tester.drag(find.byKey(const ValueKey('resolve-tree-horizontal-scrollbar')), const Offset(-120, 0));
    await tester.pumpAndSettle();

    expect(position.pixels, greaterThan(0));
    expect(tester.getRect(find.text(longName)).left, lessThan(nameLeftBefore));
    expect(tester.getRect(trailing).left, closeTo(trailingLeftBefore, 0.01));
  });

  testWidgets('file name scrollbar stays hidden when measured labels fit', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(420, 420));
    const shortName = 'short.zip';
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(24),
          child: ResolveFileTree(
            files: [FileInfo(name: shortName, size: 1024)],
            initialSelection: const [0],
            onSelectionChanged: (_) {},
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text(shortName), findsOneWidget);
    expect(find.byKey(const ValueKey('resolve-tree-horizontal-scrollbar')), findsNothing);
    expect(find.byKey(const ValueKey('resolve-tree-horizontal-scroll')), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('virtual tree lazily builds large file lists', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(760, 620));
    final files = List.generate(
      1200,
      (index) => FileInfo(path: 'folder-${index ~/ 100}', name: 'file-$index.bin', size: index),
    );
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(24),
          child: ResolveFileTree(files: files, initialSelection: const [], onSelectionChanged: (_) {}),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byWidgetPredicate((widget) => widget is VirtualTreeView), findsOneWidget);
    expect(find.text('file-1199.bin'), findsNothing);
    expect(find.byType(shad.Checkbox).evaluate().length, lessThan(100));
    final listView = tester.widget<ListView>(find.byType(ListView));
    expect(listView.childrenDelegate, isA<SliverChildBuilderDelegate>());

    final scrollable = tester.state<ScrollableState>(find.byType(Scrollable));
    scrollable.position.jumpTo(scrollable.position.maxScrollExtent);
    await tester.pump();
    expect(find.text('file-1199.bin'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('resolve-tree-expand-toggle')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('resolve-tree-expand-toggle')));
    await tester.pump(const Duration(milliseconds: 75));
    expect(find.byType(shad.Checkbox).evaluate().length, lessThan(100));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });

  test('task runtime status parses selected file progress', () {
    final status = api_task.TaskRuntimeStatus.fromJson({
      'status': 'running',
      'used': 10,
      'speed': 20,
      'downloaded': 50,
      'total': 100,
      'uploadSpeed': 0,
      'uploaded': 0,
      'extractStatus': '',
      'extractProgress': 0,
      'files': [
        {'index': 3, 'size': 100, 'downloaded': 50},
      ],
    });

    expect(status.status, api_task.Status.running);
    expect(status.files.single.index, 3);
    expect(status.files.single.size, 100);
    expect(status.files.single.downloaded, 50);
  });

  test('task transfer speeds include running downloads and active seeds', () {
    final speeds = aggregateTaskTransferSpeeds([
      _apiTransferTask(
        id: 'running-seed',
        status: api_task.Status.running,
        uploading: true,
        speed: 100,
        uploadSpeed: 10,
      ),
      _apiTransferTask(
        id: 'completed-seed',
        status: api_task.Status.done,
        uploading: true,
        speed: 999,
        uploadSpeed: 20,
      ),
      _apiTransferTask(id: 'paused-stale', status: api_task.Status.pause, uploading: true, speed: 999, uploadSpeed: 30),
      _apiTransferTask(
        id: 'running-download',
        status: api_task.Status.running,
        uploading: false,
        speed: 50,
        uploadSpeed: 40,
      ),
    ]);

    expect(speeds.downloadBytesPerSecond, 150);
    expect(speeds.uploadBytesPerSecond, 30);
  });

  test('task records keep a missing resource size unknown', () {
    final task = _apiTransferTask(
      id: 'unknown-size',
      status: api_task.Status.running,
      uploading: false,
      speed: 0,
      uploadSpeed: 0,
    );
    task.progress.used = 1704000000000000000;

    final record = TaskRecord.fromApi(task);

    expect(record.total, isNull);
    expect(record.progress, isNull);
    expect(record.isIndeterminate, isTrue);
  });

  test('task records preserve zero upload speed while uploading', () {
    final record = TaskRecord.fromApi(
      _apiTransferTask(id: 'seeding', status: api_task.Status.done, uploading: true, speed: 0, uploadSpeed: 0),
    );

    expect(record.uploading, isTrue);
    expect(record.uploadSpeed, '0 B/s');
  });

  test('task URL updates are limited to paused or failed HTTP tasks', () {
    expect(_taskRecord(id: 'http-paused', name: 'paused.bin', status: TaskStatus.paused).canUpdateUrl, isTrue);
    expect(_taskRecord(id: 'http-failed', name: 'failed.bin', status: TaskStatus.failed).canUpdateUrl, isTrue);
    expect(_taskRecord(id: 'http-running', name: 'running.bin').canUpdateUrl, isFalse);
    expect(
      _taskRecord(
        id: 'bt-paused',
        name: 'paused.torrent',
        status: TaskStatus.paused,
        protocol: api_task.Protocol.bt,
      ).canUpdateUrl,
      isFalse,
    );
  });

  test('task icons follow task type and file extension instead of status', () {
    final extensionCases = <(String, IconData)>[
      ('setup.exe', GopeedIcons.fileInstaller),
      ('mobile.apk', GopeedIcons.fileAndroid),
      ('mobile.ipa', GopeedIcons.fileIos),
      ('backup.iso', GopeedIcons.fileDiskImage),
      ('index.html', GopeedIcons.fileWeb),
      ('notes.md', GopeedIcons.fileText),
      ('manual.pdf', GopeedIcons.filePdf),
      ('report.docx', GopeedIcons.fileDocument),
      ('budget.xlsx', GopeedIcons.fileSpreadsheet),
      ('slides.pptx', GopeedIcons.filePresentation),
      ('source.tar.gz', GopeedIcons.fileArchive),
      ('photo.PNG', GopeedIcons.fileImage),
      ('track.flac', GopeedIcons.fileAudio),
      ('movie.MKV', GopeedIcons.fileVideo),
      ('main.dart', GopeedIcons.fileCode),
      ('book.epub', GopeedIcons.fileEbook),
      ('display.woff2', GopeedIcons.fileFont),
      ('cache.sqlite', GopeedIcons.fileDatabase),
      ('linux.torrent', GopeedIcons.protocolBt),
    ];
    for (final (name, icon) in extensionCases) {
      expect(
        _taskRecord(id: name, name: name).icon,
        icon,
        reason: name,
      );
    }

    expect(_taskRecord(id: 'folder', name: 'Downloads', isFolder: true).icon, GopeedIcons.folder);
    expect(
      _taskRecord(id: 'bt-folder', name: 'Linux collection', isFolder: true, protocol: api_task.Protocol.bt).icon,
      GopeedIcons.folderBt,
    );
    expect(_taskRecord(id: 'file', name: 'unknown').icon, GopeedIcons.file);
    expect(_taskRecord(id: 'bt', name: 'unknown', protocol: api_task.Protocol.bt).icon, GopeedIcons.protocolBt);
    expect(_taskRecord(id: 'ed2k', name: 'unknown', protocol: api_task.Protocol.ed2k).icon, GopeedIcons.protocolEd2k);
    final runningIcon = _taskRecord(id: 'running-video', name: 'movie.mp4').icon;
    final failedIcon = _taskRecord(id: 'failed-video', name: 'movie.mp4', status: TaskStatus.failed).icon;
    expect(failedIcon, runningIcon);
  });

  testWidgets('task cards label a missing total as unknown size', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 220));
    final record = TaskRecord.fromApi(
      _apiTransferTask(
        id: 'unknown-size-card',
        status: api_task.Status.running,
        uploading: false,
        speed: 0,
        uploadSpeed: 0,
      ),
    );

    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(24),
          child: TaskCard(
            task: record,
            selected: false,
            batchMode: false,
            selectedInBatch: false,
            onPressed: () {},
            onToggleBatch: () {},
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.text('Unknown size'), findsOneWidget);
    expect(find.textContaining('TB'), findsNothing);
  });

  testWidgets('task cards display zero upload speed while uploading', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 220));
    for (final status in [api_task.Status.running, api_task.Status.done]) {
      final record = TaskRecord.fromApi(
        _apiTransferTask(id: 'uploading-${status.name}', status: status, uploading: true, speed: 0, uploadSpeed: 0),
      );

      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: Padding(
            padding: const EdgeInsets.all(24),
            child: TaskCard(
              task: record,
              selected: false,
              batchMode: false,
              selectedInBatch: false,
              onPressed: () {},
              onToggleBatch: () {},
            ),
          ),
        ),
      );
      await tester.pump();

      expect(find.text('0 B/s'), findsOneWidget);
      expect(find.byIcon(Icons.north), findsOneWidget);
      expect(tester.takeException(), isNull);
    }
  });

  testWidgets('completed task cards show the completed status only once', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 220));

    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(24),
          child: TaskCard(
            task: _taskRecord(id: 'completed-card', name: 'completed.zip', status: TaskStatus.completed),
            selected: false,
            batchMode: false,
            selectedInBatch: false,
            onPressed: () {},
            onToggleBatch: () {},
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.text('COMPLETED'), findsOneWidget);
    expect(find.text('Downloaded'), findsNothing);
  });

  testWidgets('speed monitor displays supplied task totals without simulated updates', (WidgetTester tester) async {
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: const SizedBox(
          width: 192,
          child: SpeedMonitorCard(downloadBytesPerSecond: 10 * 1024 * 1024, uploadBytesPerSecond: 2 * 1024 * 1024),
        ),
      ),
    );
    await tester.pump(const Duration(seconds: 2));

    expect(find.text('10'), findsOneWidget);
    expect(find.text('2'), findsOneWidget);
    expect(find.text('MB/s'), findsNWidgets(2));
  });

  testWidgets('speed monitor adapts units for low transfer rates', (WidgetTester tester) async {
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: const SizedBox(
          width: 192,
          child: SpeedMonitorCard(downloadBytesPerSecond: 96 * 1024, uploadBytesPerSecond: 512),
        ),
      ),
    );

    expect(find.text('96'), findsOneWidget);
    expect(find.text('KB/s'), findsOneWidget);
    expect(find.text('512'), findsOneWidget);
    expect(find.text('B/s'), findsOneWidget);
  });

  testWidgets('task file tree maps sparse file progress by resource index', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(520, 620));
    const task = TaskRecord(
      id: 'task-1',
      name: 'album',
      status: TaskStatus.downloading,
      downloaded: '50 B',
      total: '100 B',
      progress: 0.5,
      url: 'https://example.com/album.torrent',
      storagePath: '/downloads/album',
      files: [
        TaskFileNode(path: 'disc', name: 'skipped.mp3', sizeBytes: 100),
        TaskFileNode(path: 'disc', name: 'selected.mp3', sizeBytes: 100),
      ],
      uploading: false,
    );
    final status = api_task.TaskRuntimeStatus(
      status: api_task.Status.running,
      used: 10,
      speed: 20,
      downloaded: 50,
      total: 100,
      uploadSpeed: 0,
      uploaded: 0,
      extractStatus: api_task.ExtractStatus.none,
      extractProgress: 0,
      files: [api_task.FileRuntimeStatus(index: 1, size: 100, downloaded: 50)],
    );

    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(24),
          child: TaskFileTree(task: task, runtimeStatus: status),
        ),
      ),
    );
    await tester.pump();

    expect(find.text('Progress'), findsOneWidget);
    expect(find.text('Size'), findsOneWidget);
    expect(find.text('skipped.mp3'), findsOneWidget);
    expect(find.text('selected.mp3'), findsOneWidget);
    expect(find.byIcon(GopeedIcons.folder), findsOneWidget);
    expect(find.byIcon(GopeedIcons.fileAudio), findsNWidgets(2));
    expect(tester.widget<Text>(find.text('selected.mp3')).style?.fontSize, 11);
    final compactTree = tester.widget<VirtualTreeView<dynamic>>(
      find.byWidgetPredicate((widget) => widget is VirtualTreeView),
    );
    expect(compactTree.rowHeight, 27);
    expect(find.byKey(const ValueKey('task-file-progress-file:0')), findsNothing);
    final selectedProgress = find.byKey(const ValueKey('task-file-progress-file:1'));
    expect(selectedProgress, findsOneWidget);
    final circularProgress = find.descendant(
      of: selectedProgress,
      matching: find.byType(shad.CircularProgressIndicator),
    );
    expect(circularProgress, findsOneWidget);
    expect(tester.widget<shad.CircularProgressIndicator>(circularProgress).value, 0.5);
    expect(find.descendant(of: selectedProgress, matching: find.text('100 B')), findsOneWidget);
    expect(find.byType(shad.Checkbox), findsNothing);
  });

  testWidgets('task file tree keeps a progress ring while per-file status is unavailable', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(520, 420));
    const task = TaskRecord(
      id: 'task-fallback-progress',
      name: 'pending.bin',
      status: TaskStatus.downloading,
      downloaded: '25 B',
      total: '100 B',
      progress: 0.25,
      url: 'https://example.com/pending.bin',
      storagePath: '/downloads/pending.bin',
      files: [TaskFileNode(path: '', name: 'pending.bin', sizeBytes: 100)],
      uploading: false,
    );

    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: const Padding(
          padding: EdgeInsets.all(24),
          child: TaskFileTree(task: task),
        ),
      ),
    );
    await tester.pump();

    final fileProgress = find.byKey(const ValueKey('task-file-progress-file:0'));
    final ring = find.descendant(of: fileProgress, matching: find.byType(shad.CircularProgressIndicator));
    expect(ring, findsOneWidget);
    expect(tester.widget<shad.CircularProgressIndicator>(ring).value, 0.25);
  });

  testWidgets('completed task files use a completion icon instead of a progress ring', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(520, 420));
    const task = TaskRecord(
      id: 'task-completed',
      name: 'completed.bin',
      status: TaskStatus.completed,
      downloaded: '100 B',
      total: '100 B',
      progress: 1,
      url: 'https://example.com/completed.bin',
      storagePath: '/downloads/completed.bin',
      files: [TaskFileNode(path: '', name: 'completed.bin', sizeBytes: 100)],
      uploading: false,
    );

    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: const Padding(
          padding: EdgeInsets.all(24),
          child: TaskFileTree(task: task),
        ),
      ),
    );

    expect(find.byIcon(shad.BootstrapIcons.checkCircleFill), findsOneWidget);
    expect(find.byType(shad.CircularProgressIndicator), findsNothing);
  });

  testWidgets('paused and failed task files use state icons instead of progress rings', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(520, 420));
    const pausedTask = TaskRecord(
      id: 'task-paused',
      name: 'paused.bin',
      status: TaskStatus.paused,
      downloaded: '40 B',
      total: '100 B',
      progress: 0.4,
      url: 'https://example.com/paused.bin',
      storagePath: '/downloads/paused.bin',
      files: [TaskFileNode(path: '', name: 'paused.bin', sizeBytes: 100)],
      uploading: false,
    );
    const failedTask = TaskRecord(
      id: 'task-failed',
      name: 'failed.bin',
      status: TaskStatus.failed,
      downloaded: '40 B',
      total: '100 B',
      progress: 0.4,
      url: 'https://example.com/failed.bin',
      storagePath: '/downloads/failed.bin',
      files: [TaskFileNode(path: '', name: 'failed.bin', sizeBytes: 100)],
      uploading: false,
    );

    Future<void> pumpTask(TaskRecord task) async {
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: Padding(
            padding: const EdgeInsets.all(24),
            child: TaskFileTree(task: task),
          ),
        ),
      );
      await tester.pump();
    }

    await pumpTask(pausedTask);
    expect(find.byIcon(Icons.pause_circle_filled), findsOneWidget);
    expect(find.byType(shad.CircularProgressIndicator), findsNothing);

    await pumpTask(failedTask);
    expect(find.byIcon(Icons.error_rounded), findsOneWidget);
    expect(find.byType(shad.CircularProgressIndicator), findsNothing);
  });

  testWidgets('task drawer aligns information labels beside their values', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(900, 700));
    final task = _taskRecord(
      id: 'details',
      name: 'details.zip',
      status: TaskStatus.completed,
      remaining: '1 minute remaining',
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          taskRuntimeStatusProvider(task.id).overrideWith(
            (ref) async => api_task.TaskRuntimeStatus(
              status: api_task.Status.done,
              used: 0,
              speed: 0,
              downloaded: 100,
              total: 100,
              uploadSpeed: 0,
              uploaded: 0,
              files: const [],
            ),
          ),
        ],
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: Stack(
            children: [TaskDrawer(task: task, onClose: () {}, onOpenStorage: () {}, onUpdateUrl: (_) async {})],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final labelTopLeft = tester.getTopLeft(find.text('Task Name'));
    final valueTopLeft = tester.getTopLeft(find.text('details.zip'));
    expect((labelTopLeft.dy - valueTopLeft.dy).abs(), lessThan(5));
    expect(valueTopLeft.dx, greaterThan(labelTopLeft.dx));

    final storageLabelTopLeft = tester.getTopLeft(find.text('Storage Path'));
    final storageValueFinder = find.byKey(const ValueKey('task-details-storage-value'));
    final storageValueTopLeft = tester.getTopLeft(storageValueFinder);
    expect((storageLabelTopLeft.dy - storageValueTopLeft.dy).abs(), lessThan(5));
    expect(storageValueTopLeft.dx, greaterThan(storageLabelTopLeft.dx));
    final storageValue = tester.widget<Text>(storageValueFinder);
    expect(storageValue.data, contains('\u200B'));
    expect(storageValue.data!.replaceAll('\u200B', ''), '/downloads/details.zip');
    expect(storageValue.semanticsLabel, '/downloads/details.zip');
    final urlValue = tester.widget<Text>(find.byKey(const ValueKey('task-details-url-value')));
    expect(urlValue.data, contains('\u200B'));
    expect(urlValue.data!.replaceAll('\u200B', ''), task.url);
    expect(urlValue.semanticsLabel, task.url);
    expect(find.text('Task Details'), findsOneWidget);
    expect(find.text('Details'), findsOneWidget);
    expect(find.text('Files'), findsOneWidget);
    expect(find.text('Statistics'), findsOneWidget);
    expect(find.text('Remaining'), findsOneWidget);
    expect(find.text('—'), findsOneWidget);
    expect(find.text('1 minute remaining'), findsNothing);
    expect(
      tester.getSize(find.byKey(const ValueKey('task-details-drawer'))).width,
      AppDesignTokens.taskDetailsDrawerMinWidth,
    );

    tester.view.physicalSize = const Size(1600, 900);
    await tester.pumpAndSettle();
    expect(
      tester.getSize(find.byKey(const ValueKey('task-details-drawer'))).width,
      1600 * AppDesignTokens.taskDetailsDrawerViewportRatio,
    );

    await tester.tap(find.text('Files'));
    await tester.pump();
    final browseFilesAction = find.byKey(const ValueKey('task-files-browse'));
    expect(browseFilesAction, findsOneWidget);
    final totalProgress = find.byKey(const ValueKey('task-files-total-progress'));
    expect(totalProgress, findsOneWidget);
    expect(tester.getTopLeft(browseFilesAction).dx, greaterThan(tester.getTopRight(totalProgress).dx));
    expect(
      find.descendant(of: browseFilesAction, matching: find.byIcon(Icons.snippet_folder_outlined)),
      findsOneWidget,
    );
    final browseFilesTooltipFinder = find.descendant(of: browseFilesAction, matching: find.byType(shad.Tooltip));
    final browseFilesTooltip = tester.widget<shad.Tooltip>(browseFilesTooltipFinder);
    expect((browseFilesTooltip.tooltip(tester.element(browseFilesTooltipFinder)) as Text).data, 'Browse Files');

    await tester.tap(browseFilesAction);
    await tester.pumpAndSettle();
    final browserDialog = find.byKey(const ValueKey('task-file-browser-dialog'));
    expect(browserDialog, findsOneWidget);
    expect(find.descendant(of: browserDialog, matching: find.byType(TaskFileManagerView)), findsOneWidget);
    expect(tester.getCenter(browserDialog).dx, closeTo(tester.view.physicalSize.width / 2, 1));
    expect(tester.getCenter(browserDialog).dy, closeTo(tester.view.physicalSize.height / 2, 1));
  });

  testWidgets('task drawer actions use icon buttons with hover labels', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(900, 700));
    final task = _taskRecord(id: 'editable-details', name: 'editable.zip', status: TaskStatus.paused);

    await tester.pumpWidget(
      ProviderScope(
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: Stack(
            children: [TaskDrawer(task: task, onClose: () {}, onOpenStorage: () {}, onUpdateUrl: (_) async {})],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    for (final key in [
      'task-details-close',
      'task-details-edit-url',
      'task-details-copy-url',
      'task-details-open-storage',
    ]) {
      final action = find.byKey(ValueKey(key));
      expect(action, findsOneWidget);
      expect(find.descendant(of: action, matching: find.byType(shad.IconButton)), findsOneWidget);
      expect(find.descendant(of: action, matching: find.byType(shad.Tooltip)), findsOneWidget);
    }
    expect(find.text('Copy'), findsNothing);
    expect(find.text('Edit'), findsNothing);
    expect(find.text('Open'), findsNothing);

    final editAction = find.byKey(const ValueKey('task-details-edit-url'));
    final editTooltipFinder = find.descendant(of: editAction, matching: find.byType(shad.Tooltip));
    final editTooltip = tester.widget<shad.Tooltip>(editTooltipFinder);
    expect((editTooltip.tooltip(tester.element(editTooltipFinder)) as Text).data, 'Edit');

    await tester.tap(editAction);
    await tester.pumpAndSettle();
    for (final key in ['task-details-cancel-url', 'task-details-save-url']) {
      final action = find.byKey(ValueKey(key));
      expect(action, findsOneWidget);
      expect(find.descendant(of: action, matching: find.byType(shad.IconButton)), findsOneWidget);
      expect(find.descendant(of: action, matching: find.byType(shad.Tooltip)), findsOneWidget);
    }
    expect(find.text('Cancel'), findsNothing);
    expect(find.text('Save'), findsNothing);
  });

  testWidgets('task drawer hides URL editing for non-HTTP tasks', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(900, 700));
    final task = _taskRecord(
      id: 'bt-details',
      name: 'paused.torrent',
      status: TaskStatus.paused,
      protocol: api_task.Protocol.bt,
    );

    await tester.pumpWidget(
      ProviderScope(
        child: shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: Stack(
            children: [TaskDrawer(task: task, onClose: () {}, onOpenStorage: () {}, onUpdateUrl: (_) async {})],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byKey(const ValueKey('task-details-edit-url')), findsNothing);
    expect(find.byKey(const ValueKey('task-details-copy-url')), findsOneWidget);
  });

  testWidgets('mobile task details use a standalone route without bottom navigation', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(390, 760));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          tasksControllerProvider.overrideWith(RecordingTasksController.new),
        ],
        child: const GopeedApp(),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.byType(PrimaryBottomNavigation), findsOneWidget);
    await tester.tap(find.text('first.zip'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 350));

    expect(find.text('Task Details'), findsOneWidget);
    expect(find.text('Details'), findsOneWidget);
    expect(find.text('Files'), findsOneWidget);
    expect(find.text('Statistics'), findsOneWidget);
    expect(find.byType(PrimaryBottomNavigation), findsNothing);
    expect(find.byIcon(Icons.arrow_back), findsOneWidget);

    await tester.tap(find.text('Files'));
    await tester.pump();
    expect(find.byKey(const ValueKey('task-files-browse')), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('task-files-browse')));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 350));
    expect(find.byType(TaskFileManagerView), findsOneWidget);
    expect(find.text('first.zip'), findsWidgets);
  });

  testWidgets('task file manager uses flat directories and platform-specific file actions', (
    WidgetTester tester,
  ) async {
    await _setTestSize(tester, const Size(390, 700));
    final task = _taskRecord(
      id: 'managed-files',
      name: 'bundle',
      status: TaskStatus.completed,
      isFolder: true,
      files: const [
        TaskFileNode(path: '/', name: 'root.txt', sizeBytes: 128),
        TaskFileNode(path: '/docs', name: 'guide.pdf', sizeBytes: 1024),
      ],
    );

    Future<void> pumpManager({required bool webActions, required bool desktopActions}) async {
      await tester.pumpWidget(
        shad.ShadcnApp(
          theme: AppTheme.light(),
          materialTheme: AppTheme.materialLight(),
          home: TaskFileManagerView(task: task, webActions: webActions, desktopActions: desktopActions),
        ),
      );
      await tester.pump();
    }

    await pumpManager(webActions: true, desktopActions: false);
    expect(find.text('root.txt'), findsOneWidget);
    expect(find.text('docs'), findsOneWidget);
    expect(find.text('guide.pdf'), findsNothing);
    expect(find.byIcon(GopeedIcons.fileText), findsOneWidget);
    expect(find.byIcon(GopeedIcons.folder), findsOneWidget);
    expect(find.byIcon(Icons.open_in_new), findsOneWidget);
    expect(find.byIcon(Icons.download_outlined), findsOneWidget);
    expect(find.byIcon(Icons.share_outlined), findsNothing);

    await tester.tap(find.text('docs'));
    await tester.pump();
    expect(find.text('guide.pdf'), findsOneWidget);
    expect(find.byIcon(GopeedIcons.filePdf), findsOneWidget);
    expect(find.text('1 items'), findsNothing);
    expect(find.byKey(const ValueKey('file-breadcrumb-/docs')), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('file-breadcrumb-/')));
    await tester.pump();
    expect(find.text('1 items'), findsOneWidget);

    await pumpManager(webActions: false, desktopActions: false);
    expect(find.byIcon(Icons.download_outlined), findsNothing);
    expect(find.byIcon(Icons.share_outlined), findsOneWidget);

    await pumpManager(webActions: false, desktopActions: true);
    expect(find.byIcon(Icons.download_outlined), findsNothing);
    expect(find.byIcon(Icons.share_outlined), findsNothing);
    expect(find.byIcon(Icons.open_in_new), findsOneWidget);
    expect(find.byIcon(Icons.folder_open_outlined), findsNWidgets(2));
  });

  testWidgets('open desktop task details follow refreshed file metadata', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 768));
    late RefreshingTaskFilesController tasksController;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          tasksControllerProvider.overrideWith(() => tasksController = RefreshingTaskFilesController()),
        ],
        child: const GopeedApp(),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));

    await tester.tap(find.text('fresh.zip'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 350));
    await tester.tap(find.text('Files'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 250));
    expect(find.text('late-file.bin'), findsNothing);

    tasksController.publishFiles();
    await tester.pump();

    expect(find.text('late-file.bin'), findsOneWidget);
  });

  testWidgets('mobile task cards use compact vertical spacing', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(390, 240));

    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: ListView(
          children: [
            TaskCard(
              task: _taskRecord(id: 'mobile-compact', name: 'mobile.zip'),
              selected: false,
              batchMode: false,
              selectedInBatch: false,
              onPressed: () {},
              onToggleBatch: () {},
            ),
          ],
        ),
      ),
    );

    expect(tester.getSize(find.byKey(const ValueKey('task-card-mobile-surface'))).height, lessThan(96));
  });

  testWidgets('piece map stays within narrow layouts and exposes a tapped range', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(280, 640));
    final pieces = List<TaskPieceState>.generate(
      5000,
      (index) => index < 2100 ? TaskPieceState.completed : TaskPieceState.pending,
    );
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(16),
          child: PieceMap(pieceMap: TaskPieceMap.fromStates(pieces, pieceSize: 1024)),
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 50));

    expect(tester.takeException(), isNull);
    final piecePaint = find.descendant(of: find.byType(PieceMap), matching: find.byType(CustomPaint));
    expect(tester.getSize(piecePaint).height, lessThan(240));
    await tester.tapAt(tester.getTopLeft(piecePaint) + const Offset(5, 5));
    await tester.pump();
    expect(find.textContaining('Pieces 1'), findsOneWidget);
  });

  testWidgets('large piece map fills complete rows at the default drawer width', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(440, 640));
    final pieces = List<TaskPieceState>.filled(5000, TaskPieceState.pending);
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24),
          child: PieceMap(pieceMap: TaskPieceMap.fromStates(pieces, pieceSize: 1024)),
        ),
      ),
    );
    await tester.pump();

    final piecePaint = find.descendant(of: find.byType(PieceMap), matching: find.byType(CustomPaint));
    expect(tester.getSize(piecePaint).width, 392);
    // The fixed 252-cell cap fills 9 rows at the default 28-column width.
    expect(tester.getSize(piecePaint).height, closeTo(123.96, 0.1));
    expect(tester.takeException(), isNull);
  });

  testWidgets('empty peer table keeps its sortable column headers', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(440, 500));
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: const Padding(
          padding: EdgeInsets.all(16),
          child: PeerTable(protocol: PeerTableProtocol.bt, peers: []),
        ),
      ),
    );
    await tester.pumpAndSettle();

    for (final header in [
      'Address',
      'Client',
      'Protocol',
      'Speed',
      'Upload speed',
      'Pieces',
      'Progress',
      'File relevance',
      'Source',
    ]) {
      expect(find.text(header), findsOneWidget);
    }
    expect(find.text('No connections'), findsNothing);
    expect(find.byType(Scrollbar), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('peer table displays the remote completion percentage', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 500));
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: const Padding(
          padding: EdgeInsets.all(16),
          child: PeerTable(
            protocol: PeerTableProtocol.bt,
            peers: [
              TaskPeerStats(
                address: '127.0.0.1:6881',
                client: 'Gopeed',
                downloadSpeed: 2048,
                uploadSpeed: 1024,
                pieceCount: 3,
                completion: 0.375,
                relevance: 0.125,
                source: 'pex',
                transport: 'utp',
              ),
              TaskPeerStats(
                address: '127.0.0.2:6881',
                client: 'Gopeed',
                downloadSpeed: 1024,
                uploadSpeed: 512,
                pieceCount: 8,
                completion: 1,
                relevance: 0,
                source: 'dht',
                transport: 'tcp',
              ),
            ],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Progress'), findsOneWidget);
    expect(find.text('File relevance'), findsOneWidget);
    expect(find.text('37.5%'), findsOneWidget);
    expect(find.text('12.5%'), findsOneWidget);
    expect(find.text('uTP'), findsOneWidget);
    expect(find.text('BT'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('ED2K peer table keeps protocol-specific columns', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 500));
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: const Padding(
          padding: EdgeInsets.all(16),
          child: PeerTable(
            protocol: PeerTableProtocol.ed2k,
            peers: [
              TaskPeerStats(
                address: '127.0.0.1:4662',
                client: 'aMule',
                downloadSpeed: 2048,
                uploadSpeed: 1024,
                pieceCount: 3,
                completion: null,
                relevance: null,
                source: 'kad',
                transport: 'tcp',
              ),
            ],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('ED2K'), findsOneWidget);
    expect(find.text('Progress'), findsNothing);
    expect(find.text('File relevance'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('HTTP connection lanes keep metadata and progress on one compact row', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(390, 640));
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(16),
          child: HttpConnectionLanes(
            taskDownloading: true,
            connections: const [
              HttpConnectionStats(
                downloaded: 32 * 1024 * 1024,
                total: 64 * 1024 * 1024,
                completed: false,
                failed: false,
                retryTimes: 0,
              ),
              HttpConnectionStats(
                downloaded: 12 * 1024 * 1024,
                total: 64 * 1024 * 1024,
                completed: false,
                failed: false,
                retryTimes: 2,
              ),
              HttpConnectionStats(
                downloaded: 48 * 1024 * 1024,
                total: 64 * 1024 * 1024,
                completed: true,
                failed: false,
                retryTimes: 0,
              ),
              HttpConnectionStats(
                downloaded: 8 * 1024 * 1024,
                total: 64 * 1024 * 1024,
                completed: false,
                failed: true,
                retryTimes: 3,
              ),
            ],
          ),
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 50));

    final firstLane = find.byKey(const ValueKey('http-connection-lane-0'));
    final secondLane = find.byKey(const ValueKey('http-connection-lane-1'));
    final thirdLane = find.byKey(const ValueKey('http-connection-lane-2'));
    final fourthLane = find.byKey(const ValueKey('http-connection-lane-3'));
    expect(tester.getSize(firstLane).height, 38);
    expect(tester.getSize(secondLane).height, 38);
    expect(tester.getSize(thirdLane).height, 38);
    expect(tester.getSize(fourthLane).height, 38);
    expect(find.text('#1'), findsOneWidget);
    expect(find.text('Downloading'), findsNothing);
    expect(find.text('Retrying'), findsNothing);
    expect(find.text('重试 2 次'), findsNothing);
    expect(find.text('Completed'), findsNothing);
    final firstProgress = find.byKey(const ValueKey('http-connection-progress-0'));
    final secondProgress = find.byKey(const ValueKey('http-connection-progress-1'));
    final thirdProgress = find.byKey(const ValueKey('http-connection-progress-2'));
    final fourthProgress = find.byKey(const ValueKey('http-connection-progress-3'));
    expect(tester.getSize(firstProgress).width, tester.getSize(secondProgress).width);
    expect(tester.getSize(secondProgress).width, tester.getSize(thirdProgress).width);
    expect(tester.getSize(thirdProgress).width, tester.getSize(fourthProgress).width);
    expect(tester.widget<TaskProgressBar>(firstProgress).shimmer, isTrue);
    expect(tester.widget<TaskProgressBar>(firstProgress).indeterminate, isFalse);
    expect(tester.widget<TaskProgressBar>(firstProgress).value, 0.5);
    expect(tester.widget<TaskProgressBar>(secondProgress).shimmer, isFalse);
    expect(tester.widget<TaskProgressBar>(secondProgress).indeterminate, isFalse);
    expect(tester.widget<TaskProgressBar>(thirdProgress).shimmer, isFalse);
    expect(tester.widget<TaskProgressBar>(thirdProgress).value, 0.75);
    expect(tester.widget<TaskProgressBar>(fourthProgress).shimmer, isFalse);
    expect(tester.widget<TaskProgressBar>(fourthProgress).value, 0.125);
    final retryIcon = find.byKey(const ValueKey('http-connection-status-1'));
    expect(find.descendant(of: retryIcon, matching: find.byType(shad.Tooltip)), findsOneWidget);
    final failedIcon = find.byKey(const ValueKey('http-connection-status-3'));
    expect(find.descendant(of: failedIcon, matching: find.byType(shad.Tooltip)), findsOneWidget);
    expect(
      tester.getRect(firstProgress).left - tester.getRect(find.byKey(const ValueKey('http-connection-status-0'))).right,
      closeTo(8, 0.01),
    );
    expect(tester.getRect(find.text('32 MB')).right, closeTo(tester.getRect(firstLane).right, 0.01));
    expect(tester.getRect(find.text('48 MB')).right, closeTo(tester.getRect(thirdLane).right, 0.01));
    expect(tester.takeException(), isNull);
  });

  testWidgets('HTTP connections use indeterminate progress only when total is unknown', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(390, 320));
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: const Padding(
          padding: EdgeInsets.all(16),
          child: HttpConnectionLanes(
            taskDownloading: true,
            connections: [
              HttpConnectionStats(downloaded: 1024, total: 0, completed: false, failed: false, retryTimes: 0),
              HttpConnectionStats(downloaded: 4096, total: 2048, completed: false, failed: false, retryTimes: 0),
            ],
          ),
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 50));

    final unknown = tester.widget<TaskProgressBar>(find.byKey(const ValueKey('http-connection-progress-0')));
    final overflow = tester.widget<TaskProgressBar>(find.byKey(const ValueKey('http-connection-progress-1')));
    expect(unknown.indeterminate, isTrue);
    expect(unknown.value, isNull);
    expect(overflow.indeterminate, isFalse);
    expect(overflow.value, 1);
  });

  testWidgets('HTTP connection progress stays static while its task is paused', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(390, 320));
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: const Padding(
          padding: EdgeInsets.all(16),
          child: HttpConnectionLanes(
            taskDownloading: false,
            connections: [
              HttpConnectionStats(downloaded: 1024, total: 2048, completed: false, failed: false, retryTimes: 0),
              HttpConnectionStats(downloaded: 1024, total: 0, completed: false, failed: false, retryTimes: 0),
            ],
          ),
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 50));

    final known = tester.widget<TaskProgressBar>(find.byKey(const ValueKey('http-connection-progress-0')));
    final unknown = tester.widget<TaskProgressBar>(find.byKey(const ValueKey('http-connection-progress-1')));
    expect(known.shimmer, isFalse);
    expect(known.indeterminate, isFalse);
    expect(known.value, 0.5);
    expect(unknown.shimmer, isFalse);
    expect(unknown.indeterminate, isFalse);
    expect(unknown.value, isNull);
  });

  testWidgets('task context menu exposes pause and selection operations', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(520, 620));
    var paused = false;
    var selected = false;
    final task = _taskRecord(id: 'context-task', name: 'context.zip');
    final actions = TaskContextActions(
      selected: false,
      allSelected: false,
      listeningForUpdate: false,
      onToggleSelectAll: () {},
      onToggleSelected: () => selected = true,
      onPause: () => paused = true,
      onResume: () {},
      onDelete: () {},
      onUpdateUrl: () {},
      onToggleUpdateListener: () {},
    );

    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: shad.Scaffold(
          child: Center(
            child: TaskContextMenu(
              task: task,
              actions: actions,
              child: const SizedBox(key: ValueKey('context-target'), width: 260, height: 80),
            ),
          ),
        ),
      ),
    );

    await tester.tapAt(tester.getCenter(find.byKey(const ValueKey('context-target'))), buttons: kSecondaryMouseButton);
    await tester.pumpAndSettle();
    expect(find.text('Pause'), findsOneWidget);
    expect(find.text('Select task'), findsOneWidget);

    await tester.tap(find.text('Select task'));
    await tester.pumpAndSettle();
    expect(selected, isTrue);

    await tester.tapAt(tester.getCenter(find.byKey(const ValueKey('context-target'))), buttons: kSecondaryMouseButton);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Pause'));
    await tester.pumpAndSettle();
    expect(paused, isTrue);
  });

  testWidgets('delete dialog returns keep-files switch value', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(520, 620));
    bool? keepFiles;
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Builder(
          builder: (context) => shad.PrimaryButton(
            onPressed: () async {
              keepFiles = await showTaskDeleteDialog(context, taskCount: 2, keepFiles: true);
            },
            child: const Text('Open delete dialog'),
          ),
        ),
      ),
    );

    await tester.tap(find.text('Open delete dialog'));
    await tester.pumpAndSettle();
    expect(find.text('Delete 2 tasks?'), findsOneWidget);
    expect(tester.widget<shad.Switch>(find.byType(shad.Switch)).value, isTrue);

    await tester.tap(find.byType(shad.Switch));
    await tester.pump();
    await tester.tap(find.text('Delete'));
    await tester.pumpAndSettle();
    expect(keepFiles, isFalse);
  });

  testWidgets('pending URL update asks whether to update, create, or cancel', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(520, 620));
    PendingUpdateDecision? decision;
    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Builder(
          builder: (context) => shad.PrimaryButton(
            onPressed: () async {
              decision = await showPendingUpdateDialog(context, taskName: 'archive.zip');
            },
            child: const Text('Open pending update dialog'),
          ),
        ),
      ),
    );

    await tester.tap(find.text('Open pending update dialog'));
    await tester.pumpAndSettle();
    expect(find.text('Pending Update Task Found'), findsOneWidget);
    expect(
      find.text('Task "archive.zip" is waiting for URL update. Do you want to update it with the new URL?'),
      findsOneWidget,
    );
    expect(find.text('Cancel'), findsOneWidget);
    expect(find.text('Create New Task'), findsOneWidget);
    expect(find.text('Update Task'), findsOneWidget);

    await tester.tap(find.text('Create New Task'));
    await tester.pumpAndSettle();
    expect(decision, PendingUpdateDecision.createTask);
  });

  testWidgets('captured URL update preserves the full request and stops listening after confirmation', (
    WidgetTester tester,
  ) async {
    await _setTestSize(tester, const Size(1024, 768));
    late RecordingPendingUpdateTasksController tasksController;
    final container = ProviderContainer(
      overrides: [
        appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
        appPlatformControllerProvider.overrideWith(FakePlatformController.new),
        appDeepLinkControllerProvider.overrideWith(FakeDeepLinkController.new),
        appNotificationControllerProvider.overrideWith(FakeNotificationController.new),
        tasksControllerProvider.overrideWith(() {
          tasksController = RecordingPendingUpdateTasksController();
          return tasksController;
        }),
      ],
    );
    addTearDown(container.dispose);

    await tester.pumpWidget(UncontrolledProviderScope(container: container, child: const GopeedApp()));
    await tester.pumpAndSettle();

    const listeningTask = PendingUpdateTask(id: 'paused-http', name: 'archive.zip');
    final replacementRequest = Request(
      url: 'https://example.com/replacement.zip',
      extra: const {
        'header': {'Authorization': 'Bearer replacement'},
      },
      skipVerifyCert: true,
    );
    container.read(pendingUpdateTaskProvider.notifier).set(listeningTask);
    container
        .read(pendingUpdateRequestProvider.notifier)
        .set(
          PendingUpdateRequest(
            task: listeningTask,
            createTask: CreateTask(req: replacementRequest),
          ),
        );
    await tester.pumpAndSettle();

    expect(find.text('Pending Update Task Found'), findsOneWidget);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();

    expect(tasksController.updatedTaskId, isNull);
    expect(container.read(pendingUpdateTaskProvider), same(listeningTask));
    expect(container.read(pendingUpdateRequestProvider), isNull);

    container
        .read(pendingUpdateRequestProvider.notifier)
        .set(
          PendingUpdateRequest(
            task: listeningTask,
            createTask: CreateTask(req: replacementRequest),
          ),
        );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Update Task'));
    await tester.pumpAndSettle();

    expect(tasksController.updatedTaskId, 'paused-http');
    expect(tasksController.updatedRequest, same(replacementRequest));
    expect(tasksController.updatedRequest?.skipVerifyCert, isTrue);
    expect(tasksController.updatedRequest?.extra, replacementRequest.extra);
    expect(container.read(pendingUpdateTaskProvider), isNull);
    expect(container.read(pendingUpdateRequestProvider), isNull);
  });

  testWidgets('desktop batch pause sends only selected task ids', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 768));
    late RecordingTasksController tasksController;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          tasksControllerProvider.overrideWith(() => tasksController = RecordingTasksController()),
        ],
        child: const GopeedApp(),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));

    await tester.tap(find.byIcon(Icons.checklist_rtl_outlined));
    await tester.pump(const Duration(milliseconds: 250));
    expect(
      tester.widgetList<shad.Checkbox>(find.byType(shad.Checkbox)).every((item) => item.activeColor == null),
      isTrue,
    );
    await tester.tap(find.text('first.zip'));
    await tester.pump();
    await tester.tap(find.byIcon(Icons.pause_outlined));
    await tester.pump(const Duration(milliseconds: 250));

    expect(tasksController.pausedIds, ['first']);
  });

  testWidgets('completed task batch selection only enables delete', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 768));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appRuntimeControllerProvider.overrideWith(FakeRuntimeController.new),
          tasksControllerProvider.overrideWith(CompletedTasksController.new),
        ],
        child: const GopeedApp(),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Completed'));
    await tester.pump();
    await tester.tap(find.byIcon(Icons.checklist_rtl_outlined));
    await tester.pump(const Duration(milliseconds: 250));
    await tester.tap(find.text('done.zip'));
    await tester.pump();

    final actions = tester
        .widgetList<shad.IconButton>(
          find.descendant(
            of: find.byKey(const ValueKey('tasks-top-action-buttons')),
            matching: find.byType(shad.IconButton),
          ),
        )
        .toList(growable: false);
    expect(actions, hasLength(4));
    expect(actions[0].onPressed, isNull);
    expect(actions[1].onPressed, isNull);
    expect(actions[2].onPressed, isNotNull);
    expect(actions[3].onPressed, isNotNull);
  });

  testWidgets('batch selection rebuilds only the affected task', (WidgetTester tester) async {
    final controller = TaskBatchSelectionController();
    addTearDown(controller.dispose);
    const taskIds = ['first', 'second', 'third'];
    final buildCounts = {for (final taskId in taskIds) taskId: 0};

    await tester.pumpWidget(
      Directionality(
        textDirection: TextDirection.ltr,
        child: Row(
          children: [
            for (final taskId in taskIds)
              TaskBatchSelectionBuilder(
                controller: controller,
                taskId: taskId,
                visibleTaskIds: taskIds,
                builder: (context, selected, allSelected) {
                  buildCounts[taskId] = buildCounts[taskId]! + 1;
                  return Text('$taskId:$selected:$allSelected');
                },
              ),
          ],
        ),
      ),
    );
    buildCounts.updateAll((_, _) => 0);

    controller.toggle('first');
    await tester.pump();

    expect(buildCounts, {'first': 1, 'second': 0, 'third': 0});
    expect(find.text('first:true:false'), findsOneWidget);
  });

  testWidgets('completed task selection responds immediately in batch mode', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 220));
    var selectedCount = 0;
    var openedCount = 0;

    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(24),
          child: TaskCard(
            task: _taskRecord(id: 'completed', name: 'completed.zip', status: TaskStatus.completed),
            selected: false,
            batchMode: true,
            selectedInBatch: false,
            onPressed: () => selectedCount++,
            onToggleBatch: () => selectedCount++,
            onOpen: () => openedCount++,
          ),
        ),
      ),
    );

    await tester.tap(find.text('completed.zip'));
    await tester.pump();

    expect(selectedCount, 1);
    expect(openedCount, 0);
  });

  testWidgets('completed task double click opens while folder action reveals', (WidgetTester tester) async {
    await _setTestSize(tester, const Size(1024, 220));
    var openedCount = 0;
    var revealedCount = 0;

    await tester.pumpWidget(
      shad.ShadcnApp(
        theme: AppTheme.light(),
        materialTheme: AppTheme.materialLight(),
        home: Padding(
          padding: const EdgeInsets.all(24),
          child: TaskCard(
            task: _taskRecord(id: 'completed-actions', name: 'completed.zip', status: TaskStatus.completed),
            selected: false,
            batchMode: false,
            selectedInBatch: false,
            onPressed: () {},
            onToggleBatch: () {},
            onOpen: () => openedCount++,
            onReveal: () => revealedCount++,
          ),
        ),
      ),
    );

    await tester.tap(find.text('completed.zip'));
    await tester.pump(kDoubleTapMinTime);
    await tester.tap(find.text('completed.zip'));
    await tester.pumpAndSettle();

    expect(openedCount, 1);
    expect(revealedCount, 0);

    await tester.tap(find.byIcon(Icons.folder_open_outlined));
    await tester.pump();

    expect(openedCount, 1);
    expect(revealedCount, 1);
  });
}

Future<void> _setTestSize(WidgetTester tester, Size size) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(() {
    tester.view.resetPhysicalSize();
    tester.view.resetDevicePixelRatio();
  });
}

class _ResponsiveMenuHarness extends StatelessWidget {
  const _ResponsiveMenuHarness();

  @override
  Widget build(BuildContext context) {
    return shad.ShadcnApp(
      theme: AppTheme.light(),
      materialTheme: AppTheme.materialLight(),
      home: ResponsiveMenuLayout<String>(
        title: 'Settings',
        selectedValue: 'general',
        onSelected: (_) {},
        items: const [
          ResponsiveMenuItem(value: 'general', label: 'General', icon: Icons.settings_outlined),
          ResponsiveMenuItem(value: 'advanced', label: 'Advanced', icon: Icons.tune_outlined),
        ],
        mobileContentTitleBuilder: (value) => value == 'advanced' ? 'Advanced' : 'General',
        contentBuilder: (context, selected) {
          return Text(selected == 'advanced' ? 'Advanced content' : 'General content');
        },
      ),
    );
  }
}

class _CreateTaskPageHarness extends StatelessWidget {
  const _CreateTaskPageHarness();

  @override
  Widget build(BuildContext context) {
    return shad.ShadcnApp(
      theme: AppTheme.light(),
      materialTheme: AppTheme.materialLight(),
      home: const CreateTaskWindowPage(),
    );
  }
}

class FakeTasksController extends TasksController {
  @override
  Future<TasksState> build() async => const TasksState(tasks: []);
}

class FailingTasksController extends TasksController {
  @override
  Future<TasksState> build() async => throw Exception('offline');
}

class FakeExtensionsController extends ExtensionsController {
  int removeCalls = 0;

  @override
  Future<ExtensionsState> build() async {
    final installed = api_extension.Extension(
      identity: 'extension-0',
      name: 'extension-0',
      author: 'Gopeed',
      title: 'Extension 0',
      description: 'Installed extension',
      icon: '',
      version: '1.0.0',
      homepage: '',
      repository: null,
      disabled: false,
      devMode: false,
      devPath: '',
    );
    return ExtensionsState(
      installedExtensions: [installed],
      storeExtensions: List.generate(
        8,
        (index) => StoreExtension(
          id: 'extension-$index',
          repoFullName: 'gopeed/extension-$index',
          repoUrl: 'https://github.com/gopeed/extension-$index',
          name: 'extension-$index',
          author: 'Gopeed',
          title: 'Extension $index',
          description: 'Extension description $index',
          version: '1.0.0',
          installCount: index,
          stars: index,
          topics: const [],
        ),
      ),
    );
  }

  @override
  Future<void> toggleExtension(api_extension.Extension extension, bool enabled) async {
    extension.disabled = !enabled;
    state = AsyncValue.data(
      state.requireValue.copyWith(installedExtensions: [...state.requireValue.installedExtensions]),
    );
  }

  @override
  Future<void> removeExtension(api_extension.Extension extension) async {
    removeCalls++;
  }
}

class PaginatedExtensionsController extends FakeExtensionsController {
  int loadMoreCalls = 0;

  @override
  Future<ExtensionsState> build() async {
    final initial = await super.build();
    return initial.copyWith(
      storePagination: StorePagination(page: 1, limit: 8, total: 16, totalPages: 2, hasNext: true, hasPrev: false),
    );
  }

  @override
  Future<void> loadMoreStore() async {
    final current = state.requireValue;
    if (current.loadingMoreStore || current.storePagination?.hasNext != true) return;
    loadMoreCalls++;
    state = AsyncValue.data(current.copyWith(loadingMoreStore: true));
    await Future<void>.delayed(const Duration(milliseconds: 20));
    state = AsyncValue.data(
      state.requireValue.copyWith(
        loadingMoreStore: false,
        storePagination: StorePagination(page: 2, limit: 8, total: 16, totalPages: 2, hasNext: false, hasPrev: true),
      ),
    );
  }
}

class FakeRuntimeController extends AppRuntimeController {
  @override
  Future<AppRuntimeState> build() async {
    final cfg = StartConfig()
      ..network = 'tcp'
      ..address = '127.0.0.1:9999'
      ..apiToken = ''
      ..storage = 'bolt'
      ..storageDir = ''
      ..refreshInterval = 0;
    return AppRuntimeState(startConfig: cfg, runningPort: 9999, downloaderConfig: DownloaderConfig());
  }
}

class RecordingStartConfigRuntimeController extends AppRuntimeController {
  StartConfig? savedStartConfig;

  @override
  Future<AppRuntimeState> build() async {
    final config = StartConfig()
      ..network = 'tcp'
      ..address = '192.168.1.20:4321'
      ..apiToken = 'token'
      ..storage = 'bolt'
      ..storageDir = ''
      ..refreshInterval = 0;
    return AppRuntimeState(startConfig: config, runningPort: 4321, downloaderConfig: DownloaderConfig());
  }

  @override
  Future<void> saveStartConfig(StartConfig config) async {
    savedStartConfig = StartConfig()
      ..network = config.network
      ..address = config.address
      ..apiToken = config.apiToken
      ..storage = config.storage
      ..storageDir = config.storageDir
      ..refreshInterval = config.refreshInterval;
  }
}

class FakeSettingsController extends SettingsController {
  @override
  Future<SettingsState> build() async => SettingsState(config: DownloaderConfig());

  @override
  Future<void> save(DownloaderConfig config) async {
    state = AsyncValue.data(SettingsState(config: config));
  }
}

class CategorySettingsController extends FakeSettingsController {
  @override
  Future<SettingsState> build() async {
    final config = DownloaderConfig();
    config.extra.downloadCategories = [DownloadCategory(name: '资料', path: '/tmp/data')];
    return SettingsState(config: config);
  }
}

class ProxySettingsController extends FakeSettingsController {
  @override
  Future<SettingsState> build() async {
    final config = DownloaderConfig();
    config.proxy
      ..enable = true
      ..system = false
      ..scheme = 'socks5'
      ..host = 'proxy.example.com:1080';
    return SettingsState(config: config);
  }
}

class Ed2kSettingsController extends FakeSettingsController {
  @override
  Future<SettingsState> build() async {
    final config = DownloaderConfig();
    config.protocolConfig.ed2k
      ..serverAddr = 'server-a:4661,server-b:4661'
      ..serverMet = 'https://example.com/server.met,https://mirror.example.com/server.met'
      ..nodesDat = '/tmp/nodes.dat,https://example.com/nodes.dat';
    return SettingsState(config: config);
  }
}

class FakePlatformController extends AppPlatformController {
  @override
  Future<AppPlatformState> build() async => const AppPlatformState(started: true);
}

class FakeDeepLinkController extends AppDeepLinkController {
  @override
  Future<AppDeepLinkState> build() async => const AppDeepLinkState(started: true);
}

class FakeNotificationController extends AppNotificationController {
  @override
  Future<AppNotificationState> build() async => const AppNotificationState(started: true);
}

class RecordingPendingUpdateTasksController extends TasksController {
  String? updatedTaskId;
  Request? updatedRequest;

  @override
  Future<TasksState> build() async => const TasksState(tasks: []);

  @override
  Future<void> updateRequest(String id, Request request) async {
    updatedTaskId = id;
    updatedRequest = request;
  }
}

class AvailableUpdatePlatformController extends AppPlatformController {
  @override
  Future<AppPlatformState> build() async => const AppPlatformState(
    started: true,
    updateStatus: AppUpdateStatus.available,
    latestVersion: VersionInfo(
      version: '9.9.9',
      changeLog: '# 更新日志\n\n- 测试更新',
      releaseUrl: 'https://example.com/release',
    ),
  );
}

class CheckingUpdatePlatformController extends AppPlatformController {
  @override
  Future<AppPlatformState> build() async =>
      const AppPlatformState(started: true, updateStatus: AppUpdateStatus.checking);
}

class RecordingTasksController extends TasksController {
  List<String>? pausedIds;

  @override
  Future<TasksState> build() async {
    return TasksState(
      tasks: [
        _taskRecord(id: 'first', name: 'first.zip'),
        _taskRecord(id: 'second', name: 'second.zip'),
      ],
    );
  }

  @override
  Future<void> pauseAll(List<String>? ids) async {
    pausedIds = ids == null ? null : List<String>.of(ids);
  }
}

class CompletedTasksController extends TasksController {
  @override
  Future<TasksState> build() async {
    return TasksState(
      tasks: [_taskRecord(id: 'done', name: 'done.zip', status: TaskStatus.completed)],
    );
  }
}

class RefreshingTaskFilesController extends TasksController {
  @override
  Future<TasksState> build() async {
    return TasksState(
      tasks: [_taskRecord(id: 'fresh', name: 'fresh.zip')],
    );
  }

  void publishFiles() {
    state = AsyncValue.data(
      TasksState(
        tasks: [
          _taskRecord(
            id: 'fresh',
            name: 'fresh.zip',
            files: const [TaskFileNode(path: '', name: 'late-file.bin', sizeBytes: 128)],
          ),
        ],
      ),
    );
  }
}

TaskRecord _taskRecord({
  required String id,
  required String name,
  TaskStatus status = TaskStatus.downloading,
  List<TaskFileNode> files = const [],
  api_task.Protocol protocol = api_task.Protocol.http,
  String? remaining,
  bool isFolder = false,
}) {
  return TaskRecord(
    id: id,
    name: name,
    status: status,
    downloaded: '50 B',
    total: '100 B',
    speed: '10 B/s',
    progress: 0.5,
    url: 'https://example.com/$name',
    storagePath: '/downloads/$name',
    files: files,
    uploading: false,
    isFolder: isFolder,
    protocol: protocol,
    remaining: remaining,
  );
}

api_task.Task _apiTransferTask({
  required String id,
  required api_task.Status status,
  required bool uploading,
  required int speed,
  required int uploadSpeed,
}) {
  return api_task.Task(
    id: id,
    name: id,
    meta: Meta(
      req: Request(url: 'https://example.com/$id'),
      opts: Options(),
    ),
    status: status,
    uploading: uploading,
    progress: api_task.Progress(used: 0, speed: speed, downloaded: 0, uploadSpeed: uploadSpeed, uploaded: 0),
    createdAt: DateTime.fromMillisecondsSinceEpoch(0),
    updatedAt: DateTime.fromMillisecondsSinceEpoch(0),
  );
}
