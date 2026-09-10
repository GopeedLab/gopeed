import 'capability_rpc.dart';

abstract final class NavigationMethods {
  static const showDownloadingTasks = RpcMethod<RpcUnit, RpcUnit>('navigation.tasks.downloading');
}

class AppNavigationService {
  const AppNavigationService(this._invoker);

  final CapabilityInvoker _invoker;

  Future<void> showDownloadingTasks() async {
    await _invoker.invoke(NavigationMethods.showDownloadingTasks, const RpcUnit());
  }
}
