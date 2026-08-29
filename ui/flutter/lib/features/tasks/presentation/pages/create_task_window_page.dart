import 'dart:async';
import 'dart:convert';

import 'package:desktop_drop/desktop_drop.dart';
import 'package:desktop_multi_window/desktop_multi_window.dart' as multi_window;
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart'
    show Clipboard, FilteringTextInputFormatter, TextInputAction, TextInputType, TextInputFormatter;
import 'package:flutter/widgets.dart' as flutter show Flexible, Row;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:path/path.dart' as path;
import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:window_manager/window_manager.dart';

import '../../../../api/model/create_task.dart';
import '../../../../api/model/create_task_batch.dart';
import '../../../../api/model/downloader_config.dart';
import '../../../../api/model/options.dart' as api_options;
import '../../../../api/model/request.dart';
import '../../../../api/model/resolve_result.dart';
import '../../../../api/model/resolve_task.dart';
import '../../../../core/capabilities/app_capabilities.dart';
import '../../../../core/window/app_window_chrome.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_choice_segmented_control.dart';
import '../../../../shared/widgets/app_http_headers_editor.dart';
import '../../../../shared/widgets/app_path_picker_field.dart';
import '../../../../shared/widgets/app_primary_button.dart';
import '../../../../shared/widgets/app_tooltip.dart';
import '../../../../shared/widgets/app_toast.dart';
import '../../../../l10n/l10n.dart';
import '../../../../util/util.dart';
import '../../application/pending_create_task.dart';
import '../widgets/resolve_file_tree.dart';

class CreateTaskWindowPage extends ConsumerStatefulWidget {
  const CreateTaskWindowPage({super.key, this.windowController, this.initialTask});

  final multi_window.WindowController? windowController;
  final CreateTask? initialTask;

  @override
  ConsumerState<CreateTaskWindowPage> createState() => _CreateTaskWindowPageState();
}

class _CreateTaskWindowPageState extends ConsumerState<CreateTaskWindowPage> {
  final _urlController = TextEditingController();
  final _renameController = TextEditingController();
  final _connectionsController = TextEditingController(text: '16');
  final _directoryController = TextEditingController();
  final _proxyServerController = TextEditingController();
  final _proxyPortController = TextEditingController();
  final _proxyUsernameController = TextEditingController();
  final _proxyPasswordController = TextEditingController();
  final _httpHeadersController = AppHttpHeadersController(defaultNames: const ['User-Agent', 'Cookie', 'Referer']);
  final _trackersController = TextEditingController();
  final _archivePasswordController = TextEditingController();
  final _formScrollController = ScrollController();

  String _httpMethod = 'GET';
  String _httpBody = '';
  String? _initialRawUrl;
  Map<String, String>? _initialLabels;
  RequestProxyMode _proxyMode = RequestProxyMode.follow;
  String _proxyScheme = 'http';
  bool _skipVerifyCert = false;
  bool? _autoTorrent;
  bool? _deleteTorrentAfterDownload;
  bool? _autoExtract;
  bool _deleteAfterExtract = false;
  List<DownloadCategory> _downloadCategories = const [];

  bool _showAdvanced = false;
  int _advancedScrollRequest = 0;
  bool _directDownload = false;
  int _protocolTab = 0;
  bool _creating = false;
  bool _draggingTorrent = false;
  bool _asDefaultPath = false;
  String _configuredDownloadDirectory = '';
  String _fileDataUri = '';
  bool _programmaticUrlChange = false;

