import 'package:get/get.dart';
import 'package:gopeed/app/modules/task/views/task_view.dart';

import '../modules/create/bindings/create_binding.dart';
import '../modules/create/views/create_view.dart';
import '../modules/extension/bindings/extension_binding.dart';
import '../modules/extension/views/extension_view.dart';
import '../modules/home/bindings/home_binding.dart';
import '../modules/home/views/home_view.dart';
import '../modules/root/bindings/root_binding.dart';
import '../modules/root/views/root_view.dart';
import '../modules/setting/bindings/setting_binding.dart';
import '../modules/setting/views/setting_view.dart';
import '../modules/task/bindings/task_binding.dart';

part 'app_routes.dart';

class AppPages {
  AppPages._();

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
                    name: _Paths.TASK,
                    page: () => const TaskView(),
                    transition: Transition.noTransition,
                    binding: TaskBinding()),
                GetPage(
                    name: _Paths.EXTENSION,
                    page: () => ExtensionView(),
                    transition: Transition.noTransition,
                    binding: ExtensionBinding()),
                GetPage(
                  name: _Paths.SETTING,
                  page: () => const SettingView(),
                  transition: Transition.noTransition,
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
