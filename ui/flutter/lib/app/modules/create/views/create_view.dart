import 'package:autoscale_tabbarview/autoscale_tabbarview.dart';
import 'package:desktop_drop/desktop_drop.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:get/get.dart';
import 'package:path/path.dart' as path;
import 'package:rounded_loading_button_plus/rounded_loading_button.dart';

import '../../../../api/api.dart';
import '../../../../api/model/create_task.dart';
import '../../../../api/model/options.dart';
import '../../../../api/model/request.dart';
import '../../../../api/model/resolve_result.dart';
import '../../../../database/database.dart';
import '../../../../util/input_formatter.dart';
import '../../../../util/message.dart';
import '../../../../util/util.dart';
import '../../../routes/app_pages.dart';
import '../../../views/compact_checkbox.dart';
import '../../../views/directory_selector.dart';
import '../../../views/file_list_view.dart';
import '../../app/controllers/app_controller.dart';
import '../../history/views/history_view.dart';
import '../controllers/create_controller.dart';

class CreateView extends GetView<CreateController> {
  final _confirmFormKey = GlobalKey<FormState>();

  final _urlController = TextEditingController();
  final _renameController = TextEditingController();
  final _connectionsController = TextEditingController();
  final _pathController = TextEditingController();
  final _confirmController = RoundedLoadingButtonController();
  final _httpUaController = TextEditingController();
  final _httpCookieController = TextEditingController();
  final _httpRefererController = TextEditingController();
  final _btTrackerController = TextEditingController();

  final _availableSchemes = ["http:", "https:", "magnet:"];

  CreateView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final appController = Get.find<AppController>();

    if (_connectionsController.text.isEmpty) {
      _connectionsController.text = appController
          .downloaderConfig.value.protocolConfig.http.connections
          .toString();
    }
    if (_pathController.text.isEmpty) {
      _pathController.text = appController.downloaderConfig.value.downloadDir;
    }

