import 'package:flutter/material.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import 'app_theme_color.dart';

class AppPalette extends ThemeExtension<AppPalette> {
  const AppPalette({
    required this.bg,
    required this.railBg,
    required this.sideBg,
    required this.cardBg,
    required this.cardHoverBg,
    required this.surfaceSoft,
    required this.inputBg,
    required this.footerBg,
    required this.border,
    required this.textPrimary,
    required this.textSecondary,
    required this.textMuted,
    required this.primaryActionBg,
    required this.primaryActionForeground,
    required this.primaryActionHoverBg,
    required this.brand,
    required this.brandForeground,
    required this.brandSoft,
    required this.brandTrack,
    required this.brandProgress,
    required this.accent,
    required this.accentOn,
    required this.error,
    required this.success,
    required this.toggleBg,
    required this.toggleIcon,
    required this.itemActiveBg,
    required this.filterActiveBg,
    required this.progressTrack,
    required this.taskCardSelectedBorder,
    required this.taskCardHoverBg,
    required this.taskCardAction,
    required this.taskCardProgressFill,
    required this.taskCardIconBg,
    required this.taskCardFailedIconBg,
    required this.taskMeta,
    required this.taskMetaSubtle,
    required this.taskErrorMeta,
    required this.searchHint,
    required this.headerDivider,
    required this.captionButtonHover,
    required this.captionButtonCloseHover,
  });

  static const Color lightAccent = Color(0xFF79C476);

  static const light = AppPalette(
    bg: Color(0xFFF4F7F6),
    railBg: Color(0xFFFFFFFF),
    sideBg: Color(0xFFFCFDFD),
    cardBg: Color(0xFFFFFFFF),
    cardHoverBg: Color(0xFFF5F7F5),
    surfaceSoft: Color(0xFFEDF0EE),
    inputBg: Color(0xFFE7E8E9),
    footerBg: Color(0xFFF3F4F5),
    border: Color(0x66C0C9BA),
    textPrimary: Color(0xFF191C1D),
    textSecondary: Color(0xFF5F6368),
    textMuted: Color(0xFF707A6D),
    primaryActionBg: Color(0xFF191C1D),
    primaryActionForeground: Color(0xFFFFFFFF),
    primaryActionHoverBg: Color(0xFF303435),
    brand: lightAccent,
    brandForeground: Color(0xFFFFFFFF),
    brandSoft: Color(0x1479C476),
    brandTrack: Color(0x2679C476),
    brandProgress: Color(0xFF79C476),
    accent: Color(0xFFE7E8E9),
    accentOn: Color(0xFFF4F7F6),
    error: Color(0xFFBA1A1A),
    success: Color(0xFF2E7D32),
    toggleBg: Color(0xFFFFFFFF),
    toggleIcon: Color(0xFF5F6368),
    itemActiveBg: Color(0x0F191C1D),
    filterActiveBg: Color(0x3379C476),
    progressTrack: Color(0x33C0C9BA),
    taskCardSelectedBorder: Color(0x24191C1D),
    taskCardHoverBg: Color(0xFFEAF0EC),
    taskCardAction: Color(0xFF5F6368),
    taskCardProgressFill: Color(0xFF79C476),
    taskCardIconBg: Color(0xFFF5F7F5),
    taskCardFailedIconBg: Color(0x14BA1A1A),
    taskMeta: Color(0xFF5F6368),
    taskMetaSubtle: Color(0xFF707A6D),
    taskErrorMeta: Color(0xCCBA1A1A),
    searchHint: Color(0xFF707A6D),
    headerDivider: Color(0xA3C0C9BA),
    captionButtonHover: Color(0x0F191C1D),
    captionButtonCloseHover: Color(0xFFE81123),
  );

