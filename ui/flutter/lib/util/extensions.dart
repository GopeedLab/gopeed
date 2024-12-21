extension ApplyExtension<T> on T {
  T apply(void Function(T) fn) {
    fn(this);
    return this;
  }
}

extension LetExtension<T> on T {
  R let<R>(R Function(T) fn) => fn(this);
}
