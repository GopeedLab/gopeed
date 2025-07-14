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
import '../../../../util/message.dart';
import '../../../../util/util.dart';
import '../../../views/icon_button_loading.dart';
import '../../../views/responsive_builder.dart';
import '../../../views/text_button_loading.dart';
import '../controllers/extension_controller.dart';

class ExtensionView extends GetView<ExtensionController> {
  ExtensionView({Key? key}) : super(key: key);

  final _installUrlController = TextEditingController();
  final _installBtnController = IconButtonLoadingController();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              TextButton.icon(
                onPressed: () {
                  launchUrl(Uri.parse(
                      'https://github.com/search?q=topic%3Agopeed-extension&type=repositories'));
                },
                icon: const Icon(Icons.search),
                label: Text('extensionFind'.tr),
              ),
              const SizedBox(width: 16),
              TextButton.icon(
                onPressed: () {
                  launchUrl(
                      Uri.parse('https://docs.gopeed.com/dev-extension.html'));
                },
                icon: const Icon(Icons.edit),
                label: Text('extensionDevelop'.tr),
              ),
            ],
          ),
          Obx(() => Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _installUrlController,
                      decoration: InputDecoration(
                        labelText: 'extensionInstallUrl'.tr,
                      ),
                    ),
                  ),
                  const SizedBox(width: 10),
                  IconButtonLoading(
                      controller: _installBtnController,
                      onPressed: () async {
                        if (_installUrlController.text.isEmpty) {
                          controller.tryOpenDevMode();
                          return;
                        }
                        _installBtnController.start();
                        try {
                          await installExtension(InstallExtension(
                              url: _installUrlController.text));
                          Get.snackbar('tip'.tr, 'extensionInstallSuccess'.tr);
                          await controller.load();
                        } catch (e) {
                          showErrorMessage(e);
                        } finally {
                          _installBtnController.stop();
                        }
                      },
                      icon: const Icon(Icons.download)),
                  controller.devMode.value && Util.isDesktop()
                      ? IconButton(
                          icon: const Icon(Icons.folder_open),
                          onPressed: () async {
                            var dir =
                                await FilePicker.platform.getDirectoryPath();
                            if (dir != null) {
                              try {
                                await installExtension(
                                    InstallExtension(devMode: true, url: dir));
                                Get.snackbar(
                                    'tip'.tr, 'extensionInstallSuccess'.tr);
                                await controller.load();
                              } catch (e) {
                                showErrorMessage(e);
                              }
                            }
                          })
                      : Container()
                ],
              )),
          const SizedBox(height: 16),
          Expanded(
              child: Obx(() => ListView.builder(
                    itemCount: controller.extensions.length,
                    itemBuilder: (context, index) {
                      final extension = controller.extensions[index];
                      return SizedBox(
                        height: ResponsiveBuilder.isNarrow(context) ? 152 : 112,
                        child: Card(
                          elevation: 4.0,
                          child: Column(
                            children: [
                              Expanded(
                                child: ListTile(
                                  leading: extension.icon.isEmpty
                                      ? Image.asset(
                                          "assets/extension/default_icon.png",
                                          width: 48,
                                          height: 48,
                                        )
                                      : Util.isWeb()
                                          ? Image.network(
                                              join(
                                                  '/fs/extensions/${extension.identity}/${extension.icon}'),
                                              width: 48,
                                              height: 48,
                                              headers: {
                                                'Authorization':
                                                    'Bearer ${Database.instance.getWebToken()}'
                                              },
                                            )
                                          : Image.file(
                                              extension.devMode
                                                  ? File(path.join(
                                                      extension.devPath,
                                                      extension.icon))
                                                  : File(path.join(
                                                      Util.getStorageDir(),
                                                      "extensions",
                                                      extension.identity,
                                                      extension.icon)),
                                              width: 48,
                                              height: 48,
                                            ),
                                  trailing: Switch(
                                    value: !extension.disabled,
                                    onChanged: (value) async {
                                      try {
                                        await switchExtension(
                                            extension.identity,
                                            SwitchExtension(status: value));
                                        await controller.load();
                                      } catch (e) {
                                        showErrorMessage(e);
                                      }
                                    },
                                  ),
                                  title: Row(
                                    children: () {
                                      final list = [
                                        ResponsiveBuilder.isNarrow(context)
                                            ? Expanded(
                                                child: Text(
                                                  extension.title,
                                                  overflow:
                                                      TextOverflow.ellipsis,
                                                ),
                                              )
                                            : Text(extension.title),
                                        const SizedBox(width: 8),
                                        buildChip('v${extension.version}'),
                                      ];
                                      if (extension.devMode) {
                                        list.add(const SizedBox(width: 8));
                                        list.add(buildChip(
                                          'dev',
                                          bgColor: Colors.blue.shade300,
                                        ));
                                      }
                                      return list;
                                    }(),
                                  ),
                                  subtitle: ResponsiveBuilder.isNarrow(context)
                                      ? Text(
                                          extension.description,
                                          maxLines: 2,
                                          overflow: TextOverflow.ellipsis,
                                        )
                                      : Text(extension.description),
                                ),
                              ),
                              Row(
                                mainAxisAlignment: MainAxisAlignment.end,
                                children: [
                                  extension.homepage.isNotEmpty == true
                                      ? IconButton(
                                          onPressed: () {
                                            launchUrl(
                                                Uri.parse(extension.homepage));
                                          },
                                          icon: const Icon(Icons.home))
                                      : null,
                                  extension.repository?.url.isNotEmpty == true
                                      ? IconButton(
                                          onPressed: () {
                                            launchUrl(Uri.parse(
                                                extension.repository!.url));
                                          },
                                          icon: const Icon(Icons.code))
                                      : null,
                                  extension.settings?.isNotEmpty == true
                                      ? IconButton(
                                          onPressed: () {
                                            _showSettingDialog(extension);
                                          },
                                          icon: const Icon(Icons.settings))
                                      : null,
                                  IconButton(
                                      onPressed: () {
                                        _showDeleteDialog(extension);
                                      },
                                      icon: const Icon(Icons.delete)),
                                  Obx(() => controller.updateFlags
                                          .containsKey(extension.identity)
                                      ? badges.Badge(
                                          position:
                                              badges.BadgePosition.topStart(
                                                  start: 36),
                                          child: IconButton(
                                              onPressed: () {
                                                _showUpdateDialog(extension);
                                              },
                                              icon: const Icon(Icons.refresh)))
                                      : IconButton(
                                          onPressed: () {
                                            showMessage('tip'.tr,
                                                'extensionAlreadyLatest'.tr);
                                          },
                                          icon: const Icon(Icons.refresh))),
                                ]
                                    .where((e) => e != null)
                                    .map((e) => e!)
                                    .toList(),
                              ).paddingOnly(right: 16)
                            ],
                          ),
                        ),
                      ).paddingOnly(top: 8);
                    },
                  ))),
        ],
      ).paddingAll(32),
    );
  }

  Widget buildChip(String text, {Color? bgColor}) {
    return Chip(
      padding: const EdgeInsets.all(0),
      backgroundColor: bgColor,
      label: Text(text, style: const TextStyle(fontSize: 12)),
    );
  }

  Future<void> _showSettingDialog(Extension extension) async {
    final formKey = GlobalKey<FormBuilderState>();
    final confrimController = RoundedLoadingButtonController();

    return showDialog<void>(
        context: Get.context!,
        barrierDismissible: false,
        builder: (_) => AlertDialog(
              content: Builder(builder: (context) {
                final height = MediaQuery.of(context).size.height;
                final width = MediaQuery.of(context).size.width;

                return SizedBox(
                  height: height * 0.75,
                  width: width,
                  child: FormBuilder(
                    key: formKey,
                    // autovalidateMode: AutovalidateMode.always,
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
                                                      size: 10,
                                                    )),
                                              ))
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
                    height: Get.theme.buttonTheme.height,
                  ),
                  child: ElevatedButton(
                    style:
                        ElevatedButton.styleFrom(shape: const StadiumBorder())
                            .copyWith(
                                backgroundColor: MaterialStateProperty.all(
                                    Get.theme.colorScheme.background)),
                    onPressed: () {
                      Get.back();
                    },
                    child: Text('cancel'.tr),
                  ),
                ),
                ConstrainedBox(
                  constraints: BoxConstraints.tightFor(
                    width: Get.theme.buttonTheme.minWidth,
                    height: Get.theme.buttonTheme.height,
                  ),
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
                            Get.back();
                          }
                        } catch (e) {
                          showErrorMessage(e);
                        } finally {
                          confrimController.reset();
                        }
                      },
                      controller: confrimController,
                      child: Text(
                        'confirm'.tr,
                      )),
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
        decoration: InputDecoration(
          labelText: setting.title,
        ),
        initialValue: setting.value?.toString(),
        validator: FormBuilderValidators.compose([
          requiredValidator,
        ].where((e) => e != null).map((e) => e!).toList()),
        items: setting.options!
            .map((e) => DropdownMenuItem(
                  value: e.value.toString(),
                  child: Text(e.label),
                ))
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
        builder: (_) => AlertDialog(
              title: Text('extensionDelete'.tr),
              actions: [
                TextButton(
                  child: Text('cancel'.tr),
                  onPressed: () => Get.back(),
                ),
                TextButton(
                  child: Text(
                    'confirm'.tr,
                    style: const TextStyle(color: Colors.redAccent),
                  ),
                  onPressed: () async {
                    try {
                      await deleteExtension(extension.identity);
                      await controller.load();
                      Get.back();
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
        builder: (context) => AlertDialog(
              content: Text('newVersionTitle'.trParams({
                'version': 'v${controller.updateFlags[extension.identity]!}'
              })),
              actions: [
                TextButton(
                  onPressed: () {
                    Get.back();
                  },
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
                      Get.back();
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
