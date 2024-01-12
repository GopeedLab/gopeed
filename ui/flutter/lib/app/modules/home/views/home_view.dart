import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../routes/app_pages.dart';
import '../../../views/responsive_builder.dart';
import '../controllers/home_controller.dart';

class HomeView extends GetView<HomeController> {
  const HomeView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return GetRouterOutlet.builder(builder: (context, delegate, currentRoute) {
      switch (currentRoute?.uri.path) {
        case Routes.EXTENSION:
          controller.currentIndex.value = 1;
          break;
        case Routes.SETTING:
          controller.currentIndex.value = 2;
          break;
        default:
          controller.currentIndex.value = 0;
          break;
      }

      return Scaffold(
        // extendBody: true,
        body: Row(
            // crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              !ResponsiveBuilder.isNarrow(context)
                  ? NavigationRail(
                      extended: true,
                      labelType: NavigationRailLabelType.none,
                      minExtendedWidth: 170,
                      groupAlignment: 0,
                      // useIndicator: false,
                      onDestinationSelected: (int index) {
                        controller.currentIndex.value = index;
                        switch (index) {
                          case 0:
                            delegate.offAndToNamed(Routes.TASK);
                            break;
                          case 1:
                            delegate.offAndToNamed(Routes.EXTENSION);
                            break;
                          case 2:
                            delegate.offAndToNamed(Routes.SETTING);
                            break;
                        }
                      },
                      destinations: [
                        NavigationRailDestination(
                          icon: const Icon(Icons.task),
                          selectedIcon: const Icon(Icons.task),
                          label: Text('task'.tr),
                        ),
                        NavigationRailDestination(
                          icon: const Icon(Icons.extension),
                          selectedIcon: const Icon(Icons.extension),
                          label: Text('extensions'.tr),
                        ),
                        NavigationRailDestination(
                          icon: const Icon(Icons.settings),
                          selectedIcon: const Icon(Icons.settings),
                          label: Text('setting'.tr),
                        ),
                      ],
                      selectedIndex: controller.currentIndex.value,
                      leading: const Icon(Icons.menu),
                      // trailing: const Icon(Icons.info_outline),
                    )
                  : const SizedBox.shrink(),
              Expanded(
                  child: GetRouterOutlet(
                initialRoute: Routes.TASK,
                // anchorRoute: '/',
                // filterPages: (afterAnchor) {
                //   logger.w(afterAnchor);
                //   logger.w(afterAnchor.take(1));
                //   return afterAnchor.take(1);
                // },
              ))
            ]),
        bottomNavigationBar: ResponsiveBuilder.isNarrow(context)
            ? BottomNavigationBar(
                items: <BottomNavigationBarItem>[
                  BottomNavigationBarItem(
                    icon: const Icon(Icons.task),
                    label: 'task'.tr,
                  ),
                  BottomNavigationBarItem(
                    icon: const Icon(Icons.extension),
                    label: 'extensions'.tr,
                  ),
                  BottomNavigationBarItem(
                    icon: const Icon(Icons.settings),
                    label: 'setting'.tr,
                  ),
                ],
                currentIndex: controller.currentIndex.value,
                // selectedItemColor: Get.theme.highlightColor,
                onTap: (index) {
                  controller.currentIndex.value = index;
                  switch (index) {
                    case 0:
                      delegate.offAndToNamed(Routes.TASK);
                      break;
                    case 1:
                      delegate.offAndToNamed(Routes.EXTENSION);
                      break;
                    case 2:
                      delegate.offAndToNamed(Routes.SETTING);
                      break;
                  }
                },
              )
            // StylishBottomBar(
            //         option: AnimatedBarOptions(
            //           iconSize: 32,
            //           barAnimation: BarAnimation.blink,
            //           iconStyle: IconStyle.Default,
            //           opacity: 0.3,
            //         ),
            //         items: [
            //           BottomBarItem(
            //               icon: const Icon(Icons.file_download),
            //               selectedColor: Get.theme.primaryColor,
            //               title: Text('downloading'.tr)),
            //           BottomBarItem(
            //               icon: const Icon(Icons.done),
            //               selectedColor: Get.theme.primaryColor,
            //               title: Text('downloaded'.tr)),
            //           BottomBarItem(
            //               icon: const Icon(Icons.settings),
            //               selectedColor: Get.theme.primaryColor,
            //               title: Text('setting'.tr)),
            //         ],
            //         // hasNotch: true,
            //         currentIndex: controller.currentIndex.value,
            //         onTap: (index) {
            //           switch (index) {
            //             case 0:
            //               delegate.toNamed(Routes.DOWNLOADING);
            //               controller.currentIndex.value = 0;
            //               break;
            //             case 1:
            //               delegate.toNamed(Routes.DOWNLOADED);
            //               controller.currentIndex.value = 1;
            //               break;
            //             case 2:
            //               delegate.toNamed(Routes.SETTING);
            //               controller.currentIndex.value = 2;
            //               break;
            //           }
            //         },
            //       )
            : const SizedBox.shrink(),
      );
    });
  }
}
