class Dyn:
    def __init__(self, label: str) -> None:
        self.label: str = label

    def __getattr__(self, name: str) -> str:
        return "missing:" + name


def main() -> None:
    d = Dyn("hi")
    # Declared field via getattr() goes through the switch first.
    print(getattr(d, "label"))
    # Unknown field falls through to __getattr__.
    print(getattr(d, "color"))
    print(getattr(d, "size"))


if __name__ == "__main__":
    main()
