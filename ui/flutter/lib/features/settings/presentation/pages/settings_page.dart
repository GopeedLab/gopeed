import 'dart:async';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart' show Divider, Icons, Scrollbar, ScrollbarOrientation;
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;
import 'package:url_launcher/url_launcher.dart';

import '../../../../api/model/downloader_config.dart';
import '../../../../app/application/app_appearance_controller.dart';
import '../../../../app/application/app_platform_controller.dart';
import '../../../../app/application/app_runtime_controller.dart';
import '../../../../app/application/location_keep_alive.dart';
import '../../../../core/common/start_config.dart';
import '../../../../core/utils/breakpoints.dart';
import '../../../../core/window/app_window_chrome.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_choice_segmented_control.dart';
import '../../../../shared/widgets/app_loading_button.dart';
import '../../../../shared/widgets/app_path_picker_field.dart';
import '../../../../shared/widgets/app_toast.dart';
import '../../../../shared/widgets/responsive_menu_layout.dart';
import '../../../../l10n/l10n.dart';
import '../../../../util/log_util.dart';
import '../../../../util/package_info.dart';
import '../../../../util/scheme_register/scheme_register.dart';
import '../../../../util/util.dart';
import '../../../home/presentation/widgets/primary_rail.dart';
import '../../application/settings_controller.dart';
import '../widgets/app_update_dialog.dart';
import '../widgets/download_categories_setting.dart';
import '../widgets/settings_language_select.dart';
import '../widgets/settings_content_frame.dart';
import '../widgets/settings_item.dart';
import '../widgets/settings_list_editor.dart';
import '../widgets/theme_settings_control.dart';

enum _SettingsSection { basic, downloads, advanced }

class SettingsPage extends ConsumerStatefulWidget {
  const SettingsPage({super.key, this.sectionKey});

  final String? sectionKey;

