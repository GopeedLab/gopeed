import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api_client;
import 'package:gopeed/api/model/task.dart' as api;
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';
import 'package:gopeed/features/tasks/domain/task_record.dart';
import 'package:gopeed/features/tasks/presentation/widgets/task_file_manager.dart';
import 'package:gopeed/shared/theme/app_theme.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;
import 'package:flutter/services.dart';

class _Transport implements GopeedTransport {
  @override
  String join(String path) => 'https://server.example$path';
  @override
  Future<dynamic> request(
    String path, {
    String method = 'GET',
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async => null;
}

api.Task fixture(
  String id, {
  required bool group,
  required String directory,
  String name = 'episode #2?.txt',
  bool folder = false,
}) => api.Task.fromJson({
  'id': id,
  'name': group
      ? 'folder'
      : folder
      ? 'nested-torrent'
      : name,
  'type': group ? 'group' : '',
  'parentId': group ? '' : 'group',
  'protocol': group ? '' : 'http',
  'status': 'done',
  'uploading': false,
  'createdAt': '2026-01-01T00:00:00Z',
  'updatedAt': '2026-01-01T00:00:00Z',
  'meta': {
    'req': group ? null : {'url': 'https://example.com/file'},
    'opts': {
      'path': directory,
      'name': group
          ? 'folder'
          : folder
          ? 'nested-torrent'
          : name,
      'size': 100,
    },
    if (!group)
      'res': {
        'name': folder ? 'nested-torrent' : '',
        'size': 100,
        'files': [
          {'name': name, 'path': folder ? 'subdir' : '', 'size': 100},
        ],
      },
  },
  'progress': {'used': 0, 'speed': 0, 'downloaded': 100, 'uploadSpeed': 0, 'uploaded': 0},
});
void main() {
  for (final windows in [false, true]) {
    for (final legacy in [false, true]) {
      for (final folder in [false, true]) {
        testWidgets('member URL windows=$windows legacy=$legacy folder=$folder', (tester) async {
          tester.view.physicalSize = const Size(900, 700);
          tester.view.devicePixelRatio = 1;
          addTearDown(tester.view.resetPhysicalSize);
          addTearDown(tester.view.resetDevicePixelRatio);
          api_client.setTransportForTesting(_Transport());
          String? launched;
          const channel = MethodChannel('plugins.flutter.io/url_launcher');
          tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, (call) async {
            if (call.method == 'launch') {
              launched = (call.arguments as Map)['url'] as String;
              return true;
            }
            return null;
          });
          addTearDown(() => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, null));
          final root = windows ? r'C:\Downloads\folder' : '/downloads/folder';
          final dir = windows ? r'C:\Downloads\folder\nested' : '/downloads/folder/nested';
          final group = taskRecordsFromApi([
            fixture('group', group: true, directory: legacy ? '' : root),
            fixture('child #1', group: false, directory: dir, folder: folder),
          ]).single;
          await tester.pumpWidget(
            shad.ShadcnApp(
              theme: AppTheme.light(),
              materialTheme: AppTheme.materialLight(),
              home: TaskFileManagerView(task: group, webActions: true, desktopActions: false),
            ),
          );
          await tester.pumpAndSettle();
          var current = '';
          for (final segment in group.files.single.path.split('/').where((s) => s.isNotEmpty)) {
            current += '/$segment';
            await tester.tap(find.byKey(ValueKey('managed-file-$current')));
            await tester.pumpAndSettle();
          }
          await tester.tap(find.byIcon(Icons.open_in_new));
          await tester.pumpAndSettle();
          final expected =
              'https://server.example/fs/tasks/child%20%231/${folder ? 'subdir/' : ''}episode%20%232%3F.txt';
          expect(launched, expected);
          expect(tester.takeException(), isNull);
        });
      }
    }
  }
}
