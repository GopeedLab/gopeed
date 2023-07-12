import 'package:desktop_drop/desktop_drop.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gopeed/api/model/create_task_batch.dart';
import 'package:gopeed/api/model/resolved_request.dart';
import 'package:gopeed/api/model/resource.dart';
import '../../../../api/model/resolve_result.dart';
import 'package:rounded_loading_button/rounded_loading_button.dart';

import '../../../../api/api.dart';
import '../../../../api/model/create_task.dart';
import '../../../../api/model/options.dart';
import '../../../../api/model/request.dart';
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

  CreateView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final String? filePath = Get.rootDelegate.arguments();
    if (_urlController.text.isEmpty && filePath != null) {
      _urlController.text = filePath;
      WidgetsBinding.instance.addPostFrameCallback((_) async {
        await _doResolve();
      });
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
        onDragDone: (details) {
          _urlController.text = details.files[0].path;
        },
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 16.0, horizontal: 24.0),
          child: Form(
            key: _resolveFormKey,
            autovalidateMode: AutovalidateMode.always,
            child: Column(
              children: [
                Row(
                  children: [
                    Expanded(
                      child: TextFormField(
                          autofocus: true,
                          controller: _urlController,
                          minLines: 1,
                          maxLines: 30,
                          decoration: InputDecoration(
                            hintText: _hitText(),
                            hintStyle: const TextStyle(fontSize: 12),
                            labelText: 'downloadLink'.tr,
                            icon: const Icon(Icons.link),
                            suffixIcon: IconButton(
                              onPressed: _urlController.clear,
                              icon: const Icon(Icons.clear),
                            ),
                          ),
                          validator: (v) {
                            return v!.trim().isNotEmpty
                                ? null
                                : 'downloadLinkValid'.tr;
                          }),
                    ),
                    IconButton(
                        icon: const Icon(Icons.folder_open),
                        onPressed: () async {
                          var pr = await FilePicker.platform.pickFiles(
                              type: FileType.custom,
                              allowedExtensions: ["torrent"]);
                          if (pr != null) {
                            _urlController.text = pr.files[0].path ?? "";
                          }
                        }),
                  ],
                ),
                Center(
                  child: Padding(
                      padding: const EdgeInsets.all(15.0),
                      child: ConstrainedBox(
                        constraints: const BoxConstraints.tightFor(
                          width: 150,
                          height: 40,
                        ),
                        child: RoundedLoadingButton(
                          color: Get.theme.colorScheme.secondary,
                          onPressed: _doResolve,
                          controller: _confirmController,
                          child: Text('confirm'.tr),
                        ),
                      )),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _doResolve() async {
    if (controller.isResolving.value) {
      return;
    }
    controller.isResolving.value = true;
    try {
      _confirmController.start();
      if (_resolveFormKey.currentState!.validate()) {
        final rr = await resolve(Request(
          url: _urlController.text,
        ));
        await _showResolveDialog(rr);
      }
    } catch (e) {
      showErrorMessage(e);
    } finally {
      _confirmController.reset();
      controller.isResolving.value = false;
    }
  }

  String _hitText() {
    return 'downloadLinkHit'.trParams({
      'append': Util.isDesktop() ? 'downloadLinkHitDesktop'.tr : '',
    });
  }

  Future<void> _showResolveDialog(ResolveResult rr) async {
    final files = rr.res.files;
    final appController = Get.find<AppController>();

    final createFormKey = GlobalKey<FormState>();
    final pathController = TextEditingController(
        text: appController.downloaderConfig.value.downloadDir);
    final downloadController = RoundedLoadingButtonController();
    return showDialog<void>(
        context: Get.context!,
        barrierDismissible: false,
        builder: (_) => AlertDialog(
              content: Builder(
                builder: (context) {
                  // Get available height and width of the build area of this widget. Make a choice depending on the size.
                  var height = MediaQuery.of(context).size.height;
                  var width = MediaQuery.of(context).size.width;

                  return SizedBox(
                    height: height * 0.5,
                    width: width,
                    child: Form(
                        key: createFormKey,
                        autovalidateMode: AutovalidateMode.always,
                        child: Column(
                          children: [
                            Expanded(child: FileListView(files: files)),
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
                          if (createFormKey.currentState!.validate()) {
                            // if (Util.isAndroid()) {
                            //   if (!await Permission.storage.request().isGranted) {
                            //     Get.snackbar('error'.tr,
                            //         'noStoragePermission'.tr);
                            //     return;
                            //   }
                            // }

                            final reqFiles = rr.res.files
                                .where((e) => e.req != null)
                                .toList();
                            if (reqFiles.isNotEmpty) {
                              // if the resource is a multi-task resource, call the batch download api
                              final selectReqs =
                                  controller.selectedIndexes.map((i) {
                                final file = reqFiles[i];
                                final optReq = file.req!;
                                // fill the file path into the download request
                                optReq.res = Resource(
                                    name: file.name,
                                    size: file.size,
                                    range: rr.res.range,
                                    files: [
                                      FileInfo(
                                        name: file.name,
                                        size: file.size,
                                        path: file.path,
                                      )
                                    ]);
                                return optReq;
                              }).toList();
                              await createTaskBatch(CreateTaskBatch(
                                  reqs: selectReqs,
                                  opts: Options(path: pathController.text)));
                            } else {
                              await createTask(CreateTask(
                                  rid: rr.id,
                                  opts: Options(
                                      path: pathController.text,
                                      selectFiles: controller.selectedIndexes
                                          .cast<int>())));
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
