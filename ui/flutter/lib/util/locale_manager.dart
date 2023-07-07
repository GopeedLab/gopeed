import 'package:flutter/material.dart';

Locale toLocale(String key) {
  final arr = key.split('_');
  return Locale(arr[0], arr[1]);
}

String getLocaleKey(Locale locale) {
  return '${locale.languageCode}_${locale.countryCode}';
}

const debugLocale = Locale('zh', 'CN');
const fallbackLocale = Locale('en', 'US');