  static final dark = AppPalette(
    bg: const Color(0xFF09090B),
    railBg: Colors.black,
    sideBg: const Color(0xFF111111),
    cardBg: const Color(0xFF0F0F11),
    cardHoverBg: const Color(0xFF121215),
    surfaceSoft: const Color(0xFF1F1F24),
    inputBg: const Color(0xFF111111),
    footerBg: Colors.black.withValues(alpha: 0.2),
    border: const Color(0xFF1B1B1D),
    textPrimary: const Color(0xFFFFFFFF),
    textSecondary: const Color(0xFFA1A1AA),
    textMuted: const Color(0xFF71717A),
    primaryActionBg: const Color(0xFFFAFAFA),
    primaryActionForeground: const Color(0xFF18181B),
    primaryActionHoverBg: const Color(0xFFE4E4E7),
    brand: const Color(0xFFA7F5A2),
    brandForeground: const Color(0xFF07310B),
    brandSoft: const Color(0x1AA7F5A2),
    brandTrack: const Color(0x2CA7F5A2),
    brandProgress: const Color(0xFFA7F5A2),
    accent: const Color(0xFF111111),
    accentOn: const Color(0xFF09090B),
    error: const Color(0xFFFFB4AB),
    success: const Color(0xFF81C784),
    toggleBg: Colors.white.withValues(alpha: 0.08),
    toggleIcon: const Color(0xFFA1A1AA),
    itemActiveBg: Colors.white.withValues(alpha: 0.08),
    filterActiveBg: const Color(0x3DA7F5A2),
    progressTrack: const Color(0xFF272729),
    taskCardSelectedBorder: const Color(0xFF3F3F41),
    taskCardHoverBg: const Color(0xFF171719),
    taskCardAction: const Color(0xFFC6C6C6),
    taskCardProgressFill: const Color(0xFFA7F5A2),
    taskCardIconBg: const Color(0xFF1B1B1E),
    taskCardFailedIconBg: const Color(0xFF272020),
    taskMeta: const Color(0xFF8F8F90),
    taskMetaSubtle: const Color(0xFF6B6B6C),
    taskErrorMeta: const Color(0xFFCF938C),
    searchHint: const Color(0xFF52525B),
    headerDivider: const Color(0x1AFFFFFF),
    captionButtonHover: const Color(0x0DFFFFFF),
    captionButtonCloseHover: const Color(0xFFE81123),
  );

  static AppPalette lightFor(AppThemeColor themeColor) {
    if (themeColor == AppThemeColor.green) return light;
    return _withThemeColor(light, color: themeColor.light, foreground: themeColor.lightForeground, dark: false);
  }

  static AppPalette darkFor(AppThemeColor themeColor) {
    if (themeColor == AppThemeColor.green) return dark;
    return _withThemeColor(dark, color: themeColor.dark, foreground: themeColor.darkForeground, dark: true);
  }

  static AppPalette _withThemeColor(
    AppPalette palette, {
    required Color color,
    required Color foreground,
    required bool dark,
  }) {
    return palette.copyWith(
      brand: color,
      brandForeground: foreground,
      brandSoft: color.withValues(alpha: dark ? 0.10 : 0.08),
      brandTrack: color.withValues(alpha: dark ? 0.17 : 0.15),
      brandProgress: color,
      filterActiveBg: color.withValues(alpha: dark ? 0.24 : 0.20),
      taskCardProgressFill: color,
    );
  }

  final Color bg;
  final Color railBg;
  final Color sideBg;
  final Color cardBg;
  final Color cardHoverBg;
  final Color surfaceSoft;
  final Color inputBg;
  final Color footerBg;
  final Color border;
  final Color textPrimary;
  final Color textSecondary;
  final Color textMuted;
  final Color primaryActionBg;
  final Color primaryActionForeground;
  final Color primaryActionHoverBg;
  final Color brand;
  final Color brandForeground;
  final Color brandSoft;
  final Color brandTrack;
  final Color brandProgress;
  final Color accent;
  final Color accentOn;
  final Color error;
  final Color success;
  final Color toggleBg;
  final Color toggleIcon;
  final Color itemActiveBg;
  final Color filterActiveBg;
  final Color progressTrack;
  final Color taskCardSelectedBorder;
  final Color taskCardHoverBg;
  final Color taskCardAction;
  final Color taskCardProgressFill;
  final Color taskCardIconBg;
  final Color taskCardFailedIconBg;
  final Color taskMeta;
  final Color taskMetaSubtle;
  final Color taskErrorMeta;
  final Color searchHint;
  final Color headerDivider;
  final Color captionButtonHover;
  final Color captionButtonCloseHover;

