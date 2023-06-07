import 'package:get/get.dart';

class CreateController extends GetxController {
  // final files = [].obs;
  final RxList fileInfos = [].obs;
  final RxList selectedIndexes = [].obs;
  final RxList openedFolders = [].obs;
  final isResolving = false.obs;
}
