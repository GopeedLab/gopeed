import 'capability_rpc.dart';

abstract final class NavigationMethods {
  static const showDownloadingTasks = RpcMethod<RpcUnit, RpcUnit>('navigation.tasks.downloading');
  static const showBtSettings = RpcMethod<RpcUnit, RpcUnit>('navigation.settings.bittorrent');
}

class AppNavigationService {
  const AppNavigationService(this._invoker);

  final CapabilityInvoker _invoker;

  Future<void> showDownloadingTasks() async {
    await _invoker.invoke(NavigationMethods.showDownloadingTasks, const RpcUnit());
  }

  /// Opens the settings section with the BitTorrent settings.
  Future<void> showBtSettings() async {
    await _invoker.invoke(NavigationMethods.showBtSettings, const RpcUnit());
  }
}
