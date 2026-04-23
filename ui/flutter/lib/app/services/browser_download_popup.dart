import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:get/get.dart';
import 'package:open_filex/open_filex.dart';
import 'package:path/path.dart' as path;
import 'package:window_manager/window_manager.dart';

import '../../api/api.dart' as api;
import '../../api/model/task.dart';
import '../../i18n/message.dart';
import '../../theme/theme.dart';
import '../../util/file_explorer.dart';
import '../../util/locale_manager.dart';
import '../../util/log_util.dart';
import '../../util/util.dart';
import '../modules/app/controllers/app_controller.dart';
import '../views/file_icon.dart';

const _popupWindowWidth = 760.0;
const _popupWindowHeight = 430.0;
const _popupDoneWindowWidth = 706.0;
const _popupDoneWindowHeight = 264.0;

Future<void> appendPopupDebugLog(String message) async {
  try {
    final logDir = Directory(path.join(Util.getStorageDir(), 'logs'));
    if (!await logDir.exists()) {
      await logDir.create(recursive: true);
    }
    final logFile = File(path.join(logDir.path, 'popup.log'));
    final now = DateTime.now().toIso8601String();
    await logFile.writeAsString('[$now] $message\n',
        mode: FileMode.append, flush: true);
  } catch (_) {}
}

class BrowserDownloadPopupLauncher extends GetxService {
  final Set<String> _launchedTaskIds = <String>{};

  Future<void> _writeDebugLog(String message) async {
    await appendPopupDebugLog(message);
  }

  Future<void> show(String taskId) async {
    if (!Util.isWindows()) {
      await _writeDebugLog(
          'skip popup for task=$taskId because platform is not windows');
      return;
    }

    final controller = Get.find<AppController>();
    if (!controller.downloaderConfig.value.extra.browserCapturePopup) {
      await _writeDebugLog(
          'skip popup for task=$taskId because setting is disabled');
      return;
    }

    if (!_launchedTaskIds.add(taskId)) {
      await _writeDebugLog(
          'skip popup for task=$taskId because it was already launched');
      return;
    }

    try {
      final executable = Platform.resolvedExecutable;
      final executableDir = path.dirname(executable);
      final address = controller.startConfig.value.network == 'unix'
          ? controller.startConfig.value.address
          : '${controller.startConfig.value.address.split(':').first}:${controller.runningPort.value}';

      await Process.start(
        executable,
        [
          '--download-popup',
          '--popup-task-id=$taskId',
          '--popup-network=${controller.startConfig.value.network}',
          '--popup-address=$address',
          '--popup-api-token=${controller.startConfig.value.apiToken}',
          '--popup-theme-mode=${controller.downloaderConfig.value.extra.themeMode}',
          '--popup-locale=${controller.downloaderConfig.value.extra.locale}',
        ],
        workingDirectory: executableDir,
        mode: ProcessStartMode.detached,
      );
      await _writeDebugLog(
          'launched popup for task=$taskId exe=$executable cwd=$executableDir address=$address');
    } catch (e, stackTrace) {
      _launchedTaskIds.remove(taskId);
      await _writeDebugLog('launch popup failed for task=$taskId error=$e');
      logger.w('launch browser download popup fail', e, stackTrace);
    }
  }
}

class BrowserDownloadPopupApp extends StatelessWidget {
  final String taskId;
  final String themeMode;
  final String localeKey;

  const BrowserDownloadPopupApp({
    super.key,
    required this.taskId,
    required this.themeMode,
    required this.localeKey,
  });

  @override
  Widget build(BuildContext context) {
    final resolvedThemeMode = ThemeMode.values.firstWhere(
      (item) => item.name == themeMode,
      orElse: () => ThemeMode.system,
    );
    final resolvedLocale =
        localeKey.contains('_') ? toLocale(localeKey) : fallbackLocale;
    return GetMaterialApp(
      debugShowCheckedModeBanner: false,
      theme: GopeedTheme.light,
      darkTheme: GopeedTheme.dark,
      themeMode: resolvedThemeMode,
      translations: messages,
      locale: resolvedLocale,
      fallbackLocale: fallbackLocale,
      localizationsDelegates: const [
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: messages.keys.keys.map((e) => toLocale(e)).toList(),
      home: BrowserDownloadPopupPage(taskId: taskId),
    );
  }
}

class BrowserDownloadPopupPage extends StatefulWidget {
  final String taskId;

