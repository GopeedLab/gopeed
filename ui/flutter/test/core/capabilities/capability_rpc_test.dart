import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/options.dart';
import 'package:gopeed/api/model/patch_task.dart';
import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resolve_result.dart';
import 'package:gopeed/api/model/resolve_task.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/core/window/app_window_appearance.dart';

class _Payload {
  const _Payload(this.value);

  factory _Payload.fromJson(Map<String, dynamic> json) => _Payload(json['value'] as int);

  final int value;

  Map<String, dynamic> toJson() => {'value': value};
}

void main() {
  test('serialized resolve and patch capabilities keep separate request contracts', () async {
    final codecs = createAppCapabilityCodecs();
    String? resolvedUrl;
    Map<String, dynamic>? patch;
    final registry = CapabilityRegistry(codecs)
      ..bind(GopeedMethods.resolve, (request) {
        resolvedUrl = request.req?.url;
        return ResolveResult(id: 'resolved-id', res: Resource(files: []));
      })
      ..bind(GopeedMethods.patchTask, (request) {
        patch = request;
        return const RpcUnit();
      });
    final request = ResolveTask(req: Request(url: 'magnet:?xt=urn:btih:hash'));
    final wireRequest = codecs.encode(request) as Map<String, dynamic>;
    expect(wireRequest.keys, ['req']);
    final result = codecs.decode<ResolveResult>(await registry.invoke(GopeedMethods.resolve.name, wireRequest));
    expect(result.id, 'resolved-id');
    expect(resolvedUrl, request.req?.url);

    final service = GopeedService(LocalCapabilityInvoker(registry));
    await service.patchTask('task-id', PatchTask(opts: Options(selectFiles: [1])));
    expect(patch?['id'], 'task-id');
    final patchBody = patch?['request'] as Map<String, dynamic>;
    expect((patchBody['opts'] as Map<String, dynamic>)['selectFiles'], [1]);
  });

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
