import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:get/get.dart';

import '../../../views/responsive_builder.dart';
import '../controllers/login_controller.dart';

class LoginView extends GetView<LoginController> {
  const LoginView({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final isNarrow = ResponsiveBuilder.isNarrow(context);

    return Scaffold(
      body: Container(
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [
              Get.theme.colorScheme.surface,
              Get.theme.colorScheme.surface.withOpacity(0.8),
              Get.theme.colorScheme.surface.withOpacity(0.9),
            ],
          ),
        ),
        child: Center(
          child: SingleChildScrollView(
            padding: EdgeInsets.symmetric(
              horizontal: isNarrow ? 16.0 : 32.0,
              vertical: isNarrow ? 24.0 : 32.0,
            ),
            child: Card(
              elevation: 12,
              shadowColor: Get.theme.colorScheme.shadow.withOpacity(0.3),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(isNarrow ? 20 : 24),
              ),
              color: Get.theme.colorScheme.surface,
              child: Container(
                constraints: BoxConstraints(
                  maxWidth: isNarrow ? double.infinity : 420,
                ),
                padding: EdgeInsets.all(isNarrow ? 24.0 : 40.0),
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(isNarrow ? 20 : 24),
                  border: Border.all(
                    color: Get.theme.colorScheme.outline.withOpacity(0.1),
                    width: 1,
                  ),
                ),
                child: FocusTraversalGroup(
                  policy: OrderedTraversalPolicy(),
                  child: Form(
                    key: controller.formKey,
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        // Logo/Title Section
                        Container(
                          alignment: Alignment.center,
                          margin: EdgeInsets.only(bottom: isNarrow ? 32 : 48),
                          child: Column(
                            children: [
                              Container(
                                padding: EdgeInsets.all(isNarrow ? 8 : 12),
                                decoration: BoxDecoration(
                                  borderRadius:
                                      BorderRadius.circular(isNarrow ? 20 : 24),
                                  boxShadow: [
                                    BoxShadow(
                                      color: Get.theme.colorScheme.primary
                                          .withOpacity(0.2),
                                      blurRadius: isNarrow ? 16 : 24,
                                      offset: Offset(0, isNarrow ? 8 : 12),
                                      spreadRadius: isNarrow ? 1 : 2,
                                    ),
                                  ],
                                ),
                                child: ClipRRect(
                                  borderRadius:
                                      BorderRadius.circular(isNarrow ? 16 : 20),
                                  child: SvgPicture.asset(
                                    'assets/icon/icon.svg',
                                    width: isNarrow ? 56 : 72,
                                    height: isNarrow ? 56 : 72,
                                    fit: BoxFit.cover,
                                  ),
                                ),
                              ),
                              const SizedBox(height: 8),
                              Text(
                                'Gopeed',
                                style: Get.textTheme.headlineLarge?.copyWith(
                                  fontWeight: FontWeight.w900,
                                  color: Get.theme.colorScheme.onSurface,
                                  letterSpacing: 2.0,
                                  fontSize: isNarrow ? 28 : 36,
                                ),
                              ),
                            ],
                          ),
                        ),

                        // Username Field
                        FocusTraversalOrder(
                          order: const NumericFocusOrder(1.0),
                          child: TextFormField(
                            controller: controller.usernameController,
                            autofillHints: const [
                              AutofillHints.username,
                            ],
                            autofocus: true,
                            textInputAction: TextInputAction.done,
                            onFieldSubmitted: (_) => controller.login(),
                            decoration: InputDecoration(
                              labelText: 'username'.tr,
                              labelStyle: TextStyle(
                                color: Get.theme.colorScheme.onSurface
                                    .withOpacity(0.7),
                                fontWeight: FontWeight.w500,
                                fontSize: 16,
                              ),
                              prefixIcon: Container(
                                margin: const EdgeInsets.all(12),
                                padding: const EdgeInsets.all(8),
                                decoration: BoxDecoration(
                                  color: Get.theme.colorScheme.primary
                                      .withOpacity(0.1),
                                  borderRadius: BorderRadius.circular(12),
                                ),
                                child: Icon(
                                  Icons.person_outline_rounded,
                                  color: Get.theme.colorScheme.primary,
                                  size: 20,
                                ),
                              ),
                              border: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(18),
                                borderSide: BorderSide(
                                  color: Get.theme.colorScheme.outline
                                      .withOpacity(0.2),
                                  width: 1,
                                ),
                              ),
                              enabledBorder: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(18),
                                borderSide: BorderSide(
                                  color: Get.theme.colorScheme.outline
                                      .withOpacity(0.3),
                                  width: 1.5,
                                ),
                              ),
                              focusedBorder: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(18),
                                borderSide: BorderSide(
                                  color: Get.theme.colorScheme.primary,
                                  width: 2.5,
                                ),
                              ),
                              errorBorder: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(18),
                                borderSide: BorderSide(
                                  color: Get.theme.colorScheme.error,
                                  width: 2,
                                ),
                              ),
                              focusedErrorBorder: OutlineInputBorder(
                                borderRadius: BorderRadius.circular(18),
                                borderSide: BorderSide(
                                  color: Get.theme.colorScheme.error,
                                  width: 2.5,
                                ),
                              ),
                              filled: true,
                              fillColor: Get
                                  .theme.colorScheme.surfaceContainerHighest
                                  .withOpacity(0.8),
                              contentPadding: const EdgeInsets.symmetric(
                                horizontal: 24,
                                vertical: 20,
                              ),
                            ),
                            validator: (value) {
                              if (value == null || value.trim().isEmpty) {
                                return 'username_required'.tr;
                              }
                              return null;
                            },
                          ),
                        ),

                        SizedBox(height: isNarrow ? 16 : 24),

                        // Password Field
                        FocusTraversalOrder(
                          order: const NumericFocusOrder(2.0),
                          child: Obx(() => TextFormField(
                                controller: controller.passwordController,
                                autofillHints: const [
                                  AutofillHints.password,
                                ],
                                obscureText: !controller.passwordVisible.value,
                                decoration: InputDecoration(
                                  labelText: 'password'.tr,
                                  labelStyle: TextStyle(
                                    color: Get.theme.colorScheme.onSurface
                                        .withOpacity(0.7),
                                    fontWeight: FontWeight.w500,
                                    fontSize: 16,
                                  ),
                                  prefixIcon: Container(
                                    margin: const EdgeInsets.all(12),
                                    padding: const EdgeInsets.all(8),
                                    decoration: BoxDecoration(
                                      color: Get.theme.colorScheme.primary
                                          .withOpacity(0.1),
                                      borderRadius: BorderRadius.circular(12),
                                    ),
                                    child: Icon(
                                      Icons.lock_outline_rounded,
                                      color: Get.theme.colorScheme.primary,
                                      size: 20,
                                    ),
                                  ),
                                  suffixIcon: IconButton(
                                    icon: Icon(
                                      controller.passwordVisible.value
                                          ? Icons.visibility_off_rounded
                                          : Icons.visibility_rounded,
                                      color: Get.theme.colorScheme.onSurface
                                          .withOpacity(0.6),
                                    ),
                                    onPressed:
                                        controller.togglePasswordVisibility,
                                  ),
                                  border: OutlineInputBorder(
                                    borderRadius: BorderRadius.circular(18),
                                    borderSide: BorderSide(
                                      color: Get.theme.colorScheme.outline
                                          .withOpacity(0.2),
                                      width: 1,
                                    ),
                                  ),
                                  enabledBorder: OutlineInputBorder(
                                    borderRadius: BorderRadius.circular(18),
                                    borderSide: BorderSide(
                                      color: Get.theme.colorScheme.outline
                                          .withOpacity(0.3),
                                      width: 1.5,
                                    ),
                                  ),
                                  focusedBorder: OutlineInputBorder(
                                    borderRadius: BorderRadius.circular(18),
                                    borderSide: BorderSide(
                                      color: Get.theme.colorScheme.primary,
                                      width: 2.5,
                                    ),
                                  ),
                                  errorBorder: OutlineInputBorder(
                                    borderRadius: BorderRadius.circular(18),
                                    borderSide: BorderSide(
                                      color: Get.theme.colorScheme.error,
                                      width: 2,
                                    ),
                                  ),
                                  focusedErrorBorder: OutlineInputBorder(
                                    borderRadius: BorderRadius.circular(18),
                                    borderSide: BorderSide(
                                      color: Get.theme.colorScheme.error,
                                      width: 2.5,
                                    ),
                                  ),
                                  filled: true,
                                  fillColor: Get
                                      .theme.colorScheme.surfaceContainerHighest
                                      .withOpacity(0.8),
                                  contentPadding: const EdgeInsets.symmetric(
                                    horizontal: 24,
                                    vertical: 20,
                                  ),
                                ),
                                validator: (value) {
                                  if (value == null || value.isEmpty) {
                                    return 'password_required'.tr;
                                  }
                                  return null;
                                },
                                textInputAction: TextInputAction.done,
                                onFieldSubmitted: (_) => controller.login(),
                              )),
                        ),

                        SizedBox(height: isNarrow ? 24 : 32),

                        // Login Button
                        Obx(() => ElevatedButton(
                              onPressed: controller.isLoading.value
                                  ? null
                                  : controller.login,
                              style: ElevatedButton.styleFrom(
                                padding: EdgeInsets.symmetric(
                                  vertical: isNarrow ? 16 : 18,
                                ),
                                shape: RoundedRectangleBorder(
                                  borderRadius: BorderRadius.circular(16),
                                ),
                                backgroundColor: controller.isLoading.value
                                    ? Get.theme.disabledColor
                                    : Get.theme.colorScheme.primary,
                                foregroundColor:
                                    Get.theme.colorScheme.onPrimary,
                                elevation: controller.isLoading.value ? 0 : 4,
                                shadowColor: Get.theme.colorScheme.primary
                                    .withOpacity(0.4),
                              ),
                              child: controller.isLoading.value
                                  ? SizedBox(
                                      height: 24,
                                      width: 24,
                                      child: CircularProgressIndicator(
                                        strokeWidth: 2.5,
                                        valueColor:
                                            AlwaysStoppedAnimation<Color>(Get
                                                .theme.colorScheme.onPrimary),
                                      ),
                                    )
                                  : Text(
                                      'login'.tr,
                                      style: const TextStyle(
                                        fontSize: 18,
                                        fontWeight: FontWeight.w600,
                                        letterSpacing: 0.5,
                                      ),
                                    ),
                            )),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