  const BrowserDownloadPopupPage({
    super.key,
    required this.taskId,
  });

  @override
  State<BrowserDownloadPopupPage> createState() =>
      _BrowserDownloadPopupPageState();
}

class _BrowserDownloadPopupPageState extends State<BrowserDownloadPopupPage> {
  Timer? _timer;
  Task? _task;
  bool _loading = true;
  bool _taskMissing = false;
  bool _doneLayout = false;

  @override
  void initState() {
    super.initState();
    appendPopupDebugLog('popup process init task=${widget.taskId}');
    _configureWindow();
    _refresh();
    _timer = Timer.periodic(const Duration(milliseconds: 800), (_) {
      _refresh();
    });
  }

  Future<void> _configureWindow() async {
    final windowSize = const Size(_popupWindowWidth, _popupWindowHeight);
    await windowManager.waitUntilReadyToShow(
      WindowOptions(
        size: windowSize,
        minimumSize: windowSize,
        maximumSize: windowSize,
        center: true,
        skipTaskbar: false,
      ),
      () async {
        await windowManager.setAlwaysOnTop(true);
        await windowManager.setResizable(false);
        await windowManager.setTitle(_windowTitle(false));
        await windowManager.show();
        await windowManager.focus();
        await appendPopupDebugLog('popup window shown task=${widget.taskId}');
      },
    );
  }

  Future<void> _applyWindowLayout(bool done) async {
    if (_doneLayout == done) {
      return;
    }
    _doneLayout = done;
    final size = done
        ? const Size(_popupDoneWindowWidth, _popupDoneWindowHeight)
        : const Size(_popupWindowWidth, _popupWindowHeight);
    await windowManager.setMinimumSize(size);
    await windowManager.setMaximumSize(size);
    await windowManager.setSize(size);
    await windowManager.center();
    await windowManager.setTitle(_windowTitle(done));
  }

