import 'dart:math' as math;

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
    final popupMaxHeight = isMobile ? math.min(MediaQuery.sizeOf(context).height * 0.62, 420.0) : 240.0;

    return LayoutBuilder(
      builder: (context, constraints) {
        final width = isMobile || constraints.maxWidth < 220 ? constraints.maxWidth : 220.0;
        return shad.Select<String>(
          key: const ValueKey('settings-language-select'),
          value: value,
          constraints: BoxConstraints.tightFor(width: width, height: isMobile ? 44 : 36),
          popupConstraints: BoxConstraints(maxHeight: popupMaxHeight),
          itemBuilder: (context, selectedValue) => Text(_labelFor(context, selectedValue)),
          popup: (context) => shad.SelectPopup<String>(
            items: shad.SelectItemList(
              children: [
                shad.SelectItemButton<String>(
                  value: 'system',
                  child: _LanguageOptionLabel(label: context.l10n.followSystem, mobile: isMobile),
                ),
                for (final locale in supportedLocales)
                  shad.SelectItemButton<String>(
                    value: localeConfigValue(locale),
                    child: _LanguageOptionLabel(label: lookupAppLocalizations(locale).label, mobile: isMobile),
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

class _LanguageOptionLabel extends StatelessWidget {
  const _LanguageOptionLabel({required this.label, required this.mobile});

  final String label;
  final bool mobile;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: mobile ? 28 : 20,
      child: Align(
        alignment: Alignment.centerLeft,
        child: Text(label, maxLines: 1, overflow: TextOverflow.ellipsis),
      ),
    );
  }
}
