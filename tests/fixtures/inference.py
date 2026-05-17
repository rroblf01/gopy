class Box:
    def __init__(self, value: int) -> None:
        self.value = value

    def doubled(self) -> int:
        return self.value * 2


def make_box(v: int) -> Box:
    return Box(v)


def main() -> None:
    # No annotation on `b` — the transpiler infers Box from make_box's
    # declared return type and routes `b.doubled()` to the right method.
    b = make_box(7)
    print(b.doubled())
    # Chained: nested call result still has its type known.
    print(make_box(5).doubled())


if __name__ == "__main__":
    main()
