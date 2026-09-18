import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../application/app_runtime_controller.dart';
import '../../l10n/l10n.dart';

/// Gates page content without unmounting the Navigator during startup.
class RuntimePage extends ConsumerWidget {
  const RuntimePage({super.key, required this.builder});

  final WidgetBuilder builder;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ref
        .watch(appRuntimeControllerProvider)
        .when(
          loading: () => const Scaffold(child: Center(child: CircularProgressIndicator())),
          error: (error, _) => Scaffold(
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 420),
                child: Text(context.l10n.runtimeInitializationFailed(error.toString()), textAlign: TextAlign.center),
              ),
            ),
          ),
          data: (_) => builder(context),
        );
  }
}
