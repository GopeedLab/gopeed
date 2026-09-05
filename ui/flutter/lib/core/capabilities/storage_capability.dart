import 'capability_rpc.dart';

abstract final class StorageMethods {
  static const getCreateHistory = RpcMethod<RpcUnit, List<String>>('storage.createHistory.get');
  static const saveCreateHistory = RpcMethod<List<String>, RpcUnit>('storage.createHistory.save');
  static const removeCreateHistory = RpcMethod<String, RpcUnit>('storage.createHistory.remove');
  static const clearCreateHistory = RpcMethod<RpcUnit, RpcUnit>('storage.createHistory.clear');
}

class AppStorageService {
  const AppStorageService(this._invoker);

  final CapabilityInvoker _invoker;

  Future<List<String>> getCreateHistory() => _invoker.invoke(StorageMethods.getCreateHistory, const RpcUnit());

  Future<void> saveCreateHistory(List<String> urls) async {
    if (urls.isEmpty) return;
    await _invoker.invoke(StorageMethods.saveCreateHistory, urls);
  }

  Future<void> removeCreateHistory(String url) async {
    await _invoker.invoke(StorageMethods.removeCreateHistory, url);
  }

  Future<void> clearCreateHistory() async {
    await _invoker.invoke(StorageMethods.clearCreateHistory, const RpcUnit());
  }
}
