import 'dart:convert';

import 'package:flutter/material.dart' show Icons, Scrollbar, ScrollbarOrientation, SelectableText;
import 'package:flutter/widgets.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_copy_icon_button.dart';
import '../../../../shared/widgets/app_tooltip.dart';
import 'settings_item.dart';

class McpSettingsItem extends StatelessWidget {
  const McpSettingsItem({
    super.key,
    required this.enabled,
    required this.readOnly,
    required this.onChanged,
    required this.onOpenAgentSetup,
  });

  final bool enabled;
  final bool readOnly;
  final ValueChanged<bool> onChanged;
  final VoidCallback onOpenAgentSetup;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return SettingsItem(
      title: 'MCP',
      subtitle: context.l10n.mcpDescription,
      subtitleAction: _McpAgentLink(onPressed: onOpenAgentSetup),
      titleTrailing: Icon(
        key: const ValueKey('mcp-ai-icon'),
        Icons.auto_awesome_outlined,
        size: 14,
        color: palette.brandProgress,
      ),
      child: shad.Switch(
        key: const ValueKey('mcp-endpoint-switch'),
        value: enabled,
        enabled: !readOnly,
        onChanged: onChanged,
      ),
    );
  }
}

class _McpAgentLink extends StatefulWidget {
  const _McpAgentLink({required this.onPressed});

  final VoidCallback onPressed;

  @override
  State<_McpAgentLink> createState() => _McpAgentLinkState();
}

class _McpAgentLinkState extends State<_McpAgentLink> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final color = _hovered ? palette.brandProgress : palette.brand;
    return Semantics(
      link: true,
      label: context.l10n.connectAgent,
      button: true,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        onEnter: (_) => setState(() => _hovered = true),
        onExit: (_) => setState(() => _hovered = false),
        child: GestureDetector(
          key: const ValueKey('open-mcp-agent-setup'),
          behavior: HitTestBehavior.opaque,
          onTap: widget.onPressed,
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 2),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  context.l10n.connectAgent,
                  style: TextStyle(
                    color: color,
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    decoration: TextDecoration.underline,
                    decorationColor: color,
                    decorationThickness: _hovered ? 1.5 : 1,
                  ),
                ),
                const SizedBox(width: 3),
                Icon(Icons.arrow_forward, size: 13, color: color),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

enum _McpAgent {
  codex,
  claudeCode,
  cursor,
  githubCopilot,
  windsurf,
  geminiCli,
  cline,
  openCode,
  trae,
  qoder,
  workBuddy,
  zCode,
  deepSeekHarness,
  pi,
}

extension on _McpAgent {
  String get label => switch (this) {
    _McpAgent.codex => 'Codex',
    _McpAgent.claudeCode => 'Claude Code',
    _McpAgent.cursor => 'Cursor',
    _McpAgent.githubCopilot => 'GitHub Copilot',
    _McpAgent.windsurf => 'Windsurf',
    _McpAgent.geminiCli => 'Gemini CLI',
    _McpAgent.cline => 'Cline',
    _McpAgent.openCode => 'OpenCode',
    _McpAgent.trae => 'TRAE',
    _McpAgent.qoder => 'Qoder',
    _McpAgent.workBuddy => 'WorkBuddy',
    _McpAgent.zCode => 'ZCode',
    _McpAgent.deepSeekHarness => 'DeepSeek Harness',
    _McpAgent.pi => 'Pi',
  };

