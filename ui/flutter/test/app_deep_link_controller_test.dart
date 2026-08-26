import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/app/application/app_deep_link_controller.dart';

void main() {
  test('keeps the established path-style Gopeed deep-link routes', () {
    expect(gopeedDeepLinkRoute(Uri.parse('gopeed:///create')), '/create');
    expect(gopeedDeepLinkRoute(Uri.parse('gopeed:///extension')), '/extension');
  });

  test('rejects host-style Gopeed deep-link routes', () {
    expect(gopeedDeepLinkRoute(Uri.parse('gopeed://create')), isEmpty);
    expect(gopeedDeepLinkRoute(Uri.parse('gopeed://extension')), isEmpty);
  });
}