  static AppPalette of(BuildContext context) {
    final extension = Theme.of(context).extension<AppPalette>();
    if (extension != null) {
      return extension;
    }
    return shad.Theme.of(context).brightness == Brightness.light ? light : dark;
  }

  @override
  AppPalette copyWith({
    Color? bg,
    Color? railBg,
    Color? sideBg,
    Color? cardBg,
    Color? cardHoverBg,
    Color? surfaceSoft,
    Color? inputBg,
    Color? footerBg,
    Color? border,
    Color? textPrimary,
    Color? textSecondary,
    Color? textMuted,
    Color? primaryActionBg,
    Color? primaryActionForeground,
    Color? primaryActionHoverBg,
    Color? brand,
    Color? brandForeground,
    Color? brandSoft,
    Color? brandTrack,
    Color? brandProgress,
    Color? accent,
    Color? accentOn,
    Color? error,
    Color? success,
    Color? toggleBg,
    Color? toggleIcon,
    Color? itemActiveBg,
    Color? filterActiveBg,
    Color? progressTrack,
    Color? taskCardSelectedBorder,
    Color? taskCardHoverBg,
    Color? taskCardAction,
    Color? taskCardProgressFill,
    Color? taskCardIconBg,
    Color? taskCardFailedIconBg,
    Color? taskMeta,
    Color? taskMetaSubtle,
    Color? taskErrorMeta,
    Color? searchHint,
    Color? headerDivider,
    Color? captionButtonHover,
    Color? captionButtonCloseHover,
  }) {
    return AppPalette(
      bg: bg ?? this.bg,
      railBg: railBg ?? this.railBg,
      sideBg: sideBg ?? this.sideBg,
      cardBg: cardBg ?? this.cardBg,
      cardHoverBg: cardHoverBg ?? this.cardHoverBg,
      surfaceSoft: surfaceSoft ?? this.surfaceSoft,
      inputBg: inputBg ?? this.inputBg,
      footerBg: footerBg ?? this.footerBg,
      border: border ?? this.border,
      textPrimary: textPrimary ?? this.textPrimary,
      textSecondary: textSecondary ?? this.textSecondary,
      textMuted: textMuted ?? this.textMuted,
      primaryActionBg: primaryActionBg ?? this.primaryActionBg,
      primaryActionForeground: primaryActionForeground ?? this.primaryActionForeground,
      primaryActionHoverBg: primaryActionHoverBg ?? this.primaryActionHoverBg,
      brand: brand ?? this.brand,
      brandForeground: brandForeground ?? this.brandForeground,
      brandSoft: brandSoft ?? this.brandSoft,
      brandTrack: brandTrack ?? this.brandTrack,
      brandProgress: brandProgress ?? this.brandProgress,
      accent: accent ?? this.accent,
      accentOn: accentOn ?? this.accentOn,
      error: error ?? this.error,
      success: success ?? this.success,
      toggleBg: toggleBg ?? this.toggleBg,
      toggleIcon: toggleIcon ?? this.toggleIcon,
      itemActiveBg: itemActiveBg ?? this.itemActiveBg,
      filterActiveBg: filterActiveBg ?? this.filterActiveBg,
      progressTrack: progressTrack ?? this.progressTrack,
      taskCardSelectedBorder: taskCardSelectedBorder ?? this.taskCardSelectedBorder,
      taskCardHoverBg: taskCardHoverBg ?? this.taskCardHoverBg,
      taskCardAction: taskCardAction ?? this.taskCardAction,
      taskCardProgressFill: taskCardProgressFill ?? this.taskCardProgressFill,
      taskCardIconBg: taskCardIconBg ?? this.taskCardIconBg,
      taskCardFailedIconBg: taskCardFailedIconBg ?? this.taskCardFailedIconBg,
      taskMeta: taskMeta ?? this.taskMeta,
      taskMetaSubtle: taskMetaSubtle ?? this.taskMetaSubtle,
      taskErrorMeta: taskErrorMeta ?? this.taskErrorMeta,
      searchHint: searchHint ?? this.searchHint,
      headerDivider: headerDivider ?? this.headerDivider,
      captionButtonHover: captionButtonHover ?? this.captionButtonHover,
      captionButtonCloseHover: captionButtonCloseHover ?? this.captionButtonCloseHover,
    );
  }

