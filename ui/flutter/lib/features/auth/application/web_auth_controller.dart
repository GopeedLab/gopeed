import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/api.dart' as api;
import '../../../api/model/login.dart';

final webAuthControllerProvider = Provider<WebAuthController>((ref) {
  final controller = WebAuthController();
  ref.onDispose(controller.dispose);
  return controller;
});

enum WebLoginFailure { credentials, network }

class WebAuthController extends ChangeNotifier {
  bool _loginRequired = false;
  bool _isLoggingIn = false;

  bool get loginRequired => _loginRequired;
  bool get isLoggingIn => _isLoggingIn;

  void requireLogin() {
    if (_loginRequired) return;
    _loginRequired = true;
    notifyListeners();
  }

  Future<WebLoginFailure?> login({required String username, required String password}) async {
    if (_isLoggingIn) return null;
    _isLoggingIn = true;
    notifyListeners();
    try {
      await api.login(LoginReq(username: username, password: password));
      return null;
    } on api.ApiTimeoutException {
      return WebLoginFailure.network;
    } catch (_) {
      return WebLoginFailure.credentials;
    } finally {
      _isLoggingIn = false;
      notifyListeners();
    }
  }

  void completeLogin() {
    if (!_loginRequired) return;
    _loginRequired = false;
    notifyListeners();
  }
}
