import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/create_task.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/core/window/app_window_launcher.dart';
import 'package:gopeed/core/window/window_capability_transport.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const multiWindowChannel = MethodChannel('mixin.one/desktop_multi_window');
  const windowChannels = MethodChannel('mixin.one/desktop_multi_window/channels');
  final messenger = TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  setUp(() {
    messenger.setMockMethodCallHandler(windowChannels, (call) async => null);
  });

  tearDown(() async {
    await AppWindowCapabilityHost.instance.stop();
    messenger.setMockMethodCallHandler(multiWindowChannel, null);
    messenger.setMockMethodCallHandler(windowChannels, null);
  });

  test('create-task window stays hidden until child bootstrap is ready', () async {
    final calls = <MethodCall>[];
    messenger.setMockMethodCallHandler(multiWindowChannel, (call) async {
      calls.add(call);
      if (call.method == 'createWindow') return 'create-window-id';
      return null;
    });
    messenger.setMockMethodCallHandler(windowChannels, (call) async {
      calls.add(call);
      return null;
    });

    final opened = await AppWindowLauncher.openCreateTaskWindow(
      createTask: CreateTask(req: Request(url: 'https://example.com/file.zip')),
    );

    expect(opened, isTrue);
    final createCall = calls.firstWhere((call) => call.method == 'createWindow');
    final configuration = Map<String, dynamic>.from(createCall.arguments as Map);
    expect(configuration['hiddenAtLaunch'], isTrue);
    expect(
      calls.where((call) => call.method == 'window_show' && (call.arguments as Map)['windowId'] == 'create-window-id'),
      isEmpty,
    );
  });

  test('create-task window failure reports fallback availability', () async {
    messenger.setMockMethodCallHandler(multiWindowChannel, (call) async {
      if (call.method == 'createWindow') {
        throw PlatformException(code: 'create_failed');
      }
      return null;
    });

    expect(await AppWindowLauncher.openCreateTaskWindow(), isFalse);
  });
}
