import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../api/model/create_task.dart';
import '../app/application/app_appearance_controller.dart';
import '../core/window/app_window_bootstrap.dart';
import '../core/window/app_window_frame.dart';
import '../core/window/app_window_payload.dart';
import '../core/window/window_capability_transport.dart';
import '../features/tasks/presentation/pages/create_task_window_page.dart';
import '../l10n/l10n.dart';
import '../shared/theme/app_component_themes.dart';
import '../shared/theme/app_theme.dart';
import '../shared/theme/app_theme_color.dart';

class ChildWindowApp extends StatefulWidget {
  const ChildWindowApp({super.key, required this.payload, required this.session});

  final AppWindowPayload payload;
  final ChildWindowSession session;

  @override
  State<ChildWindowApp> createState() => _ChildWindowAppState();
}

class _ChildWindowAppState extends State<ChildWindowApp> with WidgetsBindingObserver {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    widget.session.appearance.addListener(_appearanceChanged);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    widget.session.appearance.removeListener(_appearanceChanged);
    widget.session.dispose();
    super.dispose();
  }

  @override
  void didChangePlatformBrightness() {
    setState(() {});
  }

  void _appearanceChanged() {
    if (mounted) setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final appearance = widget.session.appearance.value;
    final themeColor = AppThemeColor.fromKey(appearance.themeColor);
    final themeMode = AppThemeMode.fromKey(appearance.themeMode);
    final isLightTheme = switch (themeMode) {
      AppThemeMode.system => WidgetsBinding.instance.platformDispatcher.platformBrightness == Brightness.light,
      AppThemeMode.light => true,
      AppThemeMode.dark => false,
    };
    final shadThemeMode = switch (themeMode) {
      AppThemeMode.system => shad.ThemeMode.system,
      AppThemeMode.light => shad.ThemeMode.light,
      AppThemeMode.dark => shad.ThemeMode.dark,
    };
    final locale = supportedLocaleFromConfig(appearance.locale);

    return shad.ShadcnApp(
      debugShowCheckedModeBanner: false,
      onGenerateTitle: (context) => AppWindowBootstrap.subWindowTitle(widget.payload.type, context.l10n),
      theme: AppTheme.light(themeColor),
      darkTheme: AppTheme.dark(themeColor),
      materialTheme: isLightTheme ? AppTheme.materialLight(themeColor) : AppTheme.materialDark(themeColor),
      themeMode: shadThemeMode,
      locale: locale,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      builder: (context, child) => AppComponentThemes(child: child ?? const SizedBox.shrink()),
      home: AppWindowFrame(child: Builder(builder: _buildPage)),
    );
  }

  Widget _buildPage(BuildContext context) {
    return switch (widget.payload.type) {
      AppWindowType.createTask => CreateTaskWindowPage(
        windowController: widget.session.controller,
        initialTask: widget.payload.createTask == null ? null : CreateTask.fromJson(widget.payload.createTask!),
      ),
      _ => shad.Scaffold(child: Center(child: Text(context.l10n.unsupportedWindowPayload))),
    };
  }
}
