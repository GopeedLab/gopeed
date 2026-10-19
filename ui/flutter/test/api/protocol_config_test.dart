import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/api.dart' as api;
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/core/network/gopeed/gopeed_transport.dart';

void main() {
  test('protocol jobs use the common protocol and job name API', () async {
    final transport = _Transport();
    api.setTransportForTesting(transport);
    await LocalAppCapabilities.instance.capabilities.gopeed.triggerProtocolJob('bt', 'update-trackers');
    expect(transport.path, 'api/v1/protocols/bt/jobs/update-trackers/trigger');
    expect(transport.method, 'POST');
  });

  test('mirror settings and tracker settings serialize at their backend-owned locations', () {
    final config = DownloaderConfig();
    config.githubMirror = GithubMirrorConfig(enabled: false, mirrors: []);
    config.protocolConfig.bt.customTrackers = ['udp://example.com:80/announce'];
    config.protocolConfig.bt.state.trackerSubscriptionStatuses = {
      'https://example.com/list': TrackerSubscriptionStatus(
        success: true,
        trackerCount: 12,
        lastFetchTime: DateTime.utc(2026, 10, 19),
      ),
    };
    final json = config.toJson();
    expect(json['githubMirror'], {'enabled': false, 'mirrors': []});
    expect(json['extra'], isNot(contains('githubMirror')));
    expect(json['extra'], isNot(contains('bt')));
    expect((json['protocolConfig'] as Map)['bt']['customTrackers'], ['udp://example.com:80/announce']);
    final restored = DownloaderConfig.fromJson(
      json,
    ).protocolConfig.bt.state.trackerSubscriptionStatuses['https://example.com/list']!;
    expect(restored.success, isTrue);
    expect(restored.trackerCount, 12);
    expect(restored.lastFetchTime, DateTime.utc(2026, 10, 19));
    expect(BtState.fromJson({'lastTrackerUpdateTime': '0001-01-01T00:00:00Z'}).lastTrackerUpdateTime, isNull);
  });

  test('protocol and update operations survive the shared capability codec', () {
    final codecs = createAppCapabilityCodecs();
    final payload = {'protocol': 'bt', 'name': 'update-trackers'};
    expect(codecs.decode<Map<String, dynamic>>(codecs.encode(payload)), payload);
    expect(codecs.decode<Map<String, dynamic>?>(null), isNull);
    expect(GopeedMethods.triggerProtocolJob.name, 'gopeed.protocol.job.trigger');
    expect(const RpcUnit(), isA<RpcUnit>());
  });

  test('application update checks use a dedicated API and preserve the client version', () async {
    final transport = _Transport();
    api.setTransportForTesting(transport);
    await LocalAppCapabilities.instance.capabilities.gopeed.checkAppUpdate('2.0.0-beta.2');
    expect(transport.path, 'api/v1/app/updates/check');
    expect(transport.data, {'currentVersion': '2.0.0-beta.2'});
  });

  test('update candidates use business parameters and survive capability serialization', () async {
    final transport = _Transport();
    api.setTransportForTesting(transport);
    final urls = await LocalAppCapabilities.instance.capabilities.gopeed.getAppUpdateCandidates(
      '2.0.0',
      'Gopeed-v2.0.0-windows-amd64.zip',
    );
    expect(transport.path, 'api/v1/app/updates/candidates');
    expect(transport.method, 'POST');
    expect(transport.data, {'version': '2.0.0', 'asset': 'Gopeed-v2.0.0-windows-amd64.zip'});
    expect(urls, ['https://example.com/package.zip']);
    final codecs = createAppCapabilityCodecs();
    expect(codecs.decode<List<String>>(codecs.encode(urls)), urls);
  });
}

class _Transport implements GopeedTransport {
  String? path;
  String? method;
  dynamic data;
  @override
  Future<dynamic> request(
    String path, {
    String method = 'GET',
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async {
    this.path = path;
    this.method = method;
    this.data = data;
    return {
      'code': 0,
      'data': path.endsWith('/trigger')
          ? null
          : path.endsWith('/candidates')
          ? {
              'urls': ['https://example.com/package.zip'],
            }
          : <String, dynamic>{},
    };
  }

  @override
  String join(String path) => path;
}
