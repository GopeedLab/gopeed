import 'package:flutter/material.dart';
import 'package:flutter_svg/svg.dart';
import 'package:go_router/go_router.dart';

import '../../../routes/route_names.dart';

const _sidebarItemBgColor = Color(0xFFCDCDCD);

class _SidebarItem {
  final IconData icon;
  final String label;
  final String route;

  _SidebarItem(this.icon, this.label, this.route);
}

class Sidebar extends StatefulWidget {
  const Sidebar({super.key});

  @override
  State<Sidebar> createState() => _SidebarState();
}

class _SidebarState extends State<Sidebar> {
  @override
  Widget build(BuildContext context) {
    final menuItems = [
      _SidebarItem(Icons.download, '任务', RouteNames.task),
      _SidebarItem(Icons.settings, '设置', RouteNames.settings),
    ];

    // Get current route
    final currentRoute = GoRouterState.of(context).matchedLocation;

    return Container(
      width: 90,
      color: const Color(0xFF000E4B),
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.only(top: 27),
            child: SvgPicture.asset(
              'assets/icons/gopeed_sidebar.svg',
              width: 66,
              height: 66,
            ),
          ),
          const SizedBox(height: 20),
          for (int i = 0; i < menuItems.length; i++)
            InkWell(
              onTap: () {
                if (currentRoute != menuItems[i].route) {
                  context.go(menuItems[i].route);
                }
              },
              child: Container(
                decoration: BoxDecoration(
                  color:
                      currentRoute == menuItems[i].route
                          ? const Color(0xFF000000)
                          : null,
                  borderRadius: BorderRadius.circular(8),
                ),
                width: 66,
                padding: const EdgeInsets.symmetric(vertical: 10),
                child: Column(
                  children: [
                    Icon(
                      menuItems[i].icon,
                      color:
                          currentRoute == menuItems[i].route
                              ? const Color(0xFF00E676)
                              : _sidebarItemBgColor,
                    ),
                    const SizedBox(height: 5),
                    Text(
                      menuItems[i].label,
                      style: TextStyle(
                        color:
                            currentRoute == menuItems[i].route
                                ? const Color(0xFF00E676)
                                : _sidebarItemBgColor,
                        fontSize: 12,
                      ),
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }
}
