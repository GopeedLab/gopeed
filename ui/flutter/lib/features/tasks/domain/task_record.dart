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
  const TaskFileNode({
    required this.path,
    required this.name,
    required this.sizeBytes,
    this.downloadedBytes,
    this.resourceIndex,
    this.status,
  });

  final int? resourceIndex;
  final String path;
  final String name;
  final int sizeBytes;
  final int? downloadedBytes;
  final TaskStatus? status;

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
    this.isGroup = false,
    this.members = const [],
    this.protocol,
    this.requestHeaders = const {},
    this.total,
    this.speed,
    this.uploadSpeed,
    this.remaining,
    this.downloadDuration,
    this.createdAt,
    this.doneAt,
    this.progress,
    this.error,
    this.completedLabel,
    this.downloadedBytes,
    this.totalBytes,
    this.speedBytes,
    this.uploadedBytes,
    this.uploadSpeedBytes,
    this.updatedAt,
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
  final DateTime? doneAt;
  final double? progress;
  final String url;
  final String storagePath;
  final String? error;
  final String? completedLabel;
  final int? downloadedBytes;
  final int? totalBytes;
  final int? speedBytes;
  final int? uploadedBytes;
  final int? uploadSpeedBytes;
  final DateTime? updatedAt;
  final int? remainingSeconds;
  final bool waiting;
  final List<TaskFileNode> files;
  final bool uploading;
  final bool isFolder;
  final bool isGroup;
  final List<TaskRecord> members;
  final api_task.Protocol? protocol;
  final Map<String, String> requestHeaders;

  bool get canUpdateUrl =>
      !isGroup && protocol == api_task.Protocol.http && (status == TaskStatus.paused || status == TaskStatus.failed);

  bool get canPause => isGroup ? members.any((task) => task.canPause) : status == TaskStatus.downloading;
  bool get canResume =>
      isGroup ? members.any((task) => task.canResume) : status == TaskStatus.paused || status == TaskStatus.failed;

  bool matchesSearch(String query) =>
      query.isEmpty ||
      name.toLowerCase().contains(query) ||
      url.toLowerCase().contains(query) ||
      files.any((file) => '${file.path}/${file.name}'.toLowerCase().contains(query)) ||
      members.any((member) => member.matchesSearch(query));

  bool get isIndeterminate => status == TaskStatus.downloading && !waiting && total == null;

  TaskAssetType get assetType => _assetType(name, isFolder: isFolder, protocol: protocol);

  IconData get icon => taskFileTypeIcon(name, isFolder: isFolder, protocol: protocol);

  factory TaskRecord.fromApi(api_task.Task task) {
    if (task.isGroup) return _groupRecord(task, const []);
    final totalBytes = task.meta.res?.size ?? task.meta.opts.size;
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
      speed: totalBytes > 0 || task.progress.speed > 0 ? TransferRateFormatter.format(task.progress.speed).text : null,
      uploadSpeed: task.uploading ? TransferRateFormatter.format(task.progress.uploadSpeed).text : null,
      remainingSeconds: _remainingSeconds(task, totalBytes, downloadedBytes),
      downloadDuration: task.progress.used > 0 ? Duration(microseconds: (task.progress.used + 999) ~/ 1000) : null,
      createdAt: task.createdAt,
      updatedAt: task.updatedAt,
      doneAt: task.doneAt,
      waiting: task.status == api_task.Status.wait,
      progress: progress,
      url: displayUrl,
      storagePath: storagePath,
      downloadedBytes: downloadedBytes,
      totalBytes: totalBytes > 0 ? totalBytes : null,
      speedBytes: task.progress.speed,
      uploadedBytes: task.progress.uploaded,
      uploadSpeedBytes: task.progress.uploadSpeed,
      files: _fileNodes(task.meta.res?.files, task.meta.opts.selectFiles),
      uploading: task.uploading,
      isFolder: task.meta.res?.name.isNotEmpty ?? false,
      protocol: task.protocol,
      requestHeaders: _requestHeaders(task.meta.req.extra),
    );
  }
}

List<TaskRecord> taskRecordsFromApi(List<api_task.Task> tasks) {
  final groups = {
    for (final task in tasks)
      if (task.isGroup) task.id,
  };
  final members = <String, List<TaskRecord>>{};
  final records = <String, TaskRecord>{};
  for (final task in tasks) {
    if (task.isGroup) continue;
    final record = TaskRecord.fromApi(task);
    records[task.id] = record;
    if (groups.contains(task.parentId)) {
      members.putIfAbsent(task.parentId, () => []).add(record);
    }
  }
  return [
    for (final task in tasks)
      if (task.isGroup)
        _groupRecord(task, members[task.id] ?? const [])
      else if (!groups.contains(task.parentId))
        records[task.id]!,
  ];
}

