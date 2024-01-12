import 'package:flutter/material.dart';
import 'package:flutter_foreground_task/flutter_foreground_task.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:get/get.dart';

import '../../../../i18n/message.dart';
import '../../../../theme/theme.dart';
import '../../../../util/locale_manager.dart';
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
      ),
    );
  }
}
