import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_loading_button.dart';

class LoginForm extends StatelessWidget {
  const LoginForm({
    super.key,
    required this.usernameController,
    required this.passwordController,
    required this.usernameFocusNode,
    required this.passwordFocusNode,
    required this.loading,
    required this.onSubmit,
    required this.onUsernameChanged,
    required this.onPasswordChanged,
    this.usernameError,
    this.passwordError,
    this.loginError,
  });

  final TextEditingController usernameController;
  final TextEditingController passwordController;
  final FocusNode usernameFocusNode;
  final FocusNode passwordFocusNode;
  final bool loading;
  final VoidCallback onSubmit;
  final ValueChanged<String> onUsernameChanged;
  final ValueChanged<String> onPasswordChanged;
  final String? usernameError;
  final String? passwordError;
  final String? loginError;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return AutofillGroup(
      onDisposeAction: AutofillContextAction.cancel,
      child: FocusTraversalGroup(
        policy: OrderedTraversalPolicy(),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              context.l10n.login,
              key: const ValueKey('login-title'),
              style: TextStyle(color: palette.textPrimary, fontSize: 30, fontWeight: FontWeight.w700, height: 1.15),
            ),
            const SizedBox(height: AppDesignTokens.space24),
            _LoginField(
              fieldKey: const ValueKey('login-username-field'),
              label: context.l10n.username,
              controller: usernameController,
              focusNode: usernameFocusNode,
              autofocus: true,
              enabled: !loading,
              keyboardType: TextInputType.text,
              textInputAction: TextInputAction.done,
              autofillHints: const [AutofillHints.username],
              error: usernameError,
              onChanged: onUsernameChanged,
              onSubmitted: (_) => onSubmit(),
            ),
            const SizedBox(height: AppDesignTokens.space16),
            _LoginField(
              fieldKey: const ValueKey('login-password-field'),
              label: context.l10n.password,
              controller: passwordController,
              focusNode: passwordFocusNode,
              enabled: !loading,
              keyboardType: TextInputType.visiblePassword,
              textInputAction: TextInputAction.done,
              autofillHints: const [AutofillHints.password],
              obscureText: true,
              error: passwordError,
              onChanged: onPasswordChanged,
              onSubmitted: (_) => onSubmit(),
            ),
            AnimatedSize(
              duration: const Duration(milliseconds: 160),
              curve: Curves.easeOut,
              child: loginError == null
                  ? const SizedBox(height: AppDesignTokens.space24)
                  : Padding(
                      padding: const EdgeInsets.only(top: AppDesignTokens.space12, bottom: AppDesignTokens.space12),
                      child: Text(
                        loginError!,
                        key: const ValueKey('login-error'),
                        style: TextStyle(color: palette.error, fontSize: 12, height: 1.35),
                      ),
                    ),
            ),
            Center(
              child: SizedBox(
                key: const ValueKey('login-submit-button-container'),
                width: 148,
                height: 40,
                child: AppLoadingButton(
                  key: const ValueKey('login-submit-button'),
                  onPressed: loading ? null : onSubmit,
                  loading: loading,
                  alignment: Alignment.center,
                  variant: AppLoadingButtonVariant.primary,
                  child: Text(context.l10n.login, textAlign: TextAlign.center),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _LoginField extends StatelessWidget {
  const _LoginField({
    required this.fieldKey,
    required this.label,
    required this.controller,
    required this.focusNode,
    required this.enabled,
    required this.keyboardType,
    required this.textInputAction,
    required this.autofillHints,
    required this.onChanged,
    required this.onSubmitted,
    this.autofocus = false,
    this.obscureText = false,
    this.error,
  });

  final Key fieldKey;
  final String label;
  final TextEditingController controller;
  final FocusNode focusNode;
  final bool enabled;
  final TextInputType keyboardType;
  final TextInputAction textInputAction;
  final Iterable<String> autofillHints;
  final ValueChanged<String> onChanged;
  final ValueChanged<String> onSubmitted;
  final bool autofocus;
  final bool obscureText;
  final String? error;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          label,
          style: TextStyle(color: palette.textPrimary, fontSize: 13, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: AppDesignTokens.space8),
        shad.TextField(
          key: fieldKey,
          controller: controller,
          focusNode: focusNode,
          autofocus: autofocus,
          enabled: enabled,
          keyboardType: keyboardType,
          textInputAction: textInputAction,
          autofillHints: autofillHints,
          autocorrect: false,
          enableSuggestions: false,
          obscureText: obscureText,
          onChanged: onChanged,
          onSubmitted: onSubmitted,
          border: Border.all(color: error == null ? palette.border : palette.error),
          borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
          filled: true,
          features: obscureText
              ? const [shad.InputPasswordToggleFeature(mode: shad.PasswordPeekMode.toggle)]
              : const [],
        ),
        if (error != null) ...[
          const SizedBox(height: AppDesignTokens.space4),
          Text(error!, style: TextStyle(color: palette.error, fontSize: 12, height: 1.3)),
        ],
      ],
    );
  }
}