TaskRecord _groupRecord(api_task.Task group, List<TaskRecord> members) {
  final running = members.any((task) => task.status == TaskStatus.downloading && !task.waiting);
  final failed = members.any((task) => task.status == TaskStatus.failed);
  final paused = members.isEmpty || members.any((task) => task.status == TaskStatus.paused);
  final completed = members.isNotEmpty && members.every((task) => task.status == TaskStatus.completed);
  final status = running
      ? TaskStatus.downloading
      : failed
      ? TaskStatus.failed
      : paused
      ? TaskStatus.paused
      : completed
      ? TaskStatus.completed
      : TaskStatus.downloading;
  final downloaded = members.fold(0, (total, task) => total + (task.downloadedBytes ?? 0));
  final total = members.every((task) => task.totalBytes != null || task.status == TaskStatus.completed)
      ? members.fold(0, (total, task) => total + (task.totalBytes ?? 0))
      : null;
  final speed = members.fold(
    0,
    (total, task) => total + (task.status == TaskStatus.downloading && !task.waiting ? task.speedBytes ?? 0 : 0),
  );
  final uploadSpeed = members.fold(0, (total, task) => total + (task.uploading ? task.uploadSpeedBytes ?? 0 : 0));
  final uploaded = members.fold(0, (total, task) => total + (task.uploadedBytes ?? 0));
  final pathContext = path.Context(
    style: group.meta.opts.path.contains('\\') || members.any((task) => task.storagePath.contains('\\'))
        ? path.Style.windows
        : path.Style.posix,
  );
  var root = group.meta.opts.path;
  if (root.isEmpty && members.isNotEmpty) {
    root = pathContext.dirname(members.first.storagePath);
    for (final member in members.skip(1)) {
      while (!pathContext.equals(root, member.storagePath) && !pathContext.isWithin(root, member.storagePath)) {
        final parent = pathContext.dirname(root);
        if (parent == root) break;
        root = parent;
      }
    }
  }
  final files = <TaskFileNode>[];
  for (final member in members) {
    final sourceFiles = member.isFolder
        ? member.files
        : [TaskFileNode(path: '', name: member.name, sizeBytes: member.totalBytes ?? 0)];
    for (final file in sourceFiles) {
      final fullPath = member.isFolder
          ? pathContext.join(member.storagePath, file.path, file.name)
          : member.storagePath;
      final relative = pathContext.relative(fullPath, from: root).replaceAll('\\', '/');
      files.add(
        TaskFileNode(
          resourceIndex: files.length,
          path: path.posix.dirname(relative) == '.' ? '' : path.posix.dirname(relative),
          name: path.posix.basename(relative),
          sizeBytes: file.sizeBytes,
          downloadedBytes: member.isFolder ? file.downloadedBytes : member.downloadedBytes,
          status: member.status,
        ),
      );
    }
  }
  final doneTimes = members.map((task) => task.doneAt).whereType<DateTime>().toList()..sort();
  final updates = [group.updatedAt, ...members.map((task) => task.updatedAt).whereType<DateTime>()]..sort();
  return TaskRecord(
    id: group.id,
    name: group.name,
    status: status,
    downloaded: _formatBytes(downloaded),
    total: total == null ? null : _formatBytes(total),
    downloadedBytes: downloaded,
    totalBytes: total,
    speed: TransferRateFormatter.format(speed).text,
    speedBytes: speed,
    uploadSpeed: uploadSpeed > 0 ? TransferRateFormatter.format(uploadSpeed).text : null,
    uploadedBytes: uploaded,
    uploadSpeedBytes: uploadSpeed,
    uploading: uploadSpeed > 0,
    progress: completed
        ? 1
        : total != null && total > 0
        ? (downloaded / total).clamp(0.0, 1.0)
        : null,
    remainingSeconds: total != null && speed > 0 && total > downloaded ? ((total - downloaded) / speed).ceil() : null,
    waiting: status == TaskStatus.downloading && !running,
    url: '',
    storagePath: root,
    files: files,
    isFolder: true,
    isGroup: true,
    members: members,
    createdAt: group.createdAt,
    updatedAt: updates.last,
    doneAt: completed && doneTimes.isNotEmpty ? doneTimes.last : null,
  );
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
    api_task.Protocol.hls || api_task.Protocol.http || null => TaskAssetType.file,
  };
}

String _fileExtension(String name) {
  final uri = Uri.tryParse(name);
  final filePath = uri != null && uri.hasScheme && uri.hasAuthority ? uri.path : name;
  final extension = path.extension(filePath).replaceFirst('.', '').toLowerCase();
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

List<TaskFileNode> _fileNodes(List<api_resource.FileInfo>? files, List<int> selectedIndexes) {
  if (files == null || files.isEmpty) {
    return const [];
  }
  final selected = selectedIndexes.toSet();
  return files
      .asMap()
      .entries
      .where((entry) => selected.isEmpty || selected.contains(entry.key))
      .map(
        (entry) => TaskFileNode(
          resourceIndex: entry.key,
          path: entry.value.path,
          name: entry.value.name,
          sizeBytes: entry.value.size,
        ),
      )
      .toList(growable: false);
}

String _formatBytes(int bytes) {
  return ByteSizeFormatter.format(bytes);
}
