import 'package:flutter/material.dart';

/// GpOutlineButton - 线框按钮组件
///
/// 一个带有线框边框的小型按钮组件，支持自定义图标
class GpOutlineButton extends StatelessWidget {
  /// 按钮内的图标路径或图标数据
  final dynamic icon;

  /// 点击回调
  final VoidCallback? onTap;

  /// 按钮宽度，默认42px
  final double width;

  /// 按钮高度，默认22px
  final double height;

  /// 边框颜色，默认#39CDB9
  final Color borderColor;

  /// 边框圆角，默认4px
  final double borderRadius;

  /// 边框宽度，默认1px
  final double borderWidth;

  /// 图标颜色
  final Color? iconColor;

  /// 图标大小
  final double? iconSize;

  /// 背景颜色
  final Color? backgroundColor;

  const GpOutlineButton({
    super.key,
    required this.icon,
    this.onTap,
    this.width = 42.0,
    this.height = 22.0,
    this.borderColor = const Color(0xFF39CDB9),
    this.borderRadius = 4.0,
    this.borderWidth = 1.0,
    this.iconColor,
    this.iconSize,
    this.backgroundColor,
  });

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor:
          onTap != null ? SystemMouseCursors.click : SystemMouseCursors.basic,
      child: GestureDetector(
        onTap: onTap,
        child: Container(
          width: width,
          height: height,
          decoration: BoxDecoration(
            color: backgroundColor,
            border: Border.all(color: borderColor, width: borderWidth),
            borderRadius: BorderRadius.circular(borderRadius),
          ),
          child: _buildIcon(),
        ),
      ),
    );
  }

  Widget _buildIcon() {
    if (icon == null) return const SizedBox.shrink();

    // 如果是字符串，认为是资源路径
    if (icon is String) {
      return Center(
        child: Image.asset(
          icon as String,
          width: iconSize,
          height: iconSize,
          color: iconColor,
          fit: BoxFit.contain,
        ),
      );
    }

    // 如果是IconData，使用Icon组件
    if (icon is IconData) {
      return Center(
        child: Icon(icon as IconData, color: iconColor, size: iconSize ?? 16.0),
      );
    }

    // 如果是Widget，直接使用
    if (icon is Widget) {
      return Center(child: icon as Widget);
    }

    return const SizedBox.shrink();
  }
}
