import 'dart:async';
import 'dart:convert';

import 'package:desktop_drop/desktop_drop.dart';
import 'package:desktop_multi_window/desktop_multi_window.dart' as multi_window;
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart' show Clipboard, TextInputAction, TextInputType;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:path/path.dart' as path;
import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:window_manager/window_manager.dart';

import '../../../../api/model/create_task.dart';
import '../../../../api/model/create_task_batch.dart';
import '../../../../api/model/options.dart' as api_options;
import '../../../../api/model/request.dart';
import '../../../../api/model/resolve_result.dart';
import '../../../../api/model/resolve_task.dart';
import '../../../../core/capabilities/app_capabilities.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_primary_button.dart';
import '../../../../shared/widgets/app_toast.dart';
import '../../../../shared/widgets/window/desktop_window_header.dart';
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
  final _userAgentController = TextEditingController();
  final _cookieController = TextEditingController();
  final _refererController = TextEditingController();
  final _trackersController = TextEditingController();
  final _formScrollController = ScrollController();

  bool _showAdvanced = false;
  int _advancedScrollRequest = 0;
  bool _directDownload = false;
  int _protocolTab = 0;
  bool _creating = false;
  bool _draggingTorrent = false;
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
    _userAgentController.dispose();
    _cookieController.dispose();
    _refererController.dispose();
    _trackersController.dispose();
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
        if (req.url.startsWith('magnet:') || req.url.toLowerCase().endsWith('.torrent')) {
          _protocolTab = 1;
        }
      }
      if (opts != null) {
        if (opts.name.isNotEmpty) {
          _renameController.text = opts.name;
        }
        if (opts.path.isNotEmpty) {
          _directoryController.text = opts.path;
        }
        if (opts.extra is Map<String, dynamic>) {
          final extra = api_options.OptsExtraHttp.fromJson(opts.extra as Map<String, dynamic>);
          if (extra.connections > 0) {
            _connectionsController.text = extra.connections.toString();
          }
        }
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final showCaptionControls =
        !kIsWeb && (defaultTargetPlatform == TargetPlatform.windows || defaultTargetPlatform == TargetPlatform.linux);

    final page = Scaffold(
      backgroundColor: palette.sideBg,
      child: Padding(
        padding: const EdgeInsets.only(top: AppDesignTokens.windowHeaderHeight),
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
                  const Spacer(),
                  if (showCaptionControls) const WindowCaptionControls(),
                ],
              ),
            ),
            Expanded(
              child: SingleChildScrollView(
                key: const ValueKey('create-task-form-scroll'),
                controller: _formScrollController,
                padding: const EdgeInsets.fromLTRB(24, 8, 24, 12),
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
                      child: _WindowTextField(controller: _renameController, hintText: context.l10n.keepOriginalName),
                    ),
                    const SizedBox(height: 16),
                    _FormRow(
                      label: context.l10n.connections,
                      child: _WindowTextField(controller: _connectionsController, hintText: context.l10n.enterCount),
                    ),
                    const SizedBox(height: 16),
                    _FormRow(
                      label: context.l10n.directory,
                      child: _WindowTextField(
                        controller: _directoryController,
                        hintText: context.l10n.chooseDownloadDirectory,
                        trailing: kIsWeb
                            ? null
                            : _MiniInputAction(icon: Icons.folder_open_outlined, onPressed: _pickDirectory),
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
                        padding: const EdgeInsets.only(top: 18),
                        child: Column(
                          children: [
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
                              _FormRow(
                                label: 'User-Agent',
                                child: _WindowTextField(
                                  controller: _userAgentController,
                                  hintText: context.l10n.enterValue,
                                ),
                              ),
                              const SizedBox(height: 16),
                              _FormRow(
                                label: 'Cookie',
                                child: _WindowTextField(
                                  controller: _cookieController,
                                  hintText: context.l10n.enterValue,
                                ),
                              ),
                              const SizedBox(height: 16),
                              _FormRow(
                                label: 'Referer',
                                child: _WindowTextField(
                                  controller: _refererController,
                                  hintText: context.l10n.enterValue,
                                ),
                              ),
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
        if (config.downloadDir.isNotEmpty &&
            (_directoryController.text.isEmpty || _directoryController.text == 'C:/Users/levi/Downloads')) {
          _directoryController.text = config.downloadDir;
        }
        final connections = config.protocolConfig.http.connections;
        if (connections > 0 && (_connectionsController.text.isEmpty || _connectionsController.text == '16')) {
          _connectionsController.text = connections.toString();
        }
        _directDownload = config.extra.defaultDirectDownload;
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

  Future<void> _pickDirectory() async {
    if (kIsWeb) {
      return;
    }
    final directory = await FilePicker.platform.getDirectoryPath();
    if (directory == null || directory.isEmpty) return;
    setState(() => _directoryController.text = directory);
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
    final headers = <String, String>{
      if (_userAgentController.text.trim().isNotEmpty) 'User-Agent': _userAgentController.text.trim(),
      if (_cookieController.text.trim().isNotEmpty) 'Cookie': _cookieController.text.trim(),
      if (_refererController.text.trim().isNotEmpty) 'Referer': _refererController.text.trim(),
    };
    switch (protocol) {
      case _TaskProtocol.http:
        if (headers.isNotEmpty) {
          extra = ReqExtraHttp(header: headers).toJson();
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
    return Request(url: url, extra: extra);
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
      selectFiles: selectFiles,
      extra: api_options.OptsExtraHttp(connections: connections).toJson(),
    );
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
      await ref.read(gopeedServiceProvider).createTaskBatch(CreateTaskBatch(reqs: reqs));
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
      behavior: HitTestBehavior.opaque,
      onTap: () => onChanged(!value),
      child: SizedBox(
        height: 38,
        child: Row(
          children: [
            Checkbox(
              state: value ? CheckboxState.checked : CheckboxState.unchecked,
              onChanged: (_) => onChanged(!value),
              size: 18,
            ),
            const SizedBox(width: 10),
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

class _WindowTextField extends StatelessWidget {
  const _WindowTextField({required this.controller, required this.hintText, this.trailing});

  final TextEditingController controller;
  final String hintText;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return TextField(
      controller: controller,
      hintText: hintText,
      filled: true,
      border: Border.all(color: palette.border),
      borderRadius: BorderRadius.circular(4),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      features: trailing == null ? const [] : [InputTrailingFeature(trailing!)],
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
