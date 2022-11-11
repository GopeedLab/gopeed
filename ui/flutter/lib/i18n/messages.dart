import 'package:flutter/material.dart';
import 'package:get/get.dart';

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
const fallbackLocale = Locale('en', 'US');

class _Messages extends Translations {
  @override
  Map<String, Map<String, String>> get keys => {...zhCN, ...enUS};
}
