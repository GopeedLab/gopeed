import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/window/app_window_appearance.dart';
import 'package:gopeed/core/window/window_capability_transport.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const windows = MethodChannel('mixin.one/desktop_multi_window');
  const channels = MethodChannel('mixin.one/desktop_multi_window/channels');
  const codec = StandardMethodCodec();
  final messenger = TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  tearDown(() async {
    await AppWindowCapabilityHost.instance.stop();
    messenger.setMockMethodCallHandler(windows, null);
    messenger.setMockMethodCallHandler(channels, null);
  });

  testWidgets('closed engines lose appearance subscriptions without a Dart unsubscribe', (tester) async {
    final deliveries = <String>[];
    messenger.setMockMethodCallHandler(channels, (call) async {
      if (call.method == 'invokeMethod') {
        final args = Map<String, dynamic>.from(call.arguments as Map);
        if (args['method'] == AppWindowRpcProtocol.event) deliveries.add(args['channel'] as String);
      }
      return null;
    });
    messenger.setMockMethodCallHandler(windows, (call) async {
      if (call.method == 'getAllWindows') {
        return [
          {'windowId': 'live-child', 'windowArgument': ''},
        ];
      }
      return null;
    });
    final host = AppWindowCapabilityHost.instance;
    await host.start(CapabilityRegistry(createAppCapabilityCodecs()));
    for (final id in ['closed-child', 'live-child']) {
      await messenger.handlePlatformMessage(
        channels.name,
        codec.encodeMethodCall(
          MethodCall('methodCall', {
            'channel': AppWindowRpcProtocol.channelName,
            'method': AppWindowRpcProtocol.subscribe,
            'arguments': {'windowId': id},
          }),
        ),
        (_) {},
      );
    }
    await tester.pump();
    deliveries.clear();
    await messenger.handlePlatformMessage(
      windows.name,
      codec.encodeMethodCall(const MethodCall('onWindowsChanged')),
      (_) {},
    );
    await tester.pump();
    host.updateAppearance(const AppWindowAppearance(themeMode: 'dark', themeColor: 'green', locale: 'zh'));
    await tester.pump();
    expect(deliveries, ['mixin.one/window_controller/live-child']);
  });
}
