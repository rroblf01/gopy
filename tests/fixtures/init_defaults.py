class Box:
    value: int
    label: str

    def __init__(self, value: int = 0, label: str = "untitled"):
        self.value = value
        self.label = label


def main() -> None:
    a = Box()
    print(a.value)
    print(a.label)

    b = Box(42)
    print(b.value)
    print(b.label)

    c = Box(100, "foo")
    print(c.value)
    print(c.label)


if __name__ == "__main__":
    main()