  @override
  AppPalette lerp(ThemeExtension<AppPalette>? other, double t) {
    if (other is! AppPalette) {
      return this;
    }
    return AppPalette(
      bg: Color.lerp(bg, other.bg, t)!,
      railBg: Color.lerp(railBg, other.railBg, t)!,
      sideBg: Color.lerp(sideBg, other.sideBg, t)!,
      cardBg: Color.lerp(cardBg, other.cardBg, t)!,
      cardHoverBg: Color.lerp(cardHoverBg, other.cardHoverBg, t)!,
      surfaceSoft: Color.lerp(surfaceSoft, other.surfaceSoft, t)!,
      inputBg: Color.lerp(inputBg, other.inputBg, t)!,
      footerBg: Color.lerp(footerBg, other.footerBg, t)!,
      border: Color.lerp(border, other.border, t)!,
      textPrimary: Color.lerp(textPrimary, other.textPrimary, t)!,
      textSecondary: Color.lerp(textSecondary, other.textSecondary, t)!,
      textMuted: Color.lerp(textMuted, other.textMuted, t)!,
      primaryActionBg: Color.lerp(primaryActionBg, other.primaryActionBg, t)!,
      primaryActionForeground: Color.lerp(primaryActionForeground, other.primaryActionForeground, t)!,
      primaryActionHoverBg: Color.lerp(primaryActionHoverBg, other.primaryActionHoverBg, t)!,
      brand: Color.lerp(brand, other.brand, t)!,
      brandForeground: Color.lerp(brandForeground, other.brandForeground, t)!,
      brandSoft: Color.lerp(brandSoft, other.brandSoft, t)!,
      brandTrack: Color.lerp(brandTrack, other.brandTrack, t)!,
      brandProgress: Color.lerp(brandProgress, other.brandProgress, t)!,
      accent: Color.lerp(accent, other.accent, t)!,
      accentOn: Color.lerp(accentOn, other.accentOn, t)!,
      error: Color.lerp(error, other.error, t)!,
      success: Color.lerp(success, other.success, t)!,
      toggleBg: Color.lerp(toggleBg, other.toggleBg, t)!,
      toggleIcon: Color.lerp(toggleIcon, other.toggleIcon, t)!,
      itemActiveBg: Color.lerp(itemActiveBg, other.itemActiveBg, t)!,
      filterActiveBg: Color.lerp(filterActiveBg, other.filterActiveBg, t)!,
      progressTrack: Color.lerp(progressTrack, other.progressTrack, t)!,
      taskCardSelectedBorder: Color.lerp(taskCardSelectedBorder, other.taskCardSelectedBorder, t)!,
      taskCardHoverBg: Color.lerp(taskCardHoverBg, other.taskCardHoverBg, t)!,
      taskCardAction: Color.lerp(taskCardAction, other.taskCardAction, t)!,
      taskCardProgressFill: Color.lerp(taskCardProgressFill, other.taskCardProgressFill, t)!,
      taskCardIconBg: Color.lerp(taskCardIconBg, other.taskCardIconBg, t)!,
      taskCardFailedIconBg: Color.lerp(taskCardFailedIconBg, other.taskCardFailedIconBg, t)!,
      taskMeta: Color.lerp(taskMeta, other.taskMeta, t)!,
      taskMetaSubtle: Color.lerp(taskMetaSubtle, other.taskMetaSubtle, t)!,
      taskErrorMeta: Color.lerp(taskErrorMeta, other.taskErrorMeta, t)!,
      searchHint: Color.lerp(searchHint, other.searchHint, t)!,
      headerDivider: Color.lerp(headerDivider, other.headerDivider, t)!,
      captionButtonHover: Color.lerp(captionButtonHover, other.captionButtonHover, t)!,
      captionButtonCloseHover: Color.lerp(captionButtonCloseHover, other.captionButtonCloseHover, t)!,
    );
  }
}
