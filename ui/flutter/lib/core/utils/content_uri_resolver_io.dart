import 'dart:io';

import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';
import 'package:uri_content/uri_content.dart';

Future<String> copyToCache(Uri uri) async {
  if (uri.scheme != 'content') {
    throw ArgumentError.value(uri, 'uri', 'Expected an Android content URI');
  }
  if (!Platform.isAndroid) {
    throw UnsupportedError('content URIs are only supported on Android');
  }

  final importDirectory = Directory(path.join((await getTemporaryDirectory()).path, 'torrent-imports'));
  await importDirectory.create(recursive: true);
  final output = File(path.join(importDirectory.path, 'gopeed-${DateTime.now().microsecondsSinceEpoch}.torrent'));
  final sink = output.openWrite();

  try {
    await sink.addStream(UriContent().getContentStream(uri));
    await sink.close();
  } catch (error) {
    await sink.close();
    if (await output.exists()) await output.delete();
    rethrow;
  }

  return output.path;
}
