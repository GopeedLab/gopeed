import 'package:flutter/material.dart';

class OutlinedButtonLoading extends StatefulWidget {
  final Widget child;
  final VoidCallback onPressed;
  final OutlinedButtonLoadingController controller;

  const OutlinedButtonLoading(
      {super.key,
      required this.child,
      required this.onPressed,
      required this.controller});

  @override
  State<OutlinedButtonLoading> createState() => _OutlinedButtonLoadingState();
}

class _OutlinedButtonLoadingState extends State<OutlinedButtonLoading> {
  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<bool>(
      valueListenable: widget.controller,
      builder: (context, value, child) {
        return OutlinedButton(
          key: widget.key,
          onPressed: value ? null : widget.onPressed,
          child: value
              ? const SizedBox(
                  height: 20,
                  width: 20,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                  ),
                )
              : widget.child,
        );
      },
    );
  }
}

class OutlinedButtonLoadingController extends ValueNotifier<bool> {
  OutlinedButtonLoadingController() : super(false);

  void start() {
    value = true;
  }

  void stop() {
    value = false;
  }
}
