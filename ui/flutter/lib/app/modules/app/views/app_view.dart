import 'package:flutter/material.dart';
import 'package:flutter_foreground_task/flutter_foreground_task.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:get/get.dart';
import 'package:window_manager/window_manager.dart';  // Import the required packages

import '../../../../i18n/message.dart';
import '../../../../theme/theme.dart';
import '../../../../util/locale_manager.dart';
import '../../../../util/util.dart';  // Import the required packages
import '../../../routes/app_pages.dart';
import '../controllers/app_controller.dart';

class AppView extends GetView<AppController> {
  const AppView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final config = controller.downloaderConfig.value;
    return WithForegroundTask(
      child: GetMaterialApp.router(
        useInheritedMediaQuery: true,
        debugShowCheckedModeBanner: false,
        theme: GopeedTheme.light,
        darkTheme: GopeedTheme.dark,
        themeMode: ThemeMode.values.byName(config.extra.themeMode),
        translations: messages,
        locale: toLocale(config.extra.locale),
        fallbackLocale: fallbackLocale,
        localizationsDelegates: const [
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        supportedLocales: messages.keys.keys.map((e) => toLocale(e)).toList(),
        getPages: AppPages.routes,

        // Add listening to theme changes, set the title bar color according to the current system theme, only in the case of following the system theme.
        builder: (context, child) {
          // if platform is desktop
          if (Util.isDesktop()){
            // current theme setting
            ThemeMode currentThemeSetting = ThemeMode.values.byName(config.extra.themeMode);
            // actual brightness of the UI
            Brightness brightness = Theme.of(context).brightness;
            // If the theme is set to follow the system, the title bar will use UI brightness
            if (currentThemeSetting == ThemeMode.system){
              windowManager.setBrightness(brightness);
            }
          }
          return child!;
        },

      ),
    );
  }
}
