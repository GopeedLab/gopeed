import '../../api/model/create_task.dart';
import '../../api/model/create_task_batch.dart';
import '../../api/model/downloader_config.dart';
import '../../api/model/extension.dart';
import '../../api/model/install_extension.dart';
import '../../api/model/resolve_result.dart';
import '../../api/model/resolve_task.dart';
import '../../api/model/switch_extension.dart';
import '../../api/model/task.dart';
import '../../api/model/update_check_extension_resp.dart';
import '../../api/model/update_extension_settings.dart';
import 'capability_rpc.dart';

abstract final class GopeedMethods {
  static const resolve = RpcMethod<ResolveTask, ResolveResult>('gopeed.resolve');
  static const createTask = RpcMethod<CreateTask, String>('gopeed.task.create');
  static const createTaskBatch = RpcMethod<CreateTaskBatch, List<String>>('gopeed.task.createBatch');
  static const patchTask = RpcMethod<Map<String, dynamic>, RpcUnit>('gopeed.task.patch');
  static const getTasks = RpcMethod<List<String>, List<Task>>('gopeed.task.list');
  static const getTaskStatus = RpcMethod<String, TaskRuntimeStatus>('gopeed.task.status');
  static const getTaskStats = RpcMethod<String, Map<String, dynamic>>('gopeed.task.stats');
  static const pauseTask = RpcMethod<String, RpcUnit>('gopeed.task.pause');
  static const continueTask = RpcMethod<String, RpcUnit>('gopeed.task.continue');
  static const pauseTasks = RpcMethod<Map<String, dynamic>, RpcUnit>('gopeed.task.pauseBatch');
  static const continueTasks = RpcMethod<Map<String, dynamic>, RpcUnit>('gopeed.task.continueBatch');
  static const deleteTask = RpcMethod<Map<String, dynamic>, RpcUnit>('gopeed.task.delete');
  static const deleteTasks = RpcMethod<Map<String, dynamic>, RpcUnit>('gopeed.task.deleteBatch');
  static const getConfig = RpcMethod<RpcUnit, DownloaderConfig>('gopeed.config.get');
  static const putConfig = RpcMethod<DownloaderConfig, RpcUnit>('gopeed.config.put');
  static const installExtension = RpcMethod<InstallExtension, String>('gopeed.extension.install');
  static const getExtensions = RpcMethod<RpcUnit, List<Extension>>('gopeed.extension.list');
  static const updateExtensionSettings = RpcMethod<Map<String, dynamic>, RpcUnit>('gopeed.extension.updateSettings');
  static const switchExtension = RpcMethod<Map<String, dynamic>, RpcUnit>('gopeed.extension.switch');
  static const deleteExtension = RpcMethod<String, RpcUnit>('gopeed.extension.delete');
  static const checkExtensionUpdate = RpcMethod<String, UpdateCheckExtensionResp>('gopeed.extension.checkUpdate');
  static const updateExtension = RpcMethod<String, RpcUnit>('gopeed.extension.update');
  static const testWebhook = RpcMethod<String, RpcUnit>('gopeed.webhook.test');
}

class GopeedService {
  const GopeedService(this._invoker);

  final CapabilityInvoker _invoker;

  Future<ResolveResult> resolve(ResolveTask request) => _invoker.invoke(GopeedMethods.resolve, request);

  Future<String> createTask(CreateTask request) => _invoker.invoke(GopeedMethods.createTask, request);

  Future<List<String>> createTaskBatch(CreateTaskBatch request) =>
      _invoker.invoke(GopeedMethods.createTaskBatch, request);

  Future<void> patchTask(String id, ResolveTask request) async {
    await _invoker.invoke(GopeedMethods.patchTask, {'id': id, 'request': request.toJson()});
  }

  Future<List<Task>> getTasks(List<Status> statuses) =>
      _invoker.invoke(GopeedMethods.getTasks, statuses.map((status) => status.name).toList(growable: false));

  Future<TaskRuntimeStatus> getTaskStatus(String id) => _invoker.invoke(GopeedMethods.getTaskStatus, id);

  Future<Map<String, dynamic>> getTaskStats(String id) => _invoker.invoke(GopeedMethods.getTaskStats, id);

  Future<void> pauseTask(String id) async => _invoker.invoke(GopeedMethods.pauseTask, id);

  Future<void> continueTask(String id) async => _invoker.invoke(GopeedMethods.continueTask, id);

  Future<void> pauseAllTasks(List<String>? ids) async {
    await _invoker.invoke(GopeedMethods.pauseTasks, {'ids': ids});
  }

  Future<void> continueAllTasks(List<String>? ids) async {
    await _invoker.invoke(GopeedMethods.continueTasks, {'ids': ids});
  }

  Future<void> deleteTask(String id, bool force) async {
    await _invoker.invoke(GopeedMethods.deleteTask, {'id': id, 'force': force});
  }

  Future<void> deleteTasks(List<String>? ids, bool force) async {
    await _invoker.invoke(GopeedMethods.deleteTasks, {'ids': ids, 'force': force});
  }

  Future<DownloaderConfig> getConfig() => _invoker.invoke(GopeedMethods.getConfig, const RpcUnit());

  Future<void> putConfig(DownloaderConfig config) async => _invoker.invoke(GopeedMethods.putConfig, config);

  Future<String> installExtension(InstallExtension request) => _invoker.invoke(GopeedMethods.installExtension, request);

  Future<List<Extension>> getExtensions() => _invoker.invoke(GopeedMethods.getExtensions, const RpcUnit());

  Future<void> updateExtensionSettings(String identity, UpdateExtensionSettings request) async {
    await _invoker.invoke(GopeedMethods.updateExtensionSettings, {'identity': identity, 'request': request.toJson()});
  }

  Future<void> switchExtension(String identity, SwitchExtension request) async {
    await _invoker.invoke(GopeedMethods.switchExtension, {'identity': identity, 'request': request.toJson()});
  }

  Future<void> deleteExtension(String identity) async => _invoker.invoke(GopeedMethods.deleteExtension, identity);

  Future<UpdateCheckExtensionResp> upgradeCheckExtension(String identity) =>
      _invoker.invoke(GopeedMethods.checkExtensionUpdate, identity);

  Future<void> updateExtension(String identity) async => _invoker.invoke(GopeedMethods.updateExtension, identity);

  Future<void> testWebhook(String url) async => _invoker.invoke(GopeedMethods.testWebhook, url);
}
