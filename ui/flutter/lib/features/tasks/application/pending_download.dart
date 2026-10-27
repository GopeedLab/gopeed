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
  final Request request;
  final Options options;
  ResolveResult? result;
  String? resolveError;
  String? createError;
  final Set<int> selected = {};
  final Map<int, String> createdFiles = {};
  bool created = false;
  bool included = true;

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

}
