import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../core/utils/breakpoints.dart';
import '../../../../l10n/l10n.dart';

class SettingsLanguageSelect extends StatelessWidget {
  const SettingsLanguageSelect({super.key, required this.value, required this.onChanged});

  final String value;
  final ValueChanged<String> onChanged;

  static List<Locale> get supportedLocales => AppLocalizations.supportedLocales;

  static Set<String> get supportedValues => supportedLocales.map(localeConfigValue).toSet();

  @override
  Widget build(BuildContext context) {
    final isMobile = MediaQuery.sizeOf(context).width < Breakpoints.mobile;

    return LayoutBuilder(
      builder: (context, constraints) {
        final width = isMobile || constraints.maxWidth < 220 ? constraints.maxWidth : 220.0;
        return shad.Select<String>(
          key: const ValueKey('settings-language-select'),
          value: value,
          constraints: BoxConstraints.tightFor(width: width, height: 36),
          popupConstraints: const BoxConstraints(maxHeight: 240),
          itemBuilder: (context, selectedValue) => Text(_labelFor(context, selectedValue)),
          popup: (context) => shad.SelectPopup<String>(
            items: shad.SelectItemList(
              children: [
                shad.SelectItemButton<String>(value: 'system', child: Text(context.l10n.followSystem)),
                for (final locale in supportedLocales)
                  shad.SelectItemButton<String>(
                    value: localeConfigValue(locale),
                    child: Text(lookupAppLocalizations(locale).label),
                  ),
              ],
            ),
          ),
          onChanged: (selectedValue) {
            if (selectedValue != null) {
              onChanged(selectedValue);
            }
          },
        );
      },
    );
  }

  String _labelFor(BuildContext context, String selectedValue) {
    if (selectedValue == 'system') return context.l10n.followSystem;
    final locale = localeFromConfig(selectedValue);
    if (locale != null && supportedValues.contains(localeConfigValue(locale))) {
      return lookupAppLocalizations(locale).label;
    }
    return context.l10n.followSystem;
  }
}
