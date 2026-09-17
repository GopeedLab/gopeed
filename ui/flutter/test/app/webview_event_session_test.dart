import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';
import 'package:gopeed/app/rpc/webview_rpc_service.dart';

class _Profile extends Fake implements WebViewProfile {}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  test('closed emits reason once and closes listener stream', () async {
    final page = WebViewRpcPageSession(
      pageId: 'test',
      profile: _Profile(),
      headless: true,
      debug: false,
      title: '',
      width: 1,
      height: 1,
      userAgent: '',
    );
    // Consume the expected ready failure when an unopened page is disposed.
    final pending = page.execute('1', []);
    final failure = expectLater(pending, throwsA(isA<WebViewRpcException>()));
    final events = page.events.toList();
    await page.dispose(reason: 'user');
    await page.dispose(reason: 'api');
    await failure;
    expect(await events, [
      {
        'event': 'closed',
        'data': {'reason': 'user'},
      },
    ]);
  });
}
