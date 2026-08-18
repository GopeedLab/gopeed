import 'dart:async';
import 'package:get/get.dart';
import '../../../api/api.dart' as api;
import '../../../api/model/task.dart';
import '../modules/app/controllers/app_controller.dart';
import 'location_keep_alive.dart';

class LocationKeepAliveCoordinator extends GetxService {
  Timer? _pollTimer;
  bool _reconciling = false;

  /// Overridable for testing, defaults to reading from AppController.
  bool Function() keepAliveEnabledProvider = () => Get.find<AppController>()
      .downloaderConfig
      .value
      .extra
      .backgroundLocationKeepAlive;

  /// Overridable for testing so polling tests don't need to wait 10s.
  Duration pollInterval = const Duration(seconds: 10);

  bool get _keepAliveEnabled => keepAliveEnabledProvider();

  @override
  void onInit() {
    super.onInit();
    api.setTaskChangedListener(() {
      reconcile();
    });
  }

  Future<int> _fetchGlobalRunningCount() async {
    try {
      final tasks = await api.getTasks([Status.running]);
      return tasks.length;
    } catch (_) {
      return -1;
    }
  }

  Future<void> reconcile() async {
    if (_reconciling) return;
    _reconciling = true;
    try {
      if (!_keepAliveEnabled) {
        await LocationKeepAlive.stop();
        return;
      }
      final count = await _fetchGlobalRunningCount();
      if (count < 0) return;
      if (count > 0) {
        await LocationKeepAlive.start();
      } else {
        await LocationKeepAlive.stop();
      }
    } finally {
      _reconciling = false;
    }
  }

  void startPolling() {
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(pollInterval, (_) => reconcile());
  }

  void stopPolling() {
    _pollTimer?.cancel();
  }

  @override
  void onClose() {
    stopPolling();
    super.onClose();
  }
}