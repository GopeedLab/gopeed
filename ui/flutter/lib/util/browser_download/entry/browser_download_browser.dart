import 'package:web/web.dart' as web;

void doDownload(String url, String name) {
  final anchor = web.document.createElement('a') as web.HTMLAnchorElement
    ..href = url
    ..download = name
    ..target = '_blank';
  anchor.click();
}
