import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:get/get.dart';

import '../../i18n/messages.dart';
import '../../routes/router.dart';
import '../../theme/theme.dart';
import '../home/home_controller.dart';
import 'app_controller.dart';

class AppView extends GetView<AppController> {
  const AppView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final config = controller.downloaderConfig.value;
    return GetMaterialApp.router(
      initialBinding: BindingsBuilder(() {
        Get.lazyPut<HomeController>(() => HomeController(), fenix: true);
      }),
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
      getPages: Routes.routes,
    );
  }
}
