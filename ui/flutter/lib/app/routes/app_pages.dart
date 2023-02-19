import 'package:get/get.dart';

import '../modules/create/bindings/create_binding.dart';
import '../modules/create/views/create_view.dart';
import '../modules/downloaded/bindings/downloaded_binding.dart';
import '../modules/downloaded/views/downloaded_view.dart';
import '../modules/downloading/bindings/downloading_binding.dart';
import '../modules/downloading/views/downloading_view.dart';
import '../modules/home/bindings/home_binding.dart';
import '../modules/home/views/home_view.dart';
import '../modules/root/bindings/root_binding.dart';
import '../modules/root/views/root_view.dart';
import '../modules/setting/bindings/setting_binding.dart';
import '../modules/setting/views/setting_view.dart';

part 'app_routes.dart';

class AppPages {
  AppPages._();

  static const INITIAL = Routes.ROOT;

  static final routes = [
    GetPage(
        name: _Paths.ROOT,
        participatesInRootNavigator: true,
        transition: Transition.topLevel,
        // preventDuplicates: true,
        page: () => const RootView(),
        binding: RootBinding(),
        children: [
          GetPage(
              name: _Paths.HOME,
              // participatesInRootNavigator: true,
              // transition: Transition.topLevel,
              // preventDuplicates: true,
              page: () => const HomeView(),
              binding: HomeBinding(),
              children: [
                GetPage(
                  name: _Paths.DOWNLOADED,
                  transition: Transition.noTransition,
                  page: () => const DownloadedView(),
                  binding: DownloadedBinding(),
                ),
                GetPage(
                  name: _Paths.DOWNLOADING,
                  transition: Transition.noTransition,
                  page: () => const DownloadingView(),
                  binding: DownloadingBinding(),
                ),
                GetPage(
                  name: _Paths.SETTING,
                  transition: Transition.noTransition,
                  page: () => const SettingView(),
                  binding: SettingBinding(),
                ),
              ]),
          GetPage(
            name: _Paths.CREATE,
            transition: Transition.downToUp,
            // preventDuplicates: true,
            page: () => CreateView(),
            binding: CreateBinding(),
          ),
        ]),
  ];
}
