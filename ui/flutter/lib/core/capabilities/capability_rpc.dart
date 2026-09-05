import 'dart:async';

typedef RpcDecoder<T> = T Function(Object? json);

class RpcUnit {
  const RpcUnit();
}

class RpcMethod<P, R> {
  const RpcMethod(this.name);

  final String name;
}

class CapabilityException implements Exception {
  const CapabilityException(this.code, this.message, [this.details]);

  final String code;
  final String message;
  final Object? details;

  @override
  String toString() => message;
}

class RpcCodecRegistry {
  final Map<Type, RpcDecoder<Object?>> _decoders = {};

  void register<T>(RpcDecoder<T> decoder) {
    _decoders[T] = (json) => decoder(json);
  }

  Object? encode(Object? value) {
    if (value == null || value is String || value is num || value is bool) {
      return value;
    }
    if (value is RpcUnit) {
      return const <String, Object?>{};
    }
    if (value is Enum) {
      return value.name;
    }
    if (value is Iterable) {
      return value.map(encode).toList(growable: false);
    }
    if (value is Map) {
      return {for (final entry in value.entries) entry.key.toString(): encode(entry.value)};
    }
    try {
      return encode((value as dynamic).toJson());
    } catch (_) {
      throw CapabilityException('encode_error', 'No RPC encoder registered for ${value.runtimeType}');
    }
  }

  T decode<T>(Object? value) {
    final normalized = _normalize(value);
    if (T == RpcUnit) {
      return const RpcUnit() as T;
    }
    if (normalized == null) {
      return null as T;
    }
    if (T == List<String>) {
      return (normalized as List<dynamic>).map((item) => item.toString()).toList(growable: false) as T;
    }
    if (normalized is T) {
      return normalized as T;
    }
    final decoder = _decoders[T];
    if (decoder == null) {
      throw CapabilityException('decode_error', 'No RPC decoder registered for $T');
    }
    return decoder(normalized) as T;
  }

  Object? _normalize(Object? value) {
    if (value is Map) {
      return {for (final entry in value.entries) entry.key.toString(): _normalize(entry.value)};
    }
    if (value is List) {
      return value.map(_normalize).toList(growable: false);
    }
    return value;
  }
}

abstract interface class CapabilityInvoker {
  Future<R> invoke<P, R>(RpcMethod<P, R> method, P params);
}

typedef _RawCapabilityHandler = Future<Object?> Function(Object? payload);

class CapabilityRegistry {
  CapabilityRegistry(this.codecs);

  final RpcCodecRegistry codecs;
  final Map<String, _RawCapabilityHandler> _handlers = {};
  final Map<String, Object> _localHandlers = {};

  void bind<P, R>(RpcMethod<P, R> method, FutureOr<R> Function(P params) handler) {
    if (_handlers.containsKey(method.name)) {
      throw StateError('Capability already registered: ${method.name}');
    }
    _localHandlers[method.name] = handler;
    _handlers[method.name] = (payload) async {
      final params = codecs.decode<P>(payload);
      return codecs.encode(await handler(params));
    };
  }

  Future<Object?> invoke(String operation, Object? payload) async {
    final handler = _handlers[operation];
    if (handler == null) {
      throw CapabilityException('method_not_found', 'Capability not found: $operation');
    }
    return handler(payload);
  }

  Future<R> invokeLocal<P, R>(RpcMethod<P, R> method, P params) async {
    final handler = _localHandlers[method.name];
    if (handler == null) {
      throw CapabilityException('method_not_found', 'Capability not found: ${method.name}');
    }
    return await (handler as FutureOr<R> Function(P))(params);
  }
}

class LocalCapabilityInvoker implements CapabilityInvoker {
  const LocalCapabilityInvoker(this.registry);

  final CapabilityRegistry registry;

  @override
  Future<R> invoke<P, R>(RpcMethod<P, R> method, P params) async {
    return registry.invokeLocal(method, params);
  }
}
