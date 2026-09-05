import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/api.dart' as api;
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
import '../../database/database.dart';
import '../window/app_window_appearance.dart';
import 'capability_rpc.dart';
import 'gopeed_capability.dart';
import 'storage_capability.dart';

class AppCapabilities {
  AppCapabilities(CapabilityInvoker invoker) : gopeed = GopeedService(invoker), storage = AppStorageService(invoker);

  final GopeedService gopeed;
  final AppStorageService storage;
}

class LocalAppCapabilities {
  LocalAppCapabilities._() : codecs = createAppCapabilityCodecs() {
    registry = CapabilityRegistry(codecs);
    _bindGopeed(registry);
    _bindStorage(registry);
    capabilities = AppCapabilities(LocalCapabilityInvoker(registry));
  }

  static final instance = LocalAppCapabilities._();

  final RpcCodecRegistry codecs;
  late CapabilityRegistry registry;
  late AppCapabilities capabilities;
}

final appCapabilitiesProvider = Provider<AppCapabilities>((ref) => LocalAppCapabilities.instance.capabilities);
final gopeedServiceProvider = Provider<GopeedService>((ref) => ref.watch(appCapabilitiesProvider).gopeed);
final appStorageServiceProvider = Provider<AppStorageService>((ref) => ref.watch(appCapabilitiesProvider).storage);

RpcCodecRegistry createAppCapabilityCodecs() {
  final codecs = RpcCodecRegistry();
  codecs
    ..register<AppWindowAppearance>((json) => AppWindowAppearance.fromJson(json! as Map<String, dynamic>))
    ..register<ResolveTask>((json) => ResolveTask.fromJson(json! as Map<String, dynamic>))
    ..register<ResolveResult>((json) => ResolveResult.fromJson(json! as Map<String, dynamic>))
    ..register<CreateTask>((json) => CreateTask.fromJson(json! as Map<String, dynamic>))
    ..register<CreateTaskBatch>((json) => CreateTaskBatch.fromJson(json! as Map<String, dynamic>))
    ..register<DownloaderConfig>((json) => DownloaderConfig.fromJson(json! as Map<String, dynamic>))
    ..register<TaskRuntimeStatus>((json) => TaskRuntimeStatus.fromJson(json! as Map<String, dynamic>))
    ..register<List<Task>>(
      (json) =>
          (json! as List<dynamic>).map((item) => Task.fromJson(item as Map<String, dynamic>)).toList(growable: false),
    )
    ..register<InstallExtension>((json) => InstallExtension.fromJson(json! as Map<String, dynamic>))
    ..register<UpdateExtensionSettings>((json) => UpdateExtensionSettings.fromJson(json! as Map<String, dynamic>))
    ..register<SwitchExtension>((json) => SwitchExtension.fromJson(json! as Map<String, dynamic>))
    ..register<List<Extension>>(
      (json) => (json! as List<dynamic>)
          .map((item) => Extension.fromJson(item as Map<String, dynamic>))
          .toList(growable: false),
    )
    ..register<UpdateCheckExtensionResp>((json) => UpdateCheckExtensionResp.fromJson(json! as Map<String, dynamic>));
  return codecs;
}

void _bindGopeed(CapabilityRegistry registry) {
  registry
    ..bind(GopeedMethods.resolve, api.resolve)
    ..bind(GopeedMethods.createTask, api.createTask)
    ..bind(GopeedMethods.createTaskBatch, api.createTaskBatch)
    ..bind(GopeedMethods.patchTask, (params) async {
      await api.patchTask(params['id'] as String, ResolveTask.fromJson(params['request'] as Map<String, dynamic>));
      return const RpcUnit();
    })
    ..bind(
      GopeedMethods.getTasks,
      (statuses) => api.getTasks(statuses.map((name) => Status.values.byName(name)).toList(growable: false)),
    )
    ..bind(GopeedMethods.getTaskStatus, api.getTaskStatus)
    ..bind(GopeedMethods.getTaskStats, api.getTaskStats)
    ..bind(GopeedMethods.pauseTask, (id) async {
      await api.pauseTask(id);
      return const RpcUnit();
    })
    ..bind(GopeedMethods.continueTask, (id) async {
      await api.continueTask(id);
      return const RpcUnit();
    })
    ..bind(GopeedMethods.pauseTasks, (params) async {
      await api.pauseAllTasks(_optionalStringList(params['ids']));
      return const RpcUnit();
    })
    ..bind(GopeedMethods.continueTasks, (params) async {
      await api.continueAllTasks(_optionalStringList(params['ids']));
      return const RpcUnit();
    })
    ..bind(GopeedMethods.deleteTask, (params) async {
      await api.deleteTask(params['id'] as String, params['force'] as bool? ?? false);
      return const RpcUnit();
    })
    ..bind(GopeedMethods.deleteTasks, (params) async {
      await api.deleteTasks(_optionalStringList(params['ids']), params['force'] as bool? ?? false);
      return const RpcUnit();
    })
    ..bind(GopeedMethods.getConfig, (_) => api.getConfig())
    ..bind(GopeedMethods.putConfig, (config) async {
      await api.putConfig(config);
      return const RpcUnit();
    })
    ..bind(GopeedMethods.installExtension, api.installExtension)
    ..bind(GopeedMethods.getExtensions, (_) => api.getExtensions())
    ..bind(GopeedMethods.updateExtensionSettings, (params) async {
      await api.updateExtensionSettings(
        params['identity'] as String,
        UpdateExtensionSettings.fromJson(params['request'] as Map<String, dynamic>),
      );
      return const RpcUnit();
    })
    ..bind(GopeedMethods.switchExtension, (params) async {
      await api.switchExtension(
        params['identity'] as String,
        SwitchExtension.fromJson(params['request'] as Map<String, dynamic>),
      );
      return const RpcUnit();
    })
    ..bind(GopeedMethods.deleteExtension, (identity) async {
      await api.deleteExtension(identity);
      return const RpcUnit();
    })
    ..bind(GopeedMethods.checkExtensionUpdate, api.upgradeCheckExtension)
    ..bind(GopeedMethods.updateExtension, (identity) async {
      await api.updateExtension(identity);
      return const RpcUnit();
    })
    ..bind(GopeedMethods.testWebhook, (url) async {
      await api.testWebhook(url);
      return const RpcUnit();
    });
}

void _bindStorage(CapabilityRegistry registry) {
  registry
    ..bind(StorageMethods.getCreateHistory, (_) async => Database.instance.getCreateHistory() ?? const <String>[])
    ..bind(StorageMethods.saveCreateHistory, (urls) async {
      for (final url in urls) {
        Database.instance.saveCreateHistory(url);
      }
      return const RpcUnit();
    })
    ..bind(StorageMethods.removeCreateHistory, (url) async {
      Database.instance.removeCreateHistory(url);
      return const RpcUnit();
    })
    ..bind(StorageMethods.clearCreateHistory, (_) async {
      Database.instance.clearCreateHistory();
      return const RpcUnit();
    });
}

List<String>? _optionalStringList(Object? value) {
  if (value == null) return null;
  return (value as List<dynamic>).map((item) => item.toString()).toList(growable: false);
}
