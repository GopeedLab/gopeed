import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/task.dart' as api;
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/features/tasks/application/tasks_controller.dart';

void main() {
  test('the list counts a group once and sends group IDs for every action', () async {
    api.Task task(String id) => api.Task.fromJson({
      'id': id,
      'name': '$id.txt',
      'group': {'id': 'group', 'name': 'folder', 'path': '/downloads/folder', 'createdAt': '2026-01-01T00:00:00Z'},
      'parentId': 'group',
      'protocol': 'http',
      'status': 'pause',
      'uploading': false,
      'createdAt': '2026-01-01T00:00:00Z',
      'updatedAt': '2026-01-01T00:00:00Z',
      'meta': {
        'req': {'url': 'https://example.com/$id'},
        'opts': {'path': '/downloads/folder', 'size': 100},
      },
      'progress': {'used': 0, 'speed': 0, 'downloaded': 25, 'uploaded': 0, 'uploadSpeed': 0},
    });
    final raw = [task('first'), task('second')];
    final actions = <String>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getTasks, (_) => raw)
      ..bind(GopeedMethods.pauseTask, (id) {
        actions.add('pause:$id');
        return const RpcUnit();
      })
      ..bind(GopeedMethods.continueTask, (id) {
        actions.add('resume:$id');
        return const RpcUnit();
      })
      ..bind(GopeedMethods.pauseTasks, (params) {
        actions.add('pauseBatch:${params['ids']}');
        return const RpcUnit();
      })
      ..bind(GopeedMethods.continueTasks, (params) {
        actions.add('resumeBatch:${params['ids']}');
        return const RpcUnit();
      })
      ..bind(GopeedMethods.deleteTasks, (params) {
        actions.add('delete:${params['ids']}:${params['force']}');
        raw.clear();
        return const RpcUnit();
      });
    final container = ProviderContainer(
      overrides: [appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry)))],
    );
    addTearDown(container.dispose);
    final state = await container.read(tasksControllerProvider.future);
    expect(state.tasks.length, 1);
    expect(state.tasks.single.members.length, 2);
    expect(state.tasks.single.matchesSearch('second.txt'), isTrue);
    expect(state.tasks.single.progress, 0.25);
    final controller = container.read(tasksControllerProvider.notifier);
    await controller.pause('group');
    await controller.resume('group');
    await controller.pauseAll(['group']);
    await controller.resumeAll(['group']);
    await controller.deleteSelected(['group'], force: true);
    expect(actions, [
      'pause:group',
      'resume:group',
      'pauseBatch:[group]',
      'resumeBatch:[group]',
      'delete:[group]:true',
    ]);
    expect(container.read(tasksControllerProvider).value?.tasks, isEmpty);
  });
}
