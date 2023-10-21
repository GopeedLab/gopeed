import 'package:autoscale_tabbarview/autoscale_tabbarview.dart';
import 'package:desktop_drop/desktop_drop.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:get/get.dart';
import 'package:gopeed/app/modules/history/controller/history_controller.dart';
import 'package:gopeed/app/modules/history/views/history_view.dart';
import 'package:path/path.dart' as path;
import 'package:rounded_loading_button/rounded_loading_button.dart';
import '../../../../api/api.dart';
import '../../../../api/model/create_task.dart';
import '../../../../api/model/options.dart';
import '../../../../api/model/request.dart';
import '../../../../api/model/resolve_result.dart';
import '../../../../api/model/resource.dart';
import '../../../../util/input_formatter.dart';
import '../../../../util/message.dart';
import '../../../../util/util.dart';
import '../../../routes/app_pages.dart';
import '../../../views/directory_selector.dart';
import '../../../views/file_list_view.dart';
import '../../app/controllers/app_controller.dart';
import '../controllers/create_controller.dart';

class CreateView extends GetView<CreateController> {
  final _resolveFormKey = GlobalKey<FormState>();

  final _urlController = TextEditingController();
  final _confirmController = RoundedLoadingButtonController();
  final _httpUaController = TextEditingController();
  final _httpCookieController = TextEditingController();
  final _httpRefererController = TextEditingController();
  final _btTrackerController = TextEditingController();
  final _historyController = HistoryController();

  final _availableSchemes = ["http:", "https:", "magnet:"];

  CreateView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
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
          child: Padding(
            padding:
                const EdgeInsets.symmetric(vertical: 16.0, horizontal: 24.0),
            child: Form(
              key: _resolveFormKey,
              autovalidateMode: AutovalidateMode.onUserInteraction,
              child: Column(
                children: [
                  Row(children: [
                    Expanded(
                      child: TextFormField(
                        autofocus: true,
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
                            await _historyController.getAllHistory();
                        // reversing: display last entered history first
                        List<String> reverseResultOfHistories =
                            resultOfHistories.reversed.toList();
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
                                        reverseResultOfHistories.isEmpty,
                                    historyList: ListView.builder(
                                      itemCount: reverseResultOfHistories.length,
                                      itemBuilder: (context, index) {
                                        return GestureDetector(
                                          onTap: () {
                                            _urlController.text =
                                                reverseResultOfHistories[index];
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
                                                    BorderRadius.circular(10.0),
                                              ),
                                              child: Text(
                                                reverseResultOfHistories[index]
                                                    .toString(),
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
                  ]
                      //.where((e) => e != null).map((e) => e!).toList(),
                      ),
                  Obx(
                    () => Visibility(
                      visible: controller.showAdvanced.value,
                      child: Padding(
                        padding: const EdgeInsets.only(left: 40, top: 15),
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
                                          labelText: 'Trakers',
                                          hintText: 'addTrackerHit'.tr,
                                        )),
                                  ],
                                )
                              ],
                            )
                          ],
                        ),
                      ),
                    ),
                  ),
                  Center(
                    child: Padding(
                      padding: const EdgeInsets.only(top: 15),
                      child: Column(
                        children: [
                          Padding(
                            padding: const EdgeInsets.only(right: 10),
                            child: TextButton(
                              onPressed: () {
                                controller.showAdvanced.value =
                                    !controller.showAdvanced.value;
                              },
                              child: Row(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
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
                          ),
                          SizedBox(
                            width: 150,
                            child: RoundedLoadingButton(
                              color: Get.theme.colorScheme.secondary,
                              onPressed: _doResolve,
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

  Future<void> _doResolve() async {
    if (controller.isResolving.value) {
      return;
    }
    controller.isResolving.value = true;
    try {
      _confirmController.start();
      if (_resolveFormKey.currentState!.validate()) {
        // check if is multi line urls
        final urls = Util.textToLines(_urlController.text);
        ResolveResult rr;
        bool isMultiLine = false;
        if (urls.length > 1) {
          isMultiLine = true;
          rr = ResolveResult(
              res: Resource(
                  files: urls
                      .map((u) => FileInfo(
                          name: u,
                          req: Request(url: u, extra: parseReqExtra(u))))
                      .toList()));
        } else {
          final submitUrl = Util.isWeb() && controller.fileDataUri.isNotEmpty
              ? controller.fileDataUri.value
              : _urlController.text;
          // add final submitUrl to the history
          _historyController.addHistory(submitUrl);
          rr = await resolve(Request(
            url: submitUrl,
            extra: parseReqExtra(_urlController.text),
          ));
        }
        await _showResolveDialog(rr, isMultiLine);
      }
    } catch (e) {
      showErrorMessage(e);
    } finally {
      _confirmController.reset();
      controller.isResolving.value = false;
    }
  }

  Object? parseReqExtra(String url) {
    Object? reqExtra;
    if (controller.showAdvanced.value) {
      final u = Uri.parse(_urlController.text);
      if (u.scheme.startsWith("http")) {
        reqExtra = ReqExtraHttp()
          ..header = {
            "User-Agent": _httpUaController.text,
            "Cookie": _httpCookieController.text,
            "Referer": _httpRefererController.text,
          };
      } else {
        reqExtra = ReqExtraBt()
          ..trackers = Util.textToLines(_btTrackerController.text);
      }
    }
    return reqExtra;
  }

  String _hitText() {
    return 'downloadLinkHit'.trParams({
      'append':
          Util.isDesktop() || Util.isWeb() ? 'downloadLinkHitDesktop'.tr : '',
    });
  }

  Future<void> _showResolveDialog(ResolveResult rr, bool isMuiltiLine) async {
    final appController = Get.find<AppController>();

    final createFormKey = GlobalKey<FormState>();
    final nameController = TextEditingController();
    final connectionsController = TextEditingController();
    final pathController = TextEditingController(
        text: appController.downloaderConfig.value.downloadDir);
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
                        child: Column(
                          children: [
                            Expanded(child: FileListView(files: rr.res.files)),
                            TextFormField(
                              controller: nameController,
                              decoration: InputDecoration(
                                labelText: 'rename'.tr,
                              ),
                            ),
                            TextFormField(
                              controller: connectionsController,
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
                              controller: pathController,
                            ),
                          ],
                        )),
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
                          Object? optExtra = connectionsController.text.isEmpty
                              ? null
                              : (OptsExtraHttp()
                                ..connections =
                                    int.parse(connectionsController.text));
                          if (createFormKey.currentState!.validate()) {
                            if (rr.id.isEmpty) {
                              // create task batch, there has two ways to batch create task
                              // 1. from multi line urls
                              // 2. from extension resolve result
                              await Future.wait(
                                  controller.selectedIndexes.map((index) {
                                final file = rr.res.files[index];
                                return createTask(CreateTask(
                                    req: file.req!,
                                    opt: Options(
                                        name: isMuiltiLine ? "" : file.name,
                                        path: path.join(
                                            pathController.text, rr.res.name),
                                        selectFiles: [],
                                        extra: optExtra)));
                              }));
                            } else {
                              await createTask(CreateTask(
                                  rid: rr.id,
                                  opt: Options(
                                      name: nameController.text,
                                      path: pathController.text,
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
