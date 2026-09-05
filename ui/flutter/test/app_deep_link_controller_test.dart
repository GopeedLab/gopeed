import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/app/application/app_deep_link_controller.dart';
import 'package:share_handler/share_handler.dart';

void main() {
  test('keeps the established path-style Gopeed deep-link routes', () {
    expect(gopeedDeepLinkRoute(Uri.parse('gopeed:///create')), '/create');
    expect(gopeedDeepLinkRoute(Uri.parse('gopeed:///extension')), '/extension');
  });

  test('rejects host-style Gopeed deep-link routes', () {
    expect(gopeedDeepLinkRoute(Uri.parse('gopeed://create')), isEmpty);
    expect(gopeedDeepLinkRoute(Uri.parse('gopeed://extension')), isEmpty);
  });

  test('uses a shared attachment path before an optional caption', () {
    final media = SharedMedia(
      content: 'https://example.com/caption',
      attachments: [SharedAttachment(path: '/tmp/example.torrent', type: SharedAttachmentType.file)],
    );

    expect(sharedMediaUri(media), Uri.file('/tmp/example.torrent'));
  });

  test('uses shared text when there is no attachment', () {
    final media = SharedMedia(content: '  magnet:?xt=urn:btih:example  ');

    expect(sharedMediaUri(media), Uri.parse('magnet:?xt=urn:btih:example'));
  });
}
