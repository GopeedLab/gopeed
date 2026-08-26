import 'package:flutter/widgets.dart';

class AppDesignTokens {
  const AppDesignTokens._();

  static const double windowHeaderHeight = 24;
  static const double windowRadius = 12;
  static const double railWidth = 68;
  static const double filterSidebarWidth = 192;
  static const double contentHeaderHeight = 48;
  static const double taskRowHeight = 72;
  static const double controlRadius = 4;
  static const double taskDetailsDrawerMinWidth = 440;
  static const double taskDetailsDrawerMaxWidth = 720;
  static const double taskDetailsDrawerViewportRatio = 0.42;
  static const double taskDetailsDesktopPadding = 24;
  static const double taskDetailsMobilePadding = 16;
  static const double settingsContentMaxWidth = 920;
  static const double settingsFormControlWidth = 420;
  static const double settingsNumberControlWidth = 168;
  static const double settingsItemMinLabelWidth = 160;

  static const EdgeInsets pagePadding = EdgeInsets.symmetric(horizontal: 32);
  static const EdgeInsets sidebarPadding = EdgeInsets.symmetric(horizontal: 16);

  static const double space4 = 4;
  static const double space8 = 8;
  static const double space12 = 12;
  static const double space16 = 16;
  static const double space24 = 24;
  static const double space32 = 32;

  static double taskDetailsDrawerWidth(double viewportWidth) {
    return (viewportWidth * taskDetailsDrawerViewportRatio)
        .clamp(taskDetailsDrawerMinWidth, taskDetailsDrawerMaxWidth)
        .toDouble();
  }
}
