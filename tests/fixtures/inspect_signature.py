import inspect


def add(a: int, b: int) -> int:
    return a + b


def main() -> None:
    sig = str(inspect.signature(add))
    # CPython prints "(a: int, b: int) -> int"; gopy emits "(arg0, arg1)"
    # without type info. Common ground: the parameter count, surfaced
    # through commas in the signature string.
    print(sig.count(",") + 1)


if __name__ == "__main__":
    main()
