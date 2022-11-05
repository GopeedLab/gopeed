import 'package:flutter/material.dart';
import 'package:get/get.dart';
import '../../api/api.dart';
import '../../api/model/create_task.dart';
import '../../api/model/options.dart';
import '../../api/model/request.dart';
import '../../api/model/resource.dart';
import 'create_controller.dart';
import '../../setting/setting.dart';
import '../../widget/file_list_view.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:rounded_loading_button/rounded_loading_button.dart';

import '../../routes/router.dart';
import '../../util/util.dart';
import '../../widget/directory_selector.dart';

class CreateView extends GetView<CreateController> {
  CreateView({Key? key}) : super(key: key);

  final _resolveFormKey = GlobalKey<FormState>();
  final _urlController = TextEditingController();
  final _confirmController = RoundedLoadingButtonController();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
            icon: const Icon(Icons.arrow_back),
            onPressed: () => Get.rootDelegate.offNamed(Routes.task)),
        actions: [Container()],
        title: Text('create.title'.tr),
      ),
      body: Padding(
        padding: const EdgeInsets.symmetric(vertical: 16.0, horizontal: 24.0),
        child: Form(
          key: _resolveFormKey,
          autovalidateMode: AutovalidateMode.always,
          child: Column(
            children: [
              TextFormField(
                  autofocus: true,
                  controller: _urlController,
                  minLines: 1,
                  maxLines: 15,
                  decoration: InputDecoration(
                      hintText: 'create.downloadLinkHit'.tr,
                      labelText: 'create.downloadLink'.tr,
                      icon: const Icon(Icons.link)),
                  validator: (v) {
                    return v!.trim().isNotEmpty
                        ? null
                        : 'create.downloadLinkValid'.tr;
                  }),
              // 登录按钮
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
                        onPressed: () async {
                          if (_resolveFormKey.currentState!.validate()) {
                            _confirmController.start();
                            try {
                              final res = await resolve(Request(
                                url: _urlController.text,
                              ));
                              await _showResolveDialog(res);
                            } catch (e) {
                              Get.snackbar('error'.tr, e.toString());
                            } finally {
                              _confirmController.reset();
                            }
                          }
                        },
                        controller: _confirmController,
                        child: Text('confirm'.tr),
                      ),
                    )),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _showResolveDialog(Resource res) async {
    final createFormKey = GlobalKey<FormState>();
    final pathController =
        TextEditingController(text: Setting.instance.downloadDir);
    final downloadController = RoundedLoadingButtonController();
    var fileValues = List.filled(res.files.length, true);

    return await showDialog<void>(
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
                            Expanded(
                                child: FileListView(
                              files: res.files,
                              values: fileValues,
                            )),
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
                                    Get.theme.backgroundColor)),
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
                        if (createFormKey.currentState!.validate()) {
                          downloadController.start();
                          if (Util.isAndroid()) {
                            if (!await Permission.storage.request().isGranted) {
                              Get.snackbar('error'.tr,
                                  'create.error.noStoragePermission'.tr);
                              return;
                            }
                          }
                          await createTask(CreateTask(
                              res: res,
                              opts: Options(
                                name: '',
                                path: pathController.text,
                                connections: Setting.instance.connections,
                                selectFiles: fileValues
                                    .asMap()
                                    .entries
                                    .where((e) => e.value)
                                    .map((e) => e.key)
                                    .toList(),
                              )));
                          Get.back();
                          Get.rootDelegate.offNamed(Routes.task);
                        }
                      } catch (e) {
                        Get.snackbar('error'.tr, e.toString());
                        rethrow;
                      } finally {
                        downloadController.reset();
                      }
                    },
                    controller: downloadController,
                    child: Text('create.download'.tr),
                  ),
                ),
              ],
            ));
  }
}
