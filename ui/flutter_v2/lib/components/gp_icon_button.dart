import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

/// GpIconButton - 简单的图标按钮组件
///
/// 用于TaskItem的操作按钮，支持鼠标悬浮效果
class GpIconButton extends StatefulWidget {
  /// 图标（支持 IconData 或 String 路径）
  final dynamic icon;

  /// 点击回调
  final VoidCallback? onTap;

  /// 图标颜色
  final Color? color;

  /// 图标大小
  final double? size;

  const GpIconButton({
    super.key,
    required this.icon,
    this.onTap,
    this.color,
    this.size,
  });

  @override
  State<GpIconButton> createState() => _GpIconButtonState();
}

class _GpIconButtonState extends State<GpIconButton> {
  bool _isHovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor:
          widget.onTap != null
              ? SystemMouseCursors.click
              : SystemMouseCursors.basic,
      onEnter: (_) => setState(() => _isHovered = true),
      onExit: (_) => setState(() => _isHovered = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 200),
          width: widget.size ?? 20.0,
          height: widget.size ?? 20.0,
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(4.0), // 8px圆角
            color:
                _isHovered
                    ? const Color(0xFF4883A2)
                    : Colors.transparent, // 悬浮背景色
          ),
          child: _buildIcon(),
        ),
      ),
    );
  }

  Widget _buildIcon() {
    // 如果是字符串，认为是资源路径
    if (widget.icon is String) {
      final String path = widget.icon as String;
      // 支持 svg 资源
      if (path.toLowerCase().endsWith('.svg')) {
        return Center(
          child: SvgPicture.asset(
            path,
            width: widget.size ?? 20.0,
            height: widget.size ?? 20.0,
            fit: BoxFit.contain,
            colorFilter: ColorFilter.mode(
              widget.color ?? Colors.white,
              BlendMode.srcIn,
            ),
          ),
        );
      }

      return Center(
        child: Image.asset(
          path,
          width: widget.size ?? 20.0,
          height: widget.size ?? 20.0,
          color: widget.color,
          fit: BoxFit.contain,
        ),
      );
    }

    // 如果是IconData，使用Icon组件
    if (widget.icon is IconData) {
      return Icon(
        widget.icon as IconData,
        color: widget.color ?? Colors.white,
        size: widget.size ?? 20.0,
      );
    }

    // 如果是Widget，直接使用
    if (widget.icon is Widget) {
      return Center(child: widget.icon as Widget);
    }

    return const SizedBox.shrink();
  }
}
