import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:stylish_bottom_bar/model/bar_items.dart';
import 'package:stylish_bottom_bar/stylish_bottom_bar.dart';

import '../../../routes/app_pages.dart';
import '../../../views/views/responsive_builder.dart';
import '../controllers/home_controller.dart';

class HomeView extends GetView<HomeController> {
  const HomeView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return GetRouterOutlet.builder(builder: (context, delegate, currentRoute) {
      return Scaffold(
        extendBody: true,
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
                        switch (index) {
                          case 0:
                            delegate.toNamed(Routes.DOWNLOADING);
                            controller.currentIndex.value = 0;
                            break;
                          case 1:
                            delegate.toNamed(Routes.DOWNLOADED);
                            controller.currentIndex.value = 1;
                            break;
                          case 2:
                            delegate.toNamed(Routes.SETTING);
                            controller.currentIndex.value = 2;
                            break;
                        }
                      },
                      destinations: [
                        NavigationRailDestination(
                          icon: const Icon(Icons.format_align_left_outlined),
                          selectedIcon:
                              const Icon(Icons.format_align_left_outlined),
                          label: Text('downloading'.tr),
                        ),
                        NavigationRailDestination(
                          icon: const Icon(Icons.format_align_justify_outlined),
                          selectedIcon:
                              const Icon(Icons.format_align_justify_outlined),
                          label: Text('downloaded'.tr),
                        ),
                        NavigationRailDestination(
                          icon: const Icon(Icons.settings),
                          selectedIcon: const Icon(Icons.settings),
                          label: Text('setting'.tr),
                        ),
                      ],
                      selectedIndex: controller.currentIndex.value,
                      leading: const Icon(Icons.logo_dev),
                      // trailing: const Icon(Icons.info_outline),
                    )
                  : const SizedBox.shrink(),
              Expanded(
                  child: GetRouterOutlet(
                initialRoute: Routes.DOWNLOADING,
                // anchorRoute: '/',
                // filterPages: (afterAnchor) {
                //   logger.w(afterAnchor);
                //   logger.w(afterAnchor.take(1));
                //   return afterAnchor.take(1);
                // },
              ))
            ]),
        bottomNavigationBar: ResponsiveBuilder.isNarrow(context)
            ? StylishBottomBar(
                option: AnimatedBarOptions(
                  iconSize: 32,
                  barAnimation: BarAnimation.blink,
                  iconStyle: IconStyle.Default,
                  opacity: 0.3,
                ),
                items: [
                  BottomBarItem(
                      icon: const Icon(Icons.format_align_left_outlined),
                      selectedColor: Get.theme.primaryColor,
                      title: Text('downloading'.tr)),
                  BottomBarItem(
                      icon: const Icon(Icons.format_align_justify_outlined),
                      selectedColor: Get.theme.primaryColor,
                      title: Text('downloaded'.tr)),
                  BottomBarItem(
                      icon: const Icon(Icons.settings),
                      selectedColor: Get.theme.primaryColor,
                      title: Text('setting'.tr)),
                ],
                // hasNotch: true,
                currentIndex: controller.currentIndex.value,
                onTap: (index) {
                  switch (index) {
                    case 0:
                      delegate.toNamed(Routes.DOWNLOADING);
                      controller.currentIndex.value = 0;
                      break;
                    case 1:
                      delegate.toNamed(Routes.DOWNLOADED);
                      controller.currentIndex.value = 1;
                      break;
                    case 2:
                      delegate.toNamed(Routes.SETTING);
                      controller.currentIndex.value = 2;
                      break;
                  }
                },
              )
            : const SizedBox.shrink(),
      );
    });
  }
}
