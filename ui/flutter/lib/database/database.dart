import 'dart:convert';

import 'package:gopeed/util/util.dart';
import 'package:hive/hive.dart';

import 'entity.dart';

const String _startConfig = 'startConfig';
const String _windowState = 'windowState';
const String _bookmark = 'bookmark';
const String _createHistory = 'createHistory';
const String _webToken = 'webToken';
const String _runAsMenubarApp = 'runAsMenubarApp';

class Database {
  static final Database _instance = Database._internal();

  static Database get instance => _instance;

  factory Database() {
    return _instance;
  }

  late Box box;

  Database._internal();

  Future<void> init() async {
    Hive.init(Util.getStorageDir());
    box = await Hive.openBox('database');
  }

  void save<T>(String key, T entity) {
    box.put(key, jsonEncode(entity));
  }

  T? get<T>(String key, T Function(dynamic json) fromJsonT) {
    final json = box.get(key);
    if (json == null) {
      return null;
    }
    return fromJsonT(jsonDecode(json));
  }

  void clear(String key) {
    box.delete(key);
  }

  void saveStartConfig(StartConfigEntity entity) {
    save<StartConfigEntity>(_startConfig, entity);
  }

  StartConfigEntity? getStartConfig() {
    return get<StartConfigEntity>(
        _startConfig, (json) => StartConfigEntity.fromJson(json));
  }

  /// Patch non-null fields with the original value
  void saveWindowState(WindowStateEntity entity) {
    final state = getWindowState();
    entity.isMaximized ??= state?.isMaximized;
    entity.width ??= state?.width;
    entity.height ??= state?.height;
    save<WindowStateEntity>(_windowState, entity);
  }

  WindowStateEntity? getWindowState() {
    return get<WindowStateEntity>(
        _windowState, (json) => WindowStateEntity.fromJson(json));
  }

  /// Use map to ensure that the same directory only saves the latest bookmark
  void saveBookmark(MapEntry<String, String> entry) {
    final map = getBookmark() ?? {};
    map[entry.key] = entry.value;
    save<Map<String, String>>(_bookmark, map);
  }

  Map<String, String>? getBookmark() {
    return get<Map<String, String>>(_bookmark, (json) {
      return (json as Map<String, dynamic>)
          .map((key, value) => MapEntry(key, value.toString()));
    });
  }

  void saveWebToken(String token) {
    save<String>(_webToken, token);
  }

  String? getWebToken() {
    return get<String>(_webToken, (json) => json.toString());
  }

  void saveCreateHistory(String url) {
    var list = getCreateHistory() ?? [];
    list.remove(url);
    list.insert(0, url);
    if (list.length > 64) {
      list.removeLast();
    }
    save<List<String>>(_createHistory, list);
  }

  List<String>? getCreateHistory() {
    return get<List<String>>(_createHistory, (json) {
      return (json as List<dynamic>).map((e) => e.toString()).toList();
    });
  }

  void clearCreateHistory() {
    clear(_createHistory);
  }

  void saveRunAsMenubarApp(bool value) {
    box.put(_runAsMenubarApp, value);
  }

  bool getRunAsMenubarApp() {
    return box.get(_runAsMenubarApp, defaultValue: false) as bool;
  }
}
