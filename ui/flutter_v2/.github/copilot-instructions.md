# Gopeed Flutter V2 UI - AI Coding Instructions

## Response Rules

Respond in the same language as user's question

## Project Overview

This is the Flutter V2 UI for **Gopeed** (Go Speed), a high-speed download manager built with Golang + Flutter. This Flutter app is a **desktop-focused UI redesign** with cross-platform support (macOS, Windows, Linux, iOS, Android, Web).

**Critical Context**: This Flutter UI communicates with a Go backend REST API (not in this workspace). The backend handles download operations, task management, and file I/O. The UI is purely presentational with future API integration planned.

## Architecture & Structure

### Module Organization (Feature-First)

```
lib/
├── main.dart                    # App entry, window config, router setup
├── routes/                      # Centralized routing
│   ├── app_routes.dart         # go_router configuration
│   └── route_names.dart        # Route path constants
├── modules/                     # Feature modules (screens + widgets)
│   ├── home/                   # Shell layout with sidebar
│   ├── task/                   # Download task management UI
│   └── settings/               # Settings screen
└── components/                  # Shared UI components (gp_* prefix)
```

**Key Pattern**: Features are organized in `modules/` with `screens/` and `widgets/` subdirectories. Shared components use `gp_` prefix (e.g., `GpGradientButton`, `GpIconButton`).

### Routing Architecture

- Uses `go_router` with `ShellRoute` for persistent sidebar navigation
- `HomeScreen` wraps all routes and provides the sidebar layout
- Route paths defined in `RouteNames` class (prevents magic strings)
- Current route detection via `GoRouterState.of(context).matchedLocation`

## Critical Patterns & Conventions

### Component Design System

1. **Gradient Buttons** (`GpGradientButton`): Primary actions, teal-to-green gradient (Colors: `0xFF3ACBBE` → `0xFF2CDB90`)
2. **Outline Buttons** (`GpOutlineButton`): Secondary actions, border-only with teal accent (`0xFF39CDB9`)
3. **Icon Buttons** (`GpIconButton`): Tertiary actions with hover effects (`0xFF4883A2` on hover)

### Design Implementation

- **Glass Morphism**: `TaskItem` widget uses `BackdropFilter` with blur (24px) and layered gradients
- **Color Palette**: Dark blue theme (`0xFF131956` background, `0xFF000E4B` sidebar, teal/green accents)
- **Comments Reference Design Specs**: Many widgets have detailed comments like "Rectangle 82: 主容器" mapping to Figma/design files

### Widget Documentation Style

Components use triple-slash (`///`) documentation:

```dart
/// TaskItem - 任务项组件
///
/// 一个带有毛玻璃效果和渐变边框的任务项容器
class TaskItem extends StatelessWidget {
  /// 左侧图标
  final dynamic icon;
```

**Convention**: Document widget purpose, parameters, and design intent in Chinese. Keep inline comments for implementation details.

### State Management

- Currently uses `StatefulWidget` with local state (e.g., `TaskScreen._selectedTabIndex`)
- **Future Integration**: Will integrate with backend API - prepare for state management library (GetX/Riverpod likely based on Gopeed ecosystem)

### Window Management

Desktop apps use `window_manager` package:

- Initial size: 1024x768, minimum: 800x600
- Hidden title bar (`TitleBarStyle.hidden`) for custom UI
- Configured in `main.dart` before `runApp()`

## Development Workflows

### Running the App

```bash
# Desktop (macOS/Windows/Linux)
flutter run -d macos  # or windows/linux

# Mobile
flutter run -d ios    # requires Xcode
flutter run -d android

# Web
flutter run -d chrome
```

### Building

```bash
flutter build macos --release
flutter build windows --release
flutter build linux --release
```

### Dependencies

Key packages in `pubspec.yaml`:

- `window_manager: ^0.4.3` - Desktop window control
- `flutter_svg: ^2.0.17` - SVG asset support
- `go_router: ^14.8.1` - Declarative routing

Assets in `assets/icons/` (e.g., `gopeed_sidebar.svg`, `pause.png`)

## Backend Integration (Future)

**Not Yet Implemented**: Based on parent project structure, this UI will connect to Go REST API:

- API endpoints: `/api/task`, `/api/resolve`, `/api/config` (reference: `pkg/rest/api.go` in parent)
- Task operations: create, pause, continue, delete
- Real-time progress updates via events/WebSocket

**When Implementing API**:

1. Create `lib/api/` or `lib/services/` directory
2. Use `http` or `dio` package for REST calls
3. Models should match Go backend structs (e.g., `Task`, `Progress`)
4. Replace mock data in `TaskScreen._buildTaskList()` with API calls

## Testing & Quality

- Lints: Uses `package:flutter_lints/flutter.yaml` (standard Flutter best practices)
- No tests currently - **add tests when backend integration begins**
- Desktop focus: Prioritize macOS/Windows, mobile is secondary

## Common Tasks

### Adding a New Screen

1. Create `lib/modules/[feature]/screens/[screen_name].dart`
2. Add route to `RouteNames` constant
3. Register route in `AppRoutes.getRouter()` under `ShellRoute.routes`

### Adding Shared Component

1. Create `lib/components/gp_[component_name].dart`
2. Follow naming: `Gp` prefix + PascalCase
3. Document with `///` comments
4. Support flexible sizing/theming via parameters

### Design-to-Code Workflow

When implementing from design files:

- Preserve design layer names in comments (e.g., "Rectangle 82")
- Use exact hex colors from specs
- Document gradient stops and blur sigma values
- Stack layers match design hierarchy (bottom-to-top)

## Important Notes

- **Language**: UI text is in Chinese (e.g., "下载中", "已完成", "任务失败")
- **Branch Context**: Current branch is `feat/new_ui` - this is a redesign
- **Desktop First**: Window management, mouse hover states prioritized
- **Parent Project**: Part of larger Gopeed monorepo with Go backend (outside this workspace)

---

**When uncertain**: Check `lib/modules/task/widgets/task_item.dart` for complex widget patterns, `lib/routes/app_routes.dart` for routing examples, or `lib/main.dart` for app initialization.
