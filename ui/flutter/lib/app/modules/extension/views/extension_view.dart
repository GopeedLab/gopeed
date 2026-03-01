import 'dart:io';

import 'package:badges/badges.dart' as badges;
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:form_builder_validators/form_builder_validators.dart';
import 'package:get/get.dart';
import 'package:path/path.dart' as path;
import 'package:rounded_loading_button_plus/rounded_loading_button.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../api/api.dart';
import '../../../../api/model/extension.dart';
import '../../../../api/model/install_extension.dart';
import '../../../../api/model/switch_extension.dart';
import '../../../../api/model/update_extension_settings.dart';
import '../../../../database/database.dart';
import '../../../../util/gopeed_api.dart';
import '../../../../util/message.dart';
import '../../../../util/util.dart';
import '../../../views/responsive_builder.dart';
import '../../../views/text_button_loading.dart';
import '../controllers/extension_controller.dart';

class ExtensionView extends GetView<ExtensionController> {
  ExtensionView({Key? key}) : super(key: key);

  final _installUrlController = TextEditingController();
  final _storeSearchController = TextEditingController();

  // --------------------------------------------------------------------------
  // Install from URL (manual)
  // --------------------------------------------------------------------------

  /// Auto-install triggered by a deep-link / route argument (no dialog).
  Future<void> _doInstall() async {
    if (_installUrlController.text.isEmpty) {
      controller.tryOpenDevMode();
      return;
    }
    try {
      final identity = await installExtension(
          InstallExtension(url: _installUrlController.text));
      gopeedReportExtensionInstall(identity);
      showMessage('tip'.tr, 'extensionInstallSuccess'.tr);
      await controller.load();
    } catch (e) {
      showErrorMessage(e);
    }
  }