    final String? filePath = Get.rootDelegate.arguments();
    if (_urlController.text.isEmpty) {
      if (filePath?.isNotEmpty ?? false) {
        // get file path from route arguments
        _urlController.text = filePath!;
        _urlController.selection = TextSelection.fromPosition(
            TextPosition(offset: _urlController.text.length));
      } else {
        // read clipboard
        Clipboard.getData('text/plain').then((value) {
          if (value?.text?.isNotEmpty ?? false) {
            if (_availableSchemes
                .where((e) =>
                    value!.text!.startsWith(e) ||
                    value.text!.startsWith(e.toUpperCase()))
                .isNotEmpty) {
              _urlController.text = value!.text!;
              _urlController.selection = TextSelection.fromPosition(
                  TextPosition(offset: _urlController.text.length));
              return;
            }

            recognizeMagnetUri(value!.text!);
          }
        });
      }
    }

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
            icon: const Icon(Icons.arrow_back),
            onPressed: () => Get.rootDelegate.offNamed(Routes.TASK)),
        // actions: [],
        title: Text('create'.tr),
      ),
      body: DropTarget(
        onDragDone: (details) async {
          if (!Util.isWeb()) {
            _urlController.text = details.files[0].path;
            return;
          }
          _urlController.text = details.files[0].name;
          final bytes = await details.files[0].readAsBytes();
          controller.setFileDataUri(bytes);
        },
        child: GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTap: () {
            FocusScope.of(context).requestFocus(FocusNode());
          },
          child: SingleChildScrollView(
            child: Padding(
              padding:
                  const EdgeInsets.symmetric(vertical: 16.0, horizontal: 24.0),
              child: Form(
                key: _confirmFormKey,
                autovalidateMode: AutovalidateMode.onUserInteraction,
                child: Column(
                  children: [
                    Row(children: [
                      Expanded(
                        child: TextFormField(
                          autofocus: !Util.isMobile(),
                          controller: _urlController,
                          minLines: 1,
                          maxLines: 5,
                          decoration: InputDecoration(
                            hintText: _hitText(),
                            hintStyle: const TextStyle(fontSize: 12),
                            labelText: 'downloadLink'.tr,
                            icon: const Icon(Icons.link),
                            suffixIcon: IconButton(
                              onPressed: () {
                                _urlController.clear();
                                controller.clearFileDataUri();
                              },
                              icon: const Icon(Icons.clear),
                            ),
                          ),
                          validator: (v) {
                            return v!.trim().isNotEmpty
                                ? null
                                : 'downloadLinkValid'.tr;
                          },
                          onChanged: (v) async {
                            controller.clearFileDataUri();
                            if (controller.oldUrl.value.isEmpty) {
                              recognizeMagnetUri(v);
                            }
                            controller.oldUrl.value = v;
                          },
                        ),
                      ),
                      IconButton(
                        icon: const Icon(Icons.folder_open),
                        onPressed: () async {
                          var pr = await FilePicker.platform.pickFiles(
                              type: FileType.custom,
                              allowedExtensions: ["torrent"]);
                          if (pr != null) {
                            if (!Util.isWeb()) {
                              _urlController.text = pr.files[0].path ?? "";
                              return;
                            }
                            _urlController.text = pr.files[0].name;
                            controller.setFileDataUri(pr.files[0].bytes!);
                          }
                        },
                      ),
                      IconButton(
                        icon: const Icon(Icons.history_rounded),
                        onPressed: () async {
                          List<String> resultOfHistories =
                              Database.instance.getCreateHistory() ?? [];
                          // show dialog box to list history
                          if (context.mounted) {
                            showGeneralDialog(
                              barrierColor: Colors.black.withOpacity(0.5),
                              transitionBuilder: (context, a1, a2, widget) {
                                return Transform.scale(
                                  scale: a1.value,
                                  child: Opacity(
                                    opacity: a1.value,
                                    child: HistoryView(
                                      isHistoryListEmpty:
                                          resultOfHistories.isEmpty,
                                      historyList: ListView.builder(
                                        itemCount: resultOfHistories.length,
                                        itemBuilder: (context, index) {
                                          return GestureDetector(
                                            onTap: () {
                                              _urlController.text =
                                                  resultOfHistories[index];
                                              Navigator.pop(context);
                                            },
                                            child: MouseRegion(
                                              cursor: SystemMouseCursors.click,
                                              child: Container(
                                                padding:
                                                    const EdgeInsets.symmetric(
                                                  horizontal: 8.0,
                                                  vertical: 8.0,
                                                ),
                                                margin:
                                                    const EdgeInsets.symmetric(
                                                  horizontal: 10.0,
                                                  vertical: 8.0,
                                                ),
                                                decoration: BoxDecoration(
                                                  color: Theme.of(context)
                                                      .colorScheme
                                                      .background,
                                                  borderRadius:
                                                      BorderRadius.circular(
                                                          10.0),
                                                ),
                                                child: Text(
                                                  resultOfHistories[index],
                                                ),
                                              ),
                                            ),
                                          );
                                        },
                                      ),
                                    ),
                                  ),
                                );
                              },
                              transitionDuration:
                                  const Duration(milliseconds: 250),
                              barrierDismissible: true,
                              barrierLabel: '',
                              context: context,
                              pageBuilder: (context, animation1, animation2) {
                                return const Text('PAGE BUILDER');
                              },
                            );
                          }
                        },
                      ),
                    ]),
                    Padding(
                      padding: const EdgeInsets.only(left: 40),
                      child: Column(children: [
                        TextField(
                          controller: _renameController,
                          decoration: InputDecoration(labelText: 'rename'.tr),
                        ),
                        TextField(
                          controller: _connectionsController,
                          decoration: InputDecoration(
                            labelText: 'connections'.tr,
                          ),
                          keyboardType: TextInputType.number,
                          inputFormatters: [
                            FilteringTextInputFormatter.digitsOnly,
                            NumericalRangeFormatter(min: 1, max: 256),
                          ],
                        ),
                        DirectorySelector(
                          controller: _pathController,
                        ),
                        Obx(
                          () => Visibility(
                            visible: controller.showAdvanced.value,
                            child: Column(
                              children: [
                                TabBar(
                                  controller: controller.advancedTabController,
                                  tabs: const [
                                    Tab(
                                      text: 'HTTP',
                                    ),
                                    Tab(
                                      text: 'BitTorrent',
                                    )
                                  ],
                                ),
                                AutoScaleTabBarView(
                                  controller: controller.advancedTabController,
                                  children: [
                                    Column(
                                      children: [
                                        TextFormField(
                                            controller: _httpUaController,
                                            decoration: const InputDecoration(
                                              labelText: 'User-Agent',
                                            )),
                                        TextFormField(
                                            controller: _httpCookieController,
                                            decoration: const InputDecoration(
                                              labelText: 'Cookie',
                                            )),
                                        TextFormField(
                                            controller: _httpRefererController,
                                            decoration: const InputDecoration(
                                              labelText: 'Referer',
                                            )),
                                      ],
                                    ),
                                    Column(
                                      children: [
                                        TextFormField(
                                            controller: _btTrackerController,
                                            maxLines: 5,
                                            decoration: InputDecoration(
                                              labelText: 'Trackers',
                                              hintText: 'addTrackerHit'.tr,
                                            )),
                                      ],
                                    )
                                  ],
                                )
                              ],
                            ).paddingOnly(top: 16),
                          ),
                        ),
                      ]),
                    ),
                    Center(
                      child: Padding(
                        padding: const EdgeInsets.only(top: 15),
                        child: Column(
                          children: [
                            Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                CompactCheckbox(
                                    label: 'directDownload'.tr,
                                    value: controller.directDownload.value,
                                    onChanged: (bool? value) {
                                      controller.directDownload.value =
                                          value ?? false;
                                    }),
                                TextButton(
                                  onPressed: () {
                                    controller.showAdvanced.value =
                                        !controller.showAdvanced.value;
                                  },
                                  child: Row(children: [
                                    Obx(() => Checkbox(
                                          value: controller.showAdvanced.value,
                                          onChanged: (bool? value) {
                                            controller.showAdvanced.value =
                                                value ?? false;
                                          },
                                        )),
                                    Text('advancedOptions'.tr),
                                  ]),
                                ),
                              ],
                            ),
                            SizedBox(
                              width: 150,
                              child: RoundedLoadingButton(
                                color: Get.theme.colorScheme.secondary,
                                onPressed: _doConfirm,
                                controller: _confirmController,
                                child: Text('confirm'.tr),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  // recognize magnet uri, if length == 40, auto add magnet prefix
  recognizeMagnetUri(String text) {
    if (text.length != 40) {
      return;
    }
    final exp = RegExp(r"[0-9a-fA-F]+");
    if (exp.hasMatch(text)) {
      final uri = "magnet:?xt=urn:btih:$text";
      _urlController.text = uri;
      _urlController.selection = TextSelection.fromPosition(
          TextPosition(offset: _urlController.text.length));
    }
  }

  Future<void> _doConfirm() async {
    if (controller.isConfirming.value) {
      return;
    }
    controller.isConfirming.value = true;
    try {
      _confirmController.start();
      if (_confirmFormKey.currentState!.validate()) {
        final isWebFileChosen =
            Util.isWeb() && controller.fileDataUri.isNotEmpty;
        final submitUrl = isWebFileChosen
            ? controller.fileDataUri.value
            : _urlController.text.trim();

        final urls = Util.textToLines(submitUrl);
        // Add url to the history
        if (!isWebFileChosen) {
          for (final url in urls) {
            Database.instance.saveCreateHistory(url);
          }
        }

        /*
        Check if is direct download, there has two ways to direct download
        1. Direct download option is checked
        2. Muli line urls
        */
        final isMultiLine = urls.length > 1;
        final isDirect = controller.directDownload.value || isMultiLine;
        if (isDirect) {
          await Future.wait(urls.map((url) {
            return createTask(CreateTask(
                req: Request(url: url, extra: parseReqExtra(url)),
                opt: Options(
                    name: isMultiLine ? "" : _renameController.text,
                    path: _pathController.text,
                    selectFiles: [],
                    extra: parseReqOptsExtra())));
          }));
          Get.rootDelegate.offNamed(Routes.TASK);
        } else {
          final rr = await resolve(Request(
            url: submitUrl,
            extra: parseReqExtra(_urlController.text),
          ));
          await _showResolveDialog(rr);
        }
      }
    } catch (e) {
      showErrorMessage(e);
    } finally {
      _confirmController.reset();
      controller.isConfirming.value = false;
    }
  }

  Object? parseReqExtra(String url) {
    Object? reqExtra;
    if (controller.showAdvanced.value) {
      switch (controller.advancedTabController.index) {
        case 0:
          final header = {
            "User-Agent": _httpUaController.text,
            "Cookie": _httpCookieController.text,
            "Referer": _httpRefererController.text,
          };
          header.removeWhere((key, value) => value.trim().isEmpty);
          if (header.isNotEmpty) {
            reqExtra = ReqExtraHttp()..header = header;
          }
          break;
        case 1:
          if (_btTrackerController.text.trim().isNotEmpty) {
            reqExtra = ReqExtraBt()
              ..trackers = Util.textToLines(_btTrackerController.text);
          }
          break;
      }
    }
    return reqExtra;
  }

  Object? parseReqOptsExtra() {
    return OptsExtraHttp()
      ..connections = int.tryParse(_connectionsController.text) ?? 0
      ..autoTorrent = true;
  }

  String _hitText() {
    return 'downloadLinkHit'.trParams({
      'append':
          Util.isDesktop() || Util.isWeb() ? 'downloadLinkHitDesktop'.tr : '',
    });
  }

  Future<void> _showResolveDialog(ResolveResult rr) async {
    final createFormKey = GlobalKey<FormState>();
    final downloadController = RoundedLoadingButtonController();
    return showDialog<void>(
        context: Get.context!,
        barrierDismissible: false,
        builder: (_) => AlertDialog(
              title: rr.res.name.isEmpty ? null : Text(rr.res.name),
              content: Builder(
                builder: (context) {
                  // Get available height and width of the build area of this widget. Make a choice depending on the size.
                  var height = MediaQuery.of(context).size.height;
                  var width = MediaQuery.of(context).size.width;

                  return SizedBox(
                    height: height * 0.75,
                    width: width,
                    child: Form(
                        key: createFormKey,
                        autovalidateMode: AutovalidateMode.always,
                        child: FileListView(files: rr.res.files)),
                  );
                },
              ),
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
                          downloadController.start();
                          if (controller.selectedIndexes.isEmpty) {
                            showMessage('tip'.tr, 'noFileSelected'.tr);
                            return;
                          }
                          final optExtra = parseReqOptsExtra();
                          if (createFormKey.currentState!.validate()) {
                            if (rr.id.isEmpty) {
                              // from extension resolve result
                              await Future.wait(
                                  controller.selectedIndexes.map((index) {
                                final file = rr.res.files[index];
                                return createTask(CreateTask(
                                    req: file.req!,
                                    opt: Options(
                                        name: file.name,
                                        path: path.join(_pathController.text,
                                            rr.res.name, file.path),
                                        selectFiles: [],
                                        extra: optExtra)));
                              }));
                            } else {
                              await createTask(CreateTask(
                                  rid: rr.id,
                                  opt: Options(
                                      name: _renameController.text,
                                      path: _pathController.text,
                                      selectFiles: controller.selectedIndexes,
                                      extra: optExtra)));
                            }
                            Get.back();
                            Get.rootDelegate.offNamed(Routes.TASK);
                          }
                        } catch (e) {
                          showErrorMessage(e);
                        } finally {
                          downloadController.reset();
                        }
                      },
                      controller: downloadController,
                      child: Text(
                        'download'.tr,
                        // style: controller.selectedIndexes.isEmpty
                        //     ? Get.textTheme.disabled
                        //     : Get.textTheme.titleSmall
                      )),
                ),
              ],
            ));
  }
}