  String get iconAsset => switch (this) {
    _McpAgent.codex => 'assets/agent_icons/codex.svg',
    _McpAgent.claudeCode => 'assets/agent_icons/claude_code.svg',
    _McpAgent.cursor => 'assets/agent_icons/cursor.svg',
    _McpAgent.githubCopilot => 'assets/agent_icons/github_copilot.svg',
    _McpAgent.windsurf => 'assets/agent_icons/windsurf.svg',
    _McpAgent.geminiCli => 'assets/agent_icons/gemini_cli.svg',
    _McpAgent.cline => 'assets/agent_icons/cline.svg',
    _McpAgent.openCode => 'assets/agent_icons/opencode.svg',
    _McpAgent.trae => 'assets/agent_icons/trae.svg',
    _McpAgent.qoder => 'assets/agent_icons/qoder.svg',
    _McpAgent.workBuddy => 'assets/agent_icons/codebuddy.svg',
    _McpAgent.zCode => 'assets/agent_icons/zcode.svg',
    _McpAgent.deepSeekHarness => 'assets/agent_icons/deepseek.svg',
    _McpAgent.pi => 'assets/agent_icons/pi.svg',
  };

  bool get usesThemeTint => switch (this) {
    _McpAgent.cursor ||
    _McpAgent.githubCopilot ||
    _McpAgent.windsurf ||
    _McpAgent.cline ||
    _McpAgent.openCode ||
    _McpAgent.zCode ||
    _McpAgent.pi => true,
    _ => false,
  };
}

String webMcpEndpoint(Uri pageUri) {
  return Uri(
    scheme: pageUri.scheme,
    host: pageUri.host,
    port: pageUri.hasPort ? pageUri.port : null,
    path: '/mcp',
  ).toString();
}

Future<void> showMcpAgentSetupDialog(
  BuildContext context, {
  required String endpoint,
  required String apiToken,
  required bool mcpRunning,
  bool allowManualApiTokenTemplate = false,
}) async {
  var selectedAgent = _McpAgent.codex;
  var useManualApiTokenTemplate = false;
  final overlay = const shad.DialogOverlayHandler().show<void>(
    context: context,
    alignment: Alignment.center,
    barrierDismissable: true,
    builder: (dialogContext) => StatefulBuilder(
      builder: (dialogContext, setDialogState) {
        final palette = AppPalette.of(dialogContext);
        final screenSize = MediaQuery.sizeOf(dialogContext);
        final dialogWidth = (screenSize.width - 80).clamp(220.0, 620.0);
        final contentMaxHeight = (screenSize.height - 140).clamp(120.0, 460.0);
        final templateToken = allowManualApiTokenTemplate && useManualApiTokenTemplate ? '<API_TOKEN>' : '';
        final displayToken = allowManualApiTokenTemplate ? templateToken : _maskedToken(apiToken);
        final clipboardToken = allowManualApiTokenTemplate ? templateToken : apiToken;
        final displaySnippet = _agentSnippet(selectedAgent, endpoint: endpoint, apiToken: displayToken);
        final clipboardSnippet = _agentSnippet(selectedAgent, endpoint: endpoint, apiToken: clipboardToken);

        return shad.AlertDialog(
          key: const ValueKey('mcp-agent-setup-dialog'),
          padding: const EdgeInsets.all(AppDesignTokens.space24),
          title: SizedBox(
            width: dialogWidth,
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Container(
                  width: 34,
                  height: 34,
                  decoration: BoxDecoration(
                    color: palette.brandSoft,
                    borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
                  ),
                  alignment: Alignment.center,
                  child: Icon(Icons.auto_awesome_outlined, size: 19, color: palette.brandProgress),
                ),
                const SizedBox(width: AppDesignTokens.space12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(dialogContext.l10n.connectMcpAgent),
                      const SizedBox(height: 4),
                      Text(
                        dialogContext.l10n.mcpAgentDialogDescription,
                        style: TextStyle(color: palette.textSecondary, fontSize: 12, fontWeight: FontWeight.w400),
                      ),
                    ],
                  ),
                ),
                AppTooltip(
                  message: dialogContext.l10n.close,
                  child: shad.IconButton.ghost(
                    key: const ValueKey('close-mcp-agent-setup'),
                    size: shad.ButtonSize.xSmall,
                    onPressed: () => shad.closeOverlay(dialogContext),
                    icon: const Icon(Icons.close, size: 18),
                  ),
                ),
              ],
            ),
          ),
          content: SizedBox(
            width: dialogWidth,
            child: ConstrainedBox(
              constraints: BoxConstraints(maxHeight: contentMaxHeight),
              child: SingleChildScrollView(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    if (!mcpRunning) ...[
                      Container(
                        key: const ValueKey('mcp-not-running-hint'),
                        padding: const EdgeInsets.symmetric(
                          horizontal: AppDesignTokens.space12,
                          vertical: AppDesignTokens.space8,
                        ),
                        decoration: BoxDecoration(
                          color: palette.brandSoft,
                          borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
                          border: Border.all(color: palette.brandTrack),
                        ),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Icon(Icons.info_outline, size: 16, color: palette.brandProgress),
                            const SizedBox(width: AppDesignTokens.space8),
                            Expanded(
                              child: Text(
                                dialogContext.l10n.mcpNotRunningHint,
                                style: TextStyle(color: palette.textSecondary, fontSize: 12, height: 1.4),
                              ),
                            ),
                          ],
                        ),
                      ),
                      const SizedBox(height: AppDesignTokens.space16),
                    ],
                    _McpAgentGrid(
                      selected: selectedAgent,
                      onSelected: (value) => setDialogState(() => selectedAgent = value),
                    ),
                    const SizedBox(height: AppDesignTokens.space16),
                    Row(
                      children: [
                        _McpAgentLogo(agent: selectedAgent, size: 20),
                        const SizedBox(width: AppDesignTokens.space8),
                        Expanded(
                          child: Row(
                            children: [
                              Flexible(
                                child: Text(
                                  '${selectedAgent.label} · MCP',
                                  style: TextStyle(
                                    color: palette.textPrimary,
                                    fontSize: 13,
                                    fontWeight: FontWeight.w600,
                                  ),
                                ),
                              ),
                              if (allowManualApiTokenTemplate) ...[
                                const SizedBox(width: 2),
                                _McpApiTokenTemplateHint(
                                  key: ValueKey('mcp-api-token-template-help-$useManualApiTokenTemplate'),
                                  onUseTemplate: () => setDialogState(() => useManualApiTokenTemplate = true),
                                ),
                              ],
                            ],
                          ),
                        ),
                        AppCopyIconButton(key: const ValueKey('copy-mcp-agent-snippet'), text: clipboardSnippet),
                      ],
                    ),
                    const SizedBox(height: AppDesignTokens.space8),
                    _McpSnippetBox(snippet: displaySnippet),
                  ],
                ),
              ),
            ),
          ),
        );
      },
    ),
  );
  await overlay.future;
}

