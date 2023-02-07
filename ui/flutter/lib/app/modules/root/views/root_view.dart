import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../routes/router.dart';
import 'root_controller.dart';

class RootView extends GetView<RootController> {
  const RootView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return GetRouterOutlet.builder(
      builder: (context, delegate, current) {
        return GetRouterOutlet(
          initialRoute: Routes.home,
          // anchorRoute: '/',
          // filterPages: (afterAnchor) {
          //   return afterAnchor.take(1);
          // },
        );
      },
    );
  }
}
