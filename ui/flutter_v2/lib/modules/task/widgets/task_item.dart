import 'dart:ui';
import 'package:flutter/material.dart';

/// TaskItem - 任务项组件
///
/// 一个带有毛玻璃效果和渐变边框的任务项容器，宽度自适应父组件
class TaskItem extends StatelessWidget {
  /// 左侧图标
  final dynamic icon;

  /// 任务名称
  final String? taskName;

  /// 进度值 (0.0 - 1.0)
  final double? progress;

  /// 右上角操作按钮列表
  final List<Widget>? actionButtons;

  /// 容器高度，默认106px
  final double height;

  /// 圆角大小，默认8px
  final double borderRadius;

  /// 点击回调
  final VoidCallback? onTap;

  /// 文件大小文本（如2.1MB/2.4MB）
  final String? fileSizeText;

  /// 右下角图标
  final IconData? bottomRightIcon;

  /// 右下角渐变文本
  final String? bottomRightGradientText;

  /// 右下角状态文本
  final String? bottomRightStatusText;

  const TaskItem({
    super.key,
    this.icon,
    this.taskName,
    this.progress,
    this.actionButtons,
    this.height = 106.0,
    this.borderRadius = 8.0,
    this.onTap,
    this.fileSizeText,
    this.bottomRightIcon,
    this.bottomRightGradientText,
    this.bottomRightStatusText,
  });

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor:
          onTap != null ? SystemMouseCursors.click : SystemMouseCursors.basic,
      child: GestureDetector(
        onTap: onTap,
        child: Container(
          height: height,
          child: Stack(
            children: [
              // Rectangle 82: 主容器 (最底层)
              Positioned.fill(
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(borderRadius),
                  child: BackdropFilter(
                    filter: ImageFilter.blur(sigmaX: 24.0, sigmaY: 24.0),
                    child: Container(
                      decoration: BoxDecoration(
                        // 背景色：rgba(178, 231, 254, 0.08)
                        color: const Color(0xFFB2E7FE).withOpacity(0.08),
                        borderRadius: BorderRadius.circular(borderRadius),
                        border: Border.all(
                          width: 0.5,
                          color: Colors.transparent,
                        ),
                      ),
                      child: Container(
                        decoration: BoxDecoration(
                          borderRadius: BorderRadius.circular(borderRadius),
                          // 渐变边框效果
                          gradient: const LinearGradient(
                            begin: Alignment.topLeft,
                            end: Alignment.bottomRight,
                            stops: [0.17, 0.92],
                            colors: [
                              Colors.transparent,
                              Color(0x29FFFFFF), // rgba(255, 255, 255, 0.16)
                            ],
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),

              // 蒙版组 (中间层) - 简化测试
              Positioned.fill(
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(borderRadius),
                  child: Stack(
                    children: [
                      // Rectangle 80: 渐变模糊背景 - 只保留这个
                      Positioned(
                        left: 0,
                        right: 0,
                        top: 7.23,
                        child: ImageFiltered(
                          imageFilter: ImageFilter.blur(
                            sigmaX: 40.0,
                            sigmaY: 40.0,
                          ),
                          child: Container(
                            height: 90.34,
                            decoration: BoxDecoration(
                              borderRadius: BorderRadius.circular(borderRadius),
                              gradient: const LinearGradient(
                                begin: Alignment.centerLeft, // 90deg = 从左到右
                                end: Alignment.centerRight,
                                stops: [0.0, 1.0],
                                colors: [
                                  Color(0xFF2D3975), // #2D3975 at 0%
                                  Color(0xFF266682), // #266682 at 100%
                                ],
                              ),
                            ),
                          ),
                        ),
                      ),

                      // Rectangle 81: 纯色背景蒙版 - 图层样式穿透，所以注释掉
                      // 设计稿中标明"图层样式穿透"，意味着这层不影响最终视觉效果
                      // Positioned(
                      //   left: 0,
                      //   right: 0,
                      //   top: 0,
                      //   child: Container(
                      //     height: height,
                      //     decoration: BoxDecoration(
                      //       color: const Color(0xFFB2E7FE),
                      //       borderRadius: BorderRadius.circular(borderRadius),
                      //     ),
                      //   ),
                      // ),
                    ],
                  ),
                ),
              ),

              // 内容层 (最上层)
              Positioned.fill(
                child: Container(
                  padding: const EdgeInsets.all(16.0), // 统一的16px内边距
                  child: Row(
                    children: [
                      // 左侧图标区域
                      if (icon != null)
                        Container(
                          margin: const EdgeInsets.only(
                            right: 2.0,
                          ), // 图标右边间隔2px
                          child: _buildIcon(),
                        ),

                      // 右侧内容容器
                      Expanded(
                        child: Container(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              // 任务名称区域 (与操作按钮在同一行)
                              Row(
                                children: [
                                  // 任务名称
                                  if (taskName != null)
                                    Expanded(
                                      child: Container(
                                        height: 22.0, // 设计稿高度22px
                                        child: Align(
                                          alignment:
                                              Alignment
                                                  .bottomLeft, // align-items: flex-end
                                          child: Text(
                                            taskName!,
                                            style: const TextStyle(
                                              fontSize: 16.0, // 16px
                                              fontWeight:
                                                  FontWeight.bold, // bold
                                              height:
                                                  22.0 /
                                                  16.0, // line-height: 22px
                                              color: Color(
                                                0xFFFFFFFE,
                                              ), // #FFFFFE
                                            ),
                                            maxLines: 1,
                                            overflow: TextOverflow.ellipsis,
                                          ),
                                        ),
                                      ),
                                    ),

                                  // 右侧操作按钮列表
                                  if (actionButtons != null &&
                                      actionButtons!.isNotEmpty)
                                    Row(
                                      mainAxisSize: MainAxisSize.min,
                                      children: _buildActionButtons(),
                                    ),
                                ],
                              ),

                              // 进度条
                              if (progress != null)
                                Container(
                                  margin: const EdgeInsets.only(
                                    top: 20.0,
                                  ), // 距离名称下面20px
                                  child: _buildProgressBar(),
                                ),

                              // 文件大小文本和右下角元素行
                              Container(
                                margin: const EdgeInsets.only(
                                  top: 8.0,
                                ), // 距离进度条下面8px
                                height: 16.0, // 直接指定高度为16px
                                child: Row(
                                  crossAxisAlignment:
                                      CrossAxisAlignment.center, // 垂直居中对齐
                                  children: [
                                    // 左侧文件大小文本
                                    if (fileSizeText != null &&
                                        fileSizeText!.isNotEmpty)
                                      Expanded(
                                        child: Align(
                                          alignment: Alignment.centerLeft,
                                          child: Text(
                                            fileSizeText!,
                                            style: const TextStyle(
                                              fontSize: 14.0,
                                              fontWeight: FontWeight.w500,
                                              color: Color(
                                                0xFFCCCCCC,
                                              ), // #CCCCCC
                                              letterSpacing: 0.0,
                                            ),
                                            maxLines: 1,
                                            overflow: TextOverflow.ellipsis,
                                          ),
                                        ),
                                      ),

                                    // 右下角元素组
                                    _buildBottomRightElements(),
                                  ],
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  /// 构建图标组件
  Widget _buildIcon() {
    if (icon == null) return const SizedBox.shrink();

    // 如果是字符串，认为是资源路径
    if (icon is String) {
      return Image.asset(
        icon as String,
        width: 32.0,
        height: 32.0,
        fit: BoxFit.contain,
      );
    }

    // 如果是IconData，使用Icon组件
    if (icon is IconData) {
      return Icon(icon as IconData, size: 32.0, color: Colors.white);
    }

    // 如果是Widget，直接使用
    if (icon is Widget) {
      return SizedBox(width: 32.0, height: 32.0, child: icon as Widget);
    }

    return const SizedBox.shrink();
  }

  /// 构建操作按钮列表
  List<Widget> _buildActionButtons() {
    if (actionButtons == null || actionButtons!.isEmpty) {
      return [];
    }

    List<Widget> buttons = [];
    for (int i = 0; i < actionButtons!.length; i++) {
      // 添加按钮
      buttons.add(
        SizedBox(width: 20.0, height: 20.0, child: actionButtons![i]),
      );

      // 添加间隔 (除了最后一个按钮)
      if (i < actionButtons!.length - 1) {
        buttons.add(const SizedBox(width: 16.0));
      }
    }

    return buttons;
  }

  /// 构建进度条
  Widget _buildProgressBar() {
    if (progress == null) return const SizedBox.shrink();

    return LayoutBuilder(
      builder: (context, constraints) {
        // 使用所有可用宽度，因为已经有统一的16px内边距
        final availableWidth = constraints.maxWidth;
        final progressWidth = availableWidth * progress!;

        return SizedBox(
          height: 8.0,
          child: Stack(
            children: [
              // 底部背景条
              Container(
                width: availableWidth,
                height: 8.0,
                decoration: BoxDecoration(
                  color: const Color(0xFFEBEDF3), // #EBEDF3
                  borderRadius: BorderRadius.circular(548.0), // 548px圆角
                ),
              ),

              // 进度条
              Container(
                width: progressWidth,
                height: 8.0,
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(143.0), // 143px圆角
                  gradient: const LinearGradient(
                    begin: Alignment.centerLeft, // 90deg
                    end: Alignment.centerRight,
                    stops: [0.0, 0.44, 0.99],
                    colors: [
                      Color(0xC93ACBBE), // rgba(58, 203, 190, 0.79)
                      Color(0xFF3ACBBE), // #3ACBBE at 44%
                      Color(0xFF2CDB90), // #2CDB90 at 99%
                    ],
                  ),
                  border: Border.all(width: 1.0, color: Colors.transparent),
                ),
                child: Container(
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(143.0),
                    // 渐变边框效果
                    gradient: const LinearGradient(
                      begin: Alignment.topCenter, // 180deg
                      end: Alignment.bottomCenter,
                      colors: [
                        Color(0xFF39CBBC), // #39CBBC at 0%
                        Color(0xFF2CDA92), // #2CDA92 at 100%
                      ],
                    ),
                  ),
                  child: Container(
                    margin: const EdgeInsets.all(1.0),
                    decoration: BoxDecoration(
                      borderRadius: BorderRadius.circular(142.0),
                      gradient: const LinearGradient(
                        begin: Alignment.centerLeft,
                        end: Alignment.centerRight,
                        stops: [0.0, 0.44, 0.99],
                        colors: [
                          Color(0xC93ACBBE), // rgba(58, 203, 190, 0.79)
                          Color(0xFF3ACBBE), // #3ACBBE at 44%
                          Color(0xFF2CDB90), // #2CDB90 at 99%
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  /// 构建右下角元素组
  Widget _buildBottomRightElements() {
    // 如果没有任何右下角元素，返回空
    if (bottomRightIcon == null &&
        bottomRightGradientText == null &&
        bottomRightStatusText == null) {
      return const SizedBox.shrink();
    }

    List<Widget> children = [];

    // 添加图标
    if (bottomRightIcon != null) {
      children.add(Icon(bottomRightIcon!, size: 16.0, color: Colors.white));
    }

    // 添加渐变文本
    if (bottomRightGradientText != null &&
        bottomRightGradientText!.isNotEmpty) {
      if (children.isNotEmpty) {
        children.add(const SizedBox(width: 4.0)); // 间隔4px
      }

      children.add(
        ShaderMask(
          shaderCallback:
              (bounds) => const LinearGradient(
                begin: Alignment.centerLeft,
                end: Alignment.centerRight,
                transform: GradientRotation(272 * 3.14159 / 180), // 272度转弧度
                stops: [0.73, 1.22],
                colors: [
                  Color(0xFF39CDB9), // #39CDB9 at 73%
                  Color(0xFF2EDA94), // #2EDA94 at 122%
                ],
              ).createShader(bounds),
          child: Text(
            bottomRightGradientText!,
            style: const TextStyle(
              fontSize: 14.0,
              fontWeight: FontWeight.w500,
              color: Colors.white, // 这个颜色会被shader覆盖
              letterSpacing: 0.0,
              height: 1.0, // 设置行高为1，减少额外空间
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
        ),
      );
    }

    // 添加状态文本
    if (bottomRightStatusText != null && bottomRightStatusText!.isNotEmpty) {
      if (children.isNotEmpty) {
        children.add(const SizedBox(width: 12.0)); // 间隔12px
      }

      children.add(
        Text(
          bottomRightStatusText!,
          style: const TextStyle(
            fontFamily: 'Source Han Sans',
            fontSize: 14.0,
            fontWeight: FontWeight.w500,
            color: Color(0xFFCCCCCC), // #CCCCCC
            letterSpacing: 0.0,
            height: 1.0, // 设置行高为1，减少额外空间
          ),
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
      );
    }

    return Row(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.center, // 垂直居中对齐
      textBaseline: TextBaseline.alphabetic, // 统一文本基线
      children: children,
    );
  }
}
