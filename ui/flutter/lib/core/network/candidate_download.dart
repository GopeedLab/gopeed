import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';

class DownloadChecksumException implements Exception {
  const DownloadChecksumException();
  @override
  String toString() => 'Update package checksum verification failed';
}

/// Probes backend-provided URLs concurrently, then downloads the fastest usable
/// candidate. Other candidates remain available if HEAD is unsupported or the
/// actual download fails. This layer has no GitHub URL or mirror-config rules.
Future<void> downloadFromCandidates(
  List<String> candidates,
  String savePath, {
  required ProgressCallback onReceiveProgress,
  required String expectedSha256,
  Dio? client,
  Duration probeTimeout = const Duration(seconds: 3),
}) async {
  // Reject missing or malformed trusted metadata before any network request.
  if (expectedSha256.length != 64 || !RegExp(r'^[a-fA-F0-9]{64}$').hasMatch(expectedSha256)) {
    throw const DownloadChecksumException();
  }
  final urls = candidates.toSet().toList(growable: false);
  if (urls.isEmpty) throw ArgumentError.value(candidates, 'candidates', 'No download candidates');
  final dio = client ?? Dio(BaseOptions(connectTimeout: const Duration(seconds: 10)));
  try {
    final probes = await Future.wait(
      urls.map((url) async {
        final watch = Stopwatch()..start();
        final cancel = CancelToken();
        try {
          final response = await dio
              .head<void>(
                url,
                cancelToken: cancel,
                options: Options(receiveTimeout: probeTimeout, validateStatus: (status) => status == 200),
              )
              .timeout(probeTimeout);
          if (_isPackageResponse(response.headers)) return (url: url, latency: watch.elapsed);
        } catch (_) {
          // HEAD failure does not prove that a normal GET will fail.
        } finally {
          cancel.cancel();
        }
        return (url: url, latency: null);
      }),
    );
    final available = probes.where((probe) => probe.latency != null).toList()
      ..sort((a, b) => a.latency!.compareTo(b.latency!));
    final ordered = [
      ...available.map((probe) => probe.url),
      ...probes.where((p) => p.latency == null).map((p) => p.url),
    ];
    Object? lastError;
    StackTrace? lastStack;
    for (final url in ordered) {
      try {
        onReceiveProgress(0, 0);
        final response = await dio.download(
          url,
          savePath,
          onReceiveProgress: onReceiveProgress,
          deleteOnError: true,
          options: Options(receiveTimeout: const Duration(seconds: 30), validateStatus: (status) => status == 200),
        );
        if (!_isPackageResponse(response.headers) || await File(savePath).length() == 0) {
          throw StateError('Invalid download response from $url');
        }
        final actual = await sha256.bind(File(savePath).openRead()).first;
        if (actual.toString() != expectedSha256.toLowerCase()) throw const DownloadChecksumException();
        return;
      } catch (error, stack) {
        lastError = error;
        lastStack = stack;
        final file = File(savePath);
        if (await file.exists()) await file.delete();
      }
    }
    Error.throwWithStackTrace(lastError!, lastStack!);
  } finally {
    if (client == null) dio.close(force: true);
  }
}

bool _isPackageResponse(Headers headers) {
  final contentType = headers.value(Headers.contentTypeHeader)?.split(';').first.trim().toLowerCase();
  return contentType != 'text/html' && contentType != 'application/xhtml+xml' && contentType != 'application/json';
}
