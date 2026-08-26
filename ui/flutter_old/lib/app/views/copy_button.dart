import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../util/message.dart';

class CopyButton extends StatefulWidget {
  final String? url;

  const CopyButton(this.url, {Key? key}) : super(key: key);

  @override
  State<CopyButton> createState() => _CopyButtonState();
}

class _CopyButtonState extends State<CopyButton> {
  bool success = false;

  copy() {
    final url = widget.url;
    if (url != null) {
      try {
        Clipboard.setData(ClipboardData(text: url));
        setState(() {
          success = true;
        });
        Future.delayed(const Duration(milliseconds: 300), () {
          setState(() {
            success = false;
          });
        });
      } catch (e) {
        showErrorMessage(e);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return IconButton(
      icon: success ? const Icon(Icons.check_circle) : const Icon(Icons.copy),
      onPressed: copy,
    );
  }
}
