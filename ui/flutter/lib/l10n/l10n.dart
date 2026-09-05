import 'package:flutter/widgets.dart';

import 'app_localizations.dart';

export 'app_localizations.dart';

extension AppLocalizationsContext on BuildContext {
  AppLocalizations get l10n =>
      Localizations.of<AppLocalizations>(this, AppLocalizations) ??
      appLocalizationsFor('', systemLocale: Localizations.maybeLocaleOf(this));
}

Locale? localeFromConfig(String value) {
  final normalized = value.trim().replaceAll('-', '_');
  if (normalized.isEmpty || normalized == 'system') return null;
  final parts = normalized.split('_');
  return Locale.fromSubtags(
    languageCode: parts.first.toLowerCase(),
    countryCode: parts.length > 1 ? parts[1].toUpperCase() : null,
  );
}

String localeConfigValue(Locale locale) {
  final countryCode = locale.countryCode;
  return countryCode == null || countryCode.isEmpty ? locale.languageCode : '${locale.languageCode}_$countryCode';
}

Locale? supportedLocaleFromConfig(String value) {
  final locale = localeFromConfig(value);
  if (locale == null) return null;
  final configValue = localeConfigValue(locale);
  return AppLocalizations.supportedLocales.any((candidate) => localeConfigValue(candidate) == configValue)
      ? locale
      : null;
}

AppLocalizations appLocalizationsFor(String configValue, {Locale? systemLocale}) {
  final configuredLocale = supportedLocaleFromConfig(configValue);
  if (configuredLocale != null) return lookupAppLocalizations(configuredLocale);

  final locale = systemLocale ?? WidgetsBinding.instance.platformDispatcher.locale;
  return lookupAppLocalizations(_supportedSystemLocale(locale) ?? const Locale('en'));
}

Locale? _supportedSystemLocale(Locale locale) {
  final exactValue = localeConfigValue(locale);
  for (final candidate in AppLocalizations.supportedLocales) {
    if (localeConfigValue(candidate) == exactValue) return candidate;
  }
  for (final candidate in AppLocalizations.supportedLocales) {
    if (candidate.languageCode == locale.languageCode && candidate.countryCode == null) return candidate;
  }
  return null;
}
