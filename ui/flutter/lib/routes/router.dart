import 'package:get/get.dart';
import 'package:gopeed/pages/create/create_controller.dart';
import 'package:gopeed/pages/setting/setting_view.dart';

import '../pages/create/create_view.dart';
import '../pages/home/home_controller.dart';
import '../pages/home/home_view.dart';
import '../pages/root/root_controller.dart';
import '../pages/root/root_view.dart';
import '../pages/setting/setting_controller.dart';
import '../pages/task/task_controller.dart';
import '../pages/task/task_view.dart';

abstract class _Paths {
  static const root = '/';

  static const create = '/create';

  static const home = '/home';
  static const task = '/task';
  static const setting = '/setting';
}

class Routes {
  static const root = _Paths.root;

  static const create = _Paths.create;

  static const home = _Paths.home;
  static const task = _Paths.home + _Paths.task;
  static const setting = _Paths.home + _Paths.setting;

  static final routes = [
    GetPage(
        name: _Paths.root,
        participatesInRootNavigator: true,
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
              // preventDuplicates: true,
              page: () => const HomeView(),
              binding: BindingsBuilder(() {
                Get.lazyPut<HomeController>(() => HomeController(),
                    fenix: true);
              }),
              children: [
                GetPage(
                  name: _Paths.task,
                  page: () => const TaskView(),
                  binding: BindingsBuilder(() {
                    Get.lazyPut<TaskController>(() => TaskController());
                  }),
                ),
                GetPage(
                  name: _Paths.setting,
                  page: () => const SettingView(),
                  binding: BindingsBuilder(() {
                    Get.lazyPut<SettingController>(() => SettingController());
                  }),
                )
              ]),
        ]),
  ];
}
