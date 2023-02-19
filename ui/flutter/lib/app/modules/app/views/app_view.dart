import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:get/get.dart';

import '../../../../generated/locales.g.dart';
import '../../../../i18n/messages.dart';
import '../../../../theme/theme.dart';
import '../../../routes/app_pages.dart';
import '../controllers/app_controller.dart';

class AppView extends GetView<AppController> {
  const AppView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final config = controller.downloaderConfig.value;
    return GetMaterialApp.router(
      useInheritedMediaQuery: true,
      debugShowCheckedModeBanner: false,
      theme: GopeedTheme.light,
      darkTheme: GopeedTheme.dark,
      themeMode: ThemeMode.values.byName(config.extra.themeMode),
      // translations: messages,
      translationsKeys: AppTranslation.translations,
      locale: toLocale(config.extra.locale),
      fallbackLocale: fallbackLocale,
      localizationsDelegates: const [
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales:
          AppTranslation.translations.keys.map((e) => toLocale(e)).toList(),
      getPages: AppPages.routes,
    );
  }
}
