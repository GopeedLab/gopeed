import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:stylish_bottom_bar/stylish_bottom_bar.dart';

import '../../routes/router.dart';
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
          extendBody: true, //to make floating action button notch transparent

          //to avoid the floating action button overlapping behavior,
          // when a soft keyboard is displayed
          // resizeToAvoidBottomInset: false,
          body: GetRouterOutlet(
            initialRoute: Routes.task,

            //delegate: Get.nestedKey(Routes.HOME),
            // key: Get.nestedKey(Routes.HOME),
            // key: Get.nestedKey(Routes.home),
          ),

          bottomNavigationBar: StylishBottomBar(
            items: [
              AnimatedBarItems(
                  icon: const Icon(Icons.list),
                  selectedColor: Get.theme.primaryColor,
                  title: Text('home.task'.tr)),
              AnimatedBarItems(
                  icon: const Icon(Icons.settings),
                  selectedColor: Get.theme.primaryColor,
                  title: Text('home.setting'.tr)),
            ],
            iconSize: 32,
            barAnimation: BarAnimation.fade,
            iconStyle: IconStyle.Default,
            hasNotch: true,
            opacity: 0.3,
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
          ),
          floatingActionButton: FloatingActionButton(
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
              )),
          floatingActionButtonLocation:
              FloatingActionButtonLocation.centerDocked,
        );
      },
    );
  }
}
