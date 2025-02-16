import 'package:flutter/material.dart';
import 'package:flutter_foreground_task/flutter_foreground_task.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:get/get.dart';
import 'package:flutter/services.dart';  // 导包
import 'package:window_manager/window_manager.dart';  // 导包

import '../../../../i18n/message.dart';
import '../../../../theme/theme.dart';
import '../../../../util/locale_manager.dart';
import '../../../../util/util.dart';  // 导包
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

        // 增加监听主题变化,根据当前主题设置窗口标题栏颜色
        builder: (context, child) {
          final brightness = Theme.of(context).brightness;
          if (Util.isDesktop()) {
            if (brightness == Brightness.dark) {
              windowManager.setBrightness(Brightness.dark);
            } else {
              windowManager.setBrightness(Brightness.light);
            }
          }
          return child!;
        },

      ),
    );
  }
}
