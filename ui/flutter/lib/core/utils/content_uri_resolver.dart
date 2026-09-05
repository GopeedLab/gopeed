import 'content_uri_resolver_stub.dart' if (dart.library.io) 'content_uri_resolver_io.dart' as implementation;

/// Resolves a platform content URI into a local temporary file.
abstract final class ContentUriResolver {
  static Future<String> copyToCache(Uri uri) => implementation.copyToCache(uri);
}