  @override
  ConsumerState<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends ConsumerState<SettingsPage> {
  final _downloadDirController = TextEditingController();
  final _maxRunningController = TextEditingController();
  final _httpUserAgentController = TextEditingController();
  final _httpConnectionsController = TextEditingController();
  final _btListenPortController = TextEditingController();
  final _btSeedRatioController = TextEditingController();
  final _btSeedTimeController = TextEditingController();
  final _ed2kListenPortController = TextEditingController();
  final _ed2kUdpPortController = TextEditingController();
  final _ed2kServerAddrController = TextEditingController();
  final _ed2kServerMetController = TextEditingController();
  final _ed2kNodesDatController = TextEditingController();
  final _proxyHostController = TextEditingController();
  final _proxyPortController = TextEditingController();
  final _proxyUserController = TextEditingController();
  final _proxyPasswordController = TextEditingController();
  final _customTrackersController = TextEditingController();
  final _apiAddressController = TextEditingController();
  final _apiHostController = TextEditingController();
  final _apiPortController = TextEditingController();
  final _apiTokenController = TextEditingController();

  DownloaderConfig? _config;
  StartConfig? _startConfig;
  String? _loadedSignature;
  String? _loadedStartSignature;
  Timer? _textSaveTimer;
  bool _syncingControllers = false;
  bool _syncingStartControllers = false;
  bool _savingStartConfig = false;
  String? _apiNetworkDraft;

  List<TextEditingController> get _textControllers => [
    _downloadDirController,
    _maxRunningController,
    _httpUserAgentController,
    _httpConnectionsController,
    _btListenPortController,
    _btSeedRatioController,
    _btSeedTimeController,
    _ed2kListenPortController,
    _ed2kUdpPortController,
    _ed2kServerAddrController,
    _ed2kServerMetController,
    _ed2kNodesDatController,
    _proxyHostController,
    _proxyPortController,
    _proxyUserController,
    _proxyPasswordController,
    _customTrackersController,
  ];

  List<TextEditingController> get _startTextControllers => [
    _apiAddressController,
    _apiHostController,
    _apiPortController,
    _apiTokenController,
  ];

  @override
  void initState() {
    super.initState();
    for (final controller in _textControllers) {
      controller.addListener(_scheduleTextSave);
    }
    for (final controller in _startTextControllers) {
      controller.addListener(_handleStartDraftChanged);
    }
  }

  @override
  void dispose() {
    _textSaveTimer?.cancel();
    for (final controller in _textControllers) {
      controller.removeListener(_scheduleTextSave);
      controller.dispose();
    }
    for (final controller in _startTextControllers) {
      controller.removeListener(_handleStartDraftChanged);
      controller.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final stateAsync = ref.watch(settingsControllerProvider);
    final runtimeState = ref.watch(appRuntimeControllerProvider).value;
    final isDesktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    final selectedSection = _sectionFromKey(widget.sectionKey) ?? _SettingsSection.basic;
    final content = stateAsync.when(
      loading: () => const Center(child: shad.CircularProgressIndicator()),
      error: (error, _) =>
          _ErrorState(error: error, onRetry: () => ref.read(settingsControllerProvider.notifier).reload()),
      data: (state) {
        _syncConfig(state.config);
        if (runtimeState != null) {
          _syncStartConfig(runtimeState.startConfig);
        }
        return ResponsiveMenuLayout<_SettingsSection>(
          title: context.l10n.setting,
          items: _menuItems,
          selectedValue: selectedSection,
          onSelected: (section) => _selectSection(section, isDesktop: isDesktop),
          mobileContentTitleBuilder: _sectionTitle,
          mobileContentVisible: widget.sectionKey != null,
          onMobileBack: _leaveMobileSection,
          contentBuilder: (context, section) => _buildSectionContent(section, runtimeState),
        );
      },
    );

    return shad.Scaffold(
      backgroundColor: palette.bg,
      child: isDesktop
          ? Row(
              children: [
                const PrimaryRail(activeSection: RailSection.settings),
                Expanded(child: content),
              ],
            )
          : Column(
              children: [
                Expanded(child: content),
                const PrimaryBottomNavigation(activeSection: RailSection.settings),
              ],
            ),
    );
  }

  void _selectSection(_SettingsSection section, {required bool isDesktop}) {
    final location = '/settings/${section.name}';
    if (isDesktop) {
      context.go(location);
    } else {
      context.push(location);
    }
  }

  void _leaveMobileSection() {
    if (context.canPop()) {
      context.pop();
    } else {
      context.go('/settings');
    }
  }

  _SettingsSection? _sectionFromKey(String? key) {
    for (final section in _SettingsSection.values) {
      if (section.name == key) return section;
    }
    return null;
  }

  List<ResponsiveMenuItem<_SettingsSection>> get _menuItems => [
    ResponsiveMenuItem(value: _SettingsSection.basic, label: context.l10n.basicSettings, icon: Icons.tune_outlined),
    ResponsiveMenuItem(
      value: _SettingsSection.downloads,
      label: context.l10n.downloadSettings,
      icon: Icons.download_outlined,
    ),
    ResponsiveMenuItem(
      value: _SettingsSection.advanced,
      label: context.l10n.advancedSettings,
      icon: Icons.code_outlined,
    ),
  ];

  Widget _buildSectionContent(_SettingsSection section, AppRuntimeState? runtimeState) {
    final config = _config!;
    final palette = AppPalette.of(context);
    final platform = ref.watch(appPlatformControllerProvider).value;
    final isDesktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;

    return Padding(
      padding: EdgeInsets.only(
        top: isDesktop && AppWindowChrome.reservesHeaderInset ? AppDesignTokens.windowHeaderHeight : 0,
      ),
      child: Column(
        children: [
          if (isDesktop)
            SizedBox(
              height: AppDesignTokens.contentHeaderHeight,
              child: SettingsContentFrame(
                child: SizedBox(
                  key: const ValueKey('settings-header-content'),
                  width: double.infinity,
                  child: Row(
                    children: [
                      Text(
                        _sectionTitle(section),
                        style: TextStyle(color: palette.textPrimary, fontSize: 22, fontWeight: FontWeight.w800),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          Expanded(
            child: ListView(
              key: const ValueKey('settings-section-scroll-view'),
              padding: const EdgeInsets.only(top: AppDesignTokens.space8, bottom: 28),
              children: [
                SettingsContentFrame(
                  child: SizedBox(
                    key: const ValueKey('settings-body-content'),
                    width: double.infinity,
                    child: _sectionBody(section, config, platform, runtimeState),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _sectionBody(
    _SettingsSection section,
    DownloaderConfig config,
    AppPlatformState? platform,
    AppRuntimeState? runtimeState,
  ) {
    final apiConfigDirty = _startConfig != null && _startDraftSignature() != _loadedStartSignature;
    final apiServerVisualState = _apiServerVisualState(runtimeState);
    switch (section) {
      case _SettingsSection.basic:
        return _SettingsSectionStack(
          children: [
            _SettingsBlock(
              title: context.l10n.ui,
              child: _SettingsGroup(
                children: [
                  SettingsItem(
                    title: context.l10n.theme,
                    child: ThemeModeSelector(
                      value: ref.watch(appAppearanceControllerProvider).themeMode,
                      accent: ref.watch(appAppearanceControllerProvider).themeColor,
                      onChanged: (mode) {
                        ref.read(appAppearanceControllerProvider.notifier).setThemeMode(mode);
                        _mutateConfig((next) => next.extra.themeMode = mode.key);
                      },
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.themeColor,
                    subtitle: context.l10n.themeColorDescription,
                    child: ThemeColorSelector(
                      value: ref.watch(appAppearanceControllerProvider).themeColor,
                      onChanged: (color) {
                        ref.read(appAppearanceControllerProvider.notifier).setThemeColor(color);
                        _mutateConfig((next) => next.extra.themeColor = color.key);
                      },
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.locale,
                    child: SettingsLanguageSelect(
                      value: _languageValue(config),
                      onChanged: (value) {
                        _mutateConfig((next) => next.extra.locale = value == 'system' ? '' : value);
                      },
                    ),
                  ),
                ],
              ),
            ),
            _SettingsBlock(
              title: context.l10n.general,
              child: _SettingsGroup(
                children: [
                  SettingsItem(
                    title: context.l10n.browserExtension,
                    subtitle: context.l10n.browserExtensionDescription,
                    child: _BrowserExtensionLinks(onOpen: _openBrowserExtension),
                  ),
                  if (Util.isWindows() || Util.isLinux())
                    SettingsItem(
                      title: context.l10n.launchAtStartup,
                      child: shad.Switch(
                        value: platform?.launchAtStartup ?? false,
                        onChanged: (value) => _runAction(
                          () => ref.read(appPlatformControllerProvider.notifier).setLaunchAtStartup(value),
                        ),
                      ),
                    ),
                  if (Util.isMacos())
                    SettingsItem(
                      title: context.l10n.runAsMenubarApp,
                      subtitle: context.l10n.runAsMenubarAppDesc,
                      child: shad.Switch(
                        value: platform?.runAsMenubarApp ?? false,
                        onChanged: (value) => _runAction(
                          () => ref.read(appPlatformControllerProvider.notifier).setRunAsMenubarApp(value),
                        ),
                      ),
                    ),
                  if (Util.isDesktop())
                    SettingsItem(
                      title: context.l10n.desktopNotification,
                      subtitle: context.l10n.desktopNotificationDescription,
                      child: shad.Switch(
                        value: config.extra.desktopNotification,
                        onChanged: (value) => _mutateConfig((next) => next.extra.desktopNotification = value),
                      ),
                    ),
                  if (Util.isIOS())
                    SettingsItem(
                      title: context.l10n.backgroundLocationKeepAlive,
                      subtitle: context.l10n.backgroundLocationKeepAliveDescription,
                      child: shad.Switch(
                        value: config.extra.backgroundLocationKeepAlive,
                        onChanged: (value) => unawaited(_setBackgroundLocationKeepAlive(value)),
                      ),
                    ),
                ],
              ),
            ),
            _SettingsBlock(
              title: context.l10n.about,
              child: _SettingsGroup(
                children: [
                  SettingsItem(
                    title: context.l10n.homepage,
                    child: _ExternalTextLink(
                      key: const ValueKey('gopeed-homepage'),
                      label: 'gopeed.com',
                      onPressed: () => unawaited(_openExternalUri(Uri.parse('https://gopeed.com'))),
                    ),
                  ),
                  SettingsItem(
                    title: 'GitHub',
                    child: _ExternalTextLink(
                      key: const ValueKey('gopeed-github'),
                      label: 'github.com/GopeedLab/gopeed',
                      onPressed: () => unawaited(_openExternalUri(Uri.parse('https://github.com/GopeedLab/gopeed'))),
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.contributors,
                    subtitle: context.l10n.thanksDesc,
                    child: _ExternalTextLink(
                      key: const ValueKey('gopeed-contributors'),
                      label: context.l10n.viewContributors,
                      onPressed: () => unawaited(
                        _openExternalUri(Uri.parse('https://github.com/GopeedLab/gopeed/graphs/contributors')),
                      ),
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.analyticsEnabled,
                    subtitle: context.l10n.analyticsEnabledDesc,
                    child: shad.Switch(
                      value: config.extra.analyticsEnabled,
                      onChanged: (value) => _mutateConfig((next) => next.extra.analyticsEnabled = value),
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.notifyWhenNewVersion,
                    child: shad.Switch(
                      value: config.extra.notifyWhenNewVersion,
                      onChanged: (value) => _mutateConfig((next) => next.extra.notifyWhenNewVersion = value),
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.version,
                    subtitle: context.l10n.currentVersion(_versionLabel),
                    child: _UpdateActionControl(platform: platform),
                  ),
                ],
              ),
            ),
          ],
        );
      case _SettingsSection.downloads:
        return _SettingsSectionStack(
          children: [
            _SettingsBlock(
              title: context.l10n.general,
              child: _SettingsGroup(
                children: [
                  SettingsItem(
                    title: context.l10n.downloadDir,
                    child: AppPathPickerField.downloadDirectory(
                      fieldKey: const ValueKey('download-directory-input'),
                      controller: _downloadDirController,
                      pickerKey: const ValueKey('download-directory-picker'),
                      desktopWidth: AppDesignTokens.settingsFormControlWidth,
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.downloadCategories,
                    child: DownloadCategoriesControl(
                      categories: config.extra.downloadCategories,
                      displayName: _categoryName,
                      onAdd: () => unawaited(_editDownloadCategory()),
                      onEdit: (category) => unawaited(_editDownloadCategory(category)),
                      onDelete: (category) => unawaited(_deleteDownloadCategory(category)),
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.maxRunning,
                    subtitle: context.l10n.maxRunningDescription,
                    child: _NumberSettingControl(
                      fieldKey: const ValueKey('max-running-input'),
                      controller: _maxRunningController,
                      min: 1,
                      max: 256,
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.defaultDirectDownload,
                    child: shad.Switch(
                      value: config.extra.defaultDirectDownload,
                      onChanged: (value) => _mutateConfig((next) => next.extra.defaultDirectDownload = value),
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.autoStartTasks,
                    child: shad.Switch(
                      value: config.autoStartTasks,
                      onChanged: (value) => _mutateConfig((next) => next.autoStartTasks = value),
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.autoTorrentEnable,
                    child: shad.Switch(
                      value: config.autoTorrent.enable,
                      onChanged: (value) => _mutateConfig((next) => next.autoTorrent.enable = value),
                    ),
                  ),
                  if (config.autoTorrent.enable)
                    SettingsItem(
                      title: context.l10n.autoTorrentDeleteAfterDownload,
                      child: shad.Switch(
                        value: config.autoTorrent.deleteAfterDownload,
                        onChanged: (value) => _mutateConfig((next) => next.autoTorrent.deleteAfterDownload = value),
                      ),
                    ),
                  SettingsItem(
                    title: context.l10n.autoDeleteMissingFileTasks,
                    child: shad.Switch(
                      value: config.autoDeleteMissingFileTasks,
                      onChanged: (value) => _mutateConfig((next) => next.autoDeleteMissingFileTasks = value),
                    ),
                  ),
                ],
              ),
            ),
            _SettingsBlock(
              key: const ValueKey('settings-http-block'),
              title: 'HTTP',
              child: _SettingsGroup(
                children: [
                  SettingsItem(
                    title: 'User-Agent',
                    subtitle: context.l10n.defaultUserAgentDescription,
                    child: _TextSettingControl(
                      fieldKey: const ValueKey('http-user-agent-input'),
                      controller: _httpUserAgentController,
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.connections,
                    child: _NumberSettingControl(
                      fieldKey: const ValueKey('http-connections-input'),
                      controller: _httpConnectionsController,
                      min: 1,
                      max: 256,
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.useServerCtime,
                    child: shad.Switch(
                      value: config.protocolConfig.http.useServerCtime,
                      onChanged: (value) => _mutateConfig((next) => next.protocolConfig.http.useServerCtime = value),
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.autoExtract,
                    child: shad.Switch(
                      value: config.archive.autoExtract,
                      onChanged: (value) => _mutateConfig((next) => next.archive.autoExtract = value),
                    ),
                  ),
                  if (config.archive.autoExtract)
                    SettingsItem(
                      title: context.l10n.deleteAfterExtract,
                      child: shad.Switch(
                        value: config.archive.deleteAfterExtract,
                        onChanged: (value) => _mutateConfig((next) => next.archive.deleteAfterExtract = value),
                      ),
                    ),
                ],
              ),
            ),
            _SettingsBlock(
              title: 'BitTorrent',
              child: _SettingsGroup(
                children: [
                  if (Util.isWindows())
                    SettingsItem(
                      title: context.l10n.setAsDefaultBtClient,
                      child: shad.Switch(
                        value: config.extra.defaultBtClient,
                        onChanged: (value) => unawaited(_setDefaultBtClient(value)),
                      ),
                    ),
                  SettingsItem(
                    title: context.l10n.listenPort,
                    child: _NumberSettingControl(
                      fieldKey: const ValueKey('bt-listen-port-input'),
                      controller: _btListenPortController,
                      min: 0,
                      max: 65535,
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.seedKeep,
                    child: shad.Switch(
                      value: config.protocolConfig.bt.seedKeep,
                      onChanged: (value) => _mutateConfig((next) => next.protocolConfig.bt.seedKeep = value),
                    ),
                  ),
                  if (!config.protocolConfig.bt.seedKeep) ...[
                    SettingsItem(
                      title: context.l10n.seedRatio,
                      child: _NumberSettingControl(
                        fieldKey: const ValueKey('bt-seed-ratio-input'),
                        controller: _btSeedRatioController,
                        min: 0,
                        step: 0.1,
                        decimalPlaces: 2,
                      ),
                    ),
                    SettingsItem(
                      title: context.l10n.seedTime,
                      child: _NumberSettingControl(
                        fieldKey: const ValueKey('bt-seed-time-input'),
                        controller: _btSeedTimeController,
                        min: 0,
                        max: 100000000,
                      ),
                    ),
                  ],
                  SettingsItem(
                    title: context.l10n.subscribeTracker,
                    child: _TrackerSubscriptionsControl(
                      selected: config.extra.bt.trackerSubscribeUrls,
                      autoUpdate: config.extra.bt.autoUpdateTrackers,
                      lastUpdated: config.extra.bt.lastTrackerUpdateTime,
                      onChanged: (urls) => _mutateConfig((next) => next.extra.bt.trackerSubscribeUrls = urls),
                      onAutoUpdateChanged: (value) => _mutateConfig((next) => next.extra.bt.autoUpdateTrackers = value),
                      onUpdate: _updateTrackers,
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.addTracker,
                    subtitle: context.l10n.onePerLine,
                    child: _TextSettingControl(controller: _customTrackersController, minLines: 7),
                  ),
                ],
              ),
            ),
            _SettingsBlock(
              title: 'ED2K',
              child: _SettingsGroup(
                children: [
                  SettingsItem(
                    title: context.l10n.ed2kTcpPort,
                    child: _NumberSettingControl(
                      fieldKey: const ValueKey('ed2k-tcp-port-input'),
                      controller: _ed2kListenPortController,
                      min: 0,
                      max: 65535,
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.ed2kUdpPort,
                    child: _NumberSettingControl(
                      fieldKey: const ValueKey('ed2k-udp-port-input'),
                      controller: _ed2kUdpPortController,
                      min: 0,
                      max: 65535,
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.ed2kServerList,
                    subtitle: context.l10n.onePerLine,
                    child: _TextSettingControl(
                      fieldKey: const ValueKey('ed2k-server-address-input'),
                      hintText: context.l10n.ed2kServersHint,
                      controller: _ed2kServerAddrController,
                      minLines: 5,
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.ed2kServerMet,
                    subtitle: context.l10n.onePerLine,
                    child: _TextSettingControl(
                      fieldKey: const ValueKey('ed2k-server-met-input'),
                      hintText: context.l10n.ed2kServerMetHint,
                      controller: _ed2kServerMetController,
                      minLines: 4,
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.ed2kNodesDat,
                    subtitle: context.l10n.onePerLine,
                    child: _TextSettingControl(
                      fieldKey: const ValueKey('ed2k-nodes-dat-input'),
                      hintText: context.l10n.ed2kNodesDatHint,
                      controller: _ed2kNodesDatController,
                      minLines: 4,
                    ),
                  ),
                ],
              ),
            ),
          ],
        );
      case _SettingsSection.advanced:
        return _SettingsSectionStack(
          children: [
            _SettingsBlock(
              title: context.l10n.network,
              child: _SettingsGroup(
                children: [
                  SettingsItem(
                    title: context.l10n.proxy,
                    child: AppChoiceSegmentedControl<String>(
                      value: _proxyMode,
                      buttonKeyPrefix: 'settings-choice',
                      options: [
                        AppChoiceOption(value: 'none', label: context.l10n.noProxy, icon: Icons.block_outlined),
                        AppChoiceOption(
                          value: 'system',
                          label: context.l10n.systemProxy,
                          icon: Icons.computer_outlined,
                        ),
                        AppChoiceOption(value: 'custom', label: context.l10n.customProxy, icon: Icons.tune_outlined),
                      ],
                      onChanged: (value) => _mutateConfig((next) {
                        next.proxy.enable = value != 'none';
                        next.proxy.system = value == 'system';
                      }),
                    ),
                  ),
                  if (_proxyMode == 'custom') ...[
                    SettingsItem(
                      title: context.l10n.proxyProtocol,
                      child: AppChoiceSegmentedControl<String>(
                        value: _proxyScheme,
                        buttonKeyPrefix: 'settings-choice',
                        options: const [
                          AppChoiceOption(value: 'http', label: 'HTTP', icon: Icons.http_outlined),
                          AppChoiceOption(value: 'https', label: 'HTTPS', icon: Icons.https_outlined),
                          AppChoiceOption(value: 'socks5', label: 'SOCKS5', icon: Icons.security_outlined),
                        ],
                        onChanged: (value) => _mutateConfig((next) => next.proxy.scheme = value),
                      ),
                    ),
                    SettingsItem(
                      title: context.l10n.server,
                      child: _HostPortControl(
                        hostController: _proxyHostController,
                        portController: _proxyPortController,
                        hostKey: const ValueKey('proxy-host-input'),
                        portKey: const ValueKey('proxy-port-input'),
                      ),
                    ),
                    SettingsItem(
                      title: context.l10n.username,
                      child: _TextSettingControl(controller: _proxyUserController),
                    ),
                    SettingsItem(
                      title: context.l10n.password,
                      child: _TextSettingControl(controller: _proxyPasswordController, obscureText: true),
                    ),
                  ],
                  SettingsItem(
                    title: context.l10n.githubMirrorEnable,
                    subtitle: context.l10n.githubMirrorDesc,
                    child: shad.Switch(
                      value: config.extra.githubMirror.enabled,
                      onChanged: (value) => _mutateConfig((next) => next.extra.githubMirror.enabled = value),
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.githubMirror,
                    child: SettingsListEditor(
                      entries: [
                        for (final entry in config.extra.githubMirror.mirrors.indexed)
                          if (!entry.$2.isDeleted)
                            SettingsListEntry(
                              id: 'github-mirror-${entry.$1}',
                              title: entry.$2.url,
                              subtitle: entry.$2.type.name,
                            ),
                      ],
                      onAdd: () => unawaited(_editGithubMirror()),
                      onEdit: (visibleIndex) => unawaited(_editGithubMirrorAtVisibleIndex(visibleIndex)),
                      onDelete: (visibleIndex) => unawaited(_deleteGithubMirrorAtVisibleIndex(visibleIndex)),
                    ),
                  ),
                ],
              ),
            ),
            if (!kIsWeb)
              _SettingsBlock(
                title: 'API',
                child: _SettingsGroup(
                  children: [
                    SettingsItem(
                      title: context.l10n.apiServerRuntimeStatus,
                      subtitle: _apiServerRuntimeDetails(runtimeState),
                      child: _ApiServerStatusLight(
                        key: const ValueKey('api-server-runtime-status'),
                        state: apiServerVisualState,
                        label: switch (apiServerVisualState) {
                          _ApiServerVisualState.loading => context.l10n.readingRuntimeStatus,
                          _ApiServerVisualState.running => context.l10n.apiServerRunning,
                          _ApiServerVisualState.stopped => context.l10n.apiServerStopped,
                          _ApiServerVisualState.failed => context.l10n.apiServerFailed,
                        },
                      ),
                    ),
                    SettingsItem(
                      title: context.l10n.protocol,
                      child: AppChoiceSegmentedControl<String>(
                        value: _apiNetworkValue,
                        buttonKeyPrefix: 'settings-choice',
                        options: [
                          const AppChoiceOption(value: 'tcp', label: 'TCP', icon: Icons.lan_outlined),
                          if (Util.supportUnixSocket())
                            const AppChoiceOption(value: 'unix', label: 'Unix Socket', icon: Icons.cable_outlined),
                        ],
                        onChanged: _changeApiNetwork,
                      ),
                    ),
                    if (_apiNetworkValue == 'unix')
                      SettingsItem(
                        title: context.l10n.socketPath,
                        child: _TextSettingControl(controller: _apiAddressController),
                      )
                    else
                      SettingsItem(
                        title: context.l10n.listenAddress,
                        child: _HostPortControl(
                          hostController: _apiHostController,
                          portController: _apiPortController,
                          hostKey: const ValueKey('api-host-input'),
                          portKey: const ValueKey('api-port-input'),
                        ),
                      ),
                    if (_apiNetworkValue == 'tcp')
                      SettingsItem(
                        title: context.l10n.apiToken,
                        child: _TextSettingControl(controller: _apiTokenController, obscureText: true),
                      ),
                    Align(
                      alignment: Alignment.centerRight,
                      child: Wrap(
                        spacing: 8,
                        runSpacing: 8,
                        alignment: WrapAlignment.end,
                        children: [
                          AppLoadingButton(
                            key: const ValueKey('toggle-api-server-button'),
                            onPressed: runtimeState == null ? null : _toggleApiServer,
                            loading: _savingStartConfig,
                            icon: Icon(
                              runtimeState?.apiServerState.running ?? false
                                  ? Icons.stop_circle_outlined
                                  : Icons.play_circle_outline,
                            ),
                            child: Text(
                              runtimeState?.apiServerState.running ?? false
                                  ? context.l10n.stopApiServer
                                  : context.l10n.startApiServer,
                            ),
                          ),
                          AppLoadingButton(
                            key: const ValueKey('save-api-config-button'),
                            onPressed: runtimeState == null || !apiConfigDirty ? null : _saveApiServerConfig,
                            loading: _savingStartConfig,
                            icon: const Icon(Icons.save_outlined),
                            variant: AppLoadingButtonVariant.primary,
                            child: Text(context.l10n.save),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            _SettingsBlock(
              title: context.l10n.developer,
              child: _SettingsGroup(
                children: [
                  SettingsItem(
                    title: context.l10n.webhookEnable,
                    subtitle: context.l10n.webhookDesc,
                    child: shad.Switch(
                      value: config.webhook.enable,
                      onChanged: (value) => _mutateConfig((next) => next.webhook.enable = value),
                    ),
                  ),
                  SettingsItem(
                    title: context.l10n.webhookPush,
                    subtitle: context.l10n.webhookPushDescription,
                    child: SettingsListEditor(
                      entries: [
                        for (final entry in config.webhook.urls.indexed)
                          SettingsListEntry(id: 'webhook-${entry.$1}', title: entry.$2),
                      ],
                      onAdd: () => unawaited(_editWebhook()),
                      onEdit: (index) => unawaited(_editWebhook(index)),
                      onDelete: _deleteWebhook,
                      addButtonKey: const ValueKey('add-webhook-button'),
                    ),
                  ),
                  if (Util.isDesktop()) ...[
                    SettingsItem(
                      title: context.l10n.scriptEnable,
                      subtitle: context.l10n.scriptDesc,
                      child: shad.Switch(
                        value: config.script.enable,
                        onChanged: (value) => _mutateConfig((next) => next.script.enable = value),
                      ),
                    ),
                    SettingsItem(
                      title: context.l10n.scriptExecution,
                      subtitle: context.l10n.scriptExecutionDescription,
                      child: SettingsListEditor(
                        entries: [
                          for (final entry in config.script.paths.indexed)
                            SettingsListEntry(id: 'script-${entry.$1}', title: entry.$2),
                        ],
                        onAdd: () => unawaited(_editScript()),
                        onEdit: (index) => unawaited(_editScript(index)),
                        onDelete: _deleteScript,
                        addButtonKey: const ValueKey('add-script-button'),
                      ),
                    ),
                  ],
                  SettingsItem(
                    title: context.l10n.logDirectory,
                    subtitle: _logDirectoryLabel,
                    child: shad.SecondaryButton(
                      onPressed: kIsWeb ? null : _openLogDirectory,
                      leading: const Icon(Icons.folder_open_outlined),
                      child: Text(context.l10n.open),
                    ),
                  ),
                ],
              ),
            ),
          ],
        );
    }
  }

  void _syncConfig(DownloaderConfig config) {
    final signature = _signature(config);
    if (_loadedSignature == signature && _config != null) {
      return;
    }
    _loadedSignature = signature;
    _config = DownloaderConfig.fromJson(config.toJson());
    _syncingControllers = true;
    _downloadDirController.text = _config!.downloadDir;
    _maxRunningController.text = _config!.maxRunning.clamp(1, 256).toString();
    _httpUserAgentController.text = _config!.protocolConfig.http.userAgent;
    _httpConnectionsController.text = _config!.protocolConfig.http.connections.clamp(1, 256).toString();
    _btListenPortController.text = _config!.protocolConfig.bt.listenPort.clamp(0, 65535).toString();
    _btSeedRatioController.text = _config!.protocolConfig.bt.seedRatio.toString();
    _btSeedTimeController.text = (_config!.protocolConfig.bt.seedTime ~/ 60).clamp(0, 100000000).toString();
    _ed2kListenPortController.text = _config!.protocolConfig.ed2k.listenPort.clamp(0, 65535).toString();
    _ed2kUdpPortController.text = _config!.protocolConfig.ed2k.udpPort.clamp(0, 65535).toString();
    _ed2kServerAddrController.text = _formatEd2kMultiline(_config!.protocolConfig.ed2k.serverAddr);
    _ed2kServerMetController.text = _formatEd2kMultiline(_config!.protocolConfig.ed2k.serverMet);
    _ed2kNodesDatController.text = _formatEd2kMultiline(_config!.protocolConfig.ed2k.nodesDat);
    final proxyAddress = _splitHostPort(_config!.proxy.host);
    _proxyHostController.text = proxyAddress.host;
    _proxyPortController.text = proxyAddress.port;
    _proxyUserController.text = _config!.proxy.usr;
    _proxyPasswordController.text = _config!.proxy.pwd;
    _customTrackersController.text = _config!.extra.bt.customTrackers.join('\n');
    _syncingControllers = false;
  }

  void _syncStartConfig(StartConfig config, {bool force = false}) {
    final signature = _startSignature(config);
    if (!force && _loadedStartSignature == signature && _startConfig != null) {
      return;
    }
    if (!force &&
        _startConfig != null &&
        _startConfig!.network == config.network &&
        _startConfig!.address == config.address &&
        _startConfig!.apiToken == config.apiToken) {
      _startConfig!.apiEnable = config.apiEnable;
      _loadedStartSignature = signature;
      return;
    }
    _loadedStartSignature = signature;
    _startConfig = _copyStartConfig(config);
    _apiNetworkDraft = config.network == 'unix' && Util.supportUnixSocket() ? 'unix' : 'tcp';
    _syncingStartControllers = true;
    _apiAddressController.text = _startConfig!.address;
    if (_apiNetworkDraft == 'tcp' && _looksLikeTcpAddress(_startConfig!.address)) {
      final apiAddress = _splitHostPort(_startConfig!.address);
      _apiHostController.text = apiAddress.host;
      _apiPortController.text = apiAddress.port;
    } else {
      _apiHostController.text = '127.0.0.1';
      _apiPortController.text = '9999';
    }
    _apiTokenController.text = _startConfig!.apiToken;
    _syncingStartControllers = false;
  }

  void _scheduleTextSave() {
    if (_syncingControllers || _config == null) {
      return;
    }
    _textSaveTimer?.cancel();
    _textSaveTimer = Timer(const Duration(milliseconds: 650), () {
      _applyTextControllers(_config!);
      unawaited(_saveConfig());
    });
  }

  void _handleStartDraftChanged() {
    if (_syncingStartControllers || !mounted) return;
    setState(() {});
  }

  void _mutateConfig(void Function(DownloaderConfig config) mutation) {
    final config = _config;
    if (config == null) {
      return;
    }
    setState(() => mutation(config));
    unawaited(_saveConfig());
  }

  void _applyTextControllers(DownloaderConfig config) {
    config.downloadDir = _downloadDirController.text.trim();
    config.maxRunning = _boundedInt(_maxRunningController, fallback: config.maxRunning, min: 1, max: 256);
    config.protocolConfig.http.userAgent = _httpUserAgentController.text.trim();
    config.protocolConfig.http.connections = _boundedInt(
      _httpConnectionsController,
      fallback: config.protocolConfig.http.connections,
      min: 1,
      max: 256,
    );
    config.protocolConfig.bt.listenPort = _boundedInt(
      _btListenPortController,
      fallback: config.protocolConfig.bt.listenPort,
      min: 0,
      max: 65535,
    );
    config.protocolConfig.bt.seedRatio =
        (double.tryParse(_btSeedRatioController.text.trim()) ?? config.protocolConfig.bt.seedRatio)
            .clamp(0, double.infinity)
            .toDouble();
    final seedMinutes = _boundedInt(
      _btSeedTimeController,
      fallback: config.protocolConfig.bt.seedTime ~/ 60,
      min: 0,
      max: 100000000,
    );
    config.protocolConfig.bt.seedTime = seedMinutes * 60;
    config.protocolConfig.ed2k.listenPort = _boundedInt(
      _ed2kListenPortController,
      fallback: config.protocolConfig.ed2k.listenPort,
      min: 0,
      max: 65535,
    );
    config.protocolConfig.ed2k.udpPort = _boundedInt(
      _ed2kUdpPortController,
      fallback: config.protocolConfig.ed2k.udpPort,
      min: 0,
      max: 65535,
    );
    config.protocolConfig.ed2k.serverAddr = _lines(_ed2kServerAddrController.text).join(',');
    config.protocolConfig.ed2k.serverMet = _lines(_ed2kServerMetController.text).join(',');
    config.protocolConfig.ed2k.nodesDat = _lines(_ed2kNodesDatController.text).join(',');
    config.proxy.host = _joinHostPort(_proxyHostController.text, _proxyPortController.text);
    config.proxy.usr = _proxyUserController.text.trim();
    config.proxy.pwd = _proxyPasswordController.text;
    config.extra.bt.customTrackers = _lines(_customTrackersController.text);
    config.protocolConfig.bt.trackers = {
      ...config.extra.bt.subscribeTrackers,
      ...config.extra.bt.customTrackers,
    }.toList();
  }

  void _applyStartControllers(StartConfig config) {
    config.network = _apiNetworkDraft ?? config.network;
    config.address = config.network == 'unix'
        ? _apiAddressController.text.trim()
        : _joinHostPort(_apiHostController.text, _apiPortController.text);
    config.apiToken = _apiTokenController.text.trim();
  }

  Future<void> _saveConfig() async {
    final config = _config;
    if (config == null) {
      return;
    }
    _applyTextControllers(config);
    final snapshot = DownloaderConfig.fromJson(config.toJson());
    _loadedSignature = _signature(snapshot);
    try {
      await ref.read(settingsControllerProvider.notifier).save(snapshot);
    } catch (error) {
      if (mounted) _toast(error.toString());
    }
  }

  Future<void> _saveApiServerConfig() async {
    final config = _startConfig;
    final runtimeState = ref.read(appRuntimeControllerProvider).value;
    if (config == null || runtimeState == null || _savingStartConfig) {
      return;
    }
    final snapshot = _copyStartConfig(config);
    _applyStartControllers(snapshot);
    final validationMessage = await _validateStartConfig(snapshot);
    if (validationMessage != null) {
      if (mounted) _toast(validationMessage);
      return;
    }
    if (runtimeState.apiServerState.running && !await _confirmRestartApiServer()) {
      return;
    }
    setState(() => _savingStartConfig = true);
    try {
      final controller = ref.read(appRuntimeControllerProvider.notifier);
      if (runtimeState.apiServerState.running) {
        await controller.restartApiServer(snapshot);
      } else {
        await controller.saveApiServerConfig(snapshot);
      }
      _syncStartConfig(ref.read(appRuntimeControllerProvider).requireValue.startConfig, force: true);
      if (mounted) _toast(context.l10n.apiConfigSaved, type: AppToastType.success);
    } catch (error) {
      if (mounted) _toast(error.toString());
    } finally {
      if (mounted) setState(() => _savingStartConfig = false);
    }
  }

  Future<void> _toggleApiServer() async {
    final runtimeState = ref.read(appRuntimeControllerProvider).value;
    if (runtimeState == null || _savingStartConfig) return;
    if (runtimeState.apiServerState.running && !await _confirmDisableApiServer()) return;

    setState(() => _savingStartConfig = true);
    try {
      final controller = ref.read(appRuntimeControllerProvider.notifier);
      if (runtimeState.apiServerState.running) {
        await controller.stopApiServer();
      } else {
        await controller.startApiServer();
      }
    } catch (error) {
      if (mounted) _toast(error.toString());
    } finally {
      if (mounted) setState(() => _savingStartConfig = false);
    }
  }

  Future<bool> _confirmDisableApiServer() async {
    final overlay = const shad.DialogOverlayHandler().show<bool>(
      context: context,
      alignment: Alignment.center,
      barrierDismissable: false,
      builder: (dialogContext) => shad.AlertDialog(
        key: const ValueKey('disable-api-server-dialog'),
        title: Text(dialogContext.l10n.stopListening),
        content: Text(dialogContext.l10n.disableApiServerConfirm),
        actions: [
          shad.SecondaryButton(
            key: const ValueKey('cancel-disable-api-server-button'),
            onPressed: () => shad.closeOverlay(dialogContext, false),
            child: Text(dialogContext.l10n.cancel),
          ),
          shad.DestructiveButton(
            key: const ValueKey('confirm-disable-api-server-button'),
            onPressed: () => shad.closeOverlay(dialogContext, true),
            child: Text(dialogContext.l10n.confirm),
          ),
        ],
      ),
    );
    return await overlay.future ?? false;
  }

  Future<bool> _confirmRestartApiServer() async {
    final overlay = const shad.DialogOverlayHandler().show<bool>(
      context: context,
      alignment: Alignment.center,
      barrierDismissable: false,
      builder: (dialogContext) => shad.AlertDialog(
        key: const ValueKey('restart-api-server-dialog'),
        title: Text(dialogContext.l10n.saveAndRestartApiServer),
        content: Text(dialogContext.l10n.restartApiServerConfirm),
        actions: [
          shad.SecondaryButton(
            key: const ValueKey('cancel-restart-api-server-button'),
            onPressed: () => shad.closeOverlay(dialogContext, false),
            child: Text(dialogContext.l10n.cancel),
          ),
          shad.PrimaryButton(
            key: const ValueKey('confirm-restart-api-server-button'),
            onPressed: () => shad.closeOverlay(dialogContext, true),
            child: Text(dialogContext.l10n.saveAndRestartApiServer),
          ),
        ],
      ),
    );
    return await overlay.future ?? false;
  }

  Future<String?> _validateStartConfig(StartConfig candidate) async {
    final l10n = context.l10n;
    if (candidate.network == 'unix') {
      return candidate.address.trim().isEmpty ? l10n.socketPathRequired : null;
    }

    final address = _splitHostPort(candidate.address);
    if (address.host.isEmpty) return l10n.serverRequired;
    final port = int.tryParse(address.port);
    if (port == null || port < 0 || port > 65535) return l10n.portRangeError;

    return null;
  }

  void _changeApiNetwork(String value) {
    final config = _startConfig;
    if (config == null || _apiNetworkDraft == value) return;
    setState(() {
      _apiNetworkDraft = value;
      _syncingStartControllers = true;
      if (value == 'tcp' && (_apiHostController.text.isEmpty || _apiPortController.text.isEmpty)) {
        if (_looksLikeTcpAddress(config.address)) {
          final address = _splitHostPort(config.address);
          _apiHostController.text = address.host;
          _apiPortController.text = address.port;
        } else {
          _apiHostController.text = '127.0.0.1';
          _apiPortController.text = '9999';
        }
      }
      if (value == 'unix' && (_apiAddressController.text.isEmpty || _looksLikeTcpAddress(_apiAddressController.text))) {
        _apiAddressController.text = '${Util.getStorageDir()}/$unixSocketPath';
      }
      _syncingStartControllers = false;
    });
  }

  Future<void> _runAction(Future<void> Function() action) async {
    try {
      await action();
    } catch (error) {
      if (mounted) _toast(error.toString());
    }
  }

  Future<void> _setDefaultBtClient(bool enabled) async {
    try {
      enabled ? registerDefaultTorrentClient() : unregisterDefaultTorrentClient();
      _mutateConfig((config) => config.extra.defaultBtClient = enabled);
    } catch (error) {
      if (mounted) _toast(error.toString());
    }
  }

  Future<void> _setBackgroundLocationKeepAlive(bool enabled) async {
    if (enabled && !await LocationKeepAlive.requestPermission()) {
      if (mounted) _toast(context.l10n.backgroundLocationPermissionRequired);
      return;
    }
    _mutateConfig((config) => config.extra.backgroundLocationKeepAlive = enabled);
    await LocationKeepAliveCoordinator.instance.reconcile(enabled: enabled);
  }

  List<String> _lines(String text) {
    return text.split('\n').map((line) => line.trim()).where((line) => line.isNotEmpty).toList();
  }

  String _formatEd2kMultiline(String value) {
    return value.split(RegExp(r'[,\r\n]+')).map((entry) => entry.trim()).where((entry) => entry.isNotEmpty).join('\n');
  }

  String _categoryName(DownloadCategory category) {
    if (category.name.isNotEmpty) {
      return category.name;
    }
    return switch (category.nameKey) {
      'categoryMusic' => context.l10n.categoryMusic,
      'categoryVideo' => context.l10n.categoryVideo,
      'categoryDocument' => context.l10n.categoryDocument,
      'categoryProgram' => context.l10n.categoryProgram,
      _ => context.l10n.downloadCategories,
    };
  }

  Future<void> _editDownloadCategory([DownloadCategory? category]) async {
    final draft = await showDownloadCategoryDialog(
      context,
      category: category,
      initialName: category == null ? '' : _categoryName(category),
      initialPath: _downloadDirController.text.trim(),
    );
    if (draft == null || !mounted) return;

    _mutateConfig((config) {
      final categories = List<DownloadCategory>.from(config.extra.downloadCategories);
      if (category == null) {
        categories.add(DownloadCategory(name: draft.name, path: draft.path));
      } else {
        final index = categories.indexOf(category);
        if (index < 0) return;
        final nameChanged = draft.name != _categoryName(category);
        category
          ..name = draft.name
          ..path = draft.path
          ..isDeleted = false;
        if (nameChanged) category.nameKey = null;
      }
      config.extra.downloadCategories = categories;
    });
  }

  Future<void> _deleteDownloadCategory(DownloadCategory category) async {
    final confirmed = await showDeleteDownloadCategoryDialog(context, _categoryName(category));
    if (!confirmed || !mounted) return;
    _mutateConfig((config) {
      if (category.isBuiltIn) {
        category.isDeleted = true;
        return;
      }
      config.extra.downloadCategories = List<DownloadCategory>.from(config.extra.downloadCategories)..remove(category);
    });
  }

  List<GithubMirror> get _visibleGithubMirrors =>
      _config?.extra.githubMirror.mirrors.where((mirror) => !mirror.isDeleted).toList(growable: false) ?? const [];

  Future<void> _editGithubMirror([GithubMirror? mirror]) async {
    final draft = await showGithubMirrorDialog(context, mirror: mirror);
    if (draft == null || !mounted) return;
    _mutateConfig((config) {
      final mirrors = List<GithubMirror>.from(config.extra.githubMirror.mirrors);
      if (mirror == null) {
        mirrors.add(GithubMirror(type: draft.type, url: draft.url));
      } else {
        final index = mirrors.indexOf(mirror);
        if (index < 0) return;
        mirrors[index] = GithubMirror(type: draft.type, url: draft.url, isBuiltIn: mirror.isBuiltIn, isDeleted: false);
      }
      config.extra.githubMirror.mirrors = mirrors;
    });
  }

  Future<void> _editGithubMirrorAtVisibleIndex(int index) async {
    final mirrors = _visibleGithubMirrors;
    if (index < 0 || index >= mirrors.length) return;
    await _editGithubMirror(mirrors[index]);
  }

  Future<void> _deleteGithubMirrorAtVisibleIndex(int index) async {
    final mirrors = _visibleGithubMirrors;
    if (index < 0 || index >= mirrors.length) return;
    final mirror = mirrors[index];
    if (!await showDeleteSettingsEntryDialog(context, mirror.url) || !mounted) return;
    _mutateConfig((config) {
      if (mirror.isBuiltIn) {
        mirror.isDeleted = true;
      } else {
        config.extra.githubMirror.mirrors = List<GithubMirror>.from(config.extra.githubMirror.mirrors)..remove(mirror);
      }
    });
  }

  Future<void> _editWebhook([int? index]) async {
    final urls = _config?.webhook.urls ?? const [];
    if (index != null && (index < 0 || index >= urls.length)) return;
    final value = await showTextSettingDialog(
      context,
      title: index == null ? context.l10n.addWebhook : context.l10n.editWebhook,
      fieldLabel: context.l10n.webhookUrlHint,
      initialValue: index == null ? '' : urls[index],
      requireHttpUrl: true,
      onTest: (url) => ref.read(settingsControllerProvider.notifier).testWebhook(url),
    );
    if (value == null || !mounted) return;
    _mutateConfig((config) {
      final next = List<String>.from(config.webhook.urls);
      if (index == null) {
        next.add(value);
      } else {
        next[index] = value;
      }
      config.webhook.urls = next;
    });
  }

  void _deleteWebhook(int index) {
    _mutateConfig((config) {
      if (index < 0 || index >= config.webhook.urls.length) return;
      config.webhook.urls = List<String>.from(config.webhook.urls)..removeAt(index);
    });
  }

  Future<void> _editScript([int? index]) async {
    final paths = _config?.script.paths ?? const [];
    if (index != null && (index < 0 || index >= paths.length)) return;
    final value = await showTextSettingDialog(
      context,
      title: index == null ? context.l10n.addScript : context.l10n.editScript,
      fieldLabel: context.l10n.scriptPath,
      initialValue: index == null ? '' : paths[index],
      pickPath: Util.isDesktop()
          ? () async {
              final result = await FilePicker.platform.pickFiles();
              return result?.files.firstOrNull?.path;
            }
          : null,
    );
    if (value == null || !mounted) return;
    _mutateConfig((config) {
      final next = List<String>.from(config.script.paths);
      if (index == null) {
        next.add(value);
      } else {
        next[index] = value;
      }
      config.script.paths = next;
    });
  }

  void _deleteScript(int index) {
    _mutateConfig((config) {
      if (index < 0 || index >= config.script.paths.length) return;
      config.script.paths = List<String>.from(config.script.paths)..removeAt(index);
    });
  }

  Future<void> _updateTrackers() async {
    final config = _config;
    if (config == null) return;
    await _runAction(() async {
      await ref.read(appRuntimeControllerProvider.notifier).updateTrackers(DownloaderConfig.fromJson(config.toJson()));
      if (mounted) await ref.read(settingsControllerProvider.notifier).reload(showLoading: false);
    });
  }

  String _signature(DownloaderConfig config) {
    return config.toJson().toString();
  }

  String _startSignature(StartConfig config) {
    return '${config.apiEnable}|${config.network}|${config.address}|${config.apiToken}';
  }

  String _startDraftSignature() {
    final network = _apiNetworkDraft ?? _startConfig?.network ?? 'tcp';
    final address = network == 'unix'
        ? _apiAddressController.text.trim()
        : _joinHostPort(_apiHostController.text, _apiPortController.text);
    return '${_startConfig?.apiEnable ?? false}|$network|$address|${_apiTokenController.text.trim()}';
  }

  StartConfig _copyStartConfig(StartConfig config) {
    return StartConfig()
      ..network = config.network
      ..address = config.address
      ..apiEnable = config.apiEnable
      ..storage = config.storage
      ..storageDir = config.storageDir
      ..refreshInterval = config.refreshInterval
      ..apiToken = config.apiToken
      ..webViewRpcConfig = config.webViewRpcConfig;
  }

  String get _apiNetworkValue {
    final network = _apiNetworkDraft ?? _startConfig?.network ?? 'tcp';
    if (network == 'unix' && Util.supportUnixSocket()) {
      return 'unix';
    }
    return 'tcp';
  }

  String get _proxyMode {
    final proxy = _config?.proxy;
    if (proxy == null || !proxy.enable) return 'none';
    return proxy.system ? 'system' : 'custom';
  }

  String get _proxyScheme {
    final scheme = _config?.proxy.scheme;
    return switch (scheme) {
      'https' || 'socks5' => scheme!,
      _ => 'http',
    };
  }

  String get _versionLabel {
    try {
      return packageInfo.version;
    } catch (_) {
      return '-';
    }
  }

  String get _logDirectoryLabel {
    try {
      return logsDir();
    } catch (_) {
      return context.l10n.logDirectoryUnavailable;
    }
  }

  void _openLogDirectory() {
    try {
      unawaited(launchUrl(Uri.file(logsDir()), mode: LaunchMode.externalApplication));
    } catch (error) {
      _toast(error.toString());
    }
  }

  Future<void> _openBrowserExtension(Uri uri) async {
    await _openExternalUri(uri);
  }

  Future<void> _openExternalUri(Uri uri) async {
    try {
      final opened = await launchUrl(uri, mode: LaunchMode.externalApplication);
      if (!opened && mounted) {
        _toast(context.l10n.unableOpenExternalLink);
      }
    } catch (error) {
      if (mounted) _toast(error.toString());
    }
  }

  bool _looksLikeTcpAddress(String address) {
    final separator = address.lastIndexOf(':');
    if (separator <= 0 || separator == address.length - 1) {
      return false;
    }
    return int.tryParse(address.substring(separator + 1)) != null;
  }

  ({String host, String port}) _splitHostPort(String address) {
    final value = address.trim();
    if (value.isEmpty) return (host: '', port: '');

    // Bracketed IPv6 addresses keep their internal colons, for example [::1]:9999.
    if (value.startsWith('[')) {
      final closingBracket = value.indexOf(']');
      if (closingBracket > 0 && closingBracket + 1 < value.length && value[closingBracket + 1] == ':') {
        return (host: value.substring(0, closingBracket + 1), port: value.substring(closingBracket + 2));
      }
    }

    final separator = value.lastIndexOf(':');
    if (separator <= 0 || separator == value.length - 1) {
      return (host: value, port: '');
    }
    return (host: value.substring(0, separator), port: value.substring(separator + 1));
  }

  String _joinHostPort(String hostText, String portText) {
    final host = hostText.trim();
    final port = portText.trim();
    if (host.isEmpty && port.isEmpty) return '';
    if (port.isEmpty) return host;
    return '$host:$port';
  }

  int _boundedInt(TextEditingController controller, {required int fallback, required int min, required int max}) {
    final value = int.tryParse(controller.text.trim());
    return (value ?? fallback).clamp(min, max).toInt();
  }

  String _sectionTitle(_SettingsSection section) {
    return switch (section) {
      _SettingsSection.basic => context.l10n.basicSettings,
      _SettingsSection.downloads => context.l10n.downloadSettings,
      _SettingsSection.advanced => context.l10n.advancedSettings,
    };
  }

  String? _apiServerRuntimeDetails(AppRuntimeState? runtimeState) {
    if (runtimeState == null) return null;
    final apiState = runtimeState.apiServerState;
    final details = <String>[];
    if (apiState.running) {
      details.add('${apiState.network}://${apiState.runningAddress()}');
    }
    if (apiState.lastError.isNotEmpty) {
      details.add(apiState.lastError);
    } else if (apiState.pendingApply) {
      details.add(context.l10n.apiConfigPendingApply);
    }
    return details.isEmpty ? null : details.join('\n');
  }

  _ApiServerVisualState _apiServerVisualState(AppRuntimeState? runtimeState) {
    if (runtimeState == null) return _ApiServerVisualState.loading;
    final apiState = runtimeState.apiServerState;
    if (apiState.running) return _ApiServerVisualState.running;
    if (apiState.lastError.isNotEmpty) return _ApiServerVisualState.failed;
    return _ApiServerVisualState.stopped;
  }

  String _languageValue(DownloaderConfig config) {
    final locale = supportedLocaleFromConfig(config.extra.locale);
    return locale == null ? 'system' : localeConfigValue(locale);
  }

  void _toast(String message, {AppToastType type = AppToastType.error}) {
    showAppToast(context, message, type: type);
  }
}

enum _ApiServerVisualState { loading, running, stopped, failed }

class _ApiServerStatusLight extends StatefulWidget {
  const _ApiServerStatusLight({super.key, required this.state, required this.label});

  final _ApiServerVisualState state;
  final String label;

  @override
  State<_ApiServerStatusLight> createState() => _ApiServerStatusLightState();
}

class _ApiServerStatusLightState extends State<_ApiServerStatusLight> with SingleTickerProviderStateMixin {
  late final AnimationController _pulseController;
  bool _disableAnimations = false;

  @override
  void initState() {
    super.initState();
    _pulseController = AnimationController(vsync: this, duration: const Duration(milliseconds: 1400), value: 0.35);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _disableAnimations = MediaQuery.maybeOf(context)?.disableAnimations ?? false;
    _syncPulseAnimation();
  }

  @override
  void didUpdateWidget(covariant _ApiServerStatusLight oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.state != widget.state) _syncPulseAnimation();
  }

  void _syncPulseAnimation() {
    final shouldPulse =
        !_disableAnimations &&
        (widget.state == _ApiServerVisualState.running || widget.state == _ApiServerVisualState.loading);
    if (shouldPulse) {
      if (!_pulseController.isAnimating) _pulseController.repeat(reverse: true);
      return;
    }
    _pulseController
      ..stop()
      ..value = switch (widget.state) {
        _ApiServerVisualState.running => 0.55,
        _ApiServerVisualState.loading => 0.3,
        _ApiServerVisualState.failed => 0.5,
        _ApiServerVisualState.stopped => 0,
      };
  }

  @override
  void dispose() {
    _pulseController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final color = switch (widget.state) {
      _ApiServerVisualState.running => palette.success,
      _ApiServerVisualState.failed => palette.error,
      _ApiServerVisualState.loading || _ApiServerVisualState.stopped => palette.textMuted,
    };
    final labelColor = widget.state == _ApiServerVisualState.stopped ? palette.textSecondary : color;
    return Semantics(
      label: widget.label,
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          AnimatedBuilder(
            animation: _pulseController,
            builder: (context, child) {
              final pulse = Curves.easeInOut.transform(_pulseController.value);
              final glowSize = switch (widget.state) {
                _ApiServerVisualState.running => 12.0 + pulse * 8,
                _ApiServerVisualState.loading => 10.0 + pulse * 5,
                _ApiServerVisualState.failed => 16.0,
                _ApiServerVisualState.stopped => 10.0,
              };
              final glowAlpha = switch (widget.state) {
                _ApiServerVisualState.running => 0.08 + pulse * 0.12,
                _ApiServerVisualState.loading => 0.05 + pulse * 0.06,
                _ApiServerVisualState.failed => 0.13,
                _ApiServerVisualState.stopped => 0.08,
              };
              final glowShadow = switch (widget.state) {
                _ApiServerVisualState.running => [
                  BoxShadow(
                    color: color.withValues(alpha: 0.18 + pulse * 0.2),
                    blurRadius: 4 + pulse * 8,
                    spreadRadius: pulse * 1.5,
                  ),
                ],
                _ApiServerVisualState.failed => [
                  BoxShadow(color: color.withValues(alpha: 0.28), blurRadius: 7, spreadRadius: 0.5),
                ],
                _ApiServerVisualState.loading || _ApiServerVisualState.stopped => const <BoxShadow>[],
              };
              return SizedBox.square(
                dimension: 24,
                child: Stack(
                  alignment: Alignment.center,
                  children: [
                    Container(
                      key: const ValueKey('api-server-status-glow'),
                      width: glowSize,
                      height: glowSize,
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        color: color.withValues(alpha: glowAlpha),
                        boxShadow: glowShadow,
                      ),
                    ),
                    child!,
                  ],
                ),
              );
            },
            child: AnimatedContainer(
              key: const ValueKey('api-server-status-dot'),
              duration: const Duration(milliseconds: 300),
              width: widget.state == _ApiServerVisualState.loading ? 6 : 8,
              height: widget.state == _ApiServerVisualState.loading ? 6 : 8,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: widget.state == _ApiServerVisualState.stopped ? color.withValues(alpha: 0.72) : color,
              ),
            ),
          ),
          const SizedBox(width: 6),
          Text(
            widget.label,
            style: TextStyle(color: labelColor, fontSize: 12, fontWeight: FontWeight.w700),
          ),
        ],
      ),
    );
  }
}

class _SettingsGroup extends StatelessWidget {
  const _SettingsGroup({required this.children});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(
      decoration: BoxDecoration(
        color: palette.cardBg,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: palette.border),
      ),
      child: ListView.separated(
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        padding: const EdgeInsets.all(16),
        itemCount: children.length,
        separatorBuilder: (_, _) =>
            Container(height: 1, margin: const EdgeInsets.symmetric(vertical: 12), color: palette.border),
        itemBuilder: (context, index) => children[index],
      ),
    );
  }
}

class _SettingsSectionStack extends StatelessWidget {
  const _SettingsSectionStack({required this.children});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (var index = 0; index < children.length; index++) ...[
          if (index > 0) const SizedBox(height: 18),
          children[index],
        ],
      ],
    );
  }
}

class _SettingsBlock extends StatelessWidget {
  const _SettingsBlock({super.key, required this.title, required this.child});

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(left: 2, bottom: 8),
          child: Text(
            title,
            style: TextStyle(color: palette.textSecondary, fontSize: 12, fontWeight: FontWeight.w700),
          ),
        ),
        child,
      ],
    );
  }
}

class _TextSettingControl extends StatelessWidget {
  const _TextSettingControl({
    required this.controller,
    this.fieldKey,
    this.hintText,
    this.minLines = 1,
    this.obscureText = false,
  });

  final TextEditingController controller;
  final Key? fieldKey;
  final String? hintText;
  final int minLines;
  final bool obscureText;

  @override
  Widget build(BuildContext context) {
    final desktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    return SizedBox(
      width: desktop ? AppDesignTokens.settingsFormControlWidth : double.infinity,
      child: shad.TextField(
        key: fieldKey,
        controller: controller,
        hintText: hintText,
        keyboardType: minLines > 1 ? TextInputType.multiline : null,
        obscureText: obscureText,
        minLines: minLines,
        maxLines: minLines == 1 ? 1 : minLines,
      ),
    );
  }
}

class _NumberSettingControl extends StatelessWidget {
  const _NumberSettingControl({
    required this.controller,
    required this.min,
    this.fieldKey,
    this.max,
    this.step = 1,
    this.decimalPlaces = 0,
  });

  final TextEditingController controller;
  final num min;
  final Key? fieldKey;
  final num? max;
  final double step;
  final int decimalPlaces;

  @override
  Widget build(BuildContext context) {
    final desktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    return SizedBox(
      width: desktop ? AppDesignTokens.settingsNumberControlWidth : double.infinity,
      child: _NumberInput(
        fieldKey: fieldKey,
        controller: controller,
        min: min,
        max: max,
        step: step,
        decimalPlaces: decimalPlaces,
      ),
    );
  }
}

class _HostPortControl extends StatelessWidget {
  const _HostPortControl({
    required this.hostController,
    required this.portController,
    required this.hostKey,
    required this.portKey,
  });

  final TextEditingController hostController;
  final TextEditingController portController;
  final Key hostKey;
  final Key portKey;

  @override
  Widget build(BuildContext context) {
    final desktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    return SizedBox(
      width: desktop ? AppDesignTokens.settingsFormControlWidth : double.infinity,
      child: Row(
        children: [
          Expanded(
            child: shad.TextField(key: hostKey, controller: hostController, hintText: context.l10n.server),
          ),
          const SizedBox(width: AppDesignTokens.space8),
          SizedBox(
            width: AppDesignTokens.settingsNumberControlWidth,
            child: _NumberInput(
              fieldKey: portKey,
              controller: portController,
              min: 0,
              max: 65535,
              step: 1,
              decimalPlaces: 0,
              hintText: context.l10n.port,
            ),
          ),
        ],
      ),
    );
  }
}

class _NumberInput extends StatefulWidget {
  const _NumberInput({
    required this.controller,
    required this.min,
    required this.step,
    required this.decimalPlaces,
    this.fieldKey,
    this.max,
    this.hintText,
  });

  final TextEditingController controller;
  final num min;
  final num? max;
  final double step;
  final int decimalPlaces;
  final Key? fieldKey;
  final String? hintText;

  @override
  State<_NumberInput> createState() => _NumberInputState();
}

class _NumberInputState extends State<_NumberInput> {
  bool _normalizing = false;

  @override
  void initState() {
    super.initState();
    widget.controller.addListener(_normalizeDecimalStepValue);
  }

  @override
  void didUpdateWidget(covariant _NumberInput oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller == widget.controller) return;
    oldWidget.controller.removeListener(_normalizeDecimalStepValue);
    widget.controller.addListener(_normalizeDecimalStepValue);
  }

  @override
  void dispose() {
    widget.controller.removeListener(_normalizeDecimalStepValue);
    super.dispose();
  }

  void _normalizeDecimalStepValue() {
    if (_normalizing || widget.decimalPlaces <= 0) return;
    final text = widget.controller.text;
    final separator = text.indexOf('.');
    if (separator < 0 || text.length - separator - 1 <= widget.decimalPlaces) return;
    final value = double.tryParse(text);
    if (value == null) return;

    var normalized = value.toStringAsFixed(widget.decimalPlaces);
    normalized = normalized.replaceFirst(RegExp(r'0+$'), '').replaceFirst(RegExp(r'\.$'), '');
    if (normalized == text) return;

    _normalizing = true;
    widget.controller.value = TextEditingValue(
      text: normalized,
      selection: TextSelection.collapsed(offset: normalized.length),
    );
    _normalizing = false;
  }

  @override
  Widget build(BuildContext context) {
    final formatters = widget.decimalPlaces > 0
        ? <TextInputFormatter>[_DecimalNumberFormatter(decimalPlaces: widget.decimalPlaces)]
        : <TextInputFormatter>[
            FilteringTextInputFormatter.digitsOnly,
            _NumericalRangeFormatter(min: widget.min.toInt(), max: widget.max?.toInt()),
          ];
    return shad.TextField(
      key: widget.fieldKey,
      controller: widget.controller,
      hintText: widget.hintText,
      keyboardType: widget.decimalPlaces > 0
          ? const TextInputType.numberWithOptions(decimal: true)
          : TextInputType.number,
      inputFormatters: formatters,
      features: [
        shad.InputFeature.spinner(
          step: widget.step,
          min: widget.min.toDouble(),
          max: widget.max?.toDouble(),
          invalidValue: widget.min.toDouble(),
          enableGesture: false,
        ),
      ],
    );
  }
}

/// Keeps integer text within the same inclusive bounds as the legacy settings UI.
/// Empty text is allowed while editing; the previous valid value remains in config
/// until the user enters a number again.
class _NumericalRangeFormatter extends TextInputFormatter {
  const _NumericalRangeFormatter({required this.min, this.max});

  final int min;
  final int? max;

  @override
  TextEditingValue formatEditUpdate(TextEditingValue oldValue, TextEditingValue newValue) {
    if (newValue.text.isEmpty) return newValue;
    final parsed = int.tryParse(newValue.text);
    if (parsed == null) return oldValue;
    final bounded = parsed.clamp(min, max ?? parsed).toInt();
    if (bounded == parsed) return newValue;
    final text = bounded.toString();
    return TextEditingValue(
      text: text,
      selection: TextSelection.collapsed(offset: text.length),
    );
  }
}

/// Allows a non-negative decimal number with a bounded number of fractional digits.
class _DecimalNumberFormatter extends TextInputFormatter {
  _DecimalNumberFormatter({required this.decimalPlaces}) : _pattern = RegExp('^\\d*(?:\\.\\d{0,$decimalPlaces})?\$');

  final int decimalPlaces;
  final RegExp _pattern;

  @override
  TextEditingValue formatEditUpdate(TextEditingValue oldValue, TextEditingValue newValue) {
    return _pattern.hasMatch(newValue.text) ? newValue : oldValue;
  }
}

class _ExternalTextLink extends StatefulWidget {
  const _ExternalTextLink({super.key, required this.label, required this.onPressed});

  final String label;
  final VoidCallback onPressed;

  @override
  State<_ExternalTextLink> createState() => _ExternalTextLinkState();
}

class _ExternalTextLinkState extends State<_ExternalTextLink> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final color = _hovered ? palette.brandProgress : palette.brand;
    return Semantics(
      link: true,
      label: widget.label,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        onEnter: (_) => setState(() => _hovered = true),
        onExit: (_) => setState(() => _hovered = false),
        child: GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTap: widget.onPressed,
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 2),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Flexible(
                  child: Text(
                    widget.label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: color,
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                      decoration: TextDecoration.underline,
                      decorationColor: color,
                      decorationThickness: _hovered ? 1.5 : 1,
                    ),
                  ),
                ),
                const SizedBox(width: 5),
                Icon(Icons.open_in_new, size: 13, color: color),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _BrowserExtensionLinks extends StatelessWidget {
  const _BrowserExtensionLinks({required this.onOpen});

  static final _links = <({String label, Uri uri})>[
    (
      label: 'Chrome',
      uri: Uri.parse('https://chromewebstore.google.com/detail/gopeed/mijpgljlfcapndmchhjffkpckknofcnd'),
    ),
    (
      label: 'Edge',
      uri: Uri.parse('https://microsoftedge.microsoft.com/addons/detail/dkajnckekendchdleoaenoophcobooce'),
    ),
    (label: 'Firefox', uri: Uri.parse('https://addons.mozilla.org/firefox/addon/gopeed-extension/')),
  ];

  final ValueChanged<Uri> onOpen;

  @override
  Widget build(BuildContext context) {
    final desktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      alignment: desktop ? WrapAlignment.end : WrapAlignment.start,
      children: [
        for (final link in _links)
          shad.SecondaryButton(
            key: ValueKey('browser-extension-${link.label.toLowerCase()}'),
            onPressed: () => onOpen(link.uri),
            leading: const Icon(Icons.open_in_new, size: 15),
            child: Text(link.label),
          ),
      ],
    );
  }
}

class _TrackerSubscriptionsControl extends StatefulWidget {
  const _TrackerSubscriptionsControl({
    required this.selected,
    required this.autoUpdate,
    required this.lastUpdated,
    required this.onChanged,
    required this.onAutoUpdateChanged,
    required this.onUpdate,
  });

  final List<String> selected;
  final bool autoUpdate;
  final DateTime? lastUpdated;
  final ValueChanged<List<String>> onChanged;
  final ValueChanged<bool> onAutoUpdateChanged;
  final Future<void> Function() onUpdate;

  @override
  State<_TrackerSubscriptionsControl> createState() => _TrackerSubscriptionsControlState();
}

class _TrackerSubscriptionsControlState extends State<_TrackerSubscriptionsControl> {
  final _verticalScrollController = ScrollController();
  final _horizontalScrollController = ScrollController();
  bool _updating = false;

  @override
  void dispose() {
    _verticalScrollController.dispose();
    _horizontalScrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final desktop = MediaQuery.sizeOf(context).width >= Breakpoints.mobile;
    final palette = AppPalette.of(context);
    final selectedCount = allTrackerSubscribeUrls.where(widget.selected.contains).length;
    final allSelected = selectedCount == allTrackerSubscribeUrls.length;
    final selectAllState = allSelected
        ? shad.CheckboxState.checked
        : selectedCount == 0
        ? shad.CheckboxState.unchecked
        : shad.CheckboxState.indeterminate;
    return SizedBox(
      width: desktop ? AppDesignTokens.settingsFormControlWidth : double.infinity,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Container(
            height: 210,
            decoration: BoxDecoration(
              border: Border.all(color: palette.border),
              borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
            ),
            child: Column(
              children: [
                GestureDetector(
                  key: const ValueKey('tracker-select-all'),
                  behavior: HitTestBehavior.opaque,
                  onTap: () => _toggleAll(allSelected),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                    child: Row(
                      children: [
                        shad.Checkbox(state: selectAllState, onChanged: (_) => _toggleAll(allSelected)),
                        const SizedBox(width: AppDesignTokens.checkboxLabelGap),
                        Expanded(
                          child: Text(
                            context.l10n.selectAll,
                            style: TextStyle(color: palette.textPrimary, fontSize: 12, fontWeight: FontWeight.w600),
                          ),
                        ),
                        Text(
                          '$selectedCount/${allTrackerSubscribeUrls.length}',
                          style: TextStyle(color: palette.textMuted, fontSize: 11),
                        ),
                      ],
                    ),
                  ),
                ),
                Divider(height: 1, color: palette.border),
                Expanded(
                  child: LayoutBuilder(
                    builder: (context, constraints) {
                      final labelStyle = TextStyle(color: palette.textSecondary, fontSize: 11);
                      final contentHeight = (constraints.maxHeight - AppDesignTokens.space12).clamp(
                        0.0,
                        double.infinity,
                      );
                      return Scrollbar(
                        key: const ValueKey('tracker-horizontal-scrollbar'),
                        controller: _horizontalScrollController,
                        thumbVisibility: true,
                        scrollbarOrientation: ScrollbarOrientation.bottom,
                        notificationPredicate: (notification) => notification.metrics.axis == Axis.horizontal,
                        child: Padding(
                          padding: const EdgeInsets.only(bottom: AppDesignTokens.space12),
                          child: SingleChildScrollView(
                            key: const ValueKey('tracker-horizontal-scroll-view'),
                            controller: _horizontalScrollController,
                            scrollDirection: Axis.horizontal,
                            child: SizedBox(
                              height: contentHeight,
                              child: Scrollbar(
                                key: const ValueKey('tracker-vertical-scrollbar'),
                                controller: _verticalScrollController,
                                thumbVisibility: true,
                                scrollbarOrientation: ScrollbarOrientation.right,
                                notificationPredicate: (notification) => notification.metrics.axis == Axis.vertical,
                                child: SingleChildScrollView(
                                  key: const ValueKey('tracker-vertical-scroll-view'),
                                  controller: _verticalScrollController,
                                  child: IntrinsicWidth(
                                    child: ConstrainedBox(
                                      constraints: BoxConstraints(minWidth: constraints.maxWidth),
                                      child: Padding(
                                        padding: const EdgeInsets.symmetric(vertical: 4),
                                        child: Column(
                                          crossAxisAlignment: CrossAxisAlignment.stretch,
                                          children: [
                                            for (final (index, url) in allTrackerSubscribeUrls.indexed) ...[
                                              Builder(
                                                builder: (context) {
                                                  final checked = widget.selected.contains(url);
                                                  return GestureDetector(
                                                    behavior: HitTestBehavior.opaque,
                                                    onTap: () => _toggle(url, checked),
                                                    child: Padding(
                                                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                                                      child: Row(
                                                        children: [
                                                          shad.Checkbox(
                                                            state: checked
                                                                ? shad.CheckboxState.checked
                                                                : shad.CheckboxState.unchecked,
                                                            onChanged: (_) => _toggle(url, checked),
                                                          ),
                                                          const SizedBox(width: AppDesignTokens.checkboxLabelGap),
                                                          Text(url, maxLines: 1, style: labelStyle),
                                                        ],
                                                      ),
                                                    ),
                                                  );
                                                },
                                              ),
                                              if (index != allTrackerSubscribeUrls.length - 1)
                                                Divider(height: 1, color: palette.border),
                                            ],
                                          ],
                                        ),
                                      ),
                                    ),
                                  ),
                                ),
                              ),
                            ),
                          ),
                        ),
                      );
                    },
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              AppLoadingButton(
                key: const ValueKey('tracker-update-button'),
                onPressed: _update,
                loading: _updating,
                icon: const Icon(Icons.refresh, size: 17),
                child: Text(context.l10n.newVersionUpdate),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  context.l10n.updateEveryDay,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  textAlign: TextAlign.end,
                  style: TextStyle(color: palette.textSecondary, fontSize: 12),
                ),
              ),
              const SizedBox(width: 8),
              shad.Switch(value: widget.autoUpdate, onChanged: widget.onAutoUpdateChanged),
            ],
          ),
          const SizedBox(height: 6),
          Text(
            widget.lastUpdated == null
                ? context.l10n.neverUpdated
                : context.l10n.lastUpdated(_formatDateTime(widget.lastUpdated!)),
            style: TextStyle(color: palette.textMuted, fontSize: 11),
          ),
        ],
      ),
    );
  }

  void _toggle(String url, bool checked) {
    final next = List<String>.from(widget.selected);
    checked ? next.remove(url) : next.add(url);
    widget.onChanged(next);
  }

  void _toggleAll(bool allSelected) {
    widget.onChanged(allSelected ? const [] : List<String>.from(allTrackerSubscribeUrls));
  }

  Future<void> _update() async {
    setState(() => _updating = true);
    try {
      await widget.onUpdate();
    } finally {
      if (mounted) setState(() => _updating = false);
    }
  }

  String _formatDateTime(DateTime value) {
    String two(int number) => number.toString().padLeft(2, '0');
    return '${value.year}-${two(value.month)}-${two(value.day)} ${two(value.hour)}:${two(value.minute)}:${two(value.second)}';
  }
}

class _UpdateActionControl extends ConsumerWidget {
  const _UpdateActionControl({required this.platform});

  final AppPlatformState? platform;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final status = platform?.updateStatus ?? AppUpdateStatus.idle;
    final latest = platform?.latestVersion;
    final checking = status == AppUpdateStatus.checking;

    Future<void> handlePressed() async {
      var nextVersion = latest;
      if (status != AppUpdateStatus.available || nextVersion == null) {
        nextVersion = await ref.read(appPlatformControllerProvider.notifier).checkForUpdate();
      }
      if (!context.mounted || nextVersion == null) return;
      await showAppUpdateDialog(
        context,
        versionInfo: nextVersion,
        onUpdate: ref.read(appPlatformControllerProvider.notifier).startUpdate,
      );
    }

    return _UpdateActionButton(
      available: status == AppUpdateStatus.available,
      checking: checking,
      onPressed: () => unawaited(handlePressed()),
    );
  }
}

class _UpdateActionButton extends StatelessWidget {
  const _UpdateActionButton({required this.available, required this.checking, required this.onPressed});

  final bool available;
  final bool checking;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return AppLoadingButton(
      key: const ValueKey('check-app-update-button'),
      onPressed: onPressed,
      loading: checking,
      icon: Icon(available ? Icons.system_update_alt_outlined : Icons.refresh, size: 17),
      variant: available ? AppLoadingButtonVariant.brand : AppLoadingButtonVariant.secondary,
      child: Text(available ? context.l10n.updateVersion : context.l10n.checkForUpdates),
    );
  }
}

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.error, required this.onRetry});

  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.settings_outlined, size: 32, color: palette.error),
            const SizedBox(height: 12),
            Text(
              context.l10n.unableLoadSettings,
              style: TextStyle(color: palette.textPrimary, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 8),
            Text(
              error.toString(),
              textAlign: TextAlign.center,
              style: TextStyle(color: palette.textSecondary),
            ),
            const SizedBox(height: 16),
            shad.SecondaryButton(onPressed: onRetry, child: Text(context.l10n.retry)),
          ],
        ),
      ),
    );
  }
}
