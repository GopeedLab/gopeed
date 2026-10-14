import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/widgets/app_empty_state.dart';

class TaskEmptyCreateHint extends StatelessWidget {
  const TaskEmptyCreateHint({super.key, required this.onCreateTask});

  final VoidCallback onCreateTask;

  @override
  Widget build(BuildContext context) {
    return AppEmptyStateAction(
      buttonKey: const ValueKey('tasks-empty-create-button'),
      icon: Icons.add,
      label: context.l10n.create,
      hint: context.l10n.emptyTaskCreateHint,
      onPressed: onCreateTask,
    );
  }
}