  @override
  void initState() {
    super.initState();
    _urlController.addListener(_handleUrlChanged);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final task = widget.initialTask ?? ref.read(pendingCreateTaskProvider);
      _applyInitialTask(task);
      if (widget.initialTask == null) {
        ref.read(pendingCreateTaskProvider.notifier).clear();
      }
      if (task == null || _urlController.text.trim().isEmpty) {
        unawaited(_loadClipboardUrl());
      }
    });
    unawaited(_loadDefaults());
  }

  @override
  void dispose() {
    _urlController.removeListener(_handleUrlChanged);
    _urlController.dispose();
    _renameController.dispose();
    _connectionsController.dispose();
    _directoryController.dispose();
    _proxyServerController.dispose();
    _proxyPortController.dispose();
    _proxyUsernameController.dispose();
    _proxyPasswordController.dispose();
    _httpHeadersController.dispose();
    _trackersController.dispose();
    _archivePasswordController.dispose();
    _formScrollController.dispose();
    super.dispose();
  }

  void _toggleAdvanced() {
    final expanding = !_showAdvanced;
    final request = ++_advancedScrollRequest;
    setState(() => _showAdvanced = expanding);
    if (expanding) {
      unawaited(_scrollForAdvancedOptions(request));
    } else if (_formScrollController.hasClients) {
      final position = _formScrollController.position;
      position.jumpTo(position.pixels);
    }
  }

  Future<void> _scrollForAdvancedOptions(int request) async {
    await Future<void>.delayed(_advancedScrollDelay);
    for (var attempt = 0; attempt < _advancedScrollExtentChecks; attempt++) {
      await WidgetsBinding.instance.endOfFrame;
      if (!mounted || !_showAdvanced || request != _advancedScrollRequest || !_formScrollController.hasClients) {
        return;
      }
      final position = _formScrollController.position;
      final target = (position.pixels + _advancedScrollDistance).clamp(
        position.minScrollExtent,
        position.maxScrollExtent,
      );
      if (target - position.pixels >= 1) {
        try {
          await _formScrollController.animateTo(target, duration: _advancedScrollDuration, curve: Curves.easeOutCubic);
        } catch (_) {
          // The window or scroll position can be disposed while the animation is running.
        }
        return;
      }
      if (attempt + 1 < _advancedScrollExtentChecks) {
        await Future<void>.delayed(_advancedScrollExtentRetryDelay);
      }
    }
  }

  void _handleUrlChanged() {
    if (!_programmaticUrlChange && _fileDataUri.isNotEmpty) {
      _fileDataUri = '';
    }
    _recognizeMagnetUri(_urlController.text.trim());
  }

  void _applyInitialTask(CreateTask? task) {
    if (task == null || !mounted) return;
    final req = task.req;
    final opts = task.opts;
    setState(() {
      if (req?.url.isNotEmpty == true) {
        _urlController.text = req!.url;
        _initialRawUrl = req.rawUrl;
        _initialLabels = req.labels == null ? null : Map<String, String>.of(req.labels!);
        _applyInitialProxy(req.proxy);
        _skipVerifyCert = req.skipVerifyCert;
        if (req.proxy != null || req.skipVerifyCert) {
          _showAdvanced = true;
        }
        switch (_parseProtocol(req.url)) {
          case _TaskProtocol.http:
            if (req.extra is Map) {
              final extra = ReqExtraHttp.fromJson(Map<String, dynamic>.from(req.extra! as Map));
              _httpMethod = extra.method;
              _httpBody = extra.body;
              if (extra.header.isNotEmpty) {
                _replaceHttpHeaders(extra.header);
              }
              _showAdvanced = true;
            }
            break;
          case _TaskProtocol.bt:
            _protocolTab = 1;
            if (req.extra is Map) {
              final extra = ReqExtraBt.fromJson(Map<String, dynamic>.from(req.extra! as Map));
              _trackersController.text = extra.trackers.join('\n');
              _showAdvanced = true;
            }
            break;
          case _TaskProtocol.ed2k:
          case null:
            break;
        }
      }
      if (opts != null) {
        if (opts.name.isNotEmpty) {
          _renameController.text = opts.name;
        }
        if (opts.path.isNotEmpty) {
          _directoryController.text = opts.path;
        }
        if (opts.extra is Map) {
          final extra = api_options.OptsExtraHttp.fromJson(Map<String, dynamic>.from(opts.extra! as Map));
          if (extra.connections > 0) {
            _connectionsController.text = extra.connections.toString();
          }
          _autoTorrent = extra.autoTorrent;
          _deleteTorrentAfterDownload = extra.deleteTorrentAfterDownload;
          _autoExtract = extra.autoExtract;
          _archivePasswordController.text = extra.archivePassword;
          _deleteAfterExtract = extra.deleteAfterExtract;
          if (extra.autoTorrent != null ||
              extra.deleteTorrentAfterDownload != null ||
              extra.autoExtract != null ||
              extra.archivePassword.isNotEmpty ||
              extra.deleteAfterExtract) {
            _showAdvanced = true;
          }
        }
      }
    });
  }

  void _applyInitialProxy(RequestProxy? proxy) {
    _proxyMode = proxy?.mode ?? RequestProxyMode.follow;
    const supportedSchemes = {'http', 'https', 'socks5'};
    _proxyScheme = supportedSchemes.contains(proxy?.scheme) ? proxy!.scheme : 'http';
    _proxyUsernameController.text = proxy?.usr ?? '';
    _proxyPasswordController.text = proxy?.pwd ?? '';

    final host = proxy?.host.trim() ?? '';
    if (host.isEmpty) {
      _proxyServerController.clear();
      _proxyPortController.clear();
      return;
    }
    final parsed = Uri.tryParse('$_proxyScheme://$host');
    if (parsed != null && parsed.host.isNotEmpty) {
      _proxyServerController.text = parsed.host;
      _proxyPortController.text = parsed.hasPort ? parsed.port.toString() : '';
      return;
    }
    _proxyServerController.text = host;
    _proxyPortController.clear();
  }

  void _replaceHttpHeaders(Map<String, String> headers) {
    _httpHeadersController.replace(headers);
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);

    final page = Scaffold(
      backgroundColor: palette.sideBg,
      child: Padding(
        padding: EdgeInsets.only(top: AppWindowChrome.reservesHeaderInset ? AppDesignTokens.windowHeaderHeight : 0),
        child: Column(
          children: [
            Container(
              height: 44,
              padding: const EdgeInsets.symmetric(horizontal: 24),
              child: Row(
                children: [
                  Text(
                    context.l10n.create,
                    style: TextStyle(color: palette.textPrimary, fontSize: 18, fontWeight: FontWeight.w700),
                  ),
                ],
              ),
            ),
            Expanded(
              child: SingleChildScrollView(
                key: const ValueKey('create-task-form-scroll'),
                controller: _formScrollController,
                padding: const EdgeInsets.fromLTRB(24, 8, 24, 20),
                child: Column(
                  children: [
                    _FormRow(
                      label: context.l10n.downloadLink,
                      child: DropTarget(
                        enable: !_creating,
                        onDragEntered: (_) => _setDraggingTorrent(true),
                        onDragExited: (_) => _setDraggingTorrent(false),
                        onDragDone: (details) {
                          _setDraggingTorrent(false);
                          unawaited(_handleDroppedTorrent(details));
                        },
                        child: AnimatedContainer(
                          duration: const Duration(milliseconds: 120),
                          decoration: BoxDecoration(
                            borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
                            border: _draggingTorrent ? Border.all(color: palette.brand, width: 2) : null,
                            color: _draggingTorrent ? palette.filterActiveBg : null,
                          ),
                          child: Stack(
                            children: [
                              SizedBox(
                                height: 120,
                                child: TextField(
                                  controller: _urlController,
                                  hintText: context.l10n.pasteDownloadLinks,
                                  keyboardType: TextInputType.multiline,
                                  textInputAction: TextInputAction.newline,
                                  maxLines: null,
                                  minLines: null,
                                  expands: true,
                                  textAlignVertical: TextAlignVertical.top,
                                  filled: true,
                                  border: Border.all(
                                    color: _draggingTorrent ? palette.brand : palette.border,
                                    width: _draggingTorrent ? 0 : 1,
                                  ),
                                  borderRadius: BorderRadius.circular(4),
                                ),
                              ),
                              Positioned(
                                right: 8,
                                bottom: 8,
                                child: Row(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    _MiniInputAction(icon: Icons.folder_open_outlined, onPressed: _pickTorrentFile),
                                    const SizedBox(width: 6),
                                    _MiniInputAction(icon: Icons.history, onPressed: _showCreateHistory),
                                  ],
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                    const SizedBox(height: 16),
                    _FormRow(
                      label: context.l10n.rename,
                      child: _WindowTextField(
                        key: const ValueKey('create-task-rename-input'),
                        controller: _renameController,
                        hintText: context.l10n.keepOriginalName,
                      ),
                    ),
                    const SizedBox(height: 16),
                    _FormRow(
                      label: context.l10n.connections,
                      child: _WindowTextField(controller: _connectionsController, hintText: context.l10n.enterCount),
                    ),
                    const SizedBox(height: 16),
                    _FormRow(
                      label: context.l10n.directory,
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          AppPathPickerField.downloadDirectory(
                            key: const ValueKey('create-task-directory-component'),
                            fieldKey: const ValueKey('create-task-directory-input'),
                            pickerKey: const ValueKey('create-task-directory-picker'),
                            controller: _directoryController,
                            hintText: context.l10n.chooseDownloadDirectory,
                            filled: true,
                            border: Border.all(color: palette.border),
                            borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
                            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                            pickerButtonStyle: AppPathPickerButtonStyle.outline,
                            onChanged: (_) {
                              if (_asDefaultPath) {
                                setState(() => _asDefaultPath = false);
                              }
                            },
                            onDirectoryPicked: (pickedDirectory) => setState(
                              () => _asDefaultPath = !_sameDirectory(pickedDirectory, _configuredDownloadDirectory),
                            ),
                          ),
                          const SizedBox(height: 8),
                          SizedBox(
                            key: const ValueKey('create-task-directory-options-row'),
                            width: double.infinity,
                            height: 30,
                            child: flutter.Row(
                              key: const ValueKey('create-task-directory-options-layout'),
                              mainAxisAlignment: MainAxisAlignment.spaceBetween,
                              children: [
                                _AsDefaultPathToggle(
                                  value: _asDefaultPath,
                                  onChanged: (value) => setState(() => _asDefaultPath = value),
                                ),
                                if (_downloadCategories.isNotEmpty)
                                  flutter.Flexible(
                                    child: Padding(
                                      padding: const EdgeInsetsDirectional.only(start: 16),
                                      child: _DirectoryShortcuts(
                                        shortcuts: [
                                          for (final entry in _downloadCategories.indexed)
                                            _DirectoryShortcut(
                                              index: entry.$1,
                                              label: _categoryDisplayName(context, entry.$2),
                                              path: _renderPathPlaceholders(entry.$2.path),
                                            ),
                                        ],
                                        onSelected: (shortcut) {
                                          _directoryController.text = shortcut.path;
                                          if (_asDefaultPath) {
                                            setState(() => _asDefaultPath = false);
                                          }
                                        },
                                      ),
                                    ),
                                  ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 14),
                    _FormRow(
                      label: context.l10n.directDownload,
                      child: _DirectDownloadToggle(
                        value: _directDownload,
                        onChanged: (value) => setState(() => _directDownload = value),
                      ),
                    ),
                    const SizedBox(height: 20),
                    Align(
                      alignment: Alignment.centerLeft,
                      child: GestureDetector(
                        onTap: _toggleAdvanced,
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Text(
                              context.l10n.advanced,
                              style: TextStyle(color: palette.textPrimary, fontSize: 14, fontWeight: FontWeight.w600),
                            ),
                            const SizedBox(width: 8),
                            _TripleChevronIcon(expanded: _showAdvanced, color: palette.textSecondary),
                          ],
                        ),
                      ),
                    ),
                    AnimatedCrossFade(
                      duration: _advancedExpandDuration,
                      firstCurve: Curves.easeOutCubic,
                      secondCurve: Curves.easeInCubic,
                      sizeCurve: Curves.easeOutCubic,
                      crossFadeState: _showAdvanced ? CrossFadeState.showFirst : CrossFadeState.showSecond,
                      firstChild: Padding(
                        key: const ValueKey('create-task-advanced-content'),
                        padding: const EdgeInsets.only(top: 18, bottom: 4),
                        child: Column(
                          children: [
                            _FormRow(
                              label: context.l10n.proxy,
                              child: AppChoiceSegmentedControl<RequestProxyMode>(
                                key: const ValueKey('create-task-proxy-mode'),
                                value: _proxyMode,
                                buttonKeyPrefix: 'create-task-proxy-mode',
                                options: [
                                  AppChoiceOption(
                                    value: RequestProxyMode.follow,
                                    key: 'follow',
                                    label: context.l10n.followSettings,
                                    icon: Icons.settings_suggest_outlined,
                                  ),
                                  AppChoiceOption(
                                    value: RequestProxyMode.none,
                                    key: 'none',
                                    label: context.l10n.noProxy,
                                    icon: Icons.block_outlined,
                                  ),
                                  AppChoiceOption(
                                    value: RequestProxyMode.custom,
                                    key: 'custom',
                                    label: context.l10n.customProxy,
                                    icon: Icons.tune_outlined,
                                  ),
                                ],
                                onChanged: (value) => setState(() => _proxyMode = value),
                              ),
                            ),
                            if (_proxyMode == RequestProxyMode.custom) ...[
                              const SizedBox(height: 12),
                              _FormRow(
                                label: context.l10n.proxyProtocol,
                                child: AppChoiceSegmentedControl<String>(
                                  key: const ValueKey('create-task-proxy-scheme'),
                                  value: _proxyScheme,
                                  buttonKeyPrefix: 'create-task-proxy-scheme',
                                  options: const [
                                    AppChoiceOption(value: 'http', label: 'HTTP', icon: Icons.http_outlined),
                                    AppChoiceOption(value: 'https', label: 'HTTPS', icon: Icons.https_outlined),
                                    AppChoiceOption(value: 'socks5', label: 'SOCKS5', icon: Icons.security_outlined),
                                  ],
                                  onChanged: (value) => setState(() => _proxyScheme = value),
                                ),
                              ),
                              const SizedBox(height: 12),
                              _FormRow(
                                label: context.l10n.server,
                                child: Row(
                                  children: [
                                    Expanded(
                                      flex: 618,
                                      child: _WindowTextField(
                                        key: const ValueKey('create-task-proxy-server'),
                                        controller: _proxyServerController,
                                        hintText: context.l10n.server,
                                      ),
                                    ),
                                    const SizedBox(width: 8),
                                    Expanded(
                                      flex: 382,
                                      child: _WindowTextField(
                                        key: const ValueKey('create-task-proxy-port'),
                                        controller: _proxyPortController,
                                        hintText: context.l10n.port,
                                        keyboardType: TextInputType.number,
                                        inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                              const SizedBox(height: 12),
                              _FormRow(
                                label: context.l10n.username,
                                child: Row(
                                  children: [
                                    Expanded(
                                      child: _WindowTextField(
                                        key: const ValueKey('create-task-proxy-username'),
                                        controller: _proxyUsernameController,
                                        hintText: context.l10n.username,
                                      ),
                                    ),
                                    const SizedBox(width: 8),
                                    Expanded(
                                      child: _WindowTextField(
                                        key: const ValueKey('create-task-proxy-password'),
                                        controller: _proxyPasswordController,
                                        hintText: context.l10n.password,
                                        obscureText: true,
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            ],
                            const SizedBox(height: 18),
                            Container(height: 1, color: palette.border),
                            const SizedBox(height: 16),
                            Align(
                              alignment: Alignment.centerLeft,
                              child: Container(
                                padding: const EdgeInsets.all(3),
                                decoration: BoxDecoration(
                                  color: palette.surfaceSoft,
                                  borderRadius: BorderRadius.circular(999),
                                  border: Border.all(color: palette.border),
                                ),
                                child: Row(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    _SegmentButton(
                                      label: 'HTTP',
                                      active: _protocolTab == 0,
                                      onTap: () => setState(() => _protocolTab = 0),
                                    ),
                                    _SegmentButton(
                                      label: 'BitTorrent',
                                      active: _protocolTab == 1,
                                      onTap: () => setState(() => _protocolTab = 1),
                                    ),
                                  ],
                                ),
                              ),
                            ),
                            const SizedBox(height: 16),
                            if (_protocolTab == 0) ...[
                              AppHttpHeadersEditor(
                                controller: _httpHeadersController,
                                label: context.l10n.httpHeader,
                                keyPrefix: 'create-task-http-header',
                              ),
                              const SizedBox(height: 14),
                              _FormRow(
                                label: context.l10n.skipVerifyCert,
                                child: _OptionSwitch(
                                  key: const ValueKey('create-task-skip-verify-cert'),
                                  value: _skipVerifyCert,
                                  onChanged: (value) => setState(() => _skipVerifyCert = value),
                                ),
                              ),
                              const SizedBox(height: 14),
                              _FormRow(
                                label: context.l10n.autoTorrentEnable,
                                child: _OptionSwitch(
                                  key: const ValueKey('create-task-auto-torrent'),
                                  value: _autoTorrent ?? false,
                                  onChanged: (value) => setState(() => _autoTorrent = value),
                                ),
                              ),
                              if (_autoTorrent == true) ...[
                                const SizedBox(height: 12),
                                _FormRow(
                                  label: context.l10n.autoTorrentDeleteAfterDownload,
                                  child: _OptionSwitch(
                                    key: const ValueKey('create-task-delete-torrent'),
                                    value: _deleteTorrentAfterDownload ?? false,
                                    onChanged: (value) => setState(() => _deleteTorrentAfterDownload = value),
                                  ),
                                ),
                              ],
                              const SizedBox(height: 14),
                              _FormRow(
                                label: context.l10n.autoExtract,
                                child: _OptionSwitch(
                                  key: const ValueKey('create-task-auto-extract'),
                                  value: _autoExtract ?? false,
                                  onChanged: (value) => setState(() => _autoExtract = value),
                                ),
                              ),
                              if (_autoExtract == true) ...[
                                const SizedBox(height: 12),
                                _FormRow(
                                  label: context.l10n.archivePassword,
                                  child: _WindowTextField(
                                    key: const ValueKey('create-task-archive-password'),
                                    controller: _archivePasswordController,
                                    hintText: context.l10n.archivePasswordHint,
                                    obscureText: true,
                                  ),
                                ),
                                const SizedBox(height: 12),
                                _FormRow(
                                  label: context.l10n.deleteAfterExtract,
                                  child: _OptionSwitch(
                                    key: const ValueKey('create-task-delete-after-extract'),
                                    value: _deleteAfterExtract,
                                    onChanged: (value) => setState(() => _deleteAfterExtract = value),
                                  ),
                                ),
                              ],
                            ] else ...[
                              _FormRow(
                                label: context.l10n.trackers,
                                child: SizedBox(
                                  height: 96,
                                  child: TextField(
                                    controller: _trackersController,
                                    hintText: context.l10n.oneTrackerPerLine,
                                    keyboardType: TextInputType.multiline,
                                    textInputAction: TextInputAction.newline,
                                    maxLines: null,
                                    minLines: null,
                                    expands: true,
                                    textAlignVertical: TextAlignVertical.top,
                                    filled: true,
                                    border: Border.all(color: palette.border),
                                    borderRadius: BorderRadius.circular(4),
                                  ),
                                ),
                              ),
                            ],
                          ],
                        ),
                      ),
                      secondChild: const SizedBox.shrink(),
                    ),
                  ],
                ),
              ),
            ),
            Container(
              padding: const EdgeInsets.fromLTRB(24, 12, 24, 20),
              decoration: BoxDecoration(
                border: Border(top: BorderSide(color: palette.border)),
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  SecondaryButton(
                    onPressed: _creating ? null : _closeWindow,
                    child: SizedBox(width: 68, child: Center(child: Text(context.l10n.cancel))),
                  ),
                  const SizedBox(width: 12),
                  AppPrimaryButton(
                    onPressed: _creating ? null : _confirm,
                    child: SizedBox(
                      width: 68,
                      child: Center(
                        child: _creating
                            ? const SizedBox.square(dimension: 14, child: CircularProgressIndicator())
                            : Text(context.l10n.confirm),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );

    if (!kIsWeb && defaultTargetPlatform == TargetPlatform.windows) {
      return DragToResizeArea(resizeEdgeSize: 6, child: page);
    }

    return page;
  }

  Future<void> _closeWindow() async {
    if (widget.windowController == null && mounted) {
      context.go('/');
      return;
    }
    await windowManager.close();
  }

  Future<void> _loadDefaults() async {
    try {
      final config = await ref.read(gopeedServiceProvider).getConfig();
      if (!mounted) return;
      setState(() {
        _configuredDownloadDirectory = config.downloadDir.trim();
        if (config.downloadDir.isNotEmpty &&
            (_directoryController.text.isEmpty || _directoryController.text == 'C:/Users/levi/Downloads')) {
          _directoryController.text = config.downloadDir;
        }
        if (_asDefaultPath && _sameDirectory(_directoryController.text, _configuredDownloadDirectory)) {
          _asDefaultPath = false;
        }
        final connections = config.protocolConfig.http.connections;
        if (connections > 0 && (_connectionsController.text.isEmpty || _connectionsController.text == '16')) {
          _connectionsController.text = connections.toString();
        }
        _directDownload = config.extra.defaultDirectDownload;
        _downloadCategories = config.extra.downloadCategories
            .where((category) => !category.isDeleted)
            .toList(growable: false);
      });
    } catch (_) {
      // Keep local defaults when the backend is not available yet.
    }
  }

  Future<void> _loadClipboardUrl() async {
    try {
      final data = await Clipboard.getData('text/plain');
      final value = data?.text?.trim() ?? '';
      if (!mounted || value.isEmpty || _urlController.text.trim().isNotEmpty) {
        return;
      }
      final protocol = _parseProtocol(value);
      if (protocol != null || _isBtHash(value)) {
        _setUrlText(_normalizeTaskUrl(value));
      }
    } catch (_) {
      // Clipboard access is best-effort and can be denied on some platforms.
    }
  }

  Future<void> _pickTorrentFile() async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: const ['torrent'],
      withData: kIsWeb,
    );
    final file = result?.files.single;
    if (file == null) return;

    await _applyTorrentFile(name: file.name, filePath: file.path, bytes: file.bytes);
  }

  Future<void> _handleDroppedTorrent(DropDoneDetails details) async {
    if (details.files.isEmpty) {
      return;
    }
    final file = details.files.first;
    try {
      await _applyTorrentFile(name: file.name, filePath: file.path, readBytes: () async => file.readAsBytes());
    } catch (error) {
      if (mounted) {
        _showToast(context.l10n.unableReadTorrent(error.toString()));
      }
    }
  }

  Future<void> _applyTorrentFile({
    required String name,
    String? filePath,
    List<int>? bytes,
    Future<List<int>> Function()? readBytes,
  }) async {
    if (!name.toLowerCase().endsWith('.torrent')) {
      _showToast(context.l10n.torrentOnly);
      return;
    }

    if (kIsWeb) {
      final fileBytes = bytes ?? await readBytes?.call();
      if (!mounted) return;
      if (fileBytes == null || fileBytes.isEmpty) {
        _showToast(context.l10n.torrentDataEmpty);
        return;
      }
      _setUrlText(name, fileDataUri: 'data:application/x-bittorrent;base64,${base64Encode(fileBytes)}');
      return;
    }

    if (filePath == null || filePath.isEmpty) {
      _showToast(context.l10n.torrentPathUnavailable);
      return;
    }
    _setUrlText(filePath);
  }

  Future<void> _confirm() async {
    if (_creating) {
      return;
    }
    final urls = _inputUrls();
    if (urls.isEmpty) {
      _showToast(context.l10n.downloadLinkValid);
      return;
    }
    if (!_validateAdvancedOptions()) {
      return;
    }

    setState(() {
      _creating = true;
      _draggingTorrent = false;
    });
    try {
      await _saveCreateHistory(urls);
      final isDirect = _directDownload || urls.length > 1;
      if (isDirect) {
        await Future.wait(
          urls.map((url) {
            return ref
                .read(gopeedServiceProvider)
                .createTask(
                  CreateTask(
                    req: _buildRequest(url, protocolUrl: _protocolUrlFor(url)),
                    opts: _buildOptions(name: urls.length > 1 ? '' : _renameController.text.trim()),
                  ),
                );
          }),
        );
        await _closeWindow();
        return;
      }

      final request = _buildRequest(urls.first, protocolUrl: _protocolUrlFor(urls.first));
      final options = _buildOptions();
      final result = await ref.read(gopeedServiceProvider).resolve(ResolveTask(req: request, opts: options));
      if (!mounted) return;
      _syncResolvedName(result);
      final created = await _showResolveDialog(request, result);
      if (!created) {
        if (mounted) {
          setState(() => _creating = false);
        }
        return;
      }
      await _closeWindow();
    } catch (error) {
      if (mounted) {
        _showToast(error.toString());
        setState(() => _creating = false);
      }
    }
  }

  List<String> _inputUrls() {
    return Util.textToLines(
      _submitText().trim(),
    ).map((line) => _normalizeTaskUrl(line.trim())).where((line) => line.isNotEmpty).toList();
  }

  String _submitText() {
    return kIsWeb && _fileDataUri.isNotEmpty ? _fileDataUri : _urlController.text.trim();
  }

  String _protocolUrlFor(String url) {
    return kIsWeb && _fileDataUri.isNotEmpty ? _urlController.text.trim() : url;
  }

  Request _buildRequest(String url, {String? protocolUrl}) {
    final protocol = _parseProtocol(protocolUrl ?? url);
    Object? extra;
    final headers = _httpHeadersController.toMap(requireValue: true);
    switch (protocol) {
      case _TaskProtocol.http:
        if (headers.isNotEmpty || _httpMethod != 'GET' || _httpBody.isNotEmpty) {
          extra = ReqExtraHttp(method: _httpMethod, header: headers, body: _httpBody).toJson();
        }
        break;
      case _TaskProtocol.bt:
        final trackers = Util.textToLines(
          _trackersController.text,
        ).map((line) => line.trim()).where((line) => line.isNotEmpty).toList();
        if (trackers.isNotEmpty) {
          extra = ReqExtraBt(trackers: trackers).toJson();
        }
        break;
      case _TaskProtocol.ed2k:
      case null:
        break;
    }
    return Request(
      rawUrl: _initialRawUrl,
      url: url,
      extra: extra,
      labels: _initialLabels,
      proxy: _buildProxy(),
      skipVerifyCert: _skipVerifyCert,
    );
  }

  bool _validateAdvancedOptions() {
    if (_proxyMode != RequestProxyMode.custom) {
      return true;
    }
    if (_proxyServerController.text.trim().isEmpty) {
      _showToast(context.l10n.serverRequired);
      return false;
    }
    final portText = _proxyPortController.text.trim();
    if (portText.isNotEmpty) {
      final port = int.tryParse(portText);
      if (port == null || port < 0 || port > 65535) {
        _showToast(context.l10n.portRangeError);
        return false;
      }
    }
    return true;
  }

  RequestProxy _buildProxy() {
    if (_proxyMode != RequestProxyMode.custom) {
      return RequestProxy(mode: _proxyMode);
    }
    var server = _proxyServerController.text.trim();
    final port = _proxyPortController.text.trim();
    if (server.contains(':') && !server.startsWith('[')) {
      server = '[$server]';
    }
    return RequestProxy(
      mode: RequestProxyMode.custom,
      scheme: _proxyScheme,
      host: port.isEmpty ? server : '$server:$port',
      usr: _proxyUsernameController.text.trim(),
      pwd: _proxyPasswordController.text,
    );
  }

  _TaskProtocol? _parseProtocol(String url) {
    final uppercaseUrl = url.trim().toUpperCase();
    if (uppercaseUrl.startsWith('HTTP:') || uppercaseUrl.startsWith('HTTPS:')) {
      return _TaskProtocol.http;
    }
    if (uppercaseUrl.startsWith('MAGNET:') || uppercaseUrl.endsWith('.TORRENT')) {
      return _TaskProtocol.bt;
    }
    if (uppercaseUrl.startsWith('ED2K:')) {
      return _TaskProtocol.ed2k;
    }
    return null;
  }

  String _normalizeTaskUrl(String url) {
    if (_isBtHash(url)) {
      return 'magnet:?xt=urn:btih:$url';
    }
    return url;
  }

  void _recognizeMagnetUri(String text) {
    if (!_isBtHash(text)) {
      return;
    }
    final uri = _normalizeTaskUrl(text);
    if (_urlController.text == uri) {
      return;
    }
    _setUrlText(uri);
  }

  bool _isBtHash(String text) {
    return text.length == 40 && RegExp(r'^[0-9a-fA-F]+$').hasMatch(text);
  }

  api_options.Options _buildOptions({String? name, List<int> selectFiles = const []}) {
    final connections = int.tryParse(_connectionsController.text.trim()) ?? 0;
    return api_options.Options(
      name: name ?? _renameController.text.trim(),
      path: _directoryController.text.trim(),
      asDefaultPath: _asDefaultPath,
      selectFiles: selectFiles,
      extra: api_options.OptsExtraHttp(
        connections: connections,
        autoTorrent: _autoTorrent,
        deleteTorrentAfterDownload: _deleteTorrentAfterDownload,
        autoExtract: _autoExtract,
        archivePassword: _archivePasswordController.text,
        deleteAfterExtract: _deleteAfterExtract,
      ).toJson(),
    );
  }

  bool _sameDirectory(String left, String right) {
    final normalizedLeft = left.trim();
    final normalizedRight = right.trim();
    if (normalizedLeft.isEmpty || normalizedRight.isEmpty) return false;
    return path.equals(path.normalize(normalizedLeft), path.normalize(normalizedRight));
  }

  Future<void> _submitResolved(Request request, ResolveResult result, List<int> selectedIndexes) async {
    final selected = (selectedIndexes.isEmpty ? result.res.files.asMap().keys.toList() : selectedIndexes.toList())
      ..sort();
    if (result.id.isNotEmpty) {
      await ref
          .read(gopeedServiceProvider)
          .createTask(
            CreateTask(
              rid: result.id,
              opts: _buildOptions(selectFiles: selected),
            ),
          );
      return;
    }

    if (result.res.files.isNotEmpty) {
      final reqs = selected.map((index) {
        final file = result.res.files[index];
        return CreateTaskBatchItem(
          req: file.req ?? request,
          opts: api_options.Options(
            name: file.name,
            path: path.join(_directoryController.text.trim(), result.res.name, file.path),
            extra: _buildOptions().extra,
          ),
        );
      }).toList();
      await ref.read(gopeedServiceProvider).createTaskBatch(CreateTaskBatch(reqs: reqs, opts: _buildOptions()));
      return;
    }

    await ref.read(gopeedServiceProvider).createTask(CreateTask(req: request, opts: _buildOptions()));
  }

  Future<bool> _showResolveDialog(Request request, ResolveResult result) async {
    final initialSelection = List<int>.generate(result.res.files.length, (index) => index);
    var selectedIndexes = List<int>.of(initialSelection);
    var submitting = false;
    String? errorText;

    final dialog = const DialogOverlayHandler().show<bool>(
      context: context,
      alignment: Alignment.center,
      barrierDismissable: !submitting,
      builder: (dialogContext) {
        final palette = AppPalette.of(dialogContext);
        final screenSize = MediaQuery.sizeOf(dialogContext);
        final dialogWidth = screenSize.width < 760 ? screenSize.width - 32 : 720.0;
        final treeHeight = screenSize.height < 700 ? screenSize.height * 0.55 : 440.0;

        return StatefulBuilder(
          builder: (dialogContext, setDialogState) {
            Future<void> submit() async {
              if (result.res.files.isNotEmpty && selectedIndexes.isEmpty) {
                setDialogState(() => errorText = dialogContext.l10n.selectAtLeastOneFile);
                return;
              }
              setDialogState(() {
                submitting = true;
                errorText = null;
              });
              try {
                await _submitResolved(request, result, selectedIndexes);
                if (dialogContext.mounted) {
                  await closeOverlay(dialogContext, true);
                }
              } catch (error) {
                if (!dialogContext.mounted) return;
                setDialogState(() {
                  submitting = false;
                  errorText = error.toString();
                });
              }
            }

            return AlertDialog(
              padding: const EdgeInsets.all(18),
              title: Text(
                result.res.name.trim().isEmpty ? dialogContext.l10n.resolvedFiles : result.res.name,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              content: SizedBox(
                width: dialogWidth,
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      result.res.files.isEmpty
                          ? dialogContext.l10n.resourceHasNoFiles
                          : dialogContext.l10n.resolvedFileCount(result.res.files.length),
                      style: TextStyle(color: palette.textSecondary, fontSize: 12),
                    ),
                    const SizedBox(height: 12),
                    SizedBox(
                      height: treeHeight.clamp(260.0, 460.0),
                      child: result.res.files.isEmpty
                          ? Container(
                              alignment: Alignment.center,
                              decoration: BoxDecoration(
                                color: palette.cardBg,
                                border: Border.all(color: palette.border),
                                borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
                              ),
                              child: Text(
                                dialogContext.l10n.noFiles,
                                style: TextStyle(color: palette.textMuted, fontSize: 13),
                              ),
                            )
                          : ResolveFileTree(
                              files: result.res.files,
                              initialSelection: initialSelection,
                              onSelectionChanged: (values) => selectedIndexes = values,
                            ),
                    ),
                    if (errorText != null) ...[
                      const SizedBox(height: 10),
                      Text(errorText!, style: TextStyle(color: palette.error, fontSize: 12)),
                    ],
                  ],
                ),
              ),
              actions: [
                SecondaryButton(
                  onPressed: submitting ? null : () => closeOverlay(dialogContext, false),
                  child: SizedBox(width: 68, child: Center(child: Text(dialogContext.l10n.cancel))),
                ),
                AppPrimaryButton(
                  onPressed: submitting ? null : submit,
                  child: SizedBox(
                    width: 68,
                    child: Center(
                      child: submitting
                          ? const SizedBox.square(dimension: 14, child: CircularProgressIndicator())
                          : Text(dialogContext.l10n.create),
                    ),
                  ),
                ),
              ],
            );
          },
        );
      },
    );
    final created = await dialog.future;
    return created == true;
  }

  Future<void> _showCreateHistory() async {
    List<String> histories;
    try {
      histories = await ref.read(appStorageServiceProvider).getCreateHistory();
    } catch (_) {
      histories = const [];
    }
    if (!mounted) return;
    final dialog = const DialogOverlayHandler().show<void>(
      context: context,
      alignment: Alignment.center,
      builder: (dialogContext) {
        final palette = AppPalette.of(dialogContext);
        final screenSize = MediaQuery.sizeOf(dialogContext);
        return AlertDialog(
          padding: const EdgeInsets.all(18),
          title: Text(dialogContext.l10n.createHistory),
          content: SizedBox(
            width: screenSize.width < 620 ? screenSize.width - 32 : 560,
            height: screenSize.height < 560 ? screenSize.height * 0.55 : 360,
            child: histories.isEmpty
                ? Center(
                    child: Text(dialogContext.l10n.noHistory, style: TextStyle(color: palette.textMuted, fontSize: 13)),
                  )
                : ListView.separated(
                    itemCount: histories.length,
                    separatorBuilder: (_, _) => Container(height: 1, color: palette.border),
                    itemBuilder: (context, index) {
                      final value = histories[index];
                      return GestureDetector(
                        behavior: HitTestBehavior.opaque,
                        onTap: () {
                          _setUrlText(value);
                          closeOverlay(dialogContext);
                        },
                        child: Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 10),
                          child: Text(
                            value,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(color: palette.textPrimary, fontSize: 12),
                          ),
                        ),
                      );
                    },
                  ),
          ),
          actions: [
            GhostButton(
              density: ButtonDensity.icon,
              onPressed: histories.isEmpty
                  ? null
                  : () async {
                      await ref.read(appStorageServiceProvider).clearCreateHistory();
                      if (dialogContext.mounted) closeOverlay(dialogContext);
                    },
              child: const Icon(Icons.history_toggle_off_rounded, size: 16),
            ),
            SecondaryButton(
              onPressed: () => closeOverlay(dialogContext),
              child: SizedBox(width: 68, child: Center(child: Text(dialogContext.l10n.close))),
            ),
          ],
        );
      },
    );
    await dialog.future;
  }

  void _syncResolvedName(ResolveResult result) {
    if (_renameController.text.trim().isEmpty && result.res.name.trim().isNotEmpty) {
      _renameController.text = result.res.name;
    }
  }

  void _setUrlText(String value, {String fileDataUri = ''}) {
    _programmaticUrlChange = true;
    _urlController.text = value;
    _urlController.selection = TextSelection.fromPosition(TextPosition(offset: value.length));
    _programmaticUrlChange = false;
    if (!mounted) {
      _fileDataUri = fileDataUri;
      return;
    }
    setState(() {
      _fileDataUri = fileDataUri;
      if (fileDataUri.isNotEmpty || _parseProtocol(value) == _TaskProtocol.bt) {
        _protocolTab = 1;
      }
    });
  }

  void _setDraggingTorrent(bool value) {
    if (!mounted || _draggingTorrent == value) {
      return;
    }
    setState(() => _draggingTorrent = value);
  }

  Future<void> _saveCreateHistory(List<String> urls) async {
    if (kIsWeb && _fileDataUri.isNotEmpty) {
      return;
    }
    await ref.read(appStorageServiceProvider).saveCreateHistory(urls);
  }

  String _categoryDisplayName(BuildContext context, DownloadCategory category) {
    return switch (category.nameKey) {
      'categoryMusic' => context.l10n.categoryMusic,
      'categoryVideo' => context.l10n.categoryVideo,
      'categoryDocument' => context.l10n.categoryDocument,
      'categoryProgram' => context.l10n.categoryProgram,
      _ => category.name,
    };
  }

  String _renderPathPlaceholders(String value) {
    final now = DateTime.now();
    final month = now.month.toString().padLeft(2, '0');
    final day = now.day.toString().padLeft(2, '0');
    return value
        .replaceAll('%year%', now.year.toString())
        .replaceAll('%month%', month)
        .replaceAll('%day%', day)
        .replaceAll('%date%', '${now.year}-$month-$day');
  }

  void _showToast(String message) {
    showAppToast(context, message, type: AppToastType.error);
  }
}

enum _TaskProtocol { http, bt, ed2k }

class _FormRow extends StatelessWidget {
  const _FormRow({required this.label, required this.child});

  final String label;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 104,
          child: Padding(
            padding: const EdgeInsets.only(top: 10),
            child: Text(
              label,
              style: TextStyle(color: palette.textPrimary, fontSize: 13, fontWeight: FontWeight.w500),
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(child: child),
      ],
    );
  }
}

class _DirectDownloadToggle extends StatelessWidget {
  const _DirectDownloadToggle({required this.value, required this.onChanged});

  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return GestureDetector(
      key: const ValueKey('create-task-direct-download-toggle'),
      behavior: HitTestBehavior.opaque,
      onTap: () => onChanged(!value),
      child: SizedBox(
        height: 38,
        child: Row(
          children: [
            Checkbox(
              key: const ValueKey('create-task-direct-download-checkbox'),
              state: value ? CheckboxState.checked : CheckboxState.unchecked,
              onChanged: (_) => onChanged(!value),
            ),
            const SizedBox(width: AppDesignTokens.checkboxLabelGap),
            Expanded(
              child: Text(
                context.l10n.directDownloadDescription,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(color: palette.textMuted, fontSize: 12),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _AsDefaultPathToggle extends StatelessWidget {
  const _AsDefaultPathToggle({required this.value, required this.onChanged});

  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return GestureDetector(
      key: const ValueKey('create-task-remember-download-directory'),
      behavior: HitTestBehavior.opaque,
      onTap: () => onChanged(!value),
      child: SizedBox(
        height: 24,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Checkbox(
              key: const ValueKey('create-task-remember-download-directory-checkbox'),
              state: value ? CheckboxState.checked : CheckboxState.unchecked,
              onChanged: (_) => onChanged(!value),
            ),
            const SizedBox(width: AppDesignTokens.checkboxLabelGap),
            Flexible(
              child: Text(
                key: const ValueKey('create-task-remember-download-directory-label'),
                context.l10n.rememberLastDownloadDir,
                style: TextStyle(color: palette.textSecondary, fontSize: 12, fontWeight: FontWeight.w500),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _DirectoryShortcut {
  const _DirectoryShortcut({required this.index, required this.label, required this.path});

  final int index;
  final String label;
  final String path;
}

class _DirectoryShortcuts extends StatelessWidget {
  const _DirectoryShortcuts({required this.shortcuts, required this.onSelected});

  final List<_DirectoryShortcut> shortcuts;
  final ValueChanged<_DirectoryShortcut> onSelected;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      key: const ValueKey('create-task-directory-shortcuts'),
      height: 28,
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        reverse: true,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            for (final shortcut in shortcuts) ...[
              if (shortcut.index != shortcuts.first.index) const SizedBox(width: 6),
              _DirectoryShortcutButton(
                key: ValueKey('create-task-category-${shortcut.index}'),
                shortcut: shortcut,
                onPressed: () => onSelected(shortcut),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _DirectoryShortcutButton extends StatefulWidget {
  const _DirectoryShortcutButton({super.key, required this.shortcut, required this.onPressed});

  final _DirectoryShortcut shortcut;
  final VoidCallback onPressed;

  @override
  State<_DirectoryShortcutButton> createState() => _DirectoryShortcutButtonState();
}

class _DirectoryShortcutButtonState extends State<_DirectoryShortcutButton> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return AppTooltip(
      message: widget.shortcut.path,
      child: Semantics(
        button: true,
        label: widget.shortcut.label,
        child: MouseRegion(
          cursor: SystemMouseCursors.click,
          onEnter: (_) => setState(() => _hovered = true),
          onExit: (_) => setState(() => _hovered = false),
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: widget.onPressed,
            child: AnimatedContainer(
              duration: const Duration(milliseconds: 140),
              curve: Curves.easeOutCubic,
              height: 28,
              constraints: const BoxConstraints(maxWidth: 168),
              padding: const EdgeInsets.symmetric(horizontal: 9),
              decoration: BoxDecoration(
                color: _hovered ? palette.surfaceSoft : palette.cardBg,
                borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
                border: Border.all(color: palette.border),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.folder_outlined, size: 14, color: palette.textMuted),
                  const SizedBox(width: 5),
                  Flexible(
                    child: Text(
                      widget.shortcut.label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(color: palette.textSecondary, fontSize: 11.5, fontWeight: FontWeight.w600),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _WindowTextField extends StatelessWidget {
  const _WindowTextField({
    super.key,
    required this.controller,
    required this.hintText,
    this.keyboardType,
    this.inputFormatters,
    this.obscureText = false,
  });

  final TextEditingController controller;
  final String hintText;
  final TextInputType? keyboardType;
  final List<TextInputFormatter>? inputFormatters;
  final bool obscureText;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return TextField(
      controller: controller,
      hintText: hintText,
      keyboardType: keyboardType,
      inputFormatters: inputFormatters,
      obscureText: obscureText,
      filled: true,
      border: Border.all(color: palette.border),
      borderRadius: BorderRadius.circular(4),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
    );
  }
}

class _OptionSwitch extends StatelessWidget {
  const _OptionSwitch({super.key, required this.value, required this.onChanged});

  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 38,
      child: Align(
        alignment: Alignment.centerLeft,
        child: Switch(value: value, onChanged: onChanged),
      ),
    );
  }
}

class _MiniInputAction extends StatelessWidget {
  const _MiniInputAction({required this.icon, this.onPressed});

  final IconData icon;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return GhostButton(
      density: ButtonDensity.icon,
      onPressed: onPressed,
      child: Icon(icon, size: 14, color: palette.textSecondary),
    );
  }
}

class _TripleChevronIcon extends StatelessWidget {
  const _TripleChevronIcon({required this.expanded, required this.color});

  final bool expanded;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final icon = expanded ? Icons.expand_less : Icons.expand_more;
    return SizedBox(
      width: 14,
      height: 18,
      child: Stack(
        clipBehavior: Clip.none,
        children: [
          Positioned(top: 0, left: 0, child: Icon(icon, size: 14, color: color)),
          Positioned(top: 4, left: 0, child: Icon(icon, size: 14, color: color)),
          Positioned(top: 8, left: 0, child: Icon(icon, size: 14, color: color)),
        ],
      ),
    );
  }
}

class _SegmentButton extends StatelessWidget {
  const _SegmentButton({required this.label, required this.active, required this.onTap});

  final String label;
  final bool active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return GestureDetector(
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 140),
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
        decoration: BoxDecoration(
          color: active ? palette.cardBg : const Color(0x00000000),
          borderRadius: BorderRadius.circular(999),
        ),
        child: Text(
          label,
          style: TextStyle(
            color: active ? palette.textPrimary : palette.textSecondary,
            fontSize: 12,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
    );
  }
}

const _advancedExpandDuration = Duration(milliseconds: 220);
const _advancedScrollDelay = Duration(milliseconds: 90);
const _advancedScrollDuration = Duration(milliseconds: 260);
const _advancedScrollExtentRetryDelay = Duration(milliseconds: 40);
const _advancedScrollExtentChecks = 4;
const _advancedScrollDistance = 112.0;
