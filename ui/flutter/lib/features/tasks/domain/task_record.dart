import 'package:flutter/widgets.dart';
import 'package:path/path.dart' as path;

import '../../../api/model/resource.dart' as api_resource;
import '../../../api/model/task.dart' as api_task;
import '../../../core/icons/gopeed_icons.dart';
import '../../../core/utils/byte_size_formatter.dart';
import '../../../core/utils/transfer_rate_formatter.dart';

enum TaskStatus { downloading, completed, paused, failed }

enum TaskAssetType {
  file,
  folder,
  torrentFolder,
  torrent,
  ed2k,
  installer,
  androidPackage,
  iosPackage,
  diskImage,
  web,
  text,
  pdf,
  document,
  spreadsheet,
  presentation,
  archive,
  image,
  audio,
  video,
  code,
  ebook,
  font,
  database,
}

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
    this.downloadDuration,
    this.createdAt,
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
  final Duration? downloadDuration;
  final DateTime? createdAt;
  final double? progress;
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

  TaskAssetType get assetType => _assetType(name, isFolder: isFolder, protocol: protocol);

  IconData get icon => taskFileTypeIcon(name, isFolder: isFolder, protocol: protocol);

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
      downloadDuration: task.progress.used > 0 ? Duration(microseconds: (task.progress.used + 999) ~/ 1000) : null,
      createdAt: task.createdAt,
      waiting: task.status == api_task.Status.wait,
      progress: progress,
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

const _installerExtensions = {'exe', 'msi', 'dmg', 'deb', 'rpm', 'pkg', 'appimage', 'flatpak', 'snap'};
const _androidPackageExtensions = {'apk', 'aab'};
const _iosPackageExtensions = {'ipa'};
const _diskImageExtensions = {'iso', 'img'};
const _webExtensions = {'html', 'htm', 'xhtml'};
const _textExtensions = {'txt', 'md', 'log', 'csv', 'tsv', 'json', 'yaml', 'yml', 'toml', 'ini', 'conf', 'cfg'};
const _pdfExtensions = {'pdf'};
const _documentExtensions = {'doc', 'docx', 'odt', 'rtf', 'pages'};
const _spreadsheetExtensions = {'xls', 'xlsx', 'ods', 'numbers'};
const _presentationExtensions = {'ppt', 'pptx', 'odp', 'key'};
const _archiveExtensions = {'zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz', 'zst', 'tgz', 'tbz2', 'txz', 'cab'};
const _imageExtensions = {
  'jpg',
  'jpeg',
  'png',
  'gif',
  'bmp',
  'tif',
  'tiff',
  'svg',
  'webp',
  'heic',
  'avif',
  'ico',
  'raw',
};
const _audioExtensions = {'mp3', 'wav', 'flac', 'aac', 'ogg', 'oga', 'wma', 'm4a', 'opus', 'aiff'};
const _videoExtensions = {'mp4', 'avi', 'mkv', 'mov', 'wmv', 'flv', 'webm', 'm4v', 'mpg', 'mpeg', '3gp', 'ts'};
const _codeExtensions = {
  'js',
  'jsx',
  'ts',
  'tsx',
  'css',
  'scss',
  'less',
  'xml',
  'java',
  'kt',
  'kts',
  'cpp',
  'cc',
  'c',
  'h',
  'hpp',
  'dart',
  'py',
  'rb',
  'php',
  'swift',
  'go',
  'rs',
  'sh',
  'bash',
  'zsh',
  'ps1',
  'sql',
  'vue',
  'svelte',
  'git',
};
const _ebookExtensions = {'epub', 'mobi', 'azw', 'azw3', 'fb2'};
const _fontExtensions = {'ttf', 'otf', 'woff', 'woff2'};
const _databaseExtensions = {'db', 'sqlite', 'sqlite3', 'mdb', 'accdb', 'parquet'};

