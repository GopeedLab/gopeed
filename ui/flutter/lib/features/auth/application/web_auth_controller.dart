import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/api.dart' as api;
import '../../../api/model/login.dart';
import '../../../database/database.dart';

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
    Database.instance.clearWebToken();
    if (_loginRequired) return;
    _loginRequired = true;
    notifyListeners();
  }

  Future<WebLoginFailure?> login({required String username, required String password}) async {
    if (_isLoggingIn) return null;
    _isLoggingIn = true;
    notifyListeners();
    try {
      final token = await api.login(LoginReq(username: username, password: password));
      Database.instance.saveWebToken(token);
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
