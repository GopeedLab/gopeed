import 'dart:math' as math;

import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../app/application/app_runtime_controller.dart';
import '../../../../core/window/app_window_chrome.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../application/web_auth_controller.dart';
import '../widgets/login_brand_art.dart';
import '../widgets/login_form.dart';

class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});

  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends ConsumerState<LoginPage> {
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  final _usernameFocusNode = FocusNode();
  final _passwordFocusNode = FocusNode();

  String? _usernameError;
  String? _passwordError;
  WebLoginFailure? _loginFailure;
  bool _completingLogin = false;

  @override
  void dispose() {
    _usernameController.dispose();
    _passwordController.dispose();
    _usernameFocusNode.dispose();
    _passwordFocusNode.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final username = _usernameController.text.trim();
    final password = _passwordController.text;
    setState(() {
      _usernameError = username.isEmpty ? context.l10n.username_required : null;
      _passwordError = password.isEmpty ? context.l10n.password_required : null;
      _loginFailure = null;
    });
    if (_usernameError != null || _passwordError != null) {
      if (_usernameError != null) {
        _usernameFocusNode.requestFocus();
      } else {
        _passwordFocusNode.requestFocus();
      }
      return;
    }

    final authController = ref.read(webAuthControllerProvider);
    final failure = await authController.login(username: username, password: password);
    if (!mounted) return;
    if (failure != null) {
      setState(() => _loginFailure = failure);
      return;
    }

    setState(() => _completingLogin = true);
    try {
      await ref.read(appRuntimeControllerProvider.notifier).reloadConfig();
      if (!mounted) return;
      TextInput.finishAutofillContext(shouldSave: true);
      authController.completeLogin();
    } catch (_) {
      authController.requireLogin();
      if (!mounted) return;
      setState(() => _loginFailure = WebLoginFailure.network);
    } finally {
      if (mounted) setState(() => _completingLogin = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final authController = ref.read(webAuthControllerProvider);
    return ListenableBuilder(
      listenable: authController,
      builder: (context, _) {
        final loading = authController.isLoggingIn || _completingLogin;
        return _LoginPageLayout(
          form: LoginForm(
            usernameController: _usernameController,
            passwordController: _passwordController,
            usernameFocusNode: _usernameFocusNode,
            passwordFocusNode: _passwordFocusNode,
            loading: loading,
            usernameError: _usernameError,
            passwordError: _passwordError,
            loginError: switch (_loginFailure) {
              WebLoginFailure.credentials => context.l10n.login_failed,
              WebLoginFailure.network => context.l10n.login_failed_network,
              null => null,
            },
            onUsernameChanged: (_) {
              if (_usernameError == null && _loginFailure == null) return;
              setState(() {
                _usernameError = null;
                _loginFailure = null;
              });
            },
            onPasswordChanged: (_) {
              if (_passwordError == null && _loginFailure == null) return;
              setState(() {
                _passwordError = null;
                _loginFailure = null;
              });
            },
            onSubmit: _submit,
          ),
        );
      },
    );
  }
}

class _LoginPageLayout extends StatelessWidget {
  const _LoginPageLayout({required this.form});

  final Widget form;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final topInset = AppWindowChrome.reservesHeaderInset ? AppDesignTokens.windowHeaderHeight : 0.0;
    return shad.Scaffold(
      child: ColoredBox(
        color: palette.bg,
        child: Padding(
          padding: EdgeInsets.only(top: topInset),
          child: SafeArea(
            top: false,
            child: LayoutBuilder(
              builder: (context, constraints) {
                final desktop = constraints.maxWidth >= 900;
                final horizontalPadding = desktop ? AppDesignTokens.space32 : AppDesignTokens.space16;
                final verticalPadding = desktop ? AppDesignTokens.space24 : AppDesignTokens.space16;
                final availableWidth = math.max(0.0, constraints.maxWidth - horizontalPadding * 2);
                final cardWidth = math.min(desktop ? 980.0 : 440.0, availableWidth);
                final card = SizedBox(
                  width: cardWidth,
                  child: desktop ? _DesktopLoginCard(form: form) : _CompactLoginCard(form: form),
                );
                if (desktop) {
                  return Padding(
                    padding: EdgeInsets.symmetric(horizontal: horizontalPadding, vertical: verticalPadding),
                    child: Center(
                      child: FittedBox(fit: BoxFit.scaleDown, child: card),
                    ),
                  );
                }
                return Padding(
                  padding: EdgeInsets.symmetric(horizontal: horizontalPadding, vertical: verticalPadding),
                  child: Center(
                    child: FittedBox(fit: BoxFit.scaleDown, child: card),
                  ),
                );
              },
            ),
          ),
        ),
      ),
    );
  }
}

class _DesktopLoginCard extends StatelessWidget {
  const _DesktopLoginCard({required this.form});

  final Widget form;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return shad.Card(
      key: const ValueKey('login-desktop-card'),
      padding: EdgeInsets.zero,
      borderRadius: BorderRadius.circular(AppDesignTokens.windowRadius),
      borderColor: palette.border,
      boxShadow: [
        BoxShadow(color: palette.textPrimary.withValues(alpha: 0.08), blurRadius: 36, offset: const Offset(0, 14)),
      ],
      clipBehavior: Clip.antiAlias,
      child: Row(
        children: [
          const Expanded(child: SizedBox(height: 520, child: LoginBrandArt())),
          SizedBox(
            width: 420,
            child: Padding(padding: const EdgeInsets.symmetric(horizontal: 48, vertical: 56), child: form),
          ),
        ],
      ),
    );
  }
}

class _CompactLoginCard extends StatelessWidget {
  const _CompactLoginCard({required this.form});

  final Widget form;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: 440),
      child: shad.Card(
        key: const ValueKey('login-compact-card'),
        padding: EdgeInsets.zero,
        borderRadius: BorderRadius.circular(AppDesignTokens.windowRadius),
        borderColor: palette.border,
        boxShadow: [
          BoxShadow(color: palette.textPrimary.withValues(alpha: 0.08), blurRadius: 28, offset: const Offset(0, 10)),
        ],
        clipBehavior: Clip.antiAlias,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const SizedBox(height: 190, child: LoginBrandArt(compact: true)),
            Padding(padding: const EdgeInsets.all(AppDesignTokens.space24), child: form),
          ],
        ),
      ),
    );
  }
}
