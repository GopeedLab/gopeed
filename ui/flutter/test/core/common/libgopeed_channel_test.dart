import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/common/libgopeed_channel.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('gopeed.com/libgopeed');
  final messenger = TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  tearDown(() {
    messenger.setMockMethodCallHandler(channel, null);
  });

  test('mobile invokes carry unique request IDs for async callbacks', () async {
    final requestIDs = <int>[];
    messenger.setMockMethodCallHandler(channel, (call) async {
      expect(call.method, 'invoke');
      final arguments = (call.arguments as Map<Object?, Object?>).cast<String, Object?>();
      requestIDs.add(arguments['requestID']! as int);
      return '{"code":0}';
    });

    final bridge = LibgopeedChannel();
    await Future.wait([bridge.invoke('GET', '/api/v1/info'), bridge.invoke('GET', '/api/v1/config')]);

    expect(requestIDs, [0, 1]);
  });
}
