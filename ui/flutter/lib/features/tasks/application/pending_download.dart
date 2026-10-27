import 'package:path/path.dart' as path;

import '../../../api/model/create_task.dart';
import '../../../api/model/create_task_batch.dart';
import '../../../api/model/options.dart';
import '../../../api/model/request.dart';
import '../../../api/model/resource.dart';
import '../../../api/model/resolve_result.dart';
import '../../../api/model/resolve_task.dart';
import '../../../core/capabilities/gopeed_capability.dart';

/// IDs only distinguish sources in the preview; the backend owns group IDs.
class PendingDownload {
  PendingDownload({required this.id, required this.request, required this.options});

  final String id;
  Request request;
  Options options;
  ResolveResult? result;
  String? resolveError;
  String? createError;
  final Set<int> selected = {};
  final Map<int, String> createdFiles = {};
  bool created = false;
  bool included = true;
  bool _submitting = false;
  bool _omitChecksum = false;

  bool get isGroup =>
      result != null && result!.id.isEmpty && result!.res.name.trim().isNotEmpty && result!.res.files.length > 1;
  String get name => result?.res.name.trim().isNotEmpty == true
      ? result!.res.name
      : result?.res.files.length == 1
      ? result!.res.files.single.name
      : request.url;
  List<FileInfo> get files => result?.res.files ?? const [];
  bool get hasSelection => included && !created && selected.any((index) => !createdFiles.containsKey(index));

  Future<void> resolve(GopeedService service) async {
    if (result != null || created) return;
    resolveError = null;
    try {
      result = await service.resolve(ResolveTask(req: request));
      selected.addAll(List.generate(files.isEmpty ? 1 : files.length, (index) => index));
    } catch (error) {
      resolveError = error.toString();
    }
  }

  Future<void> submit(GopeedService service, {bool direct = false}) async {
    if (_submitting) return;
    createError = null;
    if (!hasSelection || (!direct && result == null)) return;
    _submitting = true;
    if (!direct && selected.length > 1) _omitChecksum = true;
    try {
      if (direct || result!.id.isNotEmpty || files.isEmpty) {
        final opts = Options.fromJson(options.toJson());
        if (!direct && files.isNotEmpty) opts.selectFiles = selected.toList()..sort();
        if (!direct) _omitMultipleChecksums(opts);
        await service.createTask(
          CreateTask(req: direct || result!.id.isEmpty ? request : null, rid: result?.id, opts: opts),
        );
        created = true;
        return;
      }
      final indexes = selected.where((index) => !createdFiles.containsKey(index)).toList()..sort();
      final keepStructure = options.skipAutoCategorize || result!.res.name.isNotEmpty || files.length > 1;
      CreateTaskBatchItem item(int index) {
        final file = files[index];
        final req = file.req ?? request;
        final opts = Options.fromJson(options.toJson());
        opts.name = files.length == 1 && options.name.isNotEmpty ? options.name : file.name;
        final resourcePath = !isGroup && result!.res.name.trim().isNotEmpty ? result!.res.name : '';
        final pathParts = [
          if (options.path.isNotEmpty) options.path,
          if (resourcePath.isNotEmpty) resourcePath,
          file.path,
        ];
        opts.path = pathParts.isEmpty ? '' : path.joinAll(pathParts);
        opts.size = RegExp(r'^https?:', caseSensitive: false).hasMatch(req.url) ? file.size : 0;
        opts.skipAutoCategorize = keepStructure;
        // The page persists the user's selected parent directory after any
        // member succeeds. Never let a member subdirectory replace it.
        opts.asDefaultPath = false;
        _omitMultipleChecksums(opts);
        return CreateTaskBatchItem(req: req, opts: opts);
      }

      if (isGroup) {
        final response = await service.createTaskBatch(
          CreateTaskBatch(
            reqs: indexes.map(item).toList(),
            opts: options,
            group: true,
            groupName: options.name.isNotEmpty ? options.name : result!.res.name.trim(),
          ),
        );
        final errors = <String>[];
        for (final entry in response.items) {
          if (entry.index < 0 || entry.index >= indexes.length) continue;
          if (entry.taskId.isNotEmpty) createdFiles[indexes[entry.index]] = entry.taskId;
          if (entry.error.isNotEmpty && entry.taskId.isEmpty) {
            errors.add('${files[indexes[entry.index]].name}: ${entry.error}');
          }
        }
        if (errors.isNotEmpty) createError = errors.join('\n');
      } else {
        // Independent creations preserve successes when a later file fails.
        final errors = <String>[];
        for (final index in indexes) {
          final entry = item(index);
          try {
            createdFiles[index] = await service.createTask(CreateTask(req: entry.req, opts: entry.opts));
          } catch (error) {
            errors.add('${files[index].name}: $error');
          }
        }
        if (errors.isNotEmpty) createError = errors.join('\n');
      }
    } catch (error) {
      createError = error.toString();
    } finally {
      _submitting = false;
    }
  }

  void _omitMultipleChecksums(Options opts) {
    if (_omitChecksum && opts.extra is Map) {
      opts.extra = Map<String, dynamic>.from(opts.extra! as Map)..remove('checksum');
    }
  }
}
