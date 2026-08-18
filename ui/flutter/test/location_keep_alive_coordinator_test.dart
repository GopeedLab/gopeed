// This is a basic Flutter widget test.
//
// To perform an interaction with a widget in your test, use the WidgetTester
// utility that Flutter provides. For example, you can send tap and scroll
// gestures. You can also use WidgetTester to find child widgets in the widget
// tree, read text, and verify that the values of widget properties are correct.
import 'dart:convert';
import 'dart:io';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/app/services/location_keep_alive.dart';
import 'package:gopeed/app/services/location_keep_alive_coordinator.dart';

const _channel = MethodChannel('gopeed/location_keep_alive');

String _taskJson(String id, String status) => '''
{
  "id": "$id",
  "name": "test-file.zip",
  "status": "$status",
  "uploading": false,
  "progress": {
    "used": 0,
    "speed": 0,
    "downloaded": 0,
    "uploadSpeed": 0,
    "uploaded": 0
  },
  "createdAt": "2024-01-01T00:00:00.000Z",
  "updatedAt": "2024-01-01T00:00:00.000Z",
  "meta": {
    "req": { "url": "http://example.com/test-file.zip" },
    "opts": {}
  }
}
''';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  late HttpServer mockServer;
  List<String> runningTaskIds = [];
  final calls = <String>[];

  setUpAll(() async {
    mockServer = await HttpServer.bind('127.0.0.1', 0);
    mockServer.listen((req) async {
      final body = jsonEncode({
        'code': 0,
        'data': runningTaskIds.map((id) => jsonDecode(_taskJson(id, 'running'))).toList(),
      });
      req.response.headers.contentType = ContentType.json;
      req.response.write(body);
      await req.response.close();
    });
    api.init('tcp', '127.0.0.1:${mockServer.port}', '');
  });

  tearDownAll(() async {
    await mockServer.close(force: true);
  });

  setUp(() {
    runningTaskIds = [];
    calls.clear();
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(_channel, (call) async {
      calls.add(call.method);
      switch (call.method) {
        case 'start':
        case 'stop':
          return null;
        case 'requestPermission':
          return true;
        default:
          return null;
      }
    });
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(_channel, null);
  });

  group('LocationKeepAliveCoordinator', () {
    test('running task stays alive across repeated reconcile calls (tab switching)', () async {
      final coordinator = LocationKeepAliveCoordinator();
      coordinator.keepAliveEnabledProvider = () => true;
      runningTaskIds = ['task-1'];
      for (var i = 0; i < 5; i++) {
        await coordinator.reconcile();
      }

      expect(calls.contains('stop'), isFalse,
          reason: 'keep-alive must not stop while a task is running globally, '
              'regardless of how many times reconcile() runs');
      expect(calls.contains('start'), isTrue);
    });
    test('enabling the setting while a task is already running starts keep-alive immediately', () async {
      final coordinator = LocationKeepAliveCoordinator();
      runningTaskIds = ['task-1'];
      coordinator.keepAliveEnabledProvider = () => false;
      await coordinator.reconcile();
      expect(calls, contains('stop'));
      calls.clear();
      coordinator.keepAliveEnabledProvider = () => true;
      await coordinator.reconcile();
      expect(calls, contains('start'));
    });

    test('last task completing is caught by the polling fallback', () async {
      final coordinator = LocationKeepAliveCoordinator();
      coordinator.keepAliveEnabledProvider = () => true;
      coordinator.pollInterval = const Duration(milliseconds: 50);

      runningTaskIds = ['task-1'];
      await coordinator.reconcile();
      expect(calls, contains('start'));
      calls.clear();

      runningTaskIds = [];
      coordinator.startPolling();
      await Future.delayed(const Duration(milliseconds: 200));
      coordinator.stopPolling();

      expect(calls.contains('stop'), isTrue,
          reason: 'poll fallback must eventually reconcile after natural completion');
    });

    test('requestPermission forwards to native and resolves', () async {
      final granted = await LocationKeepAlive.requestPermission();
      expect(granted, isTrue);
      expect(calls, contains('requestPermission'));
    });
  });
}