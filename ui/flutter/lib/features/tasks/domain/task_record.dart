import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:path/path.dart' as path;

import '../../../api/model/resource.dart' as api_resource;
import '../../../api/model/task.dart' as api_task;
import '../../../core/utils/byte_size_formatter.dart';
import '../../../core/utils/transfer_rate_formatter.dart';

enum TaskStatus { downloading, completed, paused, failed }

enum TaskAssetType { file, archive, warning, document, repository }

class TaskFileNode {
  const TaskFileNode({required this.path, required this.name, required this.sizeBytes, this.downloadedBytes});

  final String path;
  final String name;
  final int sizeBytes;
  final int? downloadedBytes;

  double? get progress {
    final downloaded = downloadedBytes;
    if (downloaded == null || sizeBytes <= 0) return null;
    return (downloaded / sizeBytes).clamp(0.0, 1.0);
  }
}

class TaskRecord {
  const TaskRecord({
    required this.id,
    required this.name,
    required this.status,
    required this.downloaded,
    required this.assetType,
    required this.url,
    required this.storagePath,
    required this.files,
    required this.uploading,
    this.isFolder = false,
    this.protocol,
    this.requestHeaders = const {},
    this.total,
    this.speed,
    this.uploadSpeed,
    this.remaining,
    this.progress,
    this.error,
    this.completedLabel,
    this.downloadedBytes,
    this.totalBytes,
    this.speedBytes,
    this.uploadedBytes,
    this.remainingSeconds,
    this.waiting = false,
  }) : assert(!uploading || uploadSpeed != null);

  final String id;
  final String name;
  final TaskStatus status;
  final String downloaded;
  final String? total;
  final String? speed;
  final String? uploadSpeed;
  final String? remaining;
  final double? progress;
  final TaskAssetType assetType;
  final String url;
  final String storagePath;
  final String? error;
  final String? completedLabel;
  final int? downloadedBytes;
  final int? totalBytes;
  final int? speedBytes;
  final int? uploadedBytes;
  final int? remainingSeconds;
  final bool waiting;
  final List<TaskFileNode> files;
  final bool uploading;
  final bool isFolder;
  final api_task.Protocol? protocol;
  final Map<String, String> requestHeaders;

  bool get canUpdateUrl =>
      protocol == api_task.Protocol.http && (status == TaskStatus.paused || status == TaskStatus.failed);

  bool get isIndeterminate => status == TaskStatus.downloading && total == null;

  IconData get icon {
    switch (assetType) {
      case TaskAssetType.file:
        return Icons.insert_drive_file_outlined;
      case TaskAssetType.archive:
        return Icons.archive_outlined;
      case TaskAssetType.warning:
        return Icons.warning_amber_rounded;
      case TaskAssetType.document:
        return Icons.description_outlined;
      case TaskAssetType.repository:
        return Icons.data_object_outlined;
    }
  }

  factory TaskRecord.fromApi(api_task.Task task) {
    final totalBytes = task.meta.res?.size ?? 0;
    final downloadedBytes = task.progress.downloaded;
    final progress = totalBytes > 0 ? (downloadedBytes / totalBytes).clamp(0.0, 1.0) : null;
    final status = _statusFromApi(task.status);
    final name = task.name.isNotEmpty ? task.name : task.meta.res?.name;
    final fileName = (name == null || name.isEmpty) ? task.meta.req.url : name;
    final rawUrl = task.meta.req.rawUrl;
    final displayUrl = rawUrl != null && rawUrl.isNotEmpty ? rawUrl : task.meta.req.url;
    final storagePath = task.meta.opts.path.isEmpty ? fileName : path.join(task.meta.opts.path, fileName);

    return TaskRecord(
      id: task.id,
      name: fileName,
      status: status,
      downloaded: _formatBytes(downloadedBytes),
      total: totalBytes > 0 ? _formatBytes(totalBytes) : null,
      speed: task.progress.speed > 0 ? TransferRateFormatter.format(task.progress.speed).text : null,
      uploadSpeed: task.uploading ? TransferRateFormatter.format(task.progress.uploadSpeed).text : null,
      remainingSeconds: _remainingSeconds(task, totalBytes, downloadedBytes),
      waiting: task.status == api_task.Status.wait,
      progress: progress,
      assetType: _assetType(fileName, status),
      url: displayUrl,
      storagePath: storagePath,
      downloadedBytes: downloadedBytes,
      totalBytes: totalBytes > 0 ? totalBytes : null,
      speedBytes: task.progress.speed,
      uploadedBytes: task.progress.uploaded,
      files: _fileNodes(task.meta.res?.files),
      uploading: task.uploading,
      isFolder: task.meta.res?.name.isNotEmpty ?? false,
      protocol: task.protocol,
      requestHeaders: _requestHeaders(task.meta.req.extra),
    );
  }
}

Map<String, String> _requestHeaders(Object? extra) {
  if (extra is! Map) return const {};
  final header = extra['header'];
  if (header is! Map) return const {};
  return {for (final entry in header.entries) entry.key.toString(): entry.value.toString()};
}

TaskStatus _statusFromApi(api_task.Status status) {
  return switch (status) {
    api_task.Status.running || api_task.Status.wait => TaskStatus.downloading,
    api_task.Status.done => TaskStatus.completed,
    api_task.Status.pause => TaskStatus.paused,
    api_task.Status.error => TaskStatus.failed,
    api_task.Status.ready => TaskStatus.paused,
  };
}

TaskAssetType _assetType(String name, TaskStatus status) {
  if (status == TaskStatus.failed) {
    return TaskAssetType.warning;
  }
  final lower = name.toLowerCase();
  if (lower.endsWith('.zip') ||
      lower.endsWith('.rar') ||
      lower.endsWith('.7z') ||
      lower.endsWith('.tar') ||
      lower.endsWith('.gz') ||
      lower.endsWith('.iso')) {
    return TaskAssetType.archive;
  }
  if (lower.endsWith('.pdf') || lower.endsWith('.doc') || lower.endsWith('.docx')) {
    return TaskAssetType.document;
  }
  if (lower.endsWith('.git')) {
    return TaskAssetType.repository;
  }
  return TaskAssetType.file;
}

int? _remainingSeconds(api_task.Task task, int totalBytes, int downloadedBytes) {
  if (task.status != api_task.Status.running) return null;
  final speed = task.progress.speed;
  if (speed <= 0 || totalBytes <= downloadedBytes) {
    return null;
  }
  return ((totalBytes - downloadedBytes) / speed).ceil();
}

List<TaskFileNode> _fileNodes(List<api_resource.FileInfo>? files) {
  if (files == null || files.isEmpty) {
    return const [];
  }
  return files
      .map((file) => TaskFileNode(path: file.path, name: file.name, sizeBytes: file.size))
      .toList(growable: false);
}

String _formatBytes(int bytes) {
  return ByteSizeFormatter.format(bytes);
}
