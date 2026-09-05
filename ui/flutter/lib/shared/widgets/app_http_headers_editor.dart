import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../theme/app_design_tokens.dart';
import '../theme/app_palette.dart';
import '../../l10n/l10n.dart';

class AppHttpHeadersController extends ChangeNotifier {
  AppHttpHeadersController({Map<String, String>? headers, Iterable<String> defaultNames = const []}) {
    if (headers != null && headers.isNotEmpty) {
      _rows.addAll(headers.entries.map((entry) => _HttpHeaderRowControllers(name: entry.key, value: entry.value)));
    } else {
      _rows.addAll(defaultNames.map((name) => _HttpHeaderRowControllers(name: name)));
    }
    _ensureRow();
  }

  final List<_HttpHeaderRowControllers> _rows = [];

  void replace(Map<String, String> headers) {
    for (final row in _rows) {
      row.dispose();
    }
    _rows
      ..clear()
      ..addAll(headers.entries.map((entry) => _HttpHeaderRowControllers(name: entry.key, value: entry.value)));
    _ensureRow();
    notifyListeners();
  }

  Map<String, String> toMap({bool requireValue = false}) {
    final headers = <String, String>{};
    for (final row in _rows) {
      final name = row.name.text.trim();
      final value = row.value.text.trim();
      if (name.isNotEmpty && (!requireValue || value.isNotEmpty)) {
        headers[name] = value;
      }
    }
    return headers;
  }

  void add() {
    _rows.add(_HttpHeaderRowControllers());
    notifyListeners();
  }

  void removeAt(int index) {
    if (_rows.length == 1) return;
    _rows.removeAt(index).dispose();
    notifyListeners();
  }

  void _ensureRow() {
    if (_rows.isEmpty) {
      _rows.add(_HttpHeaderRowControllers());
    }
  }

  @override
  void dispose() {
    for (final row in _rows) {
      row.dispose();
    }
    super.dispose();
  }
}

class AppHttpHeadersEditor extends StatelessWidget {
  const AppHttpHeadersEditor({super.key, required this.controller, required this.label, required this.keyPrefix});

  final AppHttpHeadersController controller;
  final String label;
  final String keyPrefix;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Row(
      key: ValueKey('$keyPrefix-editor'),
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 104,
          child: Padding(
            padding: const EdgeInsets.only(top: 10),
            child: Text(
              label,
              style: TextStyle(color: palette.textPrimary, fontSize: 13, fontWeight: FontWeight.w500),
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: AnimatedBuilder(
            animation: controller,
            builder: (context, _) => Column(
              children: [
                for (var index = 0; index < controller._rows.length; index++) ...[
                  Row(
                    children: [
                      Expanded(
                        flex: 382,
                        child: _HeaderTextField(
                          key: ValueKey('$keyPrefix-name-$index'),
                          controller: controller._rows[index].name,
                          hintText: context.l10n.httpHeaderName,
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        flex: 618,
                        child: _HeaderTextField(
                          key: ValueKey('$keyPrefix-value-$index'),
                          controller: controller._rows[index].value,
                          hintText: context.l10n.httpHeaderValue,
                        ),
                      ),
                      const SizedBox(width: 4),
                      _HeaderAction(
                        key: ValueKey('$keyPrefix-remove-$index'),
                        icon: Icons.remove,
                        onPressed: controller._rows.length == 1 ? null : () => controller.removeAt(index),
                      ),
                      const SizedBox(width: 4),
                      Visibility(
                        visible: index == controller._rows.length - 1,
                        maintainAnimation: true,
                        maintainSize: true,
                        maintainState: true,
                        child: _HeaderAction(
                          key: index == controller._rows.length - 1 ? ValueKey('$keyPrefix-add') : null,
                          icon: Icons.add,
                          onPressed: controller.add,
                        ),
                      ),
                    ],
                  ),
                  if (index != controller._rows.length - 1) const SizedBox(height: 8),
                ],
              ],
            ),
          ),
        ),
      ],
    );
  }
}

class _HttpHeaderRowControllers {
  _HttpHeaderRowControllers({String name = '', String value = ''})
    : name = TextEditingController(text: name),
      value = TextEditingController(text: value);

  final TextEditingController name;
  final TextEditingController value;

  void dispose() {
    name.dispose();
    value.dispose();
  }
}

class _HeaderTextField extends StatelessWidget {
  const _HeaderTextField({super.key, required this.controller, required this.hintText});

  final TextEditingController controller;
  final String hintText;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return shad.TextField(
      controller: controller,
      hintText: hintText,
      filled: true,
      border: Border.all(color: palette.border),
      borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
    );
  }
}

class _HeaderAction extends StatelessWidget {
  const _HeaderAction({super.key, required this.icon, this.onPressed});

  final IconData icon;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return shad.GhostButton(
      density: shad.ButtonDensity.icon,
      onPressed: onPressed,
      child: Icon(icon, size: 14, color: palette.textSecondary),
    );
  }
}
