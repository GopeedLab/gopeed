import 'dart:async';

import 'package:desktop_multi_window/desktop_multi_window.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

import '../capabilities/app_capabilities.dart';
import '../capabilities/capability_rpc.dart';
import 'app_window_appearance.dart';

abstract final class AppWindowRpcProtocol {
  static const channelName = 'gopeed.app.capabilities.v1';
  static const call = 'capability.call';
  static const bootstrap = 'window.bootstrap';
  static const subscribe = 'window.subscribe';
  static const unsubscribe = 'window.unsubscribe';
  static const event = 'window.event';
  static const appearanceChanged = 'appearance.changed';
}

const _hostChannel = WindowMethodChannel(AppWindowRpcProtocol.channelName, mode: ChannelMode.unidirectional);

class WindowCapabilityInvoker implements CapabilityInvoker {
  const WindowCapabilityInvoker(this.codecs);

  final RpcCodecRegistry codecs;

  @override
  Future<R> invoke<P, R>(RpcMethod<P, R> method, P params) async {
    final raw = await _hostChannel.invokeMethod<dynamic>(AppWindowRpcProtocol.call, {
      'operation': method.name,
      'payload': codecs.encode(params),
    });
    final response = codecs.decode<Map<String, dynamic>>(raw);
    if (response['ok'] != true) {
      final error = codecs.decode<Map<String, dynamic>>(response['error']);
      throw CapabilityException(
        (error['code'] ?? 'remote_error').toString(),
        (error['message'] ?? 'Remote capability failed').toString(),
        error['details'],
      );
    }
    return codecs.decode<R>(response['data']);
  }
}

class AppWindowCapabilityHost {
  AppWindowCapabilityHost._();

  static final instance = AppWindowCapabilityHost._();

  CapabilityRegistry? _registry;
  AppWindowAppearance _appearance = const AppWindowAppearance.defaults();
  final Map<String, WindowController> _subscribers = {};

  bool get started => _registry != null;

  Future<void> start(CapabilityRegistry registry) async {
    if (_registry != null) return;
    _registry = registry;
    await _hostChannel.setMethodCallHandler(_handleCall);
  }

  Future<void> stop() async {
    _registry = null;
    _subscribers.clear();
    await _hostChannel.setMethodCallHandler(null);
  }

  void updateAppearance(AppWindowAppearance appearance) {
    if (_appearance == appearance) return;
    _appearance = appearance;
    unawaited(_broadcastAppearance());
  }

  Future<dynamic> _handleCall(MethodCall call) async {
    try {
      switch (call.method) {
        case AppWindowRpcProtocol.call:
          final args = _map(call.arguments);
          final registry = _registry;
          if (registry == null) {
            throw const CapabilityException('host_unavailable', 'Main window capabilities are not ready');
          }
          final data = await registry.invoke((args['operation'] ?? '').toString(), args['payload']);
          return {'ok': true, 'data': data};
        case AppWindowRpcProtocol.bootstrap:
          return _appearance.toJson();
        case AppWindowRpcProtocol.subscribe:
          final windowId = (_map(call.arguments)['windowId'] ?? '').toString();
          if (windowId.isEmpty) {
            throw const CapabilityException('invalid_window', 'Missing child window id');
          }
          final controller = WindowController.fromWindowId(windowId);
          _subscribers[windowId] = controller;
          scheduleMicrotask(() => _sendAppearance(windowId, controller));
          return null;
        case AppWindowRpcProtocol.unsubscribe:
          final windowId = (_map(call.arguments)['windowId'] ?? '').toString();
          _subscribers.remove(windowId);
          return null;
        default:
          throw CapabilityException('method_not_found', 'Window RPC method not found: ${call.method}');
      }
    } on CapabilityException catch (error) {
      if (call.method == AppWindowRpcProtocol.call) {
        return {
          'ok': false,
          'error': {'code': error.code, 'message': error.message, 'details': error.details},
        };
      }
      rethrow;
    } catch (error) {
      if (call.method == AppWindowRpcProtocol.call) {
        return {
          'ok': false,
          'error': {'code': 'internal_error', 'message': error.toString()},
        };
      }
      rethrow;
    }
  }

  Future<void> _broadcastAppearance() async {
    final subscribers = Map<String, WindowController>.of(_subscribers);
    await Future.wait([for (final entry in subscribers.entries) _sendAppearance(entry.key, entry.value)]);
  }

  Future<void> _sendAppearance(String windowId, WindowController controller) async {
    try {
      await controller.invokeMethod<void>(AppWindowRpcProtocol.event, {
        'name': AppWindowRpcProtocol.appearanceChanged,
        'data': _appearance.toJson(),
      });
    } catch (_) {
      _subscribers.remove(windowId);
    }
  }

  Map<String, dynamic> _map(Object? value) {
    if (value is! Map) return const {};
    return {for (final entry in value.entries) entry.key.toString(): entry.value};
  }
}

class ChildWindowSession {
  ChildWindowSession._(this.controller, this.codecs)
    : capabilities = AppCapabilities(WindowCapabilityInvoker(codecs)),
      appearance = ValueNotifier(const AppWindowAppearance.defaults());

  static Future<ChildWindowSession> connect(WindowController controller) async {
    final session = ChildWindowSession._(controller, createAppCapabilityCodecs());
    await session._initialize();
    return session;
  }

  final WindowController controller;
  final RpcCodecRegistry codecs;
  final AppCapabilities capabilities;
  final ValueNotifier<AppWindowAppearance> appearance;

  Future<void> _initialize() async {
    await controller.setWindowMethodHandler(_handleWindowCall);
    final initial = await _hostChannel.invokeMethod<dynamic>(AppWindowRpcProtocol.bootstrap);
    appearance.value = codecs.decode<AppWindowAppearance>(initial);
    await _hostChannel.invokeMethod<void>(AppWindowRpcProtocol.subscribe, {'windowId': controller.windowId});
  }

  Future<dynamic> _handleWindowCall(MethodCall call) async {
    if (call.method != AppWindowRpcProtocol.event) return null;
    final args = codecs.decode<Map<String, dynamic>>(call.arguments);
    if (args['name'] == AppWindowRpcProtocol.appearanceChanged) {
      appearance.value = codecs.decode<AppWindowAppearance>(args['data']);
    }
    return null;
  }

  void dispose() {
    unawaited(controller.setWindowMethodHandler(null));
    unawaited(_hostChannel.invokeMethod<void>(AppWindowRpcProtocol.unsubscribe, {'windowId': controller.windowId}));
    appearance.dispose();
  }
}
