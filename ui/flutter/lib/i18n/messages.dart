import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/i18n/langs/ru_RU.dart';
import 'package:gopeed/i18n/langs/fa_ir.dart';

import 'langs/en_us.dart';
import 'langs/zh_cn.dart';

Locale toLocale(String key) {
  final arr = key.split('_');
  return Locale(arr[0], arr[1]);
}

String getLocaleKey(Locale locale) {
  return '${locale.languageCode}_${locale.countryCode}';
}

final messages = _Messages();
const mainLocale = Locale('zh', 'CN');
const fallbackLocale = Locale('en', 'US');

class _Messages extends Translations {
  // just include available locales here
  @override
  Map<String, Map<String, String>> get keys =>
      {...zhCN, ...enUS, ...ruRU, ...faIR};
}
