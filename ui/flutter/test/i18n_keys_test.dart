import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/i18n/langs/ar_ar.dart';
import 'package:gopeed/i18n/langs/ca_es.dart';
import 'package:gopeed/i18n/langs/de_de.dart';
import 'package:gopeed/i18n/langs/en_us.dart';
import 'package:gopeed/i18n/langs/es_es.dart';
import 'package:gopeed/i18n/langs/fa_ir.dart';
import 'package:gopeed/i18n/langs/fr_fr.dart';
import 'package:gopeed/i18n/langs/hu_hu.dart';
import 'package:gopeed/i18n/langs/id_id.dart';
import 'package:gopeed/i18n/langs/it_it.dart';
import 'package:gopeed/i18n/langs/ja_jp.dart';
import 'package:gopeed/i18n/langs/ko_kr.dart';
import 'package:gopeed/i18n/langs/pl_pl.dart';
import 'package:gopeed/i18n/langs/pt_br.dart';
import 'package:gopeed/i18n/langs/ru_ru.dart';
import 'package:gopeed/i18n/langs/ta_ta.dart';
import 'package:gopeed/i18n/langs/tr_tr.dart';
import 'package:gopeed/i18n/langs/uk_ua.dart';
import 'package:gopeed/i18n/langs/vi_vn.dart';
import 'package:gopeed/i18n/langs/zh_cn.dart';
import 'package:gopeed/i18n/langs/zh_tw.dart';

void main() {
  group('i18n Keys Completeness Tests', () {
    final enMap = enUS['en_US']!;
    final locales = <String, Map<String, dynamic>>{
      'ar_AR': arAR['ar_AR']!,
      'ca_ES': caES['ca_ES']!,
      'de_DE': deDE['de_DE']!,
      'es_ES': esES['es_ES']!,
      'fa_IR': faIR['fa_IR']!,
      'fr_FR': frFR['fr_FR']!,
      'hu_HU': huHU['hu_HU']!,
      'id_ID': idID['id_ID']!,
      'it_IT': itIT['it_IT']!,
      'ja_JP': jaJP['ja_JP']!,
      'ko_KR': koKR['ko_KR']!,
      'pl_PL': plPL['pl_PL']!,
      'pt_BR': ptBR['pt_BR']!,
      'ru_RU': ruRU['ru_RU']!,
      'ta_TA': taTA['ta_TA']!,
      'tr_TR': trTR['tr_TR']!,
      'uk_UA': ukUA['uk_UA']!,
      'vi_VN': viVN['vi_VN']!,
      'zh_CN': zhCN['zh_CN']!,
      'zh_TW': zhTW['zh_TW']!,
    };

    for (final entry in locales.entries) {
      test('locale ${entry.key} contains all keys present in en_US', () {
        final localeMap = entry.value;
        final missing = <String>[];
        for (final key in enMap.keys) {
          if (!localeMap.containsKey(key)) {
            missing.add(key);
          }
        }
        expect(missing, isEmpty,
            reason: 'Locale ${entry.key} is missing keys: $missing');
      });
    }
  });
}