class _McpApiTokenTemplateHint extends StatelessWidget {
  const _McpApiTokenTemplateHint({super.key, required this.onUseTemplate});

  final VoidCallback onUseTemplate;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return shad.HoverCard(
      wait: const Duration(milliseconds: 180),
      debounce: const Duration(milliseconds: 350),
      popoverAlignment: Alignment.topLeft,
      anchorAlignment: Alignment.bottomLeft,
      popoverOffset: const Offset(0, 6),
      hoverBuilder: (hoverContext) {
        final hoverPalette = AppPalette.of(hoverContext);
        return Container(
          key: const ValueKey('mcp-api-token-template-hint'),
          width: 288,
          padding: const EdgeInsets.all(AppDesignTokens.space12),
          decoration: BoxDecoration(
            color: hoverPalette.cardBg,
            borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
            border: Border.all(color: hoverPalette.border),
            boxShadow: [
              BoxShadow(
                color: const Color(0xFF000000).withValues(alpha: 0.16),
                blurRadius: 16,
                offset: const Offset(0, 6),
              ),
            ],
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                hoverContext.l10n.webMcpApiTokenHint,
                style: TextStyle(color: hoverPalette.textSecondary, fontSize: 12, height: 1.45),
              ),
              const SizedBox(height: 6),
              Semantics(
                link: true,
                button: true,
                label: hoverContext.l10n.useMcpApiTokenTemplate,
                child: MouseRegion(
                  cursor: SystemMouseCursors.click,
                  child: GestureDetector(
                    key: const ValueKey('use-mcp-api-token-template'),
                    behavior: HitTestBehavior.opaque,
                    onTap: onUseTemplate,
                    child: Text(
                      hoverContext.l10n.useMcpApiTokenTemplate,
                      style: TextStyle(
                        color: hoverPalette.brand,
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                        decoration: TextDecoration.underline,
                        decorationColor: hoverPalette.brand,
                      ),
                    ),
                  ),
                ),
              ),
            ],
          ),
        );
      },
      child: Semantics(
        label: context.l10n.webMcpApiTokenHint,
        child: SizedBox(
          key: const ValueKey('mcp-api-token-template-help'),
          width: 28,
          height: 28,
          child: Icon(Icons.help_outline, size: 16, color: palette.textMuted),
        ),
      ),
    );
  }
}