IconData taskFileTypeIcon(String name, {bool isFolder = false, api_task.Protocol? protocol}) {
  return switch (_assetType(name, isFolder: isFolder, protocol: protocol)) {
    TaskAssetType.file => GopeedIcons.file,
    TaskAssetType.folder => GopeedIcons.folder,
    TaskAssetType.torrentFolder => GopeedIcons.folderBt,
    TaskAssetType.torrent => GopeedIcons.protocolBt,
    TaskAssetType.ed2k => GopeedIcons.protocolEd2k,
    TaskAssetType.installer => GopeedIcons.fileInstaller,
    TaskAssetType.androidPackage => GopeedIcons.fileAndroid,
    TaskAssetType.iosPackage => GopeedIcons.fileIos,
    TaskAssetType.diskImage => GopeedIcons.fileDiskImage,
    TaskAssetType.web => GopeedIcons.fileWeb,
    TaskAssetType.text => GopeedIcons.fileText,
    TaskAssetType.pdf => GopeedIcons.filePdf,
    TaskAssetType.document => GopeedIcons.fileDocument,
    TaskAssetType.spreadsheet => GopeedIcons.fileSpreadsheet,
    TaskAssetType.presentation => GopeedIcons.filePresentation,
    TaskAssetType.archive => GopeedIcons.fileArchive,
    TaskAssetType.image => GopeedIcons.fileImage,
    TaskAssetType.audio => GopeedIcons.fileAudio,
    TaskAssetType.video => GopeedIcons.fileVideo,
    TaskAssetType.code => GopeedIcons.fileCode,
    TaskAssetType.ebook => GopeedIcons.fileEbook,
    TaskAssetType.font => GopeedIcons.fileFont,
    TaskAssetType.database => GopeedIcons.fileDatabase,
  };
}

TaskAssetType _assetType(String name, {required bool isFolder, required api_task.Protocol? protocol}) {
  if (isFolder) {
    return protocol == api_task.Protocol.bt ? TaskAssetType.torrentFolder : TaskAssetType.folder;
  }

  final extension = _fileExtension(name);
  if (extension == 'torrent') return TaskAssetType.torrent;
  if (_installerExtensions.contains(extension)) return TaskAssetType.installer;
  if (_androidPackageExtensions.contains(extension)) return TaskAssetType.androidPackage;
  if (_iosPackageExtensions.contains(extension)) return TaskAssetType.iosPackage;
  if (_diskImageExtensions.contains(extension)) return TaskAssetType.diskImage;
  if (_webExtensions.contains(extension)) return TaskAssetType.web;
  if (_textExtensions.contains(extension)) return TaskAssetType.text;
  if (_pdfExtensions.contains(extension)) return TaskAssetType.pdf;
  if (_documentExtensions.contains(extension)) return TaskAssetType.document;
  if (_spreadsheetExtensions.contains(extension)) return TaskAssetType.spreadsheet;
  if (_presentationExtensions.contains(extension)) return TaskAssetType.presentation;
  if (_archiveExtensions.contains(extension)) return TaskAssetType.archive;
  if (_imageExtensions.contains(extension)) return TaskAssetType.image;
  if (_audioExtensions.contains(extension)) return TaskAssetType.audio;
  if (_videoExtensions.contains(extension)) return TaskAssetType.video;
  if (_codeExtensions.contains(extension)) return TaskAssetType.code;
  if (_ebookExtensions.contains(extension)) return TaskAssetType.ebook;
  if (_fontExtensions.contains(extension)) return TaskAssetType.font;
  if (_databaseExtensions.contains(extension)) return TaskAssetType.database;

  return switch (protocol) {
    api_task.Protocol.bt => TaskAssetType.torrent,
    api_task.Protocol.ed2k => TaskAssetType.ed2k,
    api_task.Protocol.http || null => TaskAssetType.file,
  };
}

String _fileExtension(String name) {
  final pathWithoutQuery = name.split(RegExp(r'[?#]')).first;
  final extension = path.extension(pathWithoutQuery).replaceFirst('.', '').toLowerCase();
  return extension.length <= 10 ? extension : '';
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
