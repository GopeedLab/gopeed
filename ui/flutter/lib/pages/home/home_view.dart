import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:stylish_bottom_bar/model/bar_items.dart';
import 'package:stylish_bottom_bar/stylish_bottom_bar.dart';

import '../../routes/router.dart';
import '../../util/responsive_builder.dart';
import 'home_controller.dart';

class HomeView extends GetView<HomeController> {
  const HomeView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return GetRouterOutlet.builder(
      builder: (context, delegate, currentRoute) {
        //This router outlet handles the appbar and the bottom navigation bar
        final currentLocation = currentRoute?.location;
        var currentIndex = 0;
        if (currentLocation?.startsWith(Routes.setting) == true) {
          currentIndex = 1;
        }
        return Scaffold(
          extendBody: true,
          //to make floating action button notch transparent

          //to avoid the floating action button overlapping behavior,
          // when a soft keyboard is displayed
          // resizeToAvoidBottomInset: false,
          // drawer:
          // drawerEnableOpenDragGesture: false,
          body: SafeArea(
              child:
                  Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
            !ResponsiveBuilder.isNarrow(context)
                ? Flexible(
                    //TODO better looking NavigationRail
                    flex: 1,
                    child: NavigationRail(
                      labelType: NavigationRailLabelType.all,
                      groupAlignment: 0,
                      onDestinationSelected: (int index) {
                        switch (index) {
                          case 0:
                            Get.rootDelegate.toNamed(Routes.create);
                            break;
                          case 1:
                            delegate.toNamed(Routes.task);
                            break;
                          case 2:
                            delegate.toNamed(Routes.setting);
                            break;
                        }
                      },
                      destinations: [
                        NavigationRailDestination(
                          icon: const Icon(Icons.add),
                          selectedIcon: const Icon(Icons.add),
                          label: Text('create.title'.tr),
                        ),
                        NavigationRailDestination(
                          icon: const Icon(Icons.list),
                          selectedIcon: const Icon(Icons.list),
                          label: Text('home.task'.tr),
                        ),
                        NavigationRailDestination(
                          icon: const Icon(Icons.settings),
                          selectedIcon: const Icon(Icons.settings),
                          label: Text('home.setting'.tr),
                        ),
                      ],
                      selectedIndex: null,
                      leading: const Icon(Icons.logo_dev),
                      // trailing: const Icon(Icons.info_outline),
                    ))
                : const Flexible(flex: 0, child: SizedBox.shrink()),
            Flexible(
                flex: 9,
                child: GetRouterOutlet(
                  initialRoute: Routes.task,
                ))
          ])),
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
                        icon: const Icon(Icons.list),
                        selectedColor: Get.theme.primaryColor,
                        title: Text('home.task'.tr)),
                    BottomBarItem(
                        icon: const Icon(Icons.settings),
                        selectedColor: Get.theme.primaryColor,
                        title: Text('home.setting'.tr)),
                  ],
                  hasNotch: true,
                  currentIndex: currentIndex,
                  onTap: (index) {
                    switch (index) {
                      case 0:
                        delegate.toNamed(Routes.task);
                        break;
                      case 1:
                        delegate.toNamed(Routes.setting);
                        break;
                    }
                  },
                )
              : null,
          // floatingActionButtonAnimator: FloatingActionButtonAnimator.scaling,
          floatingActionButton: ResponsiveBuilder.isNarrow(context)
              ? FloatingActionButton(
                  onPressed: () {
                    /* Get.isDarkMode
                    ? {
                        Get.changeTheme(GopeedTheme.light),
                        Get.changeThemeMode(ThemeMode.light)
                      }
                    : {
                        Get.changeTheme(GopeedTheme.dark),
                        Get.changeThemeMode(ThemeMode.dark)
                      }; */
                    Get.rootDelegate.toNamed(Routes.create);
                  },
                  child: const Icon(
                    Icons.add,
                  ))
              : null,
          floatingActionButtonLocation:
              FloatingActionButtonLocation.centerDocked,
        );
      },
    );
  }
}