class _McpAgentGrid extends StatelessWidget {
  const _McpAgentGrid({required this.selected, required this.onSelected});

  final _McpAgent selected;
  final ValueChanged<_McpAgent> onSelected;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final columns = switch (constraints.maxWidth) {
          >= 520 => 8,
          >= 380 => 6,
          >= 260 => 5,
          _ => 4,
        };
        return GridView.builder(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: columns,
            crossAxisSpacing: AppDesignTokens.space8,
            mainAxisSpacing: AppDesignTokens.space8,
          ),
          itemCount: _McpAgent.values.length,
          itemBuilder: (context, index) {
            final agent = _McpAgent.values[index];
            return _McpAgentButton(
              key: ValueKey('mcp-agent-${_agentKey(agent)}'),
              agent: agent,
              selected: agent == selected,
              onPressed: () => onSelected(agent),
            );
          },
        );
      },
    );
  }
}

class _McpAgentButton extends StatefulWidget {
  const _McpAgentButton({super.key, required this.agent, required this.selected, required this.onPressed});

  final _McpAgent agent;
  final bool selected;
  final VoidCallback onPressed;

  @override
  State<_McpAgentButton> createState() => _McpAgentButtonState();
}

class _McpAgentButtonState extends State<_McpAgentButton> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final background = widget.selected
        ? palette.brandSoft
        : _hovered
        ? palette.cardHoverBg
        : palette.cardBg;
    return Semantics(
      button: true,
      selected: widget.selected,
      label: widget.agent.label,
      child: AppTooltip(
        message: widget.agent.label,
        child: MouseRegion(
          cursor: SystemMouseCursors.click,
          onEnter: (_) => setState(() => _hovered = true),
          onExit: (_) => setState(() => _hovered = false),
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: widget.onPressed,
            child: AnimatedContainer(
              duration: const Duration(milliseconds: 140),
              curve: Curves.easeOut,
              decoration: BoxDecoration(
                color: background,
                borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius + 2),
                border: Border.all(
                  color: widget.selected ? palette.brandProgress : palette.border,
                  width: widget.selected ? 1.5 : 1,
                ),
              ),
              alignment: Alignment.center,
              child: _McpAgentLogo(agent: widget.agent, size: 28),
            ),
          ),
        ),
      ),
    );
  }
}

class _McpAgentLogo extends StatelessWidget {
  const _McpAgentLogo({required this.agent, required this.size});

  final _McpAgent agent;
  final double size;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return SvgPicture.asset(
      agent.iconAsset,
      width: size,
      height: size,
      fit: BoxFit.contain,
      excludeFromSemantics: true,
      colorFilter: agent.usesThemeTint ? ColorFilter.mode(palette.textPrimary, BlendMode.srcIn) : null,
    );
  }
}

class _McpSnippetBox extends StatefulWidget {
  const _McpSnippetBox({required this.snippet});

  final String snippet;

  @override
  State<_McpSnippetBox> createState() => _McpSnippetBoxState();
}

class _McpSnippetBoxState extends State<_McpSnippetBox> {
  final _horizontalController = ScrollController();
  final _verticalController = ScrollController();

