import 'package:flutter/widgets.dart';

import '../theme/app_design_tokens.dart';
import 'app_form_row.dart';

/// Related fields that share a row on wide forms and get individual labels when stacked.
class AppFormPair extends StatelessWidget {
  const AppFormPair({
    super.key,
    required this.firstLabel,
    required this.secondLabel,
    required this.first,
    required this.second,
    this.direction = Axis.horizontal,
    this.firstFlex = 1,
    this.secondFlex = 1,
  });

  final String firstLabel;
  final String secondLabel;
  final Widget first;
  final Widget second;
  final Axis direction;
  final int firstFlex;
  final int secondFlex;

  @override
  Widget build(BuildContext context) {
    if (direction == Axis.vertical) {
      return Column(
        children: [
          AppFormRow(label: firstLabel, direction: direction, child: first),
          const SizedBox(height: AppDesignTokens.space12),
          AppFormRow(label: secondLabel, direction: direction, child: second),
        ],
      );
    }
    return AppFormRow(
      label: firstLabel,
      child: Row(
        children: [
          Expanded(flex: firstFlex, child: first),
          const SizedBox(width: AppDesignTokens.space8),
          Expanded(flex: secondFlex, child: second),
        ],
      ),
    );
  }
}
