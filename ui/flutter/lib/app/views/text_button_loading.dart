import 'package:flutter/material.dart';

class TextButtonLoading extends StatefulWidget {
  final Widget child;
  final VoidCallback? onPressed;
  final TextButtonLoadingController controller;

  const TextButtonLoading(
      {Key? key,
      required this.child,
      required this.onPressed,
      required this.controller})
      : super(key: key);

  @override
  State<TextButtonLoading> createState() => _TextButtonLoadingState();
}

class _TextButtonLoadingState extends State<TextButtonLoading> {
  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<bool>(
      valueListenable: widget.controller,
      builder: (context, value, child) {
        return TextButton(
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

class TextButtonLoadingController extends ValueNotifier<bool> {
  TextButtonLoadingController() : super(false);

  void start() {
    value = true;
  }

  void stop() {
    value = false;
  }
}
