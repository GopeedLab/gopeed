import 'dart:async';

import 'package:dio/dio.dart';

import '../capabilities/gopeed_capability.dart';

/// GitHub requests use backend-generated candidates, without forwarding the
/// Downloader's authentication headers to GitHub or third-party mirrors.
class GithubClient {
  GithubClient({required GopeedService service, Dio? client, this.timeout = const Duration(seconds: 30)})
    : _service = service,
      _client = client ?? Dio(BaseOptions(connectTimeout: const Duration(seconds: 10), receiveTimeout: timeout));

  final GopeedService _service;
  final Dio _client;
  final Duration timeout;

  /// The first complete response accepted by [decode] wins. HTTP 200 alone is
  /// insufficient: a mirror's HTML error page must not hide a usable response.
  Future<T> getJson<T>(String url, {required T Function(dynamic) decode}) async {
    final urls = (await _service.getGithubCandidates(url)).toSet();
    if (urls.isEmpty) throw StateError('No GitHub candidates');
    final result = Completer<T>();
    final tokens = <CancelToken>[];
    var pending = urls.length;
    Future<void> attempt(String candidate, CancelToken token) async {
      try {
        final response = await _client
            .get<dynamic>(candidate, cancelToken: token)
            .timeout(
              timeout,
              onTimeout: () {
                token.cancel('GitHub candidate timed out');
                throw TimeoutException('GitHub candidate timed out', timeout);
              },
            );
        final value = decode(response.data);
        if (!result.isCompleted) result.complete(value);
      } catch (error, stack) {
        if (--pending == 0 && !result.isCompleted) result.completeError(error, stack);
      }
    }

    for (final candidate in urls) {
      final token = CancelToken();
      tokens.add(token);
      unawaited(attempt(candidate, token));
    }
    try {
      return await result.future;
    } finally {
      for (final token in tokens) {
        token.cancel('GitHub candidate race completed');
      }
    }
  }
}
