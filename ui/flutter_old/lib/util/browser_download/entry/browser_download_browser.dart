// ignore: avoid_web_libraries_in_flutter
import 'dart:html' as html;

void doDownload(String url, String name) {
  final anchorElement = html.AnchorElement(href: url);
  anchorElement.download = name;
  anchorElement.target = '_blank';
  anchorElement.click();
}