  Future<void> _refresh() async {
    try {
      final task = await api.getTask(widget.taskId);
      if (!mounted) {
        return;
      }
      setState(() {
        _task = task;
        _taskMissing = false;
        _loading = false;
      });
      await appendPopupDebugLog(
          'popup refresh ok task=${widget.taskId} status=${task.status.name}');
      await _applyWindowLayout(task.status == Status.done);
    } catch (e) {
      if (!mounted) {
        return;
      }
      setState(() {
        _loading = false;
        _taskMissing = true;
      });
      await appendPopupDebugLog(
          'popup refresh failed task=${widget.taskId} error=$e');
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  double _progress(Task task) {
    final total = task.meta.res?.size ?? 0;
    if (total <= 0) {
      return 0;
    }
    return (task.progress.downloaded / total).clamp(0, 1);
  }

  String _windowTitle(bool done) {
    return done ? 'popupCompleted'.tr : 'downloading'.tr;
  }

  String _statusText(Task task) {
    switch (task.status) {
      case Status.ready:
      case Status.wait:
        return 'waitingParts'.tr;
      case Status.running:
        return 'downloading'.tr;
      case Status.pause:
        return 'pause'.tr;
      case Status.error:
        return 'notificationTaskError'.tr;
      case Status.done:
        return 'popupCompleted'.tr;
    }
  }

  String _downloadedText(Task task) {
    final total = task.meta.res?.size ?? 0;
    final downloaded = Util.fmtByte(task.progress.downloaded);
    if (total <= 0) {
      return downloaded;
    }
    return '$downloaded (${(_progress(task) * 100).toStringAsFixed(2)} %)';
  }

  String _remainingText(Task task) {
    if (task.status != Status.running) {
      return '--';
    }
    final total = task.meta.res?.size ?? 0;
    final speed = task.progress.speed;
    if (total <= 0 || speed <= 0) {
      return '--';
    }
    final remainingBytes = total - task.progress.downloaded;
    if (remainingBytes <= 0) {
      return '0s';
    }
    final seconds = (remainingBytes + speed - 1) ~/ speed;
    if (seconds >= 3600) {
      final hours = seconds ~/ 3600;
      final minutes = (seconds % 3600) ~/ 60;
      return '${hours}h ${minutes}m';
    }
    if (seconds >= 60) {
      final minutes = seconds ~/ 60;
      final remainSeconds = seconds % 60;
      return '${minutes}m ${remainSeconds}s';
    }
    return '${seconds}s';
  }

  bool _recoverable(Task task) {
    return task.status != Status.done && task.progress.downloaded > 0;
  }

  String _taskPath(Task task) {
    return path.join(
        Util.safeDir(task.meta.opts.path), Util.safeDir(task.name));
  }

  Future<void> _openFolder(Task task) async {
    await FileExplorer.openAndSelectFile(_taskPath(task));
  }

  Future<void> _open(Task task) async {
    final target = _taskPath(task);
    if (task.meta.res?.name.isNotEmpty == true) {
      await _openFolder(task);
      return;
    }
    await OpenFilex.open(target);
  }

  Future<void> _pause(Task task) async {
    await api.pauseTask(task.id);
    await _refresh();
  }

  Future<void> _continue(Task task) async {
    await api.continueTask(task.id);
    await _refresh();
  }

  Future<void> _cancel(Task task) async {
    await api.deleteTask(task.id, true);
    if (mounted) {
      await windowManager.close();
    }
  }

  Widget _buildInfoRow(String label, String value,
      {Color? valueColor, bool monospace = false}) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 104,
            child: Text(
              label,
              style: TextStyle(
                fontSize: 15,
                color: theme.colorScheme.onSurface,
              ),
            ),
          ),
          Expanded(
            child: Text(
              value,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                fontSize: 15,
                color: valueColor ?? theme.colorScheme.onSurface,
                fontFamily: monospace ? 'Consolas' : null,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildLinearProgress(Task task) {
    final progress = _progress(task);
    final theme = Theme.of(context);
    return LayoutBuilder(
      builder: (context, constraints) => Container(
        height: 22,
        decoration: BoxDecoration(
          border: Border.all(color: theme.dividerColor),
          color: theme.colorScheme.surfaceContainerHighest,
        ),
        child: Align(
          alignment: Alignment.centerLeft,
          child: SizedBox(
            width: constraints.maxWidth * progress,
            child: ColoredBox(
              color: theme.colorScheme.primary,
              child: const SizedBox.expand(),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildDownloading(Task task) {
    final fileExt = task.name.contains('.')
        ? task.name.split('.').last.toUpperCase()
        : 'FILE';
    final theme = Theme.of(context);
    final textColor = theme.colorScheme.onSurface;
    return Padding(
      key: const ValueKey('downloading'),
      padding: const EdgeInsets.fromLTRB(16, 14, 16, 14),
      child: Column(
        children: [
          Container(
            width: double.infinity,
            padding: const EdgeInsets.fromLTRB(14, 14, 14, 12),
            decoration: BoxDecoration(
              border: Border.all(color: theme.dividerColor),
              color: theme.cardColor,
            ),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        task.name,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          fontSize: 16,
                          color: theme.colorScheme.primary,
                        ),
                      ),
                      const SizedBox(height: 14),
                      _buildInfoRow('taskStatus'.tr, _statusText(task),
                          valueColor: theme.colorScheme.primary),
                      _buildInfoRow(
                          'size'.tr, Util.fmtByte(task.meta.res?.size ?? 0),
                          monospace: true),
                      _buildInfoRow('downloaded'.tr, _downloadedText(task),
                          monospace: true),
                      _buildInfoRow(
                        'popupBandwidth'.tr,
                        '${Util.fmtByte(task.progress.speed)}/s',
                        monospace: true,
                      ),
                      _buildInfoRow(
                          'popupRemainingTime'.tr, _remainingText(task)),
                      _buildInfoRow(
                        'popupRecoverable'.tr,
                        _recoverable(task) ? 'yes'.tr : 'no'.tr,
                        valueColor: _recoverable(task)
                            ? theme.colorScheme.primary
                            : textColor.withValues(alpha: 0.6),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 12),
                Container(
                  width: 92,
                  height: 92,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: theme.colorScheme.surfaceContainerHighest,
                    shape: BoxShape.circle,
                  ),
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(
                        fileIcon(task.name,
                            isFolder: task.meta.res?.name.isNotEmpty == true),
                        size: 48,
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                      Container(
                        margin: const EdgeInsets.only(top: 4),
                        padding: const EdgeInsets.symmetric(
                            horizontal: 8, vertical: 2),
                        decoration: BoxDecoration(
                          color: theme.colorScheme.onSurface,
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: Text(
                          fileExt,
                          style: const TextStyle(
                            color: Colors.white,
                            fontSize: 12,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),
          _buildLinearProgress(task),
          const Spacer(),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              _PopupActionButton(
                label: task.status == Status.pause ? 'continue'.tr : 'pause'.tr,
                onPressed: task.status == Status.running
                    ? () => _pause(task)
                    : task.status == Status.pause
                        ? () => _continue(task)
                        : null,
              ),
              const SizedBox(width: 48),
              _PopupActionButton(
                label: 'cancel'.tr,
                onPressed: () => _cancel(task),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildDone(Task task) {
    final theme = Theme.of(context);
    return Padding(
      key: const ValueKey('done'),
      padding: const EdgeInsets.fromLTRB(20, 18, 20, 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 58,
                height: 58,
                decoration: BoxDecoration(
                  color: theme.colorScheme.surfaceContainerHighest,
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.insert_drive_file_rounded,
                  color: theme.colorScheme.onSurfaceVariant,
                  size: 34,
                ),
              ),
              const SizedBox(width: 18),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      task.name,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 18),
                    Text(
                      Util.fmtByte(
                          task.meta.res?.size ?? task.progress.downloaded),
                      style: const TextStyle(fontSize: 16),
                    ),
                  ],
                ),
              ),
            ],
          ),
          const Spacer(),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceEvenly,
            children: [
              _PopupActionButton(
                label: 'popupOpenFolder'.tr,
                width: 142,
                onPressed: () => _openFolder(task),
              ),
              _PopupActionButton(
                label: 'popupOpen'.tr,
                width: 142,
                onPressed: () => _open(task),
              ),
              _PopupActionButton(
                label: 'popupClose'.tr,
                width: 142,
                onPressed: () => windowManager.close(),
              ),
            ],
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final task = _task;
    return Scaffold(
      backgroundColor: Theme.of(context).scaffoldBackgroundColor,
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _taskMissing || task == null
              ? Center(
                  child: Text(
                    'unknown'.tr,
                    style: const TextStyle(fontSize: 16),
                  ),
                )
              : task.status == Status.done
                  ? _buildDone(task)
                  : _buildDownloading(task),
    );
  }
}

class _PopupActionButton extends StatelessWidget {
  final String label;
  final VoidCallback? onPressed;
  final double width;

  const _PopupActionButton({
    required this.label,
    required this.onPressed,
    this.width = 128,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return SizedBox(
      width: width,
      height: 40,
      child: OutlinedButton(
        onPressed: onPressed,
        style: OutlinedButton.styleFrom(
          foregroundColor: theme.colorScheme.onSurface,
          backgroundColor: theme.colorScheme.surface,
          side: BorderSide(
            color: onPressed == null
                ? theme.dividerColor.withValues(alpha: 0.6)
                : theme.dividerColor,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(4),
          ),
        ),
        child: Text(label),
      ),
    );
  }
}
