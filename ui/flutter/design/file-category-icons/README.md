# 文件分类图标源物料

此目录是 Gopeed 文件分类图标的唯一源物料目录，位于 Flutter 工程之外，不得加入 `ui/flutter/pubspec.yaml` 的 `assets` 配置。

## 维护限制

- 不要绕过 `$gopeed-file-category-icons` 技能直接修改、移动或批量替换这里的 SVG 与 manifest。
- 需要调整图标时，必须配套使用该技能的安全区、码位兼容、字体生成和多尺寸预览流程。
- 不要直接编辑生成的 `GopeedIcons.ttf` 或 `gopeed_icons.dart`。
- 不要复用或重排 manifest 中已有的 codepoint。

## 目录内容

- `svg/`：24×24 的可编辑 SVG 源文件。
- `manifest.json`：图标名称、SVG、稳定码位与产物路径的唯一映射。
- `preview/`：生成的 16/20/24/32px 明暗预览，用于整套视觉审查。

## 生成产物

- Flutter 字体：`ui/flutter/assets/fonts/GopeedIcons.ttf`
- Flutter 常量：`ui/flutter/lib/core/icons/gopeed_icons.dart`

所有修改都应遵循：

```text
使用 $gopeed-file-category-icons 修改 SVG/manifest
→ 重新生成完整 TTF 和 Dart 常量
→ 检查多尺寸明暗预览
→ 运行 Flutter 静态分析与测试
```
