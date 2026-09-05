import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/window/app_window_appearance.dart';

class _Payload {
  const _Payload(this.value);

  factory _Payload.fromJson(Map<String, dynamic> json) => _Payload(json['value'] as int);

  final int value;

  Map<String, dynamic> toJson() => {'value': value};
}

void main() {
  test('one method descriptor supports direct and serialized dispatch', () async {
    const method = RpcMethod<_Payload, _Payload>('test.increment');
    final codecs = RpcCodecRegistry()..register<_Payload>((json) => _Payload.fromJson(json! as Map<String, dynamic>));
    final registry = CapabilityRegistry(codecs)..bind(method, (payload) => _Payload(payload.value + 1));

    final local = await LocalCapabilityInvoker(registry).invoke(method, const _Payload(1));
    final serialized = codecs.decode<_Payload>(await registry.invoke(method.name, codecs.encode(const _Payload(4))));

    expect(local.value, 2);
    expect(serialized.value, 5);
  });

  test('appearance snapshots use the shared codec registry', () {
    final codecs = RpcCodecRegistry()
      ..register<AppWindowAppearance>((json) => AppWindowAppearance.fromJson(json! as Map<String, dynamic>));
    const appearance = AppWindowAppearance(themeMode: 'dark', themeColor: 'blue', locale: 'zh');

    expect(codecs.decode<AppWindowAppearance>(codecs.encode(appearance)), appearance);
  });

  test('unknown capabilities return a structured error', () async {
    final registry = CapabilityRegistry(RpcCodecRegistry());

    expect(
      () => registry.invoke('missing', null),
      throwsA(isA<CapabilityException>().having((error) => error.code, 'code', 'method_not_found')),
    );
  });
}
