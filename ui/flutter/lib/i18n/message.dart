import 'package:get/get.dart';

import 'langs/en_us.dart';
import 'langs/fa_ir.dart';
import 'langs/it_it.dart';
import 'langs/ja_jp.dart';
import 'langs/pl_pl.dart';
import 'langs/ru_ru.dart';
import 'langs/ta_ta.dart';
import 'langs/tr_tr.dart';
import 'langs/vi_vn.dart';
import 'langs/zh_cn.dart';
import 'langs/zh_tw.dart';

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
        ...taTA,
        ...trTR,
        ...plPL,
        ...itIT,
      };
}
