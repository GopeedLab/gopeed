import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../shared/theme/app_design_tokens.dart';

class AppUpdateNotesViewport extends StatefulWidget {
  const AppUpdateNotesViewport({super.key, required this.child});

  final Widget child;

  @override
  State<AppUpdateNotesViewport> createState() => _AppUpdateNotesViewportState();
}

class _AppUpdateNotesViewportState extends State<AppUpdateNotesViewport> {
  final _controller = ScrollController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    const fadeHeight = AppDesignTokens.space24;
    return ConstrainedBox(
      constraints: const BoxConstraints(maxHeight: 240),
      child: shad.Scrollbar(
        controller: _controller,
        thumbVisibility: true,
        child: ShaderMask(
          blendMode: BlendMode.dstIn,
          shaderCallback: (bounds) => LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            // Mask colors control opacity only; they do not tint the content.
            colors: const [shad.Colors.white, shad.Colors.white, shad.Colors.transparent],
            stops: [0, (1 - fadeHeight / bounds.height).clamp(0.0, 1.0), 1],
          ).createShader(bounds),
          child: ScrollConfiguration(
            behavior: ScrollConfiguration.of(context).copyWith(scrollbars: false),
            child: SingleChildScrollView(
              controller: _controller,
              padding: const EdgeInsetsDirectional.fromSTEB(
                AppDesignTokens.space4,
                0,
                AppDesignTokens.space16,
                // Keep the final line above the fade at the end of the scroll.
                fadeHeight,
              ),
              child: widget.child,
            ),
          ),
        ),
      ),
    );
  }
}
