import 'package:get/get.dart';

import '../pages/create/create_controller.dart';
import '../pages/create/create_view.dart';
import '../pages/downloaded/downloaded_controller.dart';
import '../pages/downloaded/downloaded_view.dart';
import '../pages/downloading/downloading_controller.dart';
import '../pages/downloading/downloading_view.dart';
import '../pages/home/home_controller.dart';
import '../pages/home/home_view.dart';
import '../pages/root/root_controller.dart';
import '../pages/root/root_view.dart';
import '../pages/setting/setting_controller.dart';
import '../pages/setting/setting_view.dart';

abstract class _Paths {
  static const root = '/';

  static const create = '/create';

  static const home = '/home';
  static const downloading = '/downloading';
  static const downloaded = '/downloaded';
  static const setting = '/setting';
}

class Routes {
  static const root = _Paths.root;

  static const create = _Paths.create;

  static const home = _Paths.home;
  static const downloading = _Paths.home + _Paths.downloading;
  static const downloaded = _Paths.home + _Paths.downloaded;

  static const setting = _Paths.home + _Paths.setting;

  static final routes = [
    GetPage(
        name: _Paths.root,
        participatesInRootNavigator: true,
        transition: Transition.topLevel,
        // preventDuplicates: true,
        page: () => const RootView(),
        binding: BindingsBuilder(() {
          Get.lazyPut<RootController>(() => RootController(), fenix: true);
        }),
        children: [
          GetPage(
            name: _Paths.create,
            transition: Transition.downToUp,
            // preventDuplicates: true,
            page: () => CreateView(),
            binding: BindingsBuilder(() {
              Get.lazyPut<CreateController>(() => CreateController(),
                  fenix: true);
            }),
          ),
          GetPage(
              name: _Paths.home,
              // transition: Transition.upToDown,
              // preventDuplicates: true,
              page: () => const HomeView(),
              binding: BindingsBuilder(() {
                Get.lazyPut<HomeController>(() => HomeController(),
                    fenix: true);
              }),
              children: [
                GetPage(
                  name: _Paths.downloading,
                  transition: Transition.noTransition,
                  page: () => const DownloadingView(),
                  binding: BindingsBuilder(() {
                    Get.lazyPut<DownloadingController>(
                      () => DownloadingController(),
                    );
                  }),
                ),
                GetPage(
                  name: _Paths.downloaded,
                  transition: Transition.noTransition,
                  page: () => const DownloadedView(),
                  binding: BindingsBuilder(() {
                    Get.lazyPut<DownloadedController>(
                      () => DownloadedController(),
                    );
                  }),
                ),
                GetPage(
                  name: _Paths.setting,
                  transition: Transition.noTransition,
                  page: () => const SettingView(),
                  binding: BindingsBuilder(() {
                    Get.lazyPut<SettingController>(() => SettingController());
                  }),
                )
              ]),
        ]),
  ];
}
