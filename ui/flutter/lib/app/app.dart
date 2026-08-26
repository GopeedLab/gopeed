import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../core/window/app_window_frame.dart';
import '../core/window/app_window_appearance.dart';
import '../core/window/window_capability_transport.dart';
import '../l10n/l10n.dart';
import '../shared/theme/app_component_themes.dart';
import '../shared/theme/app_theme.dart';
import 'application/app_appearance_controller.dart';
import 'application/app_runtime_controller.dart';
import 'router/app_router.dart';

class GopeedApp extends ConsumerStatefulWidget {
  const GopeedApp({super.key});

  @override
  ConsumerState<GopeedApp> createState() => _GopeedAppState();
}

class _GopeedAppState extends ConsumerState<GopeedApp> with WidgetsBindingObserver {
  late final GoRouter _router;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _router = AppRouter.build();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _router.dispose();
    super.dispose();
  }

  @override
  void didChangePlatformBrightness() {
    setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(appRuntimeControllerProvider, (_, next) {
      final runtime = next.value;
      if (runtime != null) {
        ref.read(appAppearanceControllerProvider.notifier).initialize(runtime.downloaderConfig.extra);
      }
    });
    final runtime = ref.watch(appRuntimeControllerProvider).value;
    if (runtime != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        ref.read(appAppearanceControllerProvider.notifier).initialize(runtime.downloaderConfig.extra);
      });
    }

    final appearance = ref.watch(appAppearanceControllerProvider);
    final themeColor = appearance.themeColor;
    final isLightTheme = appearance.resolveIsLight(WidgetsBinding.instance.platformDispatcher.platformBrightness);
    final themeMode = switch (appearance.themeMode) {
      AppThemeMode.system => shad.ThemeMode.system,
      AppThemeMode.light => shad.ThemeMode.light,
      AppThemeMode.dark => shad.ThemeMode.dark,
    };
    AppWindowCapabilityHost.instance.updateAppearance(
      AppWindowAppearance(
        themeMode: appearance.themeMode.key,
        themeColor: appearance.themeColor.key,
        locale: runtime?.downloaderConfig.extra.locale ?? '',
      ),
    );

    return shad.ShadcnApp.router(
      debugShowCheckedModeBanner: false,
      onGenerateTitle: (context) => context.l10n.appTitle,
      theme: AppTheme.light(themeColor),
      darkTheme: AppTheme.dark(themeColor),
      materialTheme: isLightTheme ? AppTheme.materialLight(themeColor) : AppTheme.materialDark(themeColor),
      themeMode: themeMode,
      locale: supportedLocaleFromConfig(runtime?.downloaderConfig.extra.locale ?? ''),
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      builder: (context, child) {
        return AppComponentThemes(child: AppWindowFrame(child: child ?? const SizedBox.shrink()));
      },
      routerConfig: _router,
    );
  }
}