  /// Manual URL install – shown when the user taps the menu item.
  Future<void> _showInstallUrlDialog() async {
    final btnController = RoundedLoadingButtonController();
    _installUrlController.clear();
    return showDialog<void>(
      context: Get.context!,
      barrierDismissible: false,
      builder: (dialogContext) => AlertDialog(
        title: Text('extensionInstallUrl'.tr),
        content: TextField(
          controller: _installUrlController,
          autofocus: true,
          decoration: InputDecoration(labelText: 'extensionInstallUrl'.tr),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(),
            child: Text('cancel'.tr),
          ),
          ConstrainedBox(
            constraints: BoxConstraints.tightFor(
              width: Get.theme.buttonTheme.minWidth,
              height: Get.theme.buttonTheme.height,
            ),
            child: RoundedLoadingButton(
              color: Get.theme.colorScheme.secondary,
              controller: btnController,
              onPressed: () async {
                if (_installUrlController.text.isEmpty) {
                  controller.tryOpenDevMode();
                  Navigator.of(dialogContext).pop();
                  return;
                }
                btnController.start();
                try {
                  final identity = await installExtension(
                      InstallExtension(url: _installUrlController.text));
                  gopeedReportExtensionInstall(identity);
                  showMessage('tip'.tr, 'extensionInstallSuccess'.tr);
                  await controller.load();
                  if (dialogContext.mounted) {
                    Navigator.of(dialogContext).pop();
                  }
                } catch (e) {
                  showErrorMessage(e);
                } finally {
                  btnController.reset();
                }
              },
              child: Text('confirm'.tr),
            ),
          ),
        ],
      ),
    );
  }

  // --------------------------------------------------------------------------
  // Build
  // --------------------------------------------------------------------------

  @override
  Widget build(BuildContext context) {
    final args = Get.rootDelegate.arguments();
    if (args is InstallExtension && !controller.pendingInstallHandled) {
      controller.pendingInstallHandled = true;
      _installUrlController.text = args.url;
      WidgetsBinding.instance.addPostFrameCallback((_) => _doInstall());
    }

    return Scaffold(
      backgroundColor: Colors.transparent,
      body: Padding(
        padding: EdgeInsets.symmetric(
            horizontal: ResponsiveBuilder.isNarrow(context) ? 16 : 32,
            vertical: 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // ── Header Row ────────────────────────────────────────────────────
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                MouseRegion(
                  cursor: SystemMouseCursors.click,
                  child: GestureDetector(
                    onTap: controller.tryOpenDevMode,
                    child: Text(
                      'extensions'.tr,
                      style: const TextStyle(
                          fontSize: 24, fontWeight: FontWeight.bold),
                    ),
                  ),
                ),
                Expanded(
                  child: Wrap(
                    alignment: WrapAlignment.end,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    spacing: 8,
                    children: [
                      TextButton.icon(
                        style: TextButton.styleFrom(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 12, vertical: 8)),
                        onPressed: () => launchUrl(Uri.parse(
                            'https://docs.gopeed.com/dev-extension.html')),
                        icon: const Icon(Icons.code, size: 16),
                        label: Text('extensionDevelop'.tr,
                            style: const TextStyle(fontSize: 13)),
                      ),
                      Tooltip(
                        message: 'extensionInstallUrl'.tr,
                        child: InkWell(
                          borderRadius: BorderRadius.circular(8),
                          onTap: _showInstallUrlDialog,
                          child: const Padding(
                            padding: EdgeInsets.all(8.0),
                            child: Icon(Icons.add_link, size: 20),
                          ),
                        ),
                      ),
                      Obx(() => controller.devMode.value && Util.isDesktop()
                          ? Tooltip(
                              message: 'extensionInstallLocal'.tr,
                              child: InkWell(
                                borderRadius: BorderRadius.circular(8),
                                onTap: () async {
                                  final dir = await FilePicker.platform
                                      .getDirectoryPath();
                                  if (dir != null) {
                                    try {
                                      await installExtension(InstallExtension(
                                          devMode: true, url: dir));
                                      showMessage('tip'.tr,
                                          'extensionInstallSuccess'.tr);
                                      await controller.load();
                                    } catch (e) {
                                      showErrorMessage(e);
                                    }
                                  }
                                },
                                child: const Padding(
                                  padding: EdgeInsets.all(8.0),
                                  child: Icon(Icons.folder_open, size: 20),
                                ),
                              ),
                            )
                          : const SizedBox.shrink()),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),

            // ── Search & Filter Row ───────────────────────────────────────────
            Builder(builder: (context) {
              final searchBox = TextField(
                controller: _storeSearchController,
                decoration: InputDecoration(
                  isDense: true,
                  hintText: 'extensionSearch'.tr,
                  prefixIcon: const Icon(Icons.search, size: 20),
                  suffixIcon: Obx(() => controller.storeQuery.value.isNotEmpty
                      ? IconButton(
                          icon: const Icon(Icons.clear, size: 18),
                          onPressed: () {
                            _storeSearchController.clear();
                            controller.onStoreQueryChanged('');
                          },
                        )
                      : const SizedBox.shrink()),
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(30),
                    borderSide: BorderSide.none,
                  ),
                  filled: true,
                  contentPadding:
                      const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
                ),
                onChanged: controller.onStoreQueryChanged,
              );

              if (ResponsiveBuilder.isNarrow(context)) {
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    searchBox,
                    const SizedBox(height: 12),
                    Row(
                      children: [
                        Expanded(child: _buildFilterDropdown(isExpanded: true)),
                        const SizedBox(width: 8),
                        Expanded(child: _buildSortDropdown(isExpanded: true)),
                      ],
                    ),
                  ],
                );
              } else {
                return Row(
                  children: [
                    Expanded(child: searchBox),
                    const SizedBox(width: 16),
                    _buildFilterDropdown(),
                    const SizedBox(width: 8),
                    _buildSortDropdown(),
                  ],
                );
              }
            }),

            const SizedBox(height: 16),

            // ── Unified list ──────────────────────────────────────────────────
            Expanded(child: _buildList(context)),
          ],
        ),
      ),
    );
  }

  // --------------------------------------------------------------------------
  // Toolbar widgets
  // --------------------------------------------------------------------------

  Widget _buildFilterDropdown({bool isExpanded = false}) {
    return Container(
      decoration: BoxDecoration(
        color: Get.theme.canvasColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Get.theme.dividerColor, width: 1),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: DropdownButtonHideUnderline(
        child: DropdownButton<ExtensionFilter>(
          value: controller.extensionFilter.value,
          isDense: true,
          isExpanded: isExpanded,
          icon: const Icon(Icons.arrow_drop_down, size: 20),
          items: ExtensionFilter.values
              .map((f) => DropdownMenuItem(
                    value: f,
                    child: Text(_filterLabel(f),
                        style: const TextStyle(fontSize: 13)),
                  ))
              .toList(),
          onChanged: (f) {
            if (f != null) controller.extensionFilter.value = f;
          },
        ),
      ),
    );
  }

  Widget _buildSortDropdown({bool isExpanded = false}) {
    final field = controller.storeSortField.value;
    return Container(
      decoration: BoxDecoration(
        color: Get.theme.canvasColor,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Get.theme.dividerColor, width: 1),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: DropdownButtonHideUnderline(
        child: DropdownButton<GopeedExtensionSortField>(
          value: field,
          isDense: true,
          isExpanded: isExpanded,
          icon: const Icon(Icons.arrow_drop_down, size: 20),
          items: GopeedExtensionSortField.values
              .map((f) => DropdownMenuItem(
                    value: f,
                    child: Text(_sortFieldLabel(f),
                        style: const TextStyle(fontSize: 13)),
                  ))
              .toList(),
          onChanged: (f) {
            if (f != null) {
              // Tapping the same field twice no longer toggles direction;
              // we instead just ensure everything is simple descending for store.
              controller.storeSortField.value = f;
              controller.storeSortOrder.value = GopeedExtensionSortOrder.desc;
              controller.loadStore(reset: true);
            }
          },
        ),
      ),
    );
  }

  String _filterLabel(ExtensionFilter f) {
    switch (f) {
      case ExtensionFilter.all:
        return 'extensionFilterAll'.tr;
      case ExtensionFilter.installed:
        return 'extensionFilterInstalled'.tr;
      case ExtensionFilter.notInstalled:
        return 'extensionFilterNotInstalled'.tr;
    }
  }

  String _sortFieldLabel(GopeedExtensionSortField field) {
    switch (field) {
      case GopeedExtensionSortField.stars:
        return 'extensionSortStars'.tr;
      case GopeedExtensionSortField.installs:
        return 'extensionSortInstalls'.tr;
      case GopeedExtensionSortField.updated:
        return 'extensionSortUpdated'.tr;
    }
  }

  // --------------------------------------------------------------------------
  // Unified list
  // --------------------------------------------------------------------------

  Widget _buildList(BuildContext context) {
    return Obx(() {
      // Accessing both observables so Obx rebuilds when either changes.
      final storeItems = controller.storeExtensions.toList();
      final installedExts = controller.extensions.toList();
      final filter = controller.extensionFilter.value;
      final loading = controller.storeLoading.value;
      final hasNext = controller.storeHasNext.value;
      final _ = controller.storeInstalling.length; // track installing state

      // Build a set of store extension ids for fast lookup.
      final storeIds = storeItems.map((e) => e.id).toSet();

      // Build a map: storeId → local Extension (if installed).
      final installedById = {
        for (final e in installedExts) e.identity: e,
      };

      // Compute visible items based on filter.
      final List<_ExtItem> items = [];

      for (final storeExt in storeItems) {
        final installedExt = installedById[storeExt.id];
        final isInstalled = installedExt != null;
        if (filter == ExtensionFilter.installed && !isInstalled) continue;
        if (filter == ExtensionFilter.notInstalled && isInstalled) continue;
        items.add(_ExtItem(storeExt: storeExt, installedExt: installedExt));
      }

      // Local-only extensions (not from store) – always show unless filter=notInstalled.
      if (filter != ExtensionFilter.notInstalled) {
        for (final ext in installedExts) {
          if (!storeIds.contains(ext.identity)) {
            items.add(_ExtItem(installedExt: ext));
          }
        }
      }

      if (items.isEmpty && loading) {
        return const Center(child: CircularProgressIndicator());
      }
      if (items.isEmpty) {
        return Center(child: Text('extensionNoResults'.tr));
      }

      return ListView.builder(
        itemCount: items.length + (hasNext ? 1 : 0),
        itemBuilder: (context, index) {
          if (index == items.length) {
            return _buildLoadMoreButton(loading);
          }
          return _buildCard(context, items[index]);
        },
      );
    });
  }

  Widget _buildLoadMoreButton(bool loading) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Center(
        child: loading
            ? const SizedBox(
                width: 24,
                height: 24,
                child: CircularProgressIndicator(strokeWidth: 2))
            : OutlinedButton(
                onPressed: controller.loadMoreStore,
                child: Text('extensionLoadMore'.tr),
              ),
      ),
    );
  }

  // --------------------------------------------------------------------------
  // Unified card (same structure as the original installed card)
  // --------------------------------------------------------------------------

  Widget _buildCard(BuildContext context, _ExtItem item) {
    final storeExt = item.storeExt;
    final ext = item.installedExt;
    final isInstalled = ext != null;
    final installing =
        storeExt != null && (controller.storeInstalling[storeExt.id] == true);

    // Leading icon
    Widget leadingIcon;
    const iconSize = 48.0;
    if (ext != null) {
      // Installed extension – use existing icon logic
      leadingIcon = ext.icon.isEmpty
          ? Image.asset('assets/extension/default_icon.png',
              width: iconSize, height: iconSize)
          : Util.isWeb()
              ? Image.network(
                  join('/fs/extensions/${ext.identity}/${ext.icon}'),
                  width: iconSize,
                  height: iconSize,
                  headers: {
                    'Authorization': 'Bearer ${Database.instance.getWebToken()}'
                  },
                )
              : Image.file(
                  ext.devMode
                      ? File(path.join(ext.devPath, ext.icon))
                      : File(path.join(Util.getStorageDir(), 'extensions',
                          ext.identity, ext.icon)),
                  width: iconSize,
                  height: iconSize,
                );
    } else {
      // Store-only – use network icon with fallback
      leadingIcon = storeExt!.icon.isEmpty
          ? Image.asset('assets/extension/default_icon.png',
              width: iconSize, height: iconSize)
          : Image.network(
              storeExt.icon,
              width: iconSize,
              height: iconSize,
              errorBuilder: (_, __, ___) => Image.asset(
                  'assets/extension/default_icon.png',
                  width: iconSize,
                  height: iconSize),
            );
    }

    // Title row chips
    final titleWidgets = <Widget>[
      Flexible(
        child: Text(
          isInstalled ? ext.title : storeExt!.title,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(fontWeight: FontWeight.bold),
        ),
      ),
      const SizedBox(width: 8),
      buildChip('v${isInstalled ? ext.version : storeExt!.version}'),
    ];
    if (isInstalled && ext.devMode) {
      titleWidgets.add(const SizedBox(width: 8));
      titleWidgets.add(buildChip('dev', bgColor: Colors.blue.shade300));
    }

    // Trailing widget
    Widget? trailing;
    if (isInstalled) {
      trailing = Switch(
        value: !ext.disabled,
        onChanged: (value) async {
          try {
            await switchExtension(ext.identity, SwitchExtension(status: value));
            await controller.load();
          } catch (e) {
            showErrorMessage(e);
          }
        },
      );
    }

    // Bottom action row
    final actions = <Widget?>[];
    if (isInstalled) {
      final hp = ext.homepage;
      final repoUrl = ext.repository?.url ?? '';
      if (hp.isNotEmpty) {
        actions.add(IconButton(
            onPressed: () => launchUrl(Uri.parse(hp)),
            icon: const Icon(Icons.home)));
      }
      if (repoUrl.isNotEmpty) {
        actions.add(IconButton(
            onPressed: () => launchUrl(Uri.parse(repoUrl)),
            icon: const Icon(Icons.code)));
      }
      if (ext.settings?.isNotEmpty == true) {
        actions.add(IconButton(
            onPressed: () => _showSettingDialog(ext),
            icon: const Icon(Icons.settings)));
      }
      actions.add(IconButton(
          onPressed: () => _showDeleteDialog(ext),
          icon: const Icon(Icons.delete)));
      actions.add(Obx(() => controller.updateFlags.containsKey(ext.identity)
          ? badges.Badge(
              position: badges.BadgePosition.topStart(start: 36),
              child: IconButton(
                  onPressed: () => _showUpdateDialog(ext),
                  icon: const Icon(Icons.refresh)))
          : IconButton(
              onPressed: () =>
                  showMessage('tip'.tr, 'extensionAlreadyLatest'.tr),
              icon: const Icon(Icons.refresh))));
    } else if (storeExt != null) {
      // Store-only: show homepage/code links + stats
      if (storeExt.homepage.isNotEmpty) {
        actions.add(IconButton(
            onPressed: () => launchUrl(Uri.parse(storeExt.homepage)),
            icon: const Icon(Icons.home)));
      }
      if (storeExt.repoUrl.isNotEmpty) {
        actions.add(IconButton(
            onPressed: () => launchUrl(Uri.parse(storeExt.repoUrl)),
            icon: const Icon(Icons.code)));

        actions.add(installing
            ? const SizedBox(
                width: 40,
                height: 40,
                child: Padding(
                  padding: EdgeInsets.all(8.0),
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              )
            : IconButton(
                icon: const Icon(Icons.download),
                tooltip: 'extensionInstallBtn'.tr,
                onPressed: () => controller.installFromStore(storeExt),
              ));
      }
    }
    Widget? bottomTrailingRow;
    if (actions.isNotEmpty) {
      bottomTrailingRow = Row(
        mainAxisSize: MainAxisSize.min,
        children: actions.where((e) => e != null).map((e) => e!).toList(),
      );
    }

    // Stars and installs for bottom-left
    Widget? bottomLeftRow;
    if (storeExt != null) {
      bottomLeftRow = Wrap(
        crossAxisAlignment: WrapCrossAlignment.center,
        spacing: 12,
        children: [
          Row(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              const Icon(Icons.star, size: 16, color: Colors.amber),
              const SizedBox(width: 4),
              Text('${storeExt.stars}',
                  style: const TextStyle(
                      fontSize: 13, color: Colors.blueGrey, height: 1.1)),
            ],
          ),
          Row(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              const Icon(Icons.download, size: 16, color: Colors.blueGrey),
              const SizedBox(width: 4),
              Text('${storeExt.installs}',
                  style: const TextStyle(
                      fontSize: 13, color: Colors.blueGrey, height: 1.1)),
            ],
          ),
        ],
      );
    }

    return Card(
      elevation: 2.0,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Container(
                  margin: const EdgeInsets.only(right: 12),
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(
                        color: Colors.grey.withOpacity(0.2), width: 1),
                  ),
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(8),
                    child: leadingIcon,
                  ),
                ),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Expanded(
                            child: Row(children: titleWidgets),
                          ),
                          if (trailing != null) trailing,
                        ],
                      ),
                      const SizedBox(height: 4),
                      Text(
                        isInstalled ? ext.description : storeExt!.description,
                        maxLines: ResponsiveBuilder.isNarrow(context) ? 2 : 3,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          fontSize: 13,
                          color: Theme.of(context).textTheme.bodySmall?.color,
                          height: 1.4,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            if (bottomLeftRow != null || bottomTrailingRow != null) ...[
              const SizedBox(height: 4),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  bottomLeftRow ?? const SizedBox.shrink(),
                  if (bottomTrailingRow != null) bottomTrailingRow,
                ],
              ),
            ]
          ],
        ),
      ),
    );
  }

  // --------------------------------------------------------------------------
  // Shared helpers
  // --------------------------------------------------------------------------

  Widget buildChip(String text, {Color? bgColor}) {
    return Chip(
      padding: const EdgeInsets.all(0),
      backgroundColor: bgColor,
      label: Text(text, style: const TextStyle(fontSize: 12)),
    );
  }

  // --------------------------------------------------------------------------
  // Dialogs (unchanged)
  // --------------------------------------------------------------------------

  Future<void> _showSettingDialog(Extension extension) async {
    final formKey = GlobalKey<FormBuilderState>();
    final confrimController = RoundedLoadingButtonController();

    return showDialog<void>(
        context: Get.context!,
        barrierDismissible: false,
        builder: (dialogContext) => AlertDialog(
              content: Builder(builder: (context) {
                final height = MediaQuery.of(context).size.height;
                final width = MediaQuery.of(context).size.width;
                return SizedBox(
                  height: height * 0.75,
                  width: width,
                  child: FormBuilder(
                    key: formKey,
                    child: Column(children: [
                      Text('setting'.tr),
                      Expanded(
                        child: SingleChildScrollView(
                          child: Column(
                              children: extension.settings!.map((e) {
                            final settingItem = _buildSettingItem(e);
                            return Row(
                              crossAxisAlignment: CrossAxisAlignment.end,
                              children: [
                                SizedBox(
                                        width: 20,
                                        child: e.description.isEmpty
                                            ? null
                                            : Tooltip(
                                                message: e.description,
                                                child: const CircleAvatar(
                                                    radius: 10,
                                                    backgroundColor:
                                                        Colors.grey,
                                                    child: Icon(
                                                        Icons.question_mark,
                                                        size: 10))))
                                    .paddingOnly(right: 10),
                                Expanded(child: settingItem),
                              ],
                            );
                          }).toList()),
                        ),
                      ),
                    ]),
                  ),
                );
              }),
              actions: [
                ConstrainedBox(
                  constraints: BoxConstraints.tightFor(
                      width: Get.theme.buttonTheme.minWidth,
                      height: Get.theme.buttonTheme.height),
                  child: ElevatedButton(
                    style:
                        ElevatedButton.styleFrom(shape: const StadiumBorder())
                            .copyWith(
                                backgroundColor: MaterialStateProperty.all(
                                    Get.theme.colorScheme.background)),
                    onPressed: () => Navigator.of(dialogContext).pop(),
                    child: Text('cancel'.tr),
                  ),
                ),
                ConstrainedBox(
                  constraints: BoxConstraints.tightFor(
                      width: Get.theme.buttonTheme.minWidth,
                      height: Get.theme.buttonTheme.height),
                  child: RoundedLoadingButton(
                      color: Get.theme.colorScheme.secondary,
                      onPressed: () async {
                        try {
                          confrimController.start();
                          if (formKey.currentState?.saveAndValidate() == true) {
                            await updateExtensionSettings(
                                extension.identity,
                                UpdateExtensionSettings(
                                    settings: formKey.currentState!.value));
                            await controller.load();
                            if (dialogContext.mounted) {
                              Navigator.of(dialogContext).pop();
                            }
                          }
                        } catch (e) {
                          showErrorMessage(e);
                        } finally {
                          confrimController.reset();
                        }
                      },
                      controller: confrimController,
                      child: Text('confirm'.tr)),
                ),
              ],
            ));
  }

  Widget _buildSettingItem(Setting setting) {
    final requiredValidator =
        setting.required ? FormBuilderValidators.required() : null;

    Widget buildTextField(TextInputFormatter? inputFormatter,
        FormFieldValidator<String>? validator, TextInputType? keyBoardType) {
      return FormBuilderTextField(
        name: setting.name,
        decoration: InputDecoration(labelText: setting.title),
        initialValue: setting.value?.toString(),
        inputFormatters: inputFormatter != null ? [inputFormatter] : null,
        keyboardType: keyBoardType,
        validator: FormBuilderValidators.compose([
          requiredValidator,
          validator,
        ].where((e) => e != null).map((e) => e!).toList()),
      );
    }

    Widget buildDropdown() {
      return FormBuilderDropdown<String>(
        name: setting.name,
        decoration: InputDecoration(labelText: setting.title),
        initialValue: setting.value?.toString(),
        validator: FormBuilderValidators.compose([
          requiredValidator,
        ].where((e) => e != null).map((e) => e!).toList()),
        items: setting.options!
            .map((e) => DropdownMenuItem(
                value: e.value.toString(), child: Text(e.label)))
            .toList(),
      );
    }

    switch (setting.type) {
      case SettingType.string:
        return setting.options?.isNotEmpty == true
            ? buildDropdown()
            : buildTextField(null, null, null);
      case SettingType.number:
        return setting.options?.isNotEmpty == true
            ? buildDropdown()
            : buildTextField(
                FilteringTextInputFormatter.allow(RegExp(r'^\d+\.?\d*')),
                FormBuilderValidators.numeric(),
                const TextInputType.numberWithOptions(decimal: true));
      case SettingType.boolean:
        return FormBuilderSwitch(
          name: setting.name,
          initialValue: (setting.value as bool?) ?? false,
          title: Text(setting.title),
          validator: requiredValidator,
        );
    }
  }

  void _showDeleteDialog(Extension extension) {
    showDialog(
        context: Get.context!,
        barrierDismissible: false,
        builder: (dialogContext) => AlertDialog(
              title: Text('extensionDelete'.tr),
              actions: [
                TextButton(
                  child: Text('cancel'.tr),
                  onPressed: () => Navigator.of(dialogContext).pop(),
                ),
                TextButton(
                  child: Text('confirm'.tr,
                      style: const TextStyle(color: Colors.redAccent)),
                  onPressed: () async {
                    try {
                      await deleteExtension(extension.identity);
                      await controller.load();
                      if (dialogContext.mounted) {
                        Navigator.of(dialogContext).pop();
                      }
                    } catch (e) {
                      showErrorMessage(e);
                    }
                  },
                ),
              ],
            ));
  }

  void _showUpdateDialog(Extension extension) {
    final confrimController = TextButtonLoadingController();
    showDialog(
        context: Get.context!,
        builder: (dialogContext) => AlertDialog(
              content: Text('newVersionTitle'.trParams({
                'version': 'v${controller.updateFlags[extension.identity]!}'
              })),
              actions: [
                TextButton(
                  onPressed: () => Navigator.of(dialogContext).pop(),
                  child: Text('newVersionLater'.tr),
                ),
                TextButtonLoading(
                  controller: confrimController,
                  onPressed: () async {
                    confrimController.start();
                    try {
                      await updateExtension(extension.identity);
                      await controller.load();
                      controller.updateFlags.remove(extension.identity);
                      if (dialogContext.mounted) {
                        Navigator.of(dialogContext).pop();
                      }
                      showMessage('tip'.tr, 'extensionUpdateSuccess'.tr);
                    } catch (e) {
                      showErrorMessage(e);
                    } finally {
                      confrimController.stop();
                    }
                  },
                  child: Text('newVersionUpdate'.tr),
                ),
              ],
            ));
  }
}

// ---------------------------------------------------------------------------
// Internal data class for unified list items
// ---------------------------------------------------------------------------

class _ExtItem {
  /// Extension from the gopeed.com store (may be null for local-only installs).
  final GopeedExtension? storeExt;

  /// Locally installed extension (may be null for store-only/not-installed).
  final Extension? installedExt;

  const _ExtItem({this.storeExt, this.installedExt})
      : assert(storeExt != null || installedExt != null);
}
