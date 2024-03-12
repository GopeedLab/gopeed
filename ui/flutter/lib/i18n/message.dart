import 'package:get/get.dart';
import 'package:gopeed/i18n/langs/fa_ir.dart';
import 'package:gopeed/i18n/langs/ja_jp.dart';
import 'package:gopeed/i18n/langs/zh_tw.dart';

import 'langs/en_us.dart';
import 'langs/ru_ru.dart';
import 'langs/zh_cn.dart';
import 'langs/vi_vn.dart';
import 'langs/tr_tr.dart';

final messages = _Messages();

class _Messages extends Translations {
  // just include available locales here
  @override
  Map<String, Map<String, String>> get keys => {
        ...zhCN,
        ...enUS,
        ...ruRU,
        ...zhTW,
        ...faIR,
        ...jaJP,
        ...viVN,
        ...trTR,
      };
}