  @override
  void didUpdateWidget(covariant _McpSnippetBox oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.snippet == oldWidget.snippet) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_horizontalController.hasClients) _horizontalController.jumpTo(0);
      if (_verticalController.hasClients) _verticalController.jumpTo(0);
    });
  }

  @override
  void dispose() {
    _horizontalController.dispose();
    _verticalController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Container(
      key: const ValueKey('mcp-agent-snippet'),
      height: 176,
      width: double.infinity,
      padding: const EdgeInsets.all(14),
      clipBehavior: Clip.hardEdge,
      decoration: BoxDecoration(
        color: palette.surfaceSoft,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: palette.border),
      ),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final contentHeight = (constraints.maxHeight - AppDesignTokens.space12).clamp(0.0, double.infinity);
          return Scrollbar(
            key: const ValueKey('mcp-snippet-horizontal-scrollbar'),
            controller: _horizontalController,
            thumbVisibility: true,
            scrollbarOrientation: ScrollbarOrientation.bottom,
            notificationPredicate: (notification) => notification.metrics.axis == Axis.horizontal,
            child: Padding(
              padding: const EdgeInsets.only(bottom: AppDesignTokens.space12),
              child: SingleChildScrollView(
                controller: _horizontalController,
                scrollDirection: Axis.horizontal,
                child: SizedBox(
                  height: contentHeight,
                  child: Scrollbar(
                    key: const ValueKey('mcp-snippet-vertical-scrollbar'),
                    controller: _verticalController,
                    thumbVisibility: true,
                    scrollbarOrientation: ScrollbarOrientation.right,
                    notificationPredicate: (notification) => notification.metrics.axis == Axis.vertical,
                    child: SingleChildScrollView(
                      controller: _verticalController,
                      child: IntrinsicWidth(
                        child: ConstrainedBox(
                          constraints: BoxConstraints(minWidth: constraints.maxWidth),
                          child: SelectableText(
                            widget.snippet,
                            style: TextStyle(
                              color: palette.textPrimary,
                              fontSize: 12,
                              height: 1.55,
                              fontFamily: 'monospace',
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}

String _agentKey(_McpAgent agent) => switch (agent) {
  _McpAgent.cursor => 'cursor',
  _McpAgent.claudeCode => 'claude-code',
  _McpAgent.codex => 'codex',
  _McpAgent.githubCopilot => 'github-copilot',
  _McpAgent.windsurf => 'windsurf',
  _McpAgent.geminiCli => 'gemini-cli',
  _McpAgent.cline => 'cline',
  _McpAgent.openCode => 'opencode',
  _McpAgent.trae => 'trae',
  _McpAgent.qoder => 'qoder',
  _McpAgent.workBuddy => 'workbuddy',
  _McpAgent.zCode => 'zcode',
  _McpAgent.deepSeekHarness => 'deepseek-harness',
  _McpAgent.pi => 'pi',
};

String _agentSnippet(_McpAgent agent, {required String endpoint, required String apiToken}) {
  return switch (agent) {
    _McpAgent.cursor => _mcpServersSnippet(endpoint: endpoint, apiToken: apiToken),
    _McpAgent.codex => _codexSnippet(endpoint: endpoint, apiToken: apiToken),
    _McpAgent.claudeCode =>
      apiToken.isNotEmpty
          ? 'claude mcp add --transport http gopeed $endpoint \\\n'
                "  --header ${_shellSingleQuote('Authorization: Bearer $apiToken')}"
          : 'claude mcp add --transport http gopeed $endpoint',
    _McpAgent.githubCopilot => _mcpServersSnippet(
      rootKey: 'servers',
      type: 'http',
      endpoint: endpoint,
      apiToken: apiToken,
    ),
    _McpAgent.windsurf => _mcpServersSnippet(urlKey: 'serverUrl', endpoint: endpoint, apiToken: apiToken),
    _McpAgent.geminiCli => _mcpServersSnippet(urlKey: 'httpUrl', endpoint: endpoint, apiToken: apiToken),
    _McpAgent.cline => _mcpServersSnippet(
      type: 'streamableHttp',
      endpoint: endpoint,
      apiToken: apiToken,
      extra: const <String, Object>{'disabled': false, 'autoApprove': <Object>[]},
    ),
    _McpAgent.openCode => _openCodeSnippet(endpoint: endpoint, apiToken: apiToken),
    _McpAgent.trae => _mcpServersSnippet(endpoint: endpoint, apiToken: apiToken),
    _McpAgent.qoder => _mcpServersSnippet(type: 'http', endpoint: endpoint, apiToken: apiToken),
    _McpAgent.workBuddy => _mcpServersSnippet(
      type: 'streamableHttp',
      endpoint: endpoint,
      apiToken: apiToken,
      extra: const <String, Object>{'timeout': 30000},
    ),
    _McpAgent.zCode => _mcpServersSnippet(type: 'http', endpoint: endpoint, apiToken: apiToken),
    _McpAgent.deepSeekHarness => _deepSeekHarnessSnippet(endpoint: endpoint, apiToken: apiToken),
    _McpAgent.pi => _piSnippet(endpoint: endpoint, apiToken: apiToken),
  };
}

String _codexSnippet({required String endpoint, required String apiToken}) {
  if (apiToken.isEmpty) {
    return 'codex mcp add gopeed --url ${_shellSingleQuote(endpoint)}';
  }
  return 'export GOPEED_API_TOKEN=${_shellSingleQuote(apiToken)}\n\n'
      'codex mcp add gopeed --url ${_shellSingleQuote(endpoint)} \\\n'
      '  --bearer-token-env-var GOPEED_API_TOKEN';
}

String _mcpServersSnippet({
  String rootKey = 'mcpServers',
  String urlKey = 'url',
  String? type,
  required String endpoint,
  required String apiToken,
  Map<String, Object> extra = const <String, Object>{},
}) {
  final server = <String, Object>{};
  if (type != null) {
    server['type'] = type;
  }
  server[urlKey] = endpoint;
  if (apiToken.isNotEmpty) {
    server['headers'] = <String, String>{'Authorization': 'Bearer $apiToken'};
  }
  server.addAll(extra);
  return const JsonEncoder.withIndent('  ').convert(<String, Object>{
    rootKey: <String, Object>{'gopeed': server},
  });
}

String _openCodeSnippet({required String endpoint, required String apiToken}) {
  final server = <String, Object>{'type': 'remote', 'url': endpoint, 'oauth': false};
  if (apiToken.isNotEmpty) {
    server['headers'] = <String, String>{'Authorization': 'Bearer $apiToken'};
  }
  return const JsonEncoder.withIndent('  ').convert(<String, Object>{
    r'$schema': 'https://opencode.ai/config.json',
    'mcp': <String, Object>{
      'servers': <String, Object>{'gopeed': server},
    },
  });
}

String _deepSeekHarnessSnippet({required String endpoint, required String apiToken}) {
  final headers = apiToken.isNotEmpty
      ? '\n        headers:\n          Authorization: ${jsonEncode('Bearer $apiToken')}'
      : '\n        headers: {}';
  return '- id: mcp-gopeed\n'
      "  name: '@deepseek-ai/dsh-mcp-client'\n"
      '  config:\n'
      '    serverName: gopeed\n'
      '    transport: streamable-http\n'
      '    url: $endpoint$headers';
}

String _piSnippet({required String endpoint, required String apiToken}) {
  final config = _mcpServersSnippet(type: 'http', endpoint: endpoint, apiToken: apiToken);
  return 'pi install npm:pi-codemcp\n\n# ~/.pi/agent/mcp.json\n$config';
}

String _maskedToken(String token) {
  if (token.isEmpty) return '';
  if (token.length <= 8) return '••••••••';
  return '${token.substring(0, 4)}••••••••${token.substring(token.length - 4)}';
}

String _shellSingleQuote(String value) => "'${value.replaceAll("'", "'\"'\"'")}'";
