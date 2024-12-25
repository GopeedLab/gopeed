import 'package:flutter/material.dart';

import '../../icon/gopeed_icons.dart';

enum SortState { none, asc, desc }

class SortIconButton extends StatefulWidget {
  final double size;
  final Color? color;
  final Color? activeColor;
  final SortState initialState;
  final Function(SortState) onStateChanged;

  const SortIconButton({
    Key? key,
    this.size = 18.0,
    this.color,
    this.activeColor,
    this.initialState = SortState.none,
    required this.onStateChanged,
  }) : super(key: key);

  @override
  State<SortIconButton> createState() => _SortIconState();
}

class _SortIconState extends State<SortIconButton> {
  late SortState _currentState;

  @override
  void initState() {
    super.initState();
    _currentState = widget.initialState;
  }

  void _toggleState() {
    setState(() {
      switch (_currentState) {
        case SortState.none:
          _currentState = SortState.asc;
          break;
        case SortState.asc:
          _currentState = SortState.desc;
          break;
        case SortState.desc:
          _currentState = SortState.none;
          break;
      }
    });
    widget.onStateChanged(_currentState);
  }

  @override
  Widget build(BuildContext context) {
    return InkWell(
      customBorder: const CircleBorder(),
      onTap: _toggleState,
      child: SizedBox(
        width: widget.size,
        height: widget.size,
        child: _currentState == SortState.none
            ? Icon(
                Gopeed.sort,
                size: widget.size,
                color: widget.color,
              )
            : Column(
                children: [
                  ClipRect(
                    child: Align(
                      alignment: Alignment.topCenter,
                      heightFactor: 0.5,
                      child: Icon(
                        Gopeed.sort,
                        size: widget.size,
                        color: _currentState == SortState.asc
                            ? (widget.activeColor ??
                                Theme.of(context).colorScheme.primary)
                            : widget.color,
                      ),
                    ),
                  ),
                  ClipRect(
                    child: Align(
                      alignment: Alignment.bottomCenter,
                      heightFactor: 0.5,
                      child: Icon(
                        Gopeed.sort,
                        size: widget.size,
                        color: _currentState == SortState.desc
                            ? (widget.activeColor ??
                                Theme.of(context).colorScheme.primary)
                            : widget.color,
                      ),
                    ),
                  ),
                ],
              ),
      ),
    );
  }
}
