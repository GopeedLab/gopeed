import 'dart:async';
import 'dart:convert';
import 'dart:ffi';
import 'dart:io';

import 'package:dio/dio.dart';

import '../../api/model/create_task.dart';
import '../../api/model/create_task_batch.dart';
import '../../api/model/downloader_config.dart';
import '../../api/model/extension.dart';
import '../../api/model/http_listen_result.dart';
import '../../api/model/info.dart';
import '../../api/model/install_extension.dart';
import '../../api/model/request.dart';
import '../../api/model/resolve_result.dart';
import '../../api/model/result.dart';
import '../../api/model/switch_extension.dart';
import '../../api/model/task.dart';
import '../../api/model/task_filter.dart';
import '../../api/model/update_check_extension_resp.dart';
import '../../api/model/update_extension_settings.dart';
import '../../util/extensions.dart';
import '../../util/util.dart';
import '../libgopeed_boot.dart';
import '../native/channel/libgopeed_channel.dart';
import '../native/ffi/libgopeed_bind.dart';
import '../native/ffi/libgopeed_ffi.dart';
import '../native/libgopeed_interface.dart';
import '../native/model/invoke_request.dart';
import '../native/model/start_config.dart';
import 'libgopeed_boot_base.dart';

LibgopeedBoot create() => LibgopeedBootNative();

class LibgopeedBootNative with LibgopeedBootBase implements LibgopeedBoot {
  late LibgopeedAbi _libgopeedAbi;
  late int _apiInstance;
  late Dio _dio;

  LibgopeedBootNative() {
    if (Util.isDesktop()) {
      var libName = "libgopeed.";
      if (Platform.isWindows) {
        libName += "dll";
      }
      if (Platform.isMacOS) {
        libName += "dylib";
      }
      if (Platform.isLinux) {
        libName += "so";
      }
      _libgopeedAbi = LibgopeedFFi(LibgopeedBind(DynamicLibrary.open(libName)));
    } else {
      _libgopeedAbi = LibgopeedChannel();
    }
  }

  Future<T> _invoke<T>(
    String method,
    List<dynamic> params,
    T Function(dynamic json)? fromJsonT,
  ) async {
    final invokeRequest = InvokeRequest(
        method: method,
        params: params.map((e) {
          if (e.toJson != null) {
            return jsonEncode(e.toJson());
          }
          return jsonEncode(e);
        }).toList());
    final resp =
        await _libgopeedAbi.invoke(_apiInstance, jsonEncode(invokeRequest));
    final result =
        Result<T>.fromJson(jsonDecode(resp), fromJsonT ??= (json) => null as T);
    return handleResult(result);
  }

  List<dynamic> _parseTaskFilter(TaskFilter? filter) {
    if (filter == null) {
      return [];
    }
    return [filter];
  }

  @override
  Future<void> init(StartConfig cfg) async {
    final result = await _libgopeedAbi.create(jsonEncode(cfg.toJson()));
    _apiInstance =
        Result.fromJson(jsonDecode(result), (data) => data as int).data!;
    _dio = createDio();
  }

  @override
  Future<HttpListenResult> startHttp() async {
    return _invoke<HttpListenResult>(
        "startHttp", [], (json) => HttpListenResult.fromJson(json));
  }

  @override
  Future<void> stopHttp() async {
    await _invoke<void>("stopHttp", [], null);
  }

  @override
  Future<void> restartHttp() async {
    await _invoke<void>("restartHttp", [], null);
  }

  @override
  Future<Info> info() async {
    return _invoke<Info>("info", [], (json) => Info.fromJson(json));
  }

  @override
  Future<ResolveResult> resolve(Request request) async {
    return _invoke<ResolveResult>(
        "resolve", [request], (json) => ResolveResult.fromJson(json));
  }

  @override
  Future<String> createTask(CreateTask createTask) async {
    return _invoke<String>(
        "createTask", [createTask], (json) => json as String);
  }

  @override
  Future<String> createTaskBatch(CreateTaskBatch createTask) async {
    return _invoke<String>(
        "createTaskBatch", [createTask], (json) => json as String);
  }

  @override
  Future<void> pauseTask(String id) async {
    await _invoke<void>("pauseTask", [id], null);
  }

  @override
  Future<void> pauseTasks(TaskFilter? filter) async {
    await _invoke<void>("pauseTasks", _parseTaskFilter(filter), null);
  }

  @override
  Future<void> continueTask(String id) async {
    await _invoke<void>("continueTask", [id], null);
  }

  @override
  Future<void> continueTasks(TaskFilter? filter) async {
    await _invoke<void>("continueTasks", _parseTaskFilter(filter), null);
  }

  @override
  Future<void> deleteTask(String id, bool force) async {
    await _invoke<void>("deleteTask", [id, force], null);
  }

  @override
  Future<void> deleteTasks(TaskFilter? filter, bool force) async {
    await _invoke<void>(
        "deleteTasks",
        _parseTaskFilter(filter).apply((it) {
          it.add(force);
        }),
        null);
  }

  @override
  Future<Task> getTask(String id) async {
    return _invoke<Task>("getTask", [id], (json) => Task.fromJson(json));
  }

  @override
  Future<List<Task>> getTasks(TaskFilter? filter) async {
    return _invoke<List<Task>>("getTasks", _parseTaskFilter(filter),
        (json) => json.map((e) => Task.fromJson(e)).toList());
  }

  @override
  Future<Map<String, dynamic>> getTaskStats(String id) async {
    return _invoke<Map<String, dynamic>>(
        "getTaskStats", [id], (json) => json as Map<String, dynamic>);
  }

  @override
  Future<DownloaderConfig> getConfig() async {
    return _invoke<DownloaderConfig>(
        "getConfig", [], (json) => DownloaderConfig.fromJson(json));
  }

  @override
  Future<void> putConfig(DownloaderConfig config) async {
    await _invoke<void>("putConfig", [config], null);
  }

  @override
  Future<void> installExtension(InstallExtension installExtension) async {
    await _invoke<void>("installExtension", [installExtension], null);
  }

  @override
  Future<List<Extension>> getExtensions() async {
    return _invoke<List<Extension>>("getExtensions", [],
        (json) => json.map((e) => Extension.fromJson(e)).toList());
  }

  @override
  Future<void> updateExtensionSettings(
      String identity, UpdateExtensionSettings updateExtensionSettings) async {
    await _invoke<void>(
        "updateExtensionSettings", [identity, updateExtensionSettings], null);
  }

  @override
  Future<void> switchExtension(
      String identity, SwitchExtension switchExtension) async {}

  @override
  Future<void> deleteExtension(String identity) async {
    await _invoke<void>("deleteExtension", [identity], null);
  }

  @override
  Future<UpdateCheckExtensionResp> upgradeCheckExtension(
      String identity) async {
    return _invoke<UpdateCheckExtensionResp>("upgradeCheckExtension",
        [identity], (json) => UpdateCheckExtensionResp.fromJson(json));
  }

  @override
  Future<void> upgradeExtension(String identity) async {
    await _invoke<void>("upgradeExtension", [identity], null);
  }

  @override
  Future<void> close() async {
    await _invoke<void>("close", [], null);
  }

  @override
  Future<Response<String>> proxyRequest<T>(String uri,
      {data, Options? options}) {
    return _dio.request(uri, data: data, options: options);
  }
}
