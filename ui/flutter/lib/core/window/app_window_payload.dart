import 'dart:convert';

enum AppWindowType { main, createTask }

class AppWindowPayload {
  const AppWindowPayload({required this.type, this.data = const {}});

  factory AppWindowPayload.main() => const AppWindowPayload(type: AppWindowType.main);

  factory AppWindowPayload.createTask({Map<String, dynamic>? createTask}) => AppWindowPayload(
    type: AppWindowType.createTask,
    data: {
      ...?createTask == null ? null : {'createTask': createTask},
    },
  );

  factory AppWindowPayload.fromRaw(String raw) {
    if (raw.trim().isEmpty) {
      return AppWindowPayload.main();
    }

    try {
      final decoded = jsonDecode(raw);
      if (decoded is Map<String, dynamic>) {
        final type = switch ((decoded['type'] ?? '').toString()) {
          'create_task' => AppWindowType.createTask,
          _ => AppWindowType.main,
        };
        return AppWindowPayload(type: type, data: decoded);
      }
    } catch (_) {
      // Ignore invalid payload and fallback to main.
    }

    return AppWindowPayload.main();
  }

  final AppWindowType type;
  final Map<String, dynamic> data;

  Map<String, dynamic>? get createTask {
    final raw = data['createTask'];
    return raw is Map<String, dynamic> ? raw : null;
  }

  String toRaw() {
    return jsonEncode({
      'type': switch (type) {
        AppWindowType.main => 'main',
        AppWindowType.createTask => 'create_task',
      },
      ...data,
    });
  }
}
