import 'package:path/path.dart' as path;

import '../../../api/model/create_task_batch.dart';
import '../../../api/model/options.dart';
import '../../../api/model/request.dart';
import '../../../api/model/resource.dart';
import '../../../api/model/resolve_result.dart';
import '../../../api/model/resolve_task.dart';
import '../../../core/capabilities/gopeed_capability.dart';

/// IDs identify sources within one preview and are never sent as group IDs.
class PendingDownload {
  PendingDownload({required this.id, required this.request, required this.options});

  final String id;
  final Request request;
  final Options options;
  ResolveResult? result;
  String? resolveError;
  String? createError;
  final Set<int> selected = {};
  bool created = false;
  bool resolving = false;

  bool get loading => result == null && resolveError == null;
  bool get isGroup => result != null && result!.id.isEmpty && result!.res.name.trim().isNotEmpty && files.length > 1;
  bool get isFtp => RegExp(r'^(ftps?|ftpes):', caseSensitive: false).hasMatch(request.url.trim());
  bool get isBt {
    if (result?.id.isEmpty == true) return false;
    final url = request.url.trim().toLowerCase();
    if (url.startsWith('http:') || url.startsWith('https:') || isFtp) return false;
    return url.startsWith('magnet:') ||
        url.startsWith('data:application/x-bittorrent;base64,') ||
        url.split(RegExp(r'[?#]')).first.endsWith('.torrent');
  }

  String get name {
    if (!isBt &&
        (!isFtp || result?.res.name.trim().isEmpty == true) &&
        result?.id.isNotEmpty == true &&
        files.length == 1 &&
        files.single.name.isNotEmpty) {
      return files.single.name;
    }
    if (result?.res.name.trim().isNotEmpty == true) return result!.res.name;
    return request.url;
  }

  List<FileInfo> get files => result?.res.files ?? const [];
  bool get hasSelection => !created && result != null && selected.isNotEmpty;

  Future<void> resolve(GopeedService service) async {
    if (result != null || resolving) return;
    resolving = true;
    resolveError = null;
    try {
      result = await service.resolve(ResolveTask(req: request));
      selected.addAll(List.generate(files.isEmpty ? 1 : files.length, (index) => index));
    } catch (error) {
      resolveError = error.toString();
    } finally {
      resolving = false;
    }
  }

  List<CreateTaskBatchItem> creationItems() {
    final indexes = selected.toList()..sort();
    if (result!.id.isNotEmpty || files.isEmpty) {
      final opts = Options.fromJson(options.toJson());
      if (files.isNotEmpty) opts.selectFiles = indexes;
      if (indexes.length > 1) _omitChecksum(opts);
      return [
        CreateTaskBatchItem(
          rid: result!.id.isEmpty ? null : result!.id,
          req: result!.id.isEmpty ? request : null,
          opts: opts,
        ),
      ];
    }
    final group = isGroup;
    final resourceName = result!.res.name;
    final parentSegments = [
      if (!group && options.path.isNotEmpty) options.path,
      if (!group && resourceName.trim().isNotEmpty) resourceName,
    ];
    final skipAutoCategorize = options.skipAutoCategorize || resourceName.isNotEmpty || files.length > 1;
    final httpUrl = RegExp(r'^https?:', caseSensitive: false);
    return indexes.map((index) {
      final file = files[index];
      final member = Options.fromJson(options.toJson());
      member.name = files.length == 1 && options.name.isNotEmpty ? options.name : file.name;
      final segments = [...parentSegments, if (file.path.isNotEmpty) file.path];
      member.path = segments.isEmpty ? '' : path.joinAll(segments);
      // The selected parent directory is persisted separately from members.
      member.asDefaultPath = false;
      member.skipAutoCategorize = skipAutoCategorize;
      final req = file.req ?? request;
      member.size = httpUrl.hasMatch(req.url) ? file.size : 0;
      if (RegExp(r'^(ftps?|ftpes):', caseSensitive: false).hasMatch(req.url)) {
        final extra = OptsExtraFtp.fromJson(Map<String, dynamic>.from(member.extra as Map? ?? const {}));
        extra.connections = extra.connections.clamp(0, 32);
        member.extra = extra.toJson();
      }
      if (indexes.length > 1) _omitChecksum(member);
      return CreateTaskBatchItem(req: req, opts: member);
    }).toList();
  }

  void _omitChecksum(Options opts) {
    if (opts.extra is Map) opts.extra = Map<String, dynamic>.from(opts.extra! as Map)..remove('checksum');
  }
}

/// A successful response completes the entire batch, including partial success.
class DownloadSubmission {
  DownloadSubmission({required this.sources, required this.request});

  final List<PendingDownload> sources;
  final CreateTaskBatch request;

  Future<void> submit(GopeedService service) async {
    for (final source in sources) {
      source.createError = null;
    }
    try {
      await service.createTaskBatch(request);
      for (final source in sources) {
        source.created = true;
      }
    } catch (error) {
      for (final source in sources) {
        source.createError = error.toString();
      }
    }
  }
}

List<DownloadSubmission> planDownloads(List<PendingDownload> sources) {
  final groups = <DownloadSubmission>[];
  final independent = <PendingDownload>[];
  for (final source in sources.where((source) => source.hasSelection)) {
    if (source.isGroup) {
      groups.add(
        DownloadSubmission(
          sources: [source],
          request: CreateTaskBatch(
            group: true,
            groupName: source.options.name.isNotEmpty ? source.options.name : source.result!.res.name.trim(),
            opts: source.options,
            reqs: source.creationItems(),
          ),
        ),
      );
    } else {
      independent.add(source);
    }
  }
  return [
    ...groups,
    if (independent.isNotEmpty)
      DownloadSubmission(
        sources: independent,
        request: CreateTaskBatch(reqs: [for (final source in independent) ...source.creationItems()]),
      ),
  ];
}
